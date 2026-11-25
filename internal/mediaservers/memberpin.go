package mediaservers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/storage"
)

// memberPins are the Members' Plex sign-ins, one per Account, polled by
// the Controller in the background from Run. A restart drops them.
type memberPins struct {
	mu        sync.Mutex
	byAccount map[identity.AccountID]*memberPin
	wake      chan struct{}
}

type memberPin struct {
	account identity.AccountID
	tv      *plextv.Member
	view    MemberPin
	every   time.Duration
	next    time.Time
	state   MemberPinState
	ended   time.Time
}

func newMemberPins() memberPins {
	return memberPins{byAccount: map[identity.AccountID]*memberPin{}, wake: make(chan struct{}, 1)}
}

// live is account's PIN while it waits.
func (ps *memberPins) live(account identity.AccountID) (MemberPin, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	p, ok := ps.byAccount[account]
	if !ok || p.state.State != PinWaiting {
		return MemberPin{}, false
	}
	return p.view, true
}

func (ps *memberPins) add(p *memberPin) {
	ps.mu.Lock()
	ps.byAccount[p.account] = p
	ps.mu.Unlock()
	select {
	case ps.wake <- struct{}{}:
	default:
	}
}

// due is the PINs to poll now, and when the next one is due, zero for
// none. It ends each PIN past its life, and forgets each ended pinKept
// ago.
func (ps *memberPins) due(now time.Time) ([]*memberPin, time.Time) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	var due []*memberPin
	var next time.Time
	for account, p := range ps.byAccount {
		switch {
		case p.state.State != PinWaiting:
			if now.After(p.ended.Add(pinKept)) {
				delete(ps.byAccount, account)
			}
		case !now.Before(p.view.ExpiresAt):
			p.end(MemberPinState{State: PinExpired}, now)
		case !now.Before(p.next):
			due = append(due, p)
		default:
			next = earlier(next, earlier(p.next, p.view.ExpiresAt))
		}
	}
	return due, next
}

func (ps *memberPins) update(f func()) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	f()
}

func (p *memberPin) end(s MemberPinState, now time.Time) { p.state, p.ended = s, now }

// pollMembers polls every waiting Member PIN at Timings.PinPoll, doubling
// on 429, until it is approved or expires, and signs the Member in.
func (m *MediaServers) pollMembers(ctx context.Context) {
	for {
		due, next := m.memberPins.due(time.Now())
		for _, p := range due {
			m.pollMember(ctx, p)
		}
		if len(due) > 0 {
			continue
		}
		if !m.memberPins.wait(ctx, next) {
			return
		}
	}
}

// wait waits for next, zero for none, or a new PIN, false once ctx ends.
func (ps *memberPins) wait(ctx context.Context, next time.Time) bool {
	var at <-chan time.Time
	if !next.IsZero() {
		timer := time.NewTimer(time.Until(next))
		defer timer.Stop()
		at = timer.C
	}
	select {
	case <-ctx.Done():
		return false
	case <-ps.wake:
	case <-at:
	}
	return true
}

func (m *MediaServers) pollMember(ctx context.Context, p *memberPin) {
	ps := &m.memberPins
	token, approved, err := p.tv.PollLegacyPin(ctx, p.view.ID)
	now := time.Now()
	if wait, limited := plextv.RateLimited(err); limited {
		ps.update(func() { p.every *= 2; p.next = now.Add(max(p.every, wait)) })
		return
	}
	switch {
	case plextv.Expired(err):
		ps.update(func() { p.end(MemberPinState{State: PinExpired}, now) })
		return
	case err != nil:
		m.cfg.Log.Warn("poll a Member's Plex sign-in", "err", err.Error())
	case approved:
		var state MemberPinState
		if state, err = m.memberApproved(ctx, p, token); err == nil {
			ps.update(func() { p.end(state, time.Now()) })
			return
		}
		m.cfg.Log.Warn("record a Member's Plex sign-in", "err", err.Error())
	}
	ps.update(func() { p.next = now.Add(p.every) })
}

// memberApproved reads the Plex account behind token and signs the Member
// in with it.
func (m *MediaServers) memberApproved(ctx context.Context, p *memberPin, token plextv.MemberToken) (MemberPinState, error) {
	u, err := p.tv.User(ctx, token)
	if err != nil {
		return MemberPinState{}, fmt.Errorf("%w: %w", ErrPlexTVUnreached, err)
	}
	return m.memberSignedIn(ctx, p.account, p.tv, u, token)
}

// memberTV is plex.tv as account's device, on the client identifier made
// at their first PIN.
func (m *MediaServers) memberTV(ctx context.Context, account identity.AccountID) (*plextv.Member, error) {
	var clientID string
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		p, err := q.GetMemberPlex(ctx, account)
		if err == nil {
			clientID = p.ClientIdentifier
			return nil
		}
		if !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		clientID = storage.NewUUID().String()
		return q.InsertMemberDevice(ctx, db.InsertMemberDeviceParams{AccountID: account, ClientIdentifier: clientID})
	})
	return m.memberDevice(clientID), err
}

func (m *MediaServers) memberDevice(clientID string) *plextv.Member {
	return plextv.NewMember(m.cfg.PlexTV, plexHeaders(clientID), m.cfg.HTTP)
}

// welcomeURL is the Watch step at the identity Address, where plex.tv
// returns the Member.
func (m *MediaServers) welcomeURL(ctx context.Context) (string, error) {
	var a identity.Address
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		a, err = identity.AddressIn(ctx, r)
		return err
	})
	return a.URL().JoinPath("welcome", "watch").String(), err
}

// StartMemberPin starts the Member's Plex sign-in, or answers the one
// still waiting.
func (s service) StartMemberPin(ctx context.Context, me identity.Account) (MemberPin, error) {
	account := me.Profile().ID
	release, err := s.m.turns.take(ctx, accountTurn(account))
	if err != nil {
		return MemberPin{}, err
	}
	defer release()
	if p, ok := s.m.memberPins.live(account); ok {
		return p, nil
	}
	if _, err := s.m.signedIn(ctx); err != nil {
		return MemberPin{}, err
	}
	tv, err := s.m.memberTV(ctx, account)
	if err != nil {
		return MemberPin{}, err
	}
	pin, err := tv.CreateLegacyPin(ctx)
	if err != nil {
		return MemberPin{}, fmt.Errorf("%w: %w", ErrPlexTVUnreached, err)
	}
	forward, err := s.m.welcomeURL(ctx)
	if err != nil {
		return MemberPin{}, err
	}
	view := MemberPin{ID: pin.ID, AuthURL: tv.AuthURL(pin.Code, forward), ExpiresAt: pin.ExpiresAt}
	s.m.memberPins.add(&memberPin{
		account: account, tv: tv, view: view, every: max(s.m.cfg.Timings.PinPoll, time.Second), next: time.Now(),
		state: MemberPinState{State: PinWaiting},
	})
	return view, nil
}

// MemberPin is where the Member's PIN id stands. Another Account's PIN
// answers expired.
func (s service) MemberPin(_ context.Context, me identity.Account, id int64) (MemberPinState, error) {
	ps := &s.m.memberPins
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for _, p := range ps.byAccount {
		if p.view.ID != id {
			continue
		}
		if p.account != me.Profile().ID {
			return MemberPinState{State: PinExpired}, nil
		}
		return p.state, nil
	}
	return MemberPinState{}, ErrPlexPinNotFound
}
