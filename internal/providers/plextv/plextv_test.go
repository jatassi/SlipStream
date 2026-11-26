package plextv_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/plex"
	"github.com/jatassi/slipstream/internal/providers/plextv"
)

const forward = "http://127.0.0.1:5599/settings/system/media-servers"

var headers = plex.Headers{ClientID: "slipstream-test-device", Product: "SlipStream", Version: "2.0.0", DeviceName: "Controller"}

// den is a second Node's front to fake-plex, which fake-plex-tv lists
// after the Controller's. Nothing dials it.
var den = fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:32401")}

// world is plex.tv's API and sign-in page and one fake-plex sharing one
// account, and a plex.tv client for a new device key.
type world struct {
	api, web, auth, plex faketest.Served
	endpoints            plextv.Endpoints
	client               *plextv.Client
}

type serveFunc func(t testing.TB, spec fake.Spec, s scenario.Scenario, opts ...faketest.Option) faketest.Served

func newWorld(t *testing.T, serve serveFunc, account fakeplextv.Account, opts ...faketest.Option) world {
	t.Helper()
	return newWorldOf(t, serve, fakeplextv.Playing(account), opts...)
}

func newWorldOf(t *testing.T, serve serveFunc, tv fake.Spec, opts ...faketest.Option) world {
	t.Helper()
	s := scenario.Scenario{Seed: "plextv"}
	shared := fake.NewShared()
	srv := fakeplex.Server{
		Name: "Den", Version: "1.43.5.11029", Identifier: "fake-plex-den", Owner: "jackson", Root: "/media",
		Published: []string{"https://203.0.113.9:32400", "https://[2001:db8::9]:32400"}, Custom: []string{"https://plex.example.test"},
		Sections: []fakeplex.Section{{Key: "1", Title: "Movies", Type: "movie"}, {Key: "2", Title: "TV Shows", Type: "show"}},
	}
	var w world
	w.plex = serve(t, fakeplex.Playing(srv), s, faketest.WithShared(shared), faketest.As("controller"))
	fronts := fake.Fronts{{Target: "plex", Node: "world", Address: den}, {Target: "plex", Node: "controller", Address: w.plex.Address}, {Target: "plex", Node: "den", Address: den}}
	w.api = serve(t, tv, s, append(opts, faketest.WithShared(shared), faketest.WithFronts(fronts))...)
	w.web = serve(t, fakeplextv.Web, s, faketest.WithShared(shared))
	w.auth = serve(t, fakeplextv.Auth, s, faketest.WithShared(shared))
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	w.endpoints = plextv.Endpoints{
		API:     endpoint.MustParseURL(w.api.Address.URL("")),
		Web:     endpoint.MustParseURL(w.web.Address.URL("")),
		Auth:    endpoint.MustParseURL(w.auth.Address.URL("/auth")),
		Desktop: endpoint.MustParseURL("https://app.plex.invalid/desktop"),
	}
	w.client = plextv.New(w.endpoints, headers, key, w.api.Client)
	return w
}

func inEachTier(t *testing.T, test func(t *testing.T, serve serveFunc)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory) })
	})
}

// page is a response from the sign-in page, its redirect not followed.
type page struct {
	status   int
	location string
}

func (w world) fetch(t *testing.T, method, to string, form url.Values) page {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, to, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c := *w.auth.Client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return page{status: resp.StatusCode, location: resp.Header.Get("Location")}
}

// approve posts form to the sign-in page's Allow.
func (w world) approve(t *testing.T, form url.Values) page {
	t.Helper()
	return w.fetch(t, http.MethodPost, w.auth.Address.URL("/auth/approve"), form)
}

