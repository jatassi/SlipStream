// Package plextv is fake-plex-tv: plex.tv as the recorded run of
// 2026-11-14 met it, in two Specs that share one account through
// fake.Shared. plextv is the JSON API on clients.plex.tv behind the run
// CA: the strong PIN with a public JWK (1800 s life), its poll with a
// device JWT and no nonce answering authToken, /api/v2/user with
// subscription.active, resources with one local connection per Node front
// of each fake-plex server and that server's own 20-character token,
// which plex.tv refuses and fake-plex takes, and the renewal, a nonce then
// a JWT signed with the device key, answering auth_token with a new 7-day
// Plex JWT even after the last one expired. plextv-auth is the sign-in
// page over plain HTTP, so a browser that doesn't trust the run CA opens
// it, and it returns to forwardUrl with no query string, as a popup or a
// redirect alike. After a sign-out the account's JWTs and its server
// tokens get 401 and a renewal 422. A JWT plex.tv issued keeps working
// after a renewal until it expires.
package plextv

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/plexaccount"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

const (
	// pinLife is a PIN's expiresIn, 1800 in the recorded run.
	pinLife = 30 * time.Minute
	// jwtLife is a Plex JWT's life, 604800 s in the recorded run.
	jwtLife = 7 * 24 * time.Hour
	// strongCode is a strong PIN's code length, 25 in the recorded run.
	strongCode = 25
)

var (
	// Approve approves {"code": "…"} as the Admin would on the sign-in
	// page, or with no code the newest waiting PIN. It answers the code.
	Approve = fake.CueSpec{Name: "plextv-approve"}
	// ExpirePins ends every PIN's life now.
	ExpirePins = fake.CueSpec{Name: "plextv-expire-pins"}
	// SignOut is the Admin removing every device from the account on
	// plex.tv: its JWTs and server tokens get 401, its renewals 422.
	SignOut = fake.CueSpec{Name: "plextv-sign-out"}
	// PlexPass sets the account's subscription to {"active": bool}.
	PlexPass = fake.CueSpec{Name: "plextv-plex-pass"}
)

// Fake is plex.tv's API. Its address is clients.plex.tv's, and Open in's
// web app is a browser-only host.
var Fake = fake.Spec{
	Name: "plextv",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.PlexTV.API = endpoint.MustParseURL(at.URL(""))
		e.PlexTV.Desktop = endpoint.MustParseURL("https://app.plex.invalid/desktop")
	},
	Online: fake.NoOnlineRun("plex.tv needs a signed-in account; the JWT flow is a recorded manual run"),
	Cues:   []fake.CueSpec{Approve, ExpirePins, SignOut, PlexPass},
}

// Auth is plex.tv's sign-in page, over plain HTTP at 127.0.0.1, so every
// browser opens it with no resolver rule and no trust in the run CA.
var Auth = fake.Spec{
	Name: "plextv-auth",
	New:  newAuth,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.PlexTV.Auth = endpoint.MustParseURL(at.URL("/auth"))
	},
	Online: fake.NoOnlineRun("the plex.tv sign-in page is a browser page; the JWT flow is a recorded manual run"),
}

// Account is the one plex.tv account a world signs in to.
type Account struct {
	Username string
	PlexPass bool
	// Relay lists a relay connection to each server when resources asks
	// for relays. The recorded run's server listed none.
	Relay bool
}

// Playing is Fake signing in to a, for a test that names it.
func Playing(a Account) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, a) }
	return spec
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	return serve(env, Account{Username: "plex-admin", PlexPass: true})
}

// device is a device key plex.tv registered when its PIN was approved.
// A sign-in after a sign-out registers the key again as a new device.
type device struct {
	clientID string
	key      ed25519.PublicKey
	kid      string
	revoked  bool
	// tokens is the server token issued to it per server identifier.
	tokens map[string]string
}

type pin struct {
	id       int64
	code     string
	clientID string
	product  string
	jwk      jwk
	created  time.Time
	expires  time.Time
	// token is the Plex JWT once the Admin approved, "" while it waits.
	token string
}

