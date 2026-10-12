// Package identity owns Accounts, sessions, the API key, the session
// lifetime setting, PIN hashing, the principal types and authentication.
// Only identity builds a principal.
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("identity", schemaFiles) }

type (
	AccountID = db.AccountID
	SessionID = db.SessionID
)

// SessionsChanged reaches an Account's streams when one of its sessions
// starts or ends.
type SessionsChanged struct {
	Account string `json:"account" format:"uuid"`
}

var SessionsChangedEvent = live.AccountEvent[SessionsChanged]("identity.sessions-changed")

// SessionsPublisher is SessionsChangedEvent bound to the hub.
type SessionsPublisher interface {
	Publish(ctx context.Context, account storage.UUID, v SessionsChanged)
}

type SessionEnder interface {
	EndSession(session storage.UUID)
}

const (
	CookieName = "slipstream_session"
	// slideDivisor makes a session slide once per lifetime/720 of use: an
	// hour at 30 days, proportionally less when a Scenario seeds a short
	// lifetime.
	slideDivisor = 720
	tokenBytes   = 32
	apiKeyPrefix = "ssk_"
)

type Service struct {
	lib      *storage.DB[file]
	pins     *pins
	dummy    string
	lifetime time.Duration
	sessions SessionsPublisher
	ends     SessionEnder
}

// New reads the session lifetime once and builds the dummy hash an unknown
// email verifies against.
func New(ctx context.Context, lib *storage.DB[file], pepper []byte, sessions SessionsPublisher, ends SessionEnder) (*Service, error) {
	days, err := db.New(lib).GetSessionLifetimeDays(ctx)
	if err != nil {
		return nil, err
	}
	s := &Service{
		lib:      lib,
		pins:     newPINs(pepper),
		lifetime: time.Duration(days) * 24 * time.Hour,
		sessions: sessions,
		ends:     ends,
	}
	if s.dummy, err = s.pins.hash(ctx, PIN{digits: "0000"}); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) HasAdmin(ctx context.Context) (bool, error) {
	return db.New(s.lib).HasAdmin(ctx)
}

// Claimed is the first Admin's Profile, the API key shown this once, and
// the cookie of its new session.
type Claimed struct {
	Profile Profile
	APIKey  string
	Cookie  *http.Cookie
}

// ClaimController creates the first Admin, the API key and a session in
// one Write. It returns ErrAlreadyClaimed when an Admin exists.
func (s *Service) ClaimController(ctx context.Context, c Claim, email, displayName string, pin PIN) (Claimed, error) {
	hash, err := s.pins.hash(ctx, pin)
	if err != nil {
		return Claimed{}, err
	}
	profile := Profile{ID: AccountID{UUID: storage.NewUUID()}, Email: normalizeEmail(email), DisplayName: displayName, Admin: true}
	apiKey := apiKeyPrefix + newToken()
	token := newToken()
	err = s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		admin, err := q.HasAdmin(ctx)
		if err != nil {
			return err
		}
		if admin {
			return ErrAlreadyClaimed
		}
		now := storage.Now()
		err = q.InsertAccount(ctx, db.InsertAccountParams{
			ID:          profile.ID,
			Email:       profile.Email,
			DisplayName: profile.DisplayName,
			Admin:       true,
			PinHash:     hash,
			CreatedAt:   now,
		})
		if err != nil {
			return err
		}
		if err := q.InsertAPIKey(ctx, db.InsertAPIKeyParams{TokenHash: digest(apiKey), CreatedAt: now}); err != nil {
			return err
		}
		return insertSession(ctx, q, profile.ID, token, c.address, now)
	})
	if err != nil {
		return Claimed{}, err
	}
	s.sessionsChanged(ctx, profile.ID)
	return Claimed{Profile: profile, APIKey: apiKey, Cookie: s.cookie(token, c.tls)}, nil
}

// SignIn verifies email and pin and starts a session. An unknown email
// verifies against the dummy hash, so both failures cost one argon2id and
// return ErrPINWrong.
func (s *Service) SignIn(ctx context.Context, a Anyone, email string, pin PIN) (Profile, *http.Cookie, error) {
	row, err := db.New(s.lib).GetAccountByEmail(ctx, normalizeEmail(email))
	known := err == nil
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return Profile{}, nil, err
	}
	hash := s.dummy
	if known {
		hash = row.PinHash
	}
	ok, err := s.pins.verify(ctx, pin, hash)
	if err != nil {
		return Profile{}, nil, err
	}
	if !ok || !known {
		return Profile{}, nil, ErrPINWrong
	}
	token := newToken()
	err = s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		now := storage.Now()
		idle := storage.FromTime(now.Time().Add(-s.lifetime))
		if err := q.DeleteIdleSessions(ctx, db.DeleteIdleSessionsParams{AccountID: row.ID, SeenBefore: idle}); err != nil {
			return err
		}
		return insertSession(ctx, q, row.ID, token, a.address, now)
	})
	if err != nil {
		return Profile{}, nil, err
	}
	s.sessionsChanged(ctx, row.ID)
	profile := Profile{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Admin: row.Admin}
	return profile, s.cookie(token, a.tls), nil
}

