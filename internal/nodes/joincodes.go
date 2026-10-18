package nodes

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// JoinCodeID names a Join Code once minted, since its text is never
// stored.
type JoinCodeID struct{ storage.UUID }

var (
	ErrJoinCodeNotFound = errors.New("no Join Code has this id")
	ErrJoinCodeClosed   = errors.New("the Join Code is used or expired")
	// ErrAddressUnusable is a Node address the link can't dial: a port 0
	// or an address given twice.
	ErrAddressUnusable = errors.New("the Node address holds an address the link can't dial")
)

// JoinCode is a minted code and when the Controller stops accepting it.
type JoinCode struct {
	ID        JoinCodeID
	Code      joincode.Code
	ExpiresAt time.Time
}

// OpenCode is a Join Code that may still pair a Node.
type OpenCode struct {
	ID                   JoinCodeID
	CreatedAt, ExpiresAt time.Time
}

// CodeState is Open until a Node pairs with the code or it expires.
type CodeState uint8

const (
	CodeOpen CodeState = iota
	CodeUsed
	CodeExpired
)

func (s CodeState) String() string { return [...]string{"open", "used", "expired"}[s] }

// JoinCodeState is a Join Code as the Add a Node dialog follows it.
type JoinCodeState struct {
	State CodeState
	// Node paired with the code, when it is Used.
	Node ID
	// Dialing is the Node address pasted for an open code, nil without one.
	Dialing *Dialing
}

// Dialing is the Node address the Controller dials for an open code.
type Dialing struct {
	Address joincode.NodeAddress
	Since   time.Time
	// Unreachable is whether dial_report has passed since without the
	// Node pairing.
	Unreachable bool
}

// MintJoinCode draws a secret and stores only its hash, expiring after the
// Join Code lifetime. The code names this Node's advertised addresses.
func (s *Service) MintJoinCode(ctx context.Context) (JoinCode, error) {
	addrs, ok := s.cfg.Link.Advertised()
	if !ok {
		return JoinCode{}, ErrNoLinkAddress
	}
	var secret [joincode.SecretSize]byte
	_, _ = rand.Read(secret[:])
	code, err := joincode.New(ed25519.PublicKey(s.cfg.Key[:]), secret, addrs.All())
	if err != nil {
		return JoinCode{}, err
	}
	hash := sha256.Sum256(secret[:])
	id := JoinCodeID{storage.NewUUID()}
	var expires storage.Time
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		lifetime, err := q.GetJoinCodeLifetime(ctx)
		if err != nil {
			return err
		}
		now := storage.Now()
		expires = storage.FromTime(now.Time().Add(JoinCodeLifetime.Value(lifetime)))
		return q.InsertJoinCode(ctx, db.InsertJoinCodeParams{ID: id.UUID, SecretHash: hash[:], CreatedAt: now, ExpiresAt: expires})
	})
	if err != nil {
		return JoinCode{}, fmt.Errorf("mint a Join Code: %w", err)
	}
	return JoinCode{ID: id, Code: code, ExpiresAt: expires.Time()}, nil
}

// JoinCodes is every open code, oldest first.
func (s *Service) JoinCodes(ctx context.Context) ([]OpenCode, error) {
	rows, err := db.New(s.cfg.Library).ListOpenJoinCodes(ctx, storage.Now())
	if err != nil {
		return nil, fmt.Errorf("list the open Join Codes: %w", err)
	}
	out := make([]OpenCode, len(rows))
	for i, r := range rows {
		out[i] = OpenCode{ID: JoinCodeID{r.ID}, CreatedAt: r.CreatedAt.Time(), ExpiresAt: r.ExpiresAt.Time()}
	}
	return out, nil
}

// JoinCodeState is the code id names now, or ErrJoinCodeNotFound once it
// was revoked or pruned.
func (s *Service) JoinCodeState(ctx context.Context, id JoinCodeID) (JoinCodeState, error) {
	row, err := db.New(s.cfg.Library).GetJoinCodeByID(ctx, id.UUID)
	if errors.Is(err, storage.ErrNoRows) {
		return JoinCodeState{}, ErrJoinCodeNotFound
	}
	if err != nil {
		return JoinCodeState{}, fmt.Errorf("read Join Code %s: %w", id, err)
	}
	now := storage.Now()
	st := JoinCodeState{State: stateOf(row, now), Node: row.UsedBy}
	if st.State != CodeOpen || row.DialAddress == nil {
		return st, nil
	}
	addr, err := joincode.ParseNodeAddress(*row.DialAddress)
	if err != nil {
		return JoinCodeState{}, fmt.Errorf("the dial address of Join Code %s: %w", id, err)
	}
	since := row.DialAt.Time()
	st.Dialing = &Dialing{Address: addr, Since: since, Unreachable: now.Time().Sub(since) >= s.cfg.Timings.DialReport}
	return st, nil
}