// signIn runs the PIN flow, the Admin allowing it on the sign-in page,
// and answers the Plex JWT.
func (w world) signIn(t *testing.T) plextv.Token {
	t.Helper()
	pin, err := w.client.CreatePin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if p := w.approve(t, url.Values{"code": {pin.Code}, "forwardUrl": {forward}}); p.status != http.StatusSeeOther {
		t.Fatalf("approve: status %d", p.status)
	}
	token, ok, err := w.client.PollPin(t.Context(), pin.ID)
	if err != nil || !ok {
		t.Fatalf("poll after approval = %v, %v", ok, err)
	}
	return token
}

func statusOf(err error) int {
	e, ok := errors.AsType[*plextv.Error](err)
	if !ok {
		return 0
	}
	return e.Status
}

func TestStrongPinWaitsForTheAdmin(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson"})
		pin, err := w.client.CreatePin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(pin.Code) != 25 || time.Until(pin.ExpiresAt).Round(time.Minute) != 30*time.Minute {
			t.Fatalf("pin = %+v, want a 25-character code living 30 minutes", pin)
		}
		if _, ok, err := w.client.PollPin(t.Context(), pin.ID); ok || err != nil {
			t.Fatalf("poll before approval = %v, %v, want waiting", ok, err)
		}
	})
}

func TestPinSignInReturnsToForwardURLWithNoQuery(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson", PlexPass: true})
		pin, err := w.client.CreatePin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		authURL := w.client.AuthURL(pin.Code, forward)
		if p := w.fetch(t, http.MethodGet, authURL, nil); p.status != http.StatusOK {
			t.Fatalf("sign-in page: status %d", p.status)
		}
		fragment := authFragment(t, authURL)
		if fragment.Get("clientID") != headers.ClientID || fragment.Get("code") != pin.Code || fragment.Get("forwardUrl") != forward {
			t.Fatalf("auth URL fragment = %v", fragment)
		}
		back := w.approve(t, url.Values{"code": {fragment.Get("code")}, "forwardUrl": {fragment.Get("forwardUrl")}})
		if back.status != http.StatusSeeOther || back.location != forward {
			t.Fatalf("after approval: status %d to %q, want %q with no query", back.status, back.location, forward)
		}
		token, ok, err := w.client.PollPin(t.Context(), pin.ID)
		if err != nil || !ok || time.Until(token.Expires).Round(time.Hour) != 7*24*time.Hour {
			t.Fatalf("poll after approval = %+v, %v, %v, want a Plex JWT living 7 days", token, ok, err)
		}
		if user, err := w.client.User(t.Context(), token.JWT); err != nil || user.Username != "jackson" || !user.PlexPass {
			t.Fatalf("user = %+v, %v", user, err)
		}
	})
}

// authFragment is the query in the sign-in page address's fragment.
func authFragment(t *testing.T, authURL string) url.Values {
	t.Helper()
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	q, err := url.ParseQuery(strings.TrimPrefix(u.Fragment, "?"))
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestServersListEachConnectionTheRelayAndATokenOnlyTheServerTakes(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson", PlexPass: true, Relay: true})
		token := w.signIn(t)
		servers, err := w.client.Servers(t.Context(), token.JWT)
		if err != nil || len(servers) != 1 {
			t.Fatalf("servers = %+v, %v", servers, err)
		}
		srv := servers[0]
		type conn struct {
			URI, Address       string
			Port               uint16
			Local, Relay, IPv6 bool
		}
		var got []conn
		for _, c := range srv.Connections {
			got = append(got, conn{c.URI.String(), c.Address, c.Port, c.Local, c.Relay, c.IPv6})
		}
		want := []conn{
			{w.plex.Address.URL(""), "127.0.0.1", w.plex.Address.AddrPort.Port(), true, false, false},
			{den.URL(""), "127.0.0.1", den.AddrPort.Port(), true, false, false},
			{"https://203.0.113.9:32400", "203.0.113.9", 32400, false, false, false},
			{"https://[2001:db8::9]:32400", "2001:db8::9", 32400, false, false, true},
			{"https://plex.example.test", "plex.example.test", 443, false, false, false},
			{"https://203-0-113-7.relay.plex.invalid:8443", "203.0.113.7", 8443, false, true, false},
		}
		if !slices.Equal(got, want) {
			t.Errorf("connections = %+v, want %+v", got, want)
		}
		if srv.Identifier != "fake-plex-den" || !srv.Owned || len(srv.Token) != 20 || srv.Token == token.JWT {
			t.Fatalf("server = %+v", srv)
		}
		base, _ := url.Parse(w.plex.Address.URL(""))
		if _, err := plex.New(base, srv.Token, headers, w.plex.Client).Sections(t.Context()); err != nil {
			t.Errorf("sections with the server token: %v", err)
		}
		if _, err := plex.New(base, token.JWT, headers, w.plex.Client).Sections(t.Context()); !plex.Refused(err) {
			t.Errorf("sections with the account JWT: %v, want 401", err)
		}
		if _, err := w.client.User(t.Context(), srv.Token); !plextv.Refused(err) {
			t.Errorf("plex.tv user with the server token: %v, want 401", err)
		}
	})
}

