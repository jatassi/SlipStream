// Package identity owns Accounts, sessions, the API key, the sign-in
// settings, PIN hashing and its limits, the principal types and
// authentication.
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
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/identity/passkey"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("identity", schemaFiles) }

type (
	AccountID = db.AccountID
	SessionID = db.SessionID
	PasskeyID = db.PasskeyID
)

// SessionsChanged reaches an Account's streams when one of its sessions
// starts or ends.
type SessionsChanged struct {
	Account string `json:"account" format:"uuid"`
}

// AccountsChanged reaches Admins when an Account is added, changed or
// deleted.
type AccountsChanged struct{}

// LinksChanged reaches Admins when a Signup link is issued, used or ends.
type LinksChanged struct{}

var (
	SessionsChangedEvent = live.AccountEvent[SessionsChanged]("identity.sessions-changed")
	AccountsChangedEvent = live.AdminEvent[AccountsChanged]("identity.accounts-changed")
	LinksChangedEvent    = live.AdminEvent[LinksChanged]("identity.links-changed")
)

const (
	CookieName = "slipstream_session"
	// slideDivisor makes a session slide once per lifetime/720 of use: an
	// hour at 30 days, proportionally less when a Scenario seeds a short
	// lifetime.
	slideDivisor = 720
	tokenBytes   = 32
	apiKeyPrefix = "ssk_"
)

type Config struct {
	Library *storage.DB[file]
	Pepper  []byte
	// Sealer seals the Signup link tokens kept for Copy link.
	Sealer *secret.Sealer
	// Hub carries identity's events and ends sessions' streams.
	Hub *live.Hub
	// Mail sends the emails whose outcome the caller waits for, and Outbox
	// those that go out after the response.
	Mail   mail.Sender
	Outbox *mail.Outbox
}

type Service struct {
	lib        *storage.DB[file]
	pins       *pins
	dummy      string
	sealer     *secret.Sealer
	hub        *live.Hub
	sessions   live.AccountPublisher[SessionsChanged]
	passkeys   live.AccountPublisher[PasskeysChanged]
	accounts   live.AdminPublisher[AccountsChanged]
	links      live.AdminPublisher[LinksChanged]
	address    live.AdminPublisher[AddressChanged]
	mail       mail.Sender
	outbox     *mail.Outbox
	shadow     *shadowLedger
	limiter    *limiter
	ceremonies *passkey.Ceremonies
}

// New builds the dummy hash an unknown email verifies against. It reads no
// setting: every setting is read at use.
func New(ctx context.Context, cfg Config) (*Service, error) {
	s := &Service{
		lib:        cfg.Library,
		pins:       newPINs(cfg.Pepper),
		sealer:     cfg.Sealer,
		hub:        cfg.Hub,
		sessions:   SessionsChangedEvent.Bind(cfg.Hub),
		passkeys:   PasskeysChangedEvent.Bind(cfg.Hub),
		accounts:   AccountsChangedEvent.Bind(cfg.Hub),
		links:      LinksChangedEvent.Bind(cfg.Hub),
		address:    AddressChangedEvent.Bind(cfg.Hub),
		mail:       cfg.Mail,
		outbox:     cfg.Outbox,
		shadow:     newShadowLedger(),
		limiter:    newLimiter(),
		ceremonies: passkey.NewCeremonies(),
	}
	var err error
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
	APIKey  APIKey
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
	apiKey := newAPIKey()
	token := newToken()
	var lifetime time.Duration
	err = s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		admin, err := q.HasAdmin(ctx)
		if err != nil {
			return err
		}
		if admin {
			return ErrAlreadyClaimed
		}
		set, err := readSettings(ctx, q)
		if err != nil {
			return err
		}
		lifetime = set.SessionLifetime
		err = q.InsertAccount(ctx, db.InsertAccountParams{
			ID:          profile.ID,
			Email:       profile.Email,
			DisplayName: profile.DisplayName,
			Admin:       true,
			PinHash:     hash,
			CreatedAt:   fx.now,
		})
		if err != nil {
			return err
		}
		if err := q.InsertAPIKey(ctx, db.InsertAPIKeyParams{TokenHash: digest(apiKey.token), CreatedAt: fx.now}); err != nil {
			return err
		}
		row, err := q.GetAccount(ctx, profile.ID)
		if err != nil {
			return err
		}
		_, err = startSession(ctx, q, fx, c.client, row, token, false)
		return err
	})
	if err != nil {
		return Claimed{}, err
	}
	return Claimed{Profile: profile, APIKey: apiKey, Cookie: cookie(token, c.https, lifetime)}, nil
}

