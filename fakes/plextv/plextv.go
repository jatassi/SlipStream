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
// after a renewal until it expires. The world holds the Admin's account,
// which owns the servers, and Members' accounts, each with its own id and
// uuid. A Member's device makes a strong PIN with no key, polled by its
// client identifier, whose approval gives a 20-character legacy token
// with no expiry, as the run of 2026-11-25 met it. Two tokens of one
// account coexist, and DELETE /api/v2/users/signout ends only the one it
// carries. The Admin's shares, the invites they leave, the Plex Home and
// a Member's accept follow the same run. plextv-web, plex.tv's own host,
// serves the share, its removal, the invite's cancel and the Home
// changes, and plextv the listings and the accept.
package plextv

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
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
	// legacyToken is a legacy token's length, 20 in the run of 2026-11-25.
	legacyToken = 20
)

var (
	// Approve approves {"code": "…", "account": "<username>"} as that
	// account would on the sign-in page, the Admin's when it names none, or
	// with no code the newest waiting PIN. It answers the code.
	Approve = fake.CueSpec{Name: "plextv-approve"}
	// ExpirePins ends every PIN's life now.
	ExpirePins = fake.CueSpec{Name: "plextv-expire-pins"}
	// SignOut is the Admin removing every device from their account on
	// plex.tv: its JWTs and server tokens get 401, its renewals 422.
	SignOut = fake.CueSpec{Name: "plextv-sign-out"}
	// PlexPass sets the Admin's subscription to {"active": bool}.
	PlexPass = fake.CueSpec{Name: "plextv-plex-pass"}
	// PlexMapping sets where Plex's own check of a server's public address
	// left it, {"server": "<name>", "state": "mapped"}, or failed or
	// pending, and stops the connect-back worker checking that server.
	PlexMapping = fake.CueSpec{Name: "plex-mapping"}
)

// ConnectBacks is the capture of each mapping the connect-back worker
// recorded, {"server": "<name>", "state": "mapped"} or failed, in
// plextv.connect-back records.
var ConnectBacks = fake.CaptureSpec{Name: "plex-connect-backs", Kind: "plextv.connect-back"}

// connectBackEvery is how often the connect-back worker checks each
// server with Remote Access on, as Plex rechecks its own mapping.
const connectBackEvery = 15 * time.Second

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
	Online:   fake.NoOnlineRun("plex.tv needs a signed-in account; the JWT flow is a recorded manual run"),
	Cues:     []fake.CueSpec{Approve, ExpirePins, SignOut, PlexPass, PlexMapping, RefuseNextShare, RefuseNextAccept, SignOutMember, HomeFull},
	Captures: []fake.CaptureSpec{ConnectBacks, Invites},
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

// Account is a plex.tv account a world holds.
type Account struct {
	Username string
	// Email is username@plex.invalid when empty.
	Email    string
	PlexPass bool
	// Relay lists a relay connection to each of the Admin's servers when
	// resources asks for relays. The recorded run's server listed none.
	Relay bool
}

// Playing is Fake holding the Admin's account, which owns the servers,
// and members' accounts, for a test that names them.
func Playing(admin Account, members ...Account) fake.Spec {
	spec := Fake
	spec.New = func(s scenario.Scenario, env fake.Env) (fake.Server, error) {
		return serve(s, env, append([]Account{admin}, members...))
	}
	return spec
}

// newServer holds the Admin's account with Plex Pass, each Scenario
// Member's account and the Scenario's other plex.tv accounts.
func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	accounts := []Account{{Username: "plex-admin", PlexPass: true}}
	for _, m := range s.Members {
		if m.Plex != nil {
			accounts = append(accounts, Account{Username: m.Plex.Username, Email: m.Plex.Email})
		}
	}
	for _, a := range s.PlexAccounts {
		accounts = append(accounts, Account{Username: a.Username, Email: a.Email})
	}
	return serve(s, env, accounts)
}

// person is an account as plex.tv keeps it, with the id and uuid it
// gave.
type person struct {
	Account
	id   int64
	uuid string
}

// device is a client plex.tv registered when its PIN was approved, with
// the key a JWT PIN carried and none for a legacy PIN. A sign-in after a
// sign-out registers the key again as a new device.
type device struct {
	clientID string
	account  *person
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
	// jwk is the device key a JWT PIN carried, zero on a legacy PIN.
	jwk     jwk
	created time.Time
	expires time.Time
	// token is the Plex JWT, or the legacy token, once its account
	// approved, "" while it waits.
	token string
}

// state is plex.tv for one world: what the API and the sign-in page
// share.
type state struct {
	mu sync.Mutex
	// accounts are the world's accounts, the Admin's first.
	accounts []*person
	key      ed25519.PrivateKey
	draw     *rand.Rand
	fronts   fake.Fronts
	owned    *plexaccount.Account
	pins     map[int64]*pin
	devices  map[string]*device
	// issued is the device and expiry of each token plex.tv gave: a Plex
	// JWT, or a legacy token, which never expires.
	issued map[string]issued
	nonces map[string]bool
	// cued are the servers whose mapping a cue or the Scenario set, which
	// the connect-back worker leaves alone.
	cued map[string]bool
	// site is the world's network, nil in a Scenario with none.
	site         *netsite.Model
	connectBacks *fake.Capture
	sharing
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
	s := &state{owned: plexaccount.Of(shared), pins: map[int64]*pin{}, devices: map[string]*device{}, issued: map[string]issued{}, nonces: map[string]bool{}, cued: map[string]bool{}}
	shared.Put(sharedKey, s)
	return s
}