func TestRenewalGivesANewJWTAndKeepsTheServerToken(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson", PlexPass: true})
		first := w.signIn(t)
		before, err := w.client.Servers(t.Context(), first.JWT)
		if err != nil {
			t.Fatal(err)
		}
		renewed, err := w.client.Renew(t.Context())
		if err != nil || renewed.JWT == first.JWT || time.Until(renewed.Expires).Round(time.Hour) != 7*24*time.Hour {
			t.Fatalf("renew = %+v, %v, want a new Plex JWT living 7 days", renewed, err)
		}
		for name, jwt := range map[string]string{"previous": first.JWT, "renewed": renewed.JWT} {
			if _, err := w.client.User(t.Context(), jwt); err != nil {
				t.Errorf("user with the %s JWT: %v", name, err)
			}
		}
		after, err := w.client.Servers(t.Context(), renewed.JWT)
		if err != nil || after[0].Token != before[0].Token {
			t.Fatalf("server token after renewal = %v, %v, want %s", after, err, before[0].Token)
		}
	})
}

func TestRenewalAfterExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t, faketest.InMemory, fakeplextv.Account{Username: "jackson", PlexPass: true})
		first := w.signIn(t)
		time.Sleep(8 * 24 * time.Hour)
		if _, err := w.client.User(t.Context(), first.JWT); !plextv.Refused(err) {
			t.Fatalf("user with an expired JWT: %v, want 401", err)
		}
		renewed, err := w.client.Renew(t.Context())
		if err != nil {
			t.Fatalf("renew after expiry: %v", err)
		}
		if _, err := w.client.User(t.Context(), renewed.JWT); err != nil {
			t.Fatalf("user with the JWT renewed after expiry: %v", err)
		}
	})
}

func TestAfterSignOutTokensGet401AndRenewal422(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson", PlexPass: true})
		token := w.signIn(t)
		servers, err := w.client.Servers(t.Context(), token.JWT)
		if err != nil {
			t.Fatal(err)
		}
		w.api.Cue(t, fakeplextv.SignOut.Name, struct{}{})
		if _, err := w.client.User(t.Context(), token.JWT); statusOf(err) != http.StatusUnauthorized || !plextv.Refused(err) {
			t.Errorf("user after sign-out: %v, want 401", err)
		}
		base, _ := url.Parse(w.plex.Address.URL(""))
		if _, err := plex.New(base, servers[0].Token, headers, w.plex.Client).Sections(t.Context()); !plex.Refused(err) {
			t.Errorf("sections with the server token after sign-out: %v, want 401", err)
		}
		if _, err := w.client.Renew(t.Context()); statusOf(err) != http.StatusUnprocessableEntity || !plextv.Refused(err) {
			t.Errorf("renew after sign-out: %v, want 422", err)
		}
		again := w.signIn(t)
		if _, err := w.client.User(t.Context(), again.JWT); err != nil {
			t.Errorf("user after signing in again: %v", err)
		}
	})
}

