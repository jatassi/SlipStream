package mediaservers

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/providers/plex"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	keyColumn = secret.NewColumn("mediaservers_plex_account.key_sealed")
	jwtColumn = secret.NewColumn("mediaservers_plex_account.jwt_sealed")
)

// pinKept is how long an expired sign-in still answers expired, for a page
// left open past the PIN's life.
const pinKept = time.Hour

// account is the one Plex account, as the Controller holds it.
type account struct {
	clientID string
	key      ed25519.PrivateKey
	// jwt is "" while signed out.
	jwt      string
	expires  time.Time
	name     string
	plexPass bool
	signedIn time.Time
	renewed  time.Time
	refused  bool
	// plexID is plex.tv's id of the account last signed in, kept across
	// sign-outs. nil before the first.
	plexID *plextv.AccountID
}

func (a account) signedInNow() bool { return a.jwt != "" }

// readAccount is the account, false when no sign-in ever started.
func (m *MediaServers) readAccount(ctx context.Context, q *db.Queries) (account, bool, error) {
	row, err := q.GetPlexAccount(ctx)
	if errors.Is(err, storage.ErrNoRows) {
		return account{}, false, nil
	}
	if err != nil {
		return account{}, false, err
	}
	key, err := m.cfg.Sealer.Open(row.KeySealed, keyColumn.Single())
	if err != nil {
		return account{}, false, fmt.Errorf("open the Plex device key: %w", err)
	}
	a := account{clientID: row.ClientIdentifier, key: ed25519.PrivateKey(key), expires: row.JwtExpiresAt.Time(),
		signedIn: row.SignedInAt.Time(), renewed: row.RenewedAt.Time(), refused: !row.RefusedAt.IsZero(), plexID: row.PlexID}
	if row.JwtSealed.Set() {
		jwt, err := m.cfg.Sealer.Open(row.JwtSealed, jwtColumn.Single())
		if err != nil {
			return account{}, false, fmt.Errorf("open the Plex sign-in: %w", err)
		}
		a.jwt, a.name, a.plexPass = string(jwt), deref(row.AccountName), *row.PlexPass
	}
	return a, true, nil
}

func (m *MediaServers) account(ctx context.Context) (account, bool, error) {
	var (
		a  account
		ok bool
	)
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		a, ok, err = m.readAccount(ctx, db.New(r))
		return err
	})
	return a, ok, err
}

// signedIn is the account while it is signed in, else ErrPlexNotSignedIn.
func (m *MediaServers) signedIn(ctx context.Context) (account, error) {
	a, ok, err := m.account(ctx)
	if err == nil && (!ok || !a.signedInNow()) {
		err = ErrPlexNotSignedIn
	}
	return a, err
}

// device is the account's device key and client identifier, made once and
// kept across sign-outs.
func (m *MediaServers) device(ctx context.Context) (account, error) {
	var a account
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		var ok bool
		var err error
		if a, ok, err = m.readAccount(ctx, q); err != nil || ok {
			return err
		}
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return fmt.Errorf("make the Plex device key: %w", err)
		}
		a = account{clientID: storage.NewUUID().String(), key: key}
		return q.InsertPlexDevice(ctx, db.InsertPlexDeviceParams{ClientIdentifier: a.clientID, KeySealed: m.cfg.Sealer.Seal(key, keyColumn.Single())})
	})
	return a, err
}

// plexTV is plex.tv for the account's device.
func (m *MediaServers) plexTV(a account) *plextv.Client {
	return plextv.New(m.cfg.PlexTV, plexHeaders(a.clientID), a.key, m.cfg.HTTP)
}

func plexHeaders(clientID string) plex.Headers {
	return plex.Headers{ClientID: clientID, Product: "SlipStream", DeviceName: "SlipStream"}
}

// fromPlexTV is a plex.tv error as the Service returns it: a refusal is
// the account's sign-in refused, anything else plex.tv not responding.
func fromPlexTV(err error) error {
	if plextv.Refused(err) {
		return &SignInRefusedError{Server: "Plex", With: RefusedPlex}
	}
	return fmt.Errorf("%w: %w", ErrPlexTVUnreached, err)
}

// pins are the sign-ins started since the Controller started. A PIN is
// polled when its page asks for its state, at most every Timings.PinPoll,
// doubling on 429.
type pins struct {
	mu sync.Mutex
	m  map[int64]*pinWait
}

type pinWait struct {
	expires time.Time
	next    time.Time
	every   time.Duration
	state   string
	// held is the sign-in of a different Plex account, unsaved until
	// ReplacePlexAccount confirms it.
	held *heldSignIn
}

type heldSignIn struct {
	token plextv.Token
	user  plextv.User
}

func (s service) StartPlexPin(ctx context.Context, forward string) (PlexPin, error) {
	a, err := s.m.device(ctx)
	if err != nil {
		return PlexPin{}, err
	}
	c := s.m.plexTV(a)
	p, err := c.CreatePin(ctx)
	if err != nil {
		return PlexPin{}, fromPlexTV(err)
	}
	ps := &s.m.pins
	ps.mu.Lock()
	defer ps.mu.Unlock()
	now := time.Now()
	if ps.m == nil {
		ps.m = map[int64]*pinWait{}
	}
	for id, w := range ps.m {
		if now.After(w.expires.Add(pinKept)) {
			delete(ps.m, id)
		}
	}
	ps.m[p.ID] = &pinWait{expires: p.ExpiresAt, next: now, every: max(s.m.cfg.Timings.PinPoll, time.Second), state: "waiting"}
	return PlexPin{ID: p.ID, AuthURL: c.AuthURL(p.Code, forward), ExpiresAt: p.ExpiresAt}, nil
}