func serve(sc scenario.Scenario, env fake.Env, accounts []Account) (fake.Server, error) {
	s := stateOf(env.Shared)
	draw := env.Streams.Stream("plextv")
	var seed [ed25519.SeedSize]byte
	for i := 0; i < len(seed); i += 8 {
		binary.LittleEndian.PutUint64(seed[i:], draw.Uint64())
	}
	s.mu.Lock()
	s.key, s.draw, s.fronts = ed25519.NewKeyFromSeed(seed[:]), draw, env.Fronts
	s.accounts = nil
	for _, a := range accounts {
		if a.Email == "" {
			a.Email = a.Username + "@plex.invalid"
		}
		s.accounts = append(s.accounts, &person{Account: a, id: 10_000_000 + draw.Int64N(90_000_000), uuid: s.word(16)})
	}
	s.mu.Unlock()
	if err := s.cues(env); err != nil {
		return nil, err
	}
	served := fake.HTTP(env, s.routes())
	if !netsite.Plays(sc) {
		return served, nil
	}
	site, err := netsite.From(env, sc)
	if err != nil {
		return nil, err
	}
	connectBacks, err := env.Capture(ConnectBacks.Name)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.site, s.connectBacks = site, connectBacks
	s.mu.Unlock()
	return connectingBack{Server: served, s: s}, nil
}

func (s *state) cues(env fake.Env) error {
	return errors.Join(
		s.sharingCues(env),
		fake.OnCue(env, Approve.Name, func(_ context.Context, a struct {
			Code    string `json:"code"`
			Account string `json:"account"`
		}) (any, error) {
			code, err := s.approve(a.Code, a.Account)
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
			for _, is := range s.issued {
				if d := is.device; d.account == s.admin() {
					d.revoked = true
					for _, token := range d.tokens {
						s.owned.Revoke(token)
					}
				}
			}
			return nil, nil
		}),
		fake.OnCue(env, PlexMapping.Name, func(_ context.Context, a struct {
			Server string `json:"server"`
			State  string `json:"state"`
		}) (any, error) {
			if !slices.Contains([]string{"mapped", "failed", "pending"}, a.State) {
				return nil, fmt.Errorf("plex-mapping: state %q is not mapped, failed or pending", a.State)
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			i := slices.IndexFunc(s.owned.Servers(), func(srv plexaccount.Server) bool { return srv.Name == a.Server })
			if i < 0 {
				return nil, fmt.Errorf("plex-mapping: the account owns no server %q", a.Server)
			}
			srv := s.owned.Servers()[i]
			s.cued[srv.Identifier] = true
			s.owned.SetMapping(srv.Identifier, plexaccount.MappingOf(a.State, s.public(srv)))
			return nil, nil
		}),
		fake.OnCue(env, PlexPass.Name, func(_ context.Context, a struct {
			Active bool `json:"active"`
		}) (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.admin().PlexPass = a.Active
			return nil, nil
		}),
	)
}

var errNoPin = errors.New("no PIN with that code is waiting")

// admin is the account that owns the servers, which s.mu holds.
func (s *state) admin() *person { return s.accounts[0] }

// approve signs the account named username in on the PIN with code, the
// Admin when username is "", or with code "" on the newest waiting PIN,
// and answers its code.
func (s *state) approve(code, username string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.admin()
	if username != "" {
		i := slices.IndexFunc(s.accounts, func(p *person) bool { return p.Username == username })
		if i < 0 {
			return "", fmt.Errorf("plex.tv holds no account %q", username)
		}
		a = s.accounts[i]
	}
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
	d := &device{clientID: p.clientID, account: a, key: p.jwk.key, kid: p.jwk.Kid, tokens: map[string]string{}}
	if p.legacy() {
		p.token = s.word(legacyToken)
		s.issued[p.token] = issued{device: d}
		return p.code, nil
	}
	s.devices[d.kid] = d
	p.token = s.mint(d, now)
	return p.code, nil
}

// mint is a new Plex JWT for d, which s.mu holds.
func (s *state) mint(d *device, now time.Time) string {
	token := signJWT(s.key, map[string]any{
		"aud": []string{"plex.tv", d.clientID}, "iss": "plex.tv", "iat": now.Unix(), "exp": now.Add(jwtLife).Unix(),
		"nonce": s.word(32), "thumbprint": d.kid, "user": map[string]any{"id": d.account.id, "uuid": d.account.uuid},
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

// legacy is whether p is a PIN with no device key, whose approval gives a
// legacy token.
func (p *pin) legacy() bool { return p.jwk.key == nil }