func TestExpiredPin(t *testing.T) {
	t.Run("past its life", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t, faketest.InMemory, fakeplextv.Account{Username: "jackson"})
			pin, err := w.client.CreatePin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(30 * time.Minute)
			if _, _, err := w.client.PollPin(t.Context(), pin.ID); !plextv.Expired(err) {
				t.Fatalf("poll after 30 minutes: %v, want expired", err)
			}
		})
	})
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson"})
		pin, err := w.client.CreatePin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		w.api.Cue(t, fakeplextv.ExpirePins.Name, struct{}{})
		if p := w.approve(t, url.Values{"code": {pin.Code}, "forwardUrl": {forward}}); p.status != http.StatusNotFound {
			t.Errorf("approve an expired PIN: status %d, want 404", p.status)
		}
		if _, _, err := w.client.PollPin(t.Context(), pin.ID); !plextv.Expired(err) {
			t.Fatalf("poll an expired PIN: %v, want expired", err)
		}
	})
}

func TestRateLimitedPollCarriesItsWait(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		limit := fault.Rule{ID: "limit", Target: "plextv", Route: "pin", Fault: fault.Status{Code: http.StatusTooManyRequests, RetryAfter: 30 * time.Second}, Times: 1}
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson"}, faketest.WithFaults(limit))
		pin, err := w.client.CreatePin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = w.client.PollPin(t.Context(), pin.ID)
		if wait, ok := plextv.RateLimited(err); !ok || wait != 30*time.Second {
			t.Fatalf("rate limited poll = %v, %v, %v, want 30s", wait, ok, err)
		}
		if _, _, err := w.client.PollPin(t.Context(), pin.ID); err != nil {
			t.Fatalf("poll after the limit: %v", err)
		}
	})
}

// TestAMemberSignsInWithALegacyPin approves a Member's legacy PIN on the
// sign-in page's account picker: the token is the Member's own, a second
// sign-in's token lives beside it, and signing one out leaves the other
// and the Admin's JWT working.
func TestAMemberSignsInWithALegacyPin(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorldOf(t, serve, fakeplextv.Playing(fakeplextv.Account{Username: "jackson", PlexPass: true}, fakeplextv.Account{Username: "ana", Email: "ana@example.test"}))
		jwt := w.signIn(t).JWT
		admin, err := w.client.User(t.Context(), jwt)
		if err != nil {
			t.Fatal(err)
		}
		member := plextv.NewMember(w.endpoints, plex.Headers{ClientID: "slipstream-member-ana", Product: "SlipStream"}, w.api.Client)
		first, second := w.memberSignIn(t, member, "ana"), w.memberSignIn(t, member, "ana")
		for _, token := range []plextv.MemberToken{first, second} {
			u, err := member.User(t.Context(), token)
			if err != nil || u.Username != "ana" || u.Email != "ana@example.test" || u.PlexPass || u.ID == 0 || u.ID == admin.ID || u.UUID == "" || u.UUID == admin.UUID {
				t.Fatalf("member user = %+v, %v, want ana apart from the Admin %+v", u, err, admin)
			}
		}
		if err := member.SignOut(t.Context(), first); err != nil {
			t.Fatal(err)
		}
		if _, err := member.User(t.Context(), first); !plextv.Refused(err) {
			t.Errorf("user with the signed-out token: %v, want 401", err)
		}
		if _, err := member.User(t.Context(), second); err != nil {
			t.Errorf("user with the other token after one signed out: %v", err)
		}
		if _, err := w.client.Servers(t.Context(), jwt); err != nil {
			t.Errorf("the Admin's servers after a Member signed out: %v", err)
		}
	})
}