// state is plex.tv for one world: what the API and the sign-in page
// share.
type state struct {
	mu      sync.Mutex
	account Account
	key     ed25519.PrivateKey
	draw    *rand.Rand
	fronts  fake.Fronts
	owned   *plexaccount.Account
	pins    map[int64]*pin
	devices map[string]*device
	// issued is the device and expiry of each Plex JWT plex.tv signed.
	issued map[string]issued
	nonces map[string]bool
}

type issued struct {
	device  *device
	expires time.Time
}

const sharedKey = "plextv"

// stateOf is shared's plex.tv. The world builds its fakes one after
// another, so the first Spec to ask puts it.
func stateOf(shared *fake.Shared) *state {
	if v, ok := shared.Get(sharedKey); ok {
		s, _ := v.(*state)
		return s
	}
	s := &state{owned: plexaccount.Of(shared), pins: map[int64]*pin{}, devices: map[string]*device{}, issued: map[string]issued{}, nonces: map[string]bool{}}
	shared.Put(sharedKey, s)
	return s
}

func serve(env fake.Env, a Account) (fake.Server, error) {
	s := stateOf(env.Shared)
	draw := env.Streams.Stream("plextv")
	var seed [ed25519.SeedSize]byte
	for i := 0; i < len(seed); i += 8 {
		binary.LittleEndian.PutUint64(seed[i:], draw.Uint64())
	}
	s.mu.Lock()
	s.account, s.key, s.draw, s.fronts = a, ed25519.NewKeyFromSeed(seed[:]), draw, env.Fronts
	s.mu.Unlock()
	if err := s.cues(env); err != nil {
		return nil, err
	}
	return fake.HTTP(env, s.routes()), nil
}

func (s *state) cues(env fake.Env) error {
	return errors.Join(
		fake.OnCue(env, Approve.Name, func(_ context.Context, a struct {
			Code string `json:"code"`
		}) (any, error) {
			code, err := s.approve(a.Code)
			return map[string]string{"code": code}, err
		}),
		fake.OnCue(env, ExpirePins.Name, func(context.Context, struct{}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			now := time.Now()
			for _, p := range s.pins {
				if p.expires.After(now) {
					p.expires = now
				}
			}
			return nil, nil
		}),
		fake.OnCue(env, SignOut.Name, func(context.Context, struct{}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			for _, d := range s.devices {
				d.revoked = true
				for _, token := range d.tokens {
					s.owned.Revoke(token)
				}
			}
			return nil, nil
		}),
		fake.OnCue(env, PlexPass.Name, func(_ context.Context, a struct {
			Active bool `json:"active"`
		}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.account.PlexPass = a.Active
			return nil, nil
		}),
	)
}

var errNoPin = errors.New("no PIN with that code is waiting")

// approve signs the account in on the PIN with code, or with code "" on
// the newest waiting PIN, and answers its code.
func (s *state) approve(code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var p *pin
	for _, q := range s.pins {
		if now.Before(q.expires) && q.token == "" && (q.code == code || (code == "" && (p == nil || q.created.After(p.created)))) {
			p = q
		}
	}
	if p == nil {
		return "", errNoPin
	}
	d := &device{clientID: p.clientID, key: p.jwk.key, kid: p.jwk.Kid, tokens: map[string]string{}}
	s.devices[d.kid] = d
	p.token = s.mint(d, now)
	return p.code, nil
}

// mint is a new Plex JWT for d, which s.mu holds.
func (s *state) mint(d *device, now time.Time) string {
	token := signJWT(s.key, map[string]any{
		"aud": []string{"plex.tv", d.clientID}, "iss": "plex.tv", "iat": now.Unix(), "exp": now.Add(jwtLife).Unix(),
		"nonce": s.word(32), "thumbprint": d.kid, "user": map[string]any{"id": 1, "uuid": s.word(16)},
	})
	s.issued[token] = issued{device: d, expires: now.Add(jwtLife)}
	return token
}

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// word is n characters drawn from alphabet, which s.mu holds.
func (s *state) word(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[s.draw.IntN(len(alphabet))]
	}
	return string(b)
}