// SignOut ends a's session and its open streams and returns the cookie
// that clears it.
func (s *Service) SignOut(ctx context.Context, a Account) (*http.Cookie, error) {
	err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).DeleteSession(ctx, a.session)
	})
	if err != nil {
		return nil, err
	}
	s.ends.EndSession(a.session.UUID)
	s.sessionsChanged(ctx, a.profile.ID)
	return clearCookie(a.tls), nil
}

// Request is what authentication reads from an HTTP request.
type Request struct {
	// APIKey is the X-Api-Key header, "" when absent.
	APIKey string
	// Session is the session cookie's value, "" when absent.
	Session string
	Address netip.Addr
	TLS     bool
}

// Authenticate resolves r's Caller. A request with an API key
// authenticates by it alone. Otherwise the session cookie is looked up;
// the cookie returned, when not nil, re-sends a session that slid or
// clears one that is gone or expired.
func (s *Service) Authenticate(ctx context.Context, r Request) (Caller, *http.Cookie, error) {
	c := Caller{client: client{address: r.Address, tls: r.TLS}}
	if r.APIKey != "" {
		stored, err := db.New(s.lib).GetAPIKeyHash(ctx)
		if err != nil && !errors.Is(err, storage.ErrNoRows) {
			return Caller{}, nil, err
		}
		c.kind = wrongAPIKey
		if subtle.ConstantTimeCompare(digest(r.APIKey), stored) == 1 {
			c.kind = apiKeyCaller
		}
		return c, nil, nil
	}
	if r.Session == "" {
		return c, nil, nil
	}
	row, err := db.New(s.lib).GetSession(ctx, digest(r.Session))
	if errors.Is(err, storage.ErrNoRows) {
		return c, clearCookie(r.TLS), nil
	}
	if err != nil {
		return Caller{}, nil, err
	}
	u, err := s.use(ctx, row.SessionID, row.AccountID, row.LastSeenAt)
	if err != nil {
		return Caller{}, nil, err
	}
	if u == expired {
		return c, clearCookie(r.TLS), nil
	}
	c.kind = sessionCaller
	c.account = Account{
		client:  c.client,
		profile: Profile{ID: row.AccountID, Email: row.Email, DisplayName: row.DisplayName, Admin: row.Admin},
		session: row.SessionID,
	}
	if u == slid {
		return c, s.cookie(r.Session, r.TLS), nil
	}
	return c, nil, nil
}

// Renew counts a's open stream as use of its session, under the same rule
// as a request. A session that expired or is gone returns ErrSignedOut.
func (s *Service) Renew(ctx context.Context, a Account) error {
	seen, err := db.New(s.lib).GetSessionLastSeen(ctx, a.session)
	if errors.Is(err, storage.ErrNoRows) {
		return ErrSignedOut
	}
	if err != nil {
		return err
	}
	u, err := s.use(ctx, a.session, a.profile.ID, seen)
	if err != nil {
		return err
	}
	if u == expired {
		return ErrSignedOut
	}
	return nil
}

type sessionUse uint8

const (
	current sessionUse = iota
	slid
	expired
)

// use is one use of a session last seen at seen. An expired session is
// deleted and its streams end. One idle longer than lifetime/720 slides.
func (s *Service) use(ctx context.Context, session SessionID, account AccountID, seen storage.Time) (sessionUse, error) {
	now := storage.Now()
	idle := now.Time().Sub(seen.Time())
	switch {
	case idle > s.lifetime:
		err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return db.New(tx).DeleteSession(ctx, session)
		})
		if err != nil {
			return 0, err
		}
		s.ends.EndSession(session.UUID)
		s.sessionsChanged(ctx, account)
		return expired, nil
	case idle > s.lifetime/slideDivisor:
		err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return db.New(tx).TouchSession(ctx, db.TouchSessionParams{LastSeenAt: now, ID: session})
		})
		return slid, err
	}
	return current, nil
}

func (s *Service) sessionsChanged(ctx context.Context, account AccountID) {
	s.sessions.Publish(ctx, account.UUID, SessionsChanged{Account: account.String()})
}

// cookie builds the session cookie: slipstream_session=<token>; Path=/;
// Max-Age=<lifetime>; HttpOnly; SameSite=Lax, plus Secure over TLS.
func (s *Service) cookie(token string, secure bool) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // Secure only over TLS: a LAN Node is reached by plain HTTP
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(s.lifetime / time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func clearCookie(secure bool) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // Secure only over TLS: a LAN Node is reached by plain HTTP
		Name:     CookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func insertSession(ctx context.Context, q *db.Queries, account AccountID, token string, address netip.Addr, now storage.Time) error {
	return q.InsertSession(ctx, db.InsertSessionParams{
		ID:         SessionID{UUID: storage.NewUUID()},
		TokenHash:  digest(token),
		AccountID:  account,
		Address:    address.String(),
		CreatedAt:  now,
		LastSeenAt: now,
	})
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func newToken() string {
	b := make([]byte, tokenBytes)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// digest is how a session token and the API key are stored: only as their
// SHA-256.
func digest(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