// memberSignIn runs member's legacy PIN, the account named username
// allowing it on the sign-in page, and answers the Member's token.
func (w world) memberSignIn(t *testing.T, member *plextv.Member, username string) plextv.MemberToken {
	t.Helper()
	pin, err := member.CreateLegacyPin(t.Context())
	if err != nil {
		t.Fatalf("legacy PIN = %+v, %v", pin, err)
	}
	if _, ok, err := member.PollLegacyPin(t.Context(), pin.ID); ok || err != nil {
		t.Fatalf("poll before approval = %v, %v, want waiting", ok, err)
	}
	code := authFragment(t, member.AuthURL(pin.Code, forward)).Get("code")
	if p := w.approve(t, url.Values{"code": {code}, "account": {username}, "forwardUrl": {forward}}); p.status != http.StatusSeeOther {
		t.Fatalf("approve as %s: status %d", username, p.status)
	}
	token, ok, err := member.PollLegacyPin(t.Context(), pin.ID)
	if err != nil || !ok {
		t.Fatalf("poll after approval = %v, %v, want a token", ok, err)
	}
	return token
}

// sent is one request a plex.tv client sent: its host as a route names
// it, api or web, and its path with the query.
type sent struct {
	method, host, path, body string
}

// recorder sends each request to the fake serving its host and keeps
// what it sent.
type recorder struct {
	mu    sync.Mutex
	hosts map[string]struct {
		name string
		rt   http.RoundTripper
	}
	sent []sent
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	h, ok := r.hosts[req.URL.Host]
	if !ok {
		return nil, fmt.Errorf("no fake serves %s", req.URL.Host)
	}
	var body []byte
	if req.Body != nil {
		var err error
		if body, err = io.ReadAll(req.Body); err != nil {
			return nil, fmt.Errorf("read the request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	r.mu.Lock()
	r.sent = append(r.sent, sent{req.Method, h.name, req.URL.RequestURI(), string(body)})
	r.mu.Unlock()
	resp, err := h.rt.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", h.name, err)
	}
	return resp, nil
}

// take is what was sent since the last take.
func (r *recorder) take() []sent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.sent
	r.sent = nil
	return out
}

// sharing is a world whose Admin signed in, holding Members ana and bo,
// in memory with plex.tv's two hosts at their own names, and a recorder
// of every request to them.
type sharing struct {
	world
	rec     *recorder
	http    *http.Client
	admin   string
	machine string
	member  *plextv.Member
	ana, bo plextv.User
	anaTok  plextv.MemberToken
}

func newSharing(t *testing.T, plexPass bool) sharing {
	t.Helper()
	tv := fakeplextv.Playing(fakeplextv.Account{Username: "jackson", PlexPass: plexPass},
		fakeplextv.Account{Username: "ana", Email: "ana@example.test"}, fakeplextv.Account{Username: "bo", Email: "bo@example.test"})
	w := sharing{world: newWorldOf(t, faketest.InMemory, tv), machine: "fake-plex-den"}
	w.rec = &recorder{hosts: map[string]struct {
		name string
		rt   http.RoundTripper
	}{
		"plextv.fake.localhost":     {"api", w.api.Client.Transport},
		"plextv-web.fake.localhost": {"web", w.web.Client.Transport},
	}}
	w.http = &http.Client{Transport: w.rec}
	w.endpoints.API = endpoint.MustParseURL("https://plextv.fake.localhost")
	w.endpoints.Web = endpoint.MustParseURL("https://plextv-web.fake.localhost")
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	w.client = plextv.New(w.endpoints, headers, key, w.http)
	w.admin = w.signIn(t).JWT
	w.member = plextv.NewMember(w.endpoints, plex.Headers{ClientID: "slipstream-member", Product: "SlipStream"}, w.http)
	w.anaTok = w.memberSignIn(t, w.member, "ana")
	if w.ana, err = w.member.User(t.Context(), w.anaTok); err != nil {
		t.Fatal(err)
	}
	if w.bo, err = w.member.User(t.Context(), w.memberSignIn(t, w.member, "bo")); err != nil {
		t.Fatal(err)
	}
	w.rec.take()
	return w
}

func (w sharing) sections(t *testing.T) []plextv.SectionID {
	t.Helper()
	ids, err := w.client.Sections(t.Context(), w.admin, w.machine)
	if err != nil || len(ids) != 2 {
		t.Fatalf("sections = %v, %v", ids, err)
	}
	return []plextv.SectionID{ids["1"], ids["2"]}
}

func (w sharing) share(t *testing.T, member plextv.User) plextv.ShareID {
	t.Helper()
	id, err := w.client.Share(t.Context(), w.admin, w.machine, member.ID, w.sections(t))
	if err != nil || id == "" {
		t.Fatalf("share with %s = %q, %v", member.Username, id, err)
	}
	return id
}

// listedInvite is an invite as invites/requested lists it: the account's
// id, its server flag and how many pending shares it names.
type listedInvite struct {
	id, server string
	servers    int
}

// invited are the invites invites/requested lists.
func (w sharing) invited(t *testing.T) []listedInvite {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://plextv.fake.localhost/api/invites/requested", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Plex-Token", w.admin)
	resp, err := w.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		Invites []struct {
			ID      string     `xml:"id,attr"`
			Server  string     `xml:"server,attr"`
			Servers []struct{} `xml:"Server"`
		} `xml:"Invite"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	var invites []listedInvite
	for _, inv := range out.Invites {
		invites = append(invites, listedInvite{inv.ID, inv.Server, len(inv.Servers)})
	}
	return invites
}

func idOf(u plextv.User) string { return strconv.FormatInt(int64(u.ID), 10) }

func who(u plextv.User) plextv.Who { return plextv.Who{ID: u.ID, UUID: u.UUID} }

// wireRun is what the wire table's steps learn as they go.
type wireRun struct {
	sharing
	sections          []plextv.SectionID
	anaShare, boShare plextv.ShareID
}

// wireSteps drive each sharing route once, in the order SlipStream calls
// them, with the requests the run of 2026-11-25 and its RUNBOOK sent.
var wireSteps = []struct {
	name string
	do   func(t *testing.T, r *wireRun)
	want func(r *wireRun) []sent
}{
	{"sections", func(t *testing.T, r *wireRun) { r.sections = r.sharing.sections(t) }, func(r *wireRun) []sent {
		return []sent{{"GET", "api", "/api/servers/" + r.machine, ""}}
	}},
	{"share", func(t *testing.T, r *wireRun) { r.anaShare = r.share(t, r.ana) }, func(r *wireRun) []sent {
		body := fmt.Sprintf(`{"server_id":%q,"shared_server":{"library_section_ids":[%s,%s],"invited_id":%d},"sharing_settings":{}}`, r.machine, r.sections[0], r.sections[1], r.ana.ID)
		return []sent{{"GET", "api", "/api/servers/" + r.machine, ""}, {"POST", "web", "/api/servers/" + r.machine + "/shared_servers", body}}
	}},
	{"find a pending share", func(t *testing.T, r *wireRun) {
		r.wantFound(t, r.ana, plextv.Found{Share: r.anaShare})
	}, func(*wireRun) []sent { return findSent }},
	{"accept", func(t *testing.T, r *wireRun) {
		if ok, err := r.member.Accept(t.Context(), r.anaTok, r.anaShare); !ok || err != nil {
			t.Fatalf("accept = %v, %v", ok, err)
		}
		r.wantFound(t, r.ana, plextv.Found{Share: r.anaShare, Accepted: true})
	}, func(r *wireRun) []sent {
		return append([]sent{
			{"GET", "api", "/api/v2/shared_servers/invites/received/pending", ""}, {"POST", "api", "/api/v2/shared_servers/" + string(r.anaShare) + "/accept", ""},
		}, findSent...)
	}},
	{"unshare before accept", func(t *testing.T, r *wireRun) {
		r.boShare = r.share(t, r.bo)
		r.rec.take()
		if err := r.client.Unshare(t.Context(), r.admin, r.machine, r.boShare, who(r.bo)); err != nil {
			t.Fatal(err)
		}
	}, func(r *wireRun) []sent {
		return []sent{
			{"DELETE", "web", "/api/servers/" + r.machine + "/shared_servers/" + string(r.boShare), ""}, {"GET", "api", "/api/invites/requested", ""},
			{"DELETE", "web", "/api/invites/requested/" + idOf(r.bo) + "?friend=1&home=0&server=0", ""},
		}
	}},
	{"unshare after accept", func(t *testing.T, r *wireRun) {
		if err := r.client.Unshare(t.Context(), r.admin, r.machine, r.anaShare, who(r.ana)); err != nil {
			t.Fatal(err)
		}
	}, func(r *wireRun) []sent {
		return []sent{{"DELETE", "web", "/api/servers/" + r.machine + "/shared_servers/" + string(r.anaShare), ""}, {"GET", "api", "/api/invites/requested", ""}}
	}},
	{"invite into the Home", func(t *testing.T, r *wireRun) {
		if err := r.client.InviteHome(t.Context(), r.admin, plextv.Invitee{Who: who(r.ana), Username: "ana", Email: "ana@example.test"}); err != nil {
			t.Fatal(err)
		}
	}, func(*wireRun) []sent {
		return []sent{{"POST", "web", "/api/home/users?invitedEmail=ana%40example.test", ""}}
	}},
	{"leave the Home", func(t *testing.T, r *wireRun) {
		if err := r.client.LeaveHome(t.Context(), r.admin, who(r.ana)); err != nil {
			t.Fatal(err)
		}
	}, func(r *wireRun) []sent {
		return []sent{{"GET", "api", "/api/home/users", ""}, {"DELETE", "web", "/api/home/users/" + idOf(r.ana), ""}}
	}},
}

// findSent is FindShare's three listings.
var findSent = []sent{
	{"GET", "api", "/api/v2/shared_servers/owned/accepted", ""}, {"GET", "api", "/api/v2/shared_servers/owned/pending", ""},
	{"GET", "api", "/api/home/users", ""},
}

func (w sharing) wantFound(t *testing.T, member plextv.User, want plextv.Found) {
	t.Helper()
	if found, err := w.client.FindShare(t.Context(), w.admin, w.machine, who(member)); err != nil || found != want {
		t.Fatalf("found for %s = %+v, %v, want %+v", member.Username, found, err, want)
	}
}

func TestSharingCallsMatchTheRecordedRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &wireRun{sharing: newSharing(t, true)}
		for _, step := range wireSteps {
			step.do(t, r)
			if got, want := r.rec.take(), step.want(r); !slices.EqualFunc(got, want, sameRequest) {
				t.Errorf("%s sent %+v, want %+v", step.name, got, want)
			}
		}
	})
}

// sameRequest compares bodies as JSON values.
func sameRequest(a, b sent) bool {
	if a.body == "" || b.body == "" {
		return a == b
	}
	var x, y any
	if json.Unmarshal([]byte(a.body), &x) != nil || json.Unmarshal([]byte(b.body), &y) != nil {
		return false
	}
	a.body, b.body = "", ""
	return a == b && reflect.DeepEqual(x, y)
}

// sharingAnswers are plex.tv's answers to the sharing calls as the run
// of 2026-11-25 met them, and how the client reads each.
var sharingAnswers = []struct {
	name     string
	plexPass bool
	run      func(t *testing.T, w sharing)
}{
	{"a second share adopts the first", true, func(t *testing.T, w sharing) {
		first := w.share(t, w.ana)
		if again := w.share(t, w.ana); again != first {
			t.Fatalf("second share = %s, want %s", again, first)
		}
	}},
	{"a Home invite without Plex Pass is declined", false, func(t *testing.T, w sharing) {
		err := w.client.InviteHome(t.Context(), w.admin, plextv.Invitee{Who: who(w.ana), Email: "ana@example.test"})
		if !plextv.Declined(err) || plextv.Refused(err) || statusOf(err) != http.StatusForbidden {
			t.Fatalf("invite = %v, want the Plex Pass 403 declined", err)
		}
	}},
	{"a Home invite into a full Home is declined", true, func(t *testing.T, w sharing) {
		w.api.Cue(t, fakeplextv.HomeFull.Name, struct{}{})
		err := w.client.InviteHome(t.Context(), w.admin, plextv.Invitee{Who: who(w.ana), Email: "ana@example.test"})
		if !plextv.Declined(err) || plextv.Refused(err) || statusOf(err) != http.StatusForbidden {
			t.Fatalf("invite = %v, want the Home limit 403 declined", err)
		}
	}},
	{"a signed-out Admin is refused", true, func(t *testing.T, w sharing) {
		sections := w.sections(t)
		w.api.Cue(t, fakeplextv.SignOut.Name, struct{}{})
		_, err := w.client.Share(t.Context(), w.admin, w.machine, w.ana.ID, sections)
		if !plextv.Refused(err) || plextv.Declined(err) {
			t.Fatalf("share = %v, want 401 refused", err)
		}
	}},
	{"unshare of a share plex.tv no longer has", true, func(t *testing.T, w sharing) {
		if err := w.client.Unshare(t.Context(), w.admin, w.machine, "123456789", who(w.ana)); err != nil {
			t.Fatal(err)
		}
	}},
	{"a cancel of an invite plex.tv no longer has", true, func(t *testing.T, w sharing) {
		id := w.share(t, w.ana)
		if _, err := w.web.Faults.Set(fault.Rule{ID: "gone", Target: "plextv-web", Route: "cancel-invite", Fault: fault.Status{Code: http.StatusNotFound}, Times: 1}); err != nil {
			t.Fatal(err)
		}
		if err := w.client.Unshare(t.Context(), w.admin, w.machine, id, who(w.ana)); err != nil {
			t.Fatal(err)
		}
	}},
	{"leaving a Home the account isn't in", true, func(t *testing.T, w sharing) {
		if err := w.client.LeaveHome(t.Context(), w.admin, who(w.ana)); err != nil {
			t.Fatal(err)
		}
	}},
	{"leaving a Home plex.tv answers 404", true, func(t *testing.T, w sharing) {
		if err := w.client.InviteHome(t.Context(), w.admin, plextv.Invitee{Who: who(w.ana), Email: "ana@example.test"}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.web.Faults.Set(fault.Rule{ID: "gone", Target: "plextv-web", Route: "leave-home", Fault: fault.Status{Code: http.StatusNotFound}, Times: 1}); err != nil {
			t.Fatal(err)
		}
		if err := w.client.LeaveHome(t.Context(), w.admin, who(w.ana)); err != nil {
			t.Fatal(err)
		}
	}},
	{"an unshare before accept leaves no invite", true, func(t *testing.T, w sharing) {
		id := w.share(t, w.ana)
		if got := w.invited(t); !slices.Equal(got, []listedInvite{{idOf(w.ana), "1", 1}}) {
			t.Fatalf("invited after share = %+v, want ana's with the share", got)
		}
		if err := w.client.Unshare(t.Context(), w.admin, w.machine, id, who(w.ana)); err != nil {
			t.Fatal(err)
		}
		if got := w.invited(t); len(got) != 0 {
			t.Errorf("invited after unshare = %+v, want none", got)
		}
		if ok, err := w.member.Accept(t.Context(), w.anaTok, id); ok || err != nil {
			t.Errorf("accept after unshare = %v, %v, want not pending", ok, err)
		}
		if found, err := w.client.FindShare(t.Context(), w.admin, w.machine, who(w.ana)); err != nil || found != (plextv.Found{}) {
			t.Errorf("found after unshare = %+v, %v, want none", found, err)
		}
	}},
}

func TestSharingAnswers(t *testing.T) {
	for _, c := range sharingAnswers {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { c.run(t, newSharing(t, c.plexPass)) })
		})
	}
}