// SignedIn is any way in: a PIN, a Signup link, a Reset link.
// PasskeyPrompt is the "Sign in faster here" sheet, which the same Write
// marks shown on the new session, so Not now holds until the next sign-in.
type SignedIn struct {
	Profile       Profile
	Cookie        *http.Cookie
	PINRequired   bool
	KeyPrompt     KeyPrompt
	PasskeyPrompt bool
}

var ErrAccountDisabled = errors.New("identity: the Account is turned off")

// SignIn verifies email and pin and starts a session. An unknown email
// verifies against the dummy hash and keeps a shadow ledger, so it answers
// exactly as a real Account does. A turned-off Account is refused only
// once its PIN checks out, so the refusal reveals nothing without it.
func (s *Service) SignIn(ctx context.Context, a Anyone, email string, pin PIN) (SignedIn, error) {
	set, err := readSettings(ctx, db.New(s.lib))
	if err != nil {
		return SignedIn{}, err
	}
	sub, err := s.subjectByEmail(ctx, email)
	if err != nil {
		return SignedIn{}, err
	}
	pass, err := s.checkPIN(ctx, a.client, sub, pin, set)
	if err != nil {
		return SignedIn{}, err
	}
	token := newToken()
	var out SignedIn
	disabled := false
	err = s.afterPIN(ctx, pass, func(ctx context.Context, q *db.Queries, fx *effects) error {
		row, err := q.GetAccount(ctx, sub.account)
		if err != nil {
			return err
		}
		if !row.DisabledAt.IsZero() {
			disabled = true
			return nil
		}
		out, err = startSession(ctx, q, fx, a.client, row, token, false)
		return err
	})
	if err != nil {
		return SignedIn{}, err
	}
	if disabled {
		return SignedIn{}, ErrAccountDisabled
	}
	out.Cookie = cookie(token, a.https, set.SessionLifetime)
	return out, nil
}

func (s *Service) signInBy(ctx context.Context, c client, fn func(ctx context.Context, q *db.Queries, fx *effects) (AccountID, error)) (SignedIn, error) {
	token := newToken()
	var out SignedIn
	var lifetime time.Duration
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		id, err := fn(ctx, q, fx)
		if err != nil {
			return err
		}
		set, err := readSettings(ctx, q)
		if err != nil {
			return err
		}
		lifetime = set.SessionLifetime
		row, err := q.GetAccount(ctx, id)
		if err != nil {
			return err
		}
		out, err = startSession(ctx, q, fx, c, row, token, false)
		return err
	})
	if err != nil {
		return SignedIn{}, err
	}
	out.Cookie = cookie(token, c.https, lifetime)
	return out, nil
}

func (s *Service) subjectByEmail(ctx context.Context, email string) (subject, error) {
	email = normalizeEmail(email)
	row, err := db.New(s.lib).GetAccountByEmail(ctx, email)
	if errors.Is(err, storage.ErrNoRows) {
		return subject{email: email, hash: s.dummy}, nil
	}
	if err != nil {
		return subject{}, err
	}
	return subjectOf(row), nil
}

func profileOf(row db.IdentityAccount) Profile {
	return Profile{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Admin: row.Admin}
}

// SignOut ends b's session and its open streams and returns the cookie
// that clears it.
func (s *Service) SignOut(ctx context.Context, b Bearer) (*http.Cookie, error) {
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		fx.endSession(b.s.profile.ID, b.s.session, live.SignedOut)
		return q.DeleteSession(ctx, b.s.session)
	})
	if err != nil {
		return nil, err
	}
	return clearCookie(b.https), nil
}

// Request is what authentication reads from an HTTP request.
type Request struct {
	// APIKey is the X-Api-Key header, "" when absent.
	APIKey string
	// Session is the session cookie's value, "" when absent.
	Session string
	// Address and HTTPS are the client's, after http.trusted_proxies.
	Address netip.Addr
	HTTPS   bool
	// Host is the request's Host, its port included.
	Host      string
	UserAgent string
}

// Unauthenticated is r's caller as nobody, its credentials unread.
func (r Request) Unauthenticated() Caller {
	return Caller{client: client{address: r.Address, https: r.HTTPS, host: r.Host, userAgent: r.UserAgent}}
}