func (s service) PlexPin(ctx context.Context, id int64) (PlexPinState, error) {
	ps := &s.m.pins
	ps.mu.Lock()
	defer ps.mu.Unlock()
	w, ok := ps.m[id]
	if !ok {
		return PlexPinState{}, ErrPlexPinNotFound
	}
	now := time.Now()
	switch {
	case w.state != "waiting" || now.Before(w.next):
	case !now.Before(w.expires):
		w.state = "expired"
	default:
		state, err := s.m.poll(ctx, id, w, now)
		if err != nil {
			return PlexPinState{}, err
		}
		w.state = state
	}
	return PlexPinState{State: w.state}, nil
}

// poll asks plex.tv for the PIN once, and stores the account once the
// Admin approved it.
func (m *MediaServers) poll(ctx context.Context, id int64, w *pinWait, now time.Time) (string, error) {
	a, _, err := m.account(ctx)
	if err != nil {
		return "", err
	}
	c := m.plexTV(a)
	token, approved, err := c.PollPin(ctx, id)
	if wait, limited := plextv.RateLimited(err); limited {
		w.every *= 2
		w.next = now.Add(max(w.every, wait))
		return "waiting", nil
	}
	switch {
	case plextv.Expired(err):
		return "expired", nil
	case err != nil:
		return "", fromPlexTV(err)
	case !approved:
		w.next = now.Add(w.every)
		return "waiting", nil
	}
	user, err := c.User(ctx, token.JWT)
	if err != nil {
		return "", fromPlexTV(err)
	}
	if a.plexID != nil && *a.plexID != user.ID {
		w.held = &heldSignIn{token: token, user: user}
		return "other_account", nil
	}
	if err := m.signInToPlex(ctx, token, user, false); err != nil {
		return "", err
	}
	return "signed_in", nil
}

// signInToPlex stores the account's sign-in, after removing every Plex
// server and Member token of the account it replaces in the same Write.
func (m *MediaServers) signInToPlex(ctx context.Context, token plextv.Token, user plextv.User, replace bool) error {
	name := cmpOr(user.Username, user.Title)
	var out signedOut
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if replace {
			var err error
			if out, err = m.signOutIn(ctx, q); err != nil {
				return err
			}
		}
		return q.SignInToPlex(ctx, db.SignInToPlexParams{
			JwtSealed: m.cfg.Sealer.Seal([]byte(token.JWT), jwtColumn.Single()), JwtExpiresAt: storage.FromTime(token.Expires),
			AccountName: &name, PlexPass: &user.PlexPass, PlexID: &user.ID, SignedInAt: storage.Now(),
		})
	})
	if err != nil {
		return err
	}
	if replace {
		m.afterSignOut(ctx, out)
	}
	m.cfg.Log.Info("signed in to Plex", "account", name)
	m.senders.wakeAll()
	if err := m.reread(ctx); err != nil {
		m.cfg.Log.Warn("read the Plex servers again after a sign-in", "err", err.Error())
	}
	return nil
}

// ReplacePlexAccount confirms the sign-in of pin, held as other_account:
// the Plex servers and Members' access to them go, in the Write that signs
// SlipStream in to the new account.
func (s service) ReplacePlexAccount(ctx context.Context, pin int64) (AccountView, error) {
	ps := &s.m.pins
	ps.mu.Lock()
	defer ps.mu.Unlock()
	w, ok := ps.m[pin]
	if !ok || w.held == nil {
		return AccountView{}, ErrPlexPinNotFound
	}
	if err := s.m.signInToPlex(ctx, w.held.token, w.held.user, true); err != nil {
		return AccountView{}, err
	}
	w.held, w.state = nil, "signed_in"
	return s.PlexAccount(ctx)
}

func (s service) PlexAccount(ctx context.Context) (AccountView, error) {
	a, err := s.m.signedIn(ctx)
	if err != nil {
		return AccountView{}, err
	}
	return AccountView{Name: a.name, PlexPass: a.plexPass, SignedInAt: a.signedIn, ExpiresAt: a.expires, RenewedAt: timeOrNil(storage.FromTime(a.renewed)), Refused: a.refused}, nil
}

// SignOutOfPlex forgets the account's sign-in, removes every Plex server
// and clears every Member's Plex token in the same Write. The device key
// and plex.tv id stay for the next sign-in.
func (s service) SignOutOfPlex(ctx context.Context) error {
	var out signedOut
	err := s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		var err error
		if out, err = s.m.signOutIn(ctx, q); err != nil {
			return err
		}
		return q.SignOutOfPlex(ctx)
	})
	if err != nil {
		return err
	}
	s.m.afterSignOut(ctx, out)
	return nil
}

// signedOut is what a sign-out's Write removed.
type signedOut struct {
	servers []db.MediaserversServer
	tokens  []clearedToken
}

// signOutIn deletes every Plex server, whose access rows go with them, and
// clears every Member's Plex token.
func (m *MediaServers) signOutIn(ctx context.Context, q *db.Queries) (signedOut, error) {
	var out signedOut
	var err error
	if out.servers, err = q.ListPlexServers(ctx); err != nil {
		return out, err
	}
	if err := q.DeletePlexServers(ctx); err != nil {
		return out, err
	}
	out.tokens, err = m.memberTokens(ctx, q)
	return out, err
}

func (m *MediaServers) afterSignOut(ctx context.Context, out signedOut) {
	m.signOut(ctx, out.tokens)
	ids := make([]link.ServerID, len(out.servers))
	reaching := make([]role.NodeID, len(out.servers))
	for i, r := range out.servers {
		ids[i], reaching[i] = r.ID, r.ReachingNodeID
	}
	m.cfg.Touch(reaching...)
	m.changed(ctx, ids...)
	if len(out.servers) > 0 {
		m.accessChanged(ctx, identity.AccountID{})
	}
}
