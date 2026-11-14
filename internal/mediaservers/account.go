package mediaservers

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

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
		signedIn: row.SignedInAt.Time(), renewed: row.RenewedAt.Time(), refused: !row.RefusedAt.IsZero()}
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
	name := cmpOr(user.Username, user.Title)
	err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).SignInToPlex(ctx, db.SignInToPlexParams{
			JwtSealed: m.cfg.Sealer.Seal([]byte(token.JWT), jwtColumn.Single()), JwtExpiresAt: storage.FromTime(token.Expires),
			AccountName: &name, PlexPass: &user.PlexPass, SignedInAt: storage.Now(),
		})
	})
	if err != nil {
		return "", err
	}
	m.cfg.Log.Info("signed in to Plex", "account", name)
	m.senders.wakeAll()
	if err := m.reread(ctx); err != nil {
		m.cfg.Log.Warn("read the Plex servers again after a sign-in", "err", err.Error())
	}
	return "signed_in", nil
}

func (s service) PlexAccount(ctx context.Context) (AccountView, error) {
	a, err := s.m.signedIn(ctx)
	if err != nil {
		return AccountView{}, err
	}
	return AccountView{Name: a.name, PlexPass: a.plexPass, SignedInAt: a.signedIn, ExpiresAt: a.expires, RenewedAt: timeOrNil(storage.FromTime(a.renewed)), Refused: a.refused}, nil
}

// SignOutOfPlex forgets the account's sign-in and removes every Plex
// server in the same Write. The device key stays for the next sign-in.
func (s service) SignOutOfPlex(ctx context.Context) error {
	var removed []db.MediaserversServer
	err := s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		var err error
		if removed, err = q.ListPlexServers(ctx); err != nil {
			return err
		}
		if err := q.DeletePlexServers(ctx); err != nil {
			return err
		}
		return q.SignOutOfPlex(ctx)
	})
	if err != nil {
		return err
	}
	ids := make([]link.ServerID, len(removed))
	reaching := make([]role.NodeID, len(removed))
	for i, r := range removed {
		ids[i], reaching[i] = r.ID, r.ReachingNodeID
	}
	s.m.cfg.Touch(reaching...)
	s.m.changed(ctx, ids...)
	return nil
}