func stateOf(row db.NodesJoinCode, now storage.Time) CodeState {
	switch {
	case !row.UsedBy.IsZero():
		return CodeUsed
	case row.ExpiresAt.Compare(now) <= 0:
		return CodeExpired
	}
	return CodeOpen
}

// RevokeJoinCode deletes the code, so a later Pair with it is refused as
// unknown, and stops dialing for it.
func (s *Service) RevokeJoinCode(ctx context.Context, id JoinCodeID) error {
	var deleted int64
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		deleted, err = db.New(tx).DeleteJoinCode(ctx, id.UUID)
		return err
	})
	if err != nil {
		return fmt.Errorf("revoke Join Code %s: %w", id, err)
	}
	if deleted == 0 {
		return ErrJoinCodeNotFound
	}
	s.stopInvite(id)
	return nil
}

// PruneJoinCodes deletes every Join Code that expired more than one Join
// Code lifetime ago. The grace keeps Pair's answer to a late retry of a
// used code.
func (s *Service) PruneJoinCodes(ctx context.Context) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		lifetime, err := q.GetJoinCodeLifetime(ctx)
		if err != nil {
			return err
		}
		return q.DeleteJoinCodesExpiredBefore(ctx, storage.FromTime(time.Now().Add(-JoinCodeLifetime.Value(lifetime))))
	})
	if err != nil {
		return fmt.Errorf("prune Join Codes: %w", err)
	}
	return nil
}

// DialForCode stores addr on the open code id and has the link invite the
// Node there, which pairs over the Controller's session. It returns at
// once; the invite runs until the code expires or dial_invite passes,
// whichever is sooner. Pasting the same address again changes nothing.
func (s *Service) DialForCode(ctx context.Context, id JoinCodeID, addr joincode.NodeAddress) error {
	target, err := targetOf(addr)
	if err != nil {
		return err
	}
	var until time.Time
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetJoinCodeByID(ctx, id.UUID)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrJoinCodeNotFound
		}
		if err != nil {
			return err
		}
		now := storage.Now()
		if stateOf(row, now) != CodeOpen {
			return ErrJoinCodeClosed
		}
		until = now.Time().Add(s.cfg.Timings.DialInvite)
		if expires := row.ExpiresAt.Time(); expires.Before(until) {
			until = expires
		}
		text := addr.String()
		if row.DialAddress != nil && *row.DialAddress == text {
			return nil
		}
		return q.SetJoinCodeDial(ctx, db.SetJoinCodeDialParams{ID: id.UUID, DialAddress: &text, DialAt: now})
	})
	if err != nil {
		return fmt.Errorf("dial for Join Code %s: %w", id, err)
	}
	s.invite(id, target, until)
	return nil
}

// targetOf is addr as the link dials it.
func targetOf(addr joincode.NodeAddress) (link.Target, error) {
	key, err := link.KeyFromBytes(addr.Key())
	if err != nil {
		return link.Target{}, err
	}
	addrs, err := link.ParseAddresses(addr.Addresses())
	if err != nil || len(addrs.All()) != len(addr.Addresses()) {
		return link.Target{}, ErrAddressUnusable
	}
	return link.Target{Key: key, Addresses: addrs}, nil
}

// inviting is the one invite running for a code.
type inviting struct {
	target link.Target
	stop   context.CancelFunc
}

// invite runs Link.Invite for code id until until, replacing an invite to
// another target and leaving one to the same target running.
func (s *Service) invite(id JoinCodeID, t link.Target, until time.Time) {
	ctx, stop := context.WithDeadline(context.Background(), until)
	run := &inviting{target: t, stop: stop}
	s.mu.Lock()
	if old := s.invites[id]; old != nil {
		if old.target == t {
			s.mu.Unlock()
			stop()
			return
		}
		old.stop()
	}
	s.invites[id] = run
	s.mu.Unlock()
	go func() {
		defer stop()
		err := s.cfg.Link.Invite(ctx, t)
		s.mu.Lock()
		if s.invites[id] == run {
			delete(s.invites, id)
		}
		s.mu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) {
			s.cfg.Log.Warn("the Node at a pasted Node address did not pair", "joinCode", id.String(), "node", t.Key.Fingerprint(), "err", err.Error())
		}
	}()
}

func (s *Service) stopInvite(id JoinCodeID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run := s.invites[id]; run != nil {
		run.stop()
		delete(s.invites, id)
	}
}