// Authenticate resolves r's Caller. A request with an API key
// authenticates by it alone. Otherwise the session cookie is looked up;
// the cookie returned, when not nil, re-sends a session that slid or
// clears one that is gone or expired.
func (s *Service) Authenticate(ctx context.Context, r Request) (Caller, *http.Cookie, error) {
	c := r.Unauthenticated()
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
		return c, clearCookie(r.HTTPS), nil
	}
	if err != nil {
		return Caller{}, nil, err
	}
	lifetime := SessionLifetime.Value(row.SessionLifetimeSeconds)
	u, err := s.use(ctx, row.SessionID, row.AccountID, row.LastSeenAt, lifetime)
	if err != nil {
		return Caller{}, nil, err
	}
	if u == expired {
		return c, clearCookie(r.HTTPS), nil
	}
	window := ReconfirmWindow.Value(row.ReconfirmWindowSeconds)
	c.kind = sessionCaller
	c.s = signedIn{
		profile:     Profile{ID: row.AccountID, Email: row.Email, DisplayName: row.DisplayName, Admin: row.Admin},
		session:     row.SessionID,
		confirmed:   time.Since(row.ConfirmedAt.Time()) <= window,
		pinRequired: row.PinRequired,
		keyPrompt:   row.KeyPrompt,
	}
	if u == slid {
		return c, cookie(r.Session, r.HTTPS, lifetime), nil
	}
	return c, nil, nil
}

// Renew counts a's open stream as use of its session, under the same rule
// as a request. A session that expired or is gone returns ErrSignedOut.
func (s *Service) Renew(ctx context.Context, a Account) error {
	row, err := db.New(s.lib).GetSessionUse(ctx, a.s.session)
	if errors.Is(err, storage.ErrNoRows) {
		return ErrSignedOut
	}
	if err != nil {
		return err
	}
	u, err := s.use(ctx, a.s.session, a.s.profile.ID, row.LastSeenAt, SessionLifetime.Value(row.SessionLifetimeSeconds))
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
func (s *Service) use(ctx context.Context, session SessionID, account AccountID, seen storage.Time, lifetime time.Duration) (sessionUse, error) {
	now := storage.Now()
	idle := now.Time().Sub(seen.Time())
	switch {
	case idle > lifetime:
		err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
			fx.endSession(account, session, live.Expired)
			return q.DeleteSession(ctx, session)
		})
		return expired, err
	case idle > lifetime/slideDivisor:
		err := s.write(ctx, func(ctx context.Context, q *db.Queries, _ *effects) error {
			return q.TouchSession(ctx, db.TouchSessionParams{LastSeenAt: now, ID: session})
		})
		return slid, err
	}
	return current, nil
}

// PruneSessions deletes every session idle past the session lifetime, for
// every Account, and ends its streams as a use of it would.
func (s *Service) PruneSessions(ctx context.Context) error {
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		set, err := readSettings(ctx, q)
		if err != nil {
			return err
		}
		before := storage.FromTime(fx.now.Time().Add(-set.SessionLifetime))
		pruned, err := q.ListIdleSessions(ctx, before)
		if err != nil {
			return err
		}
		for _, p := range pruned {
			fx.endSession(p.AccountID, p.ID, live.Expired)
		}
		return q.DeleteIdleSessions(ctx, before)
	})
	if err != nil {
		return fmt.Errorf("prune idle sessions: %w", err)
	}
	return nil
}

// cookie builds the session cookie: slipstream_session=<token>; Path=/;
// Max-Age=<lifetime>; HttpOnly; SameSite=Lax, plus Secure over HTTPS.
func cookie(token string, secure bool, lifetime time.Duration) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // Secure only over HTTPS: a LAN Node is reached by plain HTTP
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(lifetime / time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func clearCookie(secure bool) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // Secure only over HTTPS: a LAN Node is reached by plain HTTP
		Name:     CookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// startSession is the one session insert.
func startSession(ctx context.Context, q *db.Queries, fx *effects, c client, account db.IdentityAccount, token string, pinRequired bool) (SignedIn, error) {
	prompt := false
	if rp, v := c.relyingParty(); v.Available && account.KeyPrompt == KeyPromptAsk {
		held, err := q.CountPasskeysAt(ctx, db.CountPasskeysAtParams{AccountID: account.ID, RelyingParty: rp.ID()})
		if err != nil {
			return SignedIn{}, err
		}
		prompt = held == 0
	}
	err := q.InsertSession(ctx, db.InsertSessionParams{
		ID:          SessionID{UUID: storage.NewUUID()},
		TokenHash:   digest(token),
		AccountID:   account.ID,
		Address:     c.address.String(),
		CreatedAt:   fx.now,
		LastSeenAt:  fx.now,
		ConfirmedAt: fx.now,
		Device:      deviceLabel(c.userAgent),
		PinRequired: pinRequired,
		Prompted:    prompt,
	})
	if err != nil {
		return SignedIn{}, err
	}
	fx.sessionsChanged(account.ID)
	if err := q.StampSignIn(ctx, db.StampSignInParams{LastSignInAt: fx.now, ID: account.ID}); err != nil {
		return SignedIn{}, err
	}
	return SignedIn{Profile: profileOf(account), PINRequired: pinRequired, KeyPrompt: account.KeyPrompt, PasskeyPrompt: prompt}, nil
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
