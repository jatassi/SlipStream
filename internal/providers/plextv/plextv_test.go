package plextv_test

import (
	"crypto/ed25519"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
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
	api, auth, plex faketest.Served
	client          *plextv.Client
}

type serveFunc func(t testing.TB, spec fake.Spec, s scenario.Scenario, opts ...faketest.Option) faketest.Served

func newWorld(t *testing.T, serve serveFunc, account fakeplextv.Account, opts ...faketest.Option) world {
	t.Helper()
	s := scenario.Scenario{Seed: "plextv"}
	shared := fake.NewShared()
	srv := fakeplex.Server{Name: "Den", Version: "1.43.5.11029", Identifier: "fake-plex-den", Owner: "jackson", Root: "/media"}
	var w world
	w.plex = serve(t, fakeplex.Playing(srv), s, faketest.WithShared(shared), faketest.As("controller"))
	fronts := fake.Fronts{{Target: "plex", Node: "world", Address: den}, {Target: "plex", Node: "controller", Address: w.plex.Address}, {Target: "plex", Node: "den", Address: den}}
	w.api = serve(t, fakeplextv.Playing(account), s, append(opts, faketest.WithShared(shared), faketest.WithFronts(fronts))...)
	w.auth = serve(t, fakeplextv.Auth, s, faketest.WithShared(shared))
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	e := plextv.Endpoints{
		API:     endpoint.MustParseURL(w.api.Address.URL("")),
		Auth:    endpoint.MustParseURL(w.auth.Address.URL("/auth")),
		Desktop: endpoint.MustParseURL("https://app.plex.invalid/desktop"),
	}
	w.client = plextv.New(e, headers, key, w.api.Client)
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

// submit posts form to the sign-in page's path.
func (w world) submit(t *testing.T, path string, form url.Values) page {
	t.Helper()
	return w.fetch(t, http.MethodPost, w.auth.Address.URL(path), form)
}

// signIn runs the PIN flow, the Admin allowing it on the sign-in page,
// and answers the Plex JWT.
func (w world) signIn(t *testing.T) plextv.Token {
	t.Helper()
	pin, err := w.client.CreatePin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if p := w.submit(t, "/auth/approve", url.Values{"code": {pin.Code}, "forwardUrl": {forward}}); p.status != http.StatusSeeOther {
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
		back := w.submit(t, "/auth/approve", url.Values{"code": {fragment.Get("code")}, "forwardUrl": {fragment.Get("forwardUrl")}})
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

func TestDeclineReturnsWithTheAccountSignedOut(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson", PlexPass: true})
		pin, err := w.client.CreatePin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		back := w.submit(t, "/auth/decline", url.Values{"forwardUrl": {forward}})
		if back.location != forward {
			t.Fatalf("after decline: to %q", back.location)
		}
		if _, ok, err := w.client.PollPin(t.Context(), pin.ID); ok || err != nil {
			t.Fatalf("poll after decline = %v, %v, want waiting", ok, err)
		}
	})
}

func TestServersListOneLocalConnectionPerNodeAndATokenOnlyTheServerTakes(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson", PlexPass: true, Relay: true})
		token := w.signIn(t)
		servers, err := w.client.Servers(t.Context(), token.JWT)
		if err != nil || len(servers) != 1 {
			t.Fatalf("servers = %+v, %v", servers, err)
		}
		srv := servers[0]
		var uris []string
		for _, c := range srv.Connections {
			if !c.Local {
				t.Errorf("connection %s is not local", c.URI)
			}
			uris = append(uris, c.URI.String())
		}
		if want := []string{w.plex.Address.URL(""), den.URL("")}; strings.Join(uris, " ") != strings.Join(want, " ") {
			t.Errorf("connections = %v, want %v, the relay dropped", uris, want)
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
		if p := w.submit(t, "/auth/approve", url.Values{"code": {pin.Code}, "forwardUrl": {forward}}); p.status != http.StatusNotFound {
			t.Errorf("approve an expired PIN: status %d, want 404", p.status)
		}
		if _, _, err := w.client.PollPin(t.Context(), pin.ID); !plextv.Expired(err) {
			t.Fatalf("poll an expired PIN: %v, want expired", err)
		}
	})
}

func TestApproveCueSignsInTheWaitingPin(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson"})
		pin, err := w.client.CreatePin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		w.api.Cue(t, fakeplextv.Approve.Name, struct{}{})
		if _, ok, err := w.client.PollPin(t.Context(), pin.ID); !ok || err != nil {
			t.Fatalf("poll after the approve cue = %v, %v", ok, err)
		}
	})
}

func TestPlexPassFollowsTheSubscription(t *testing.T) {
	inEachTier(t, func(t *testing.T, serve serveFunc) {
		w := newWorld(t, serve, fakeplextv.Account{Username: "jackson"})
		token := w.signIn(t)
		if user, err := w.client.User(t.Context(), token.JWT); err != nil || user.PlexPass {
			t.Fatalf("user without Plex Pass = %+v, %v", user, err)
		}
		w.api.Cue(t, fakeplextv.PlexPass.Name, map[string]bool{"active": true})
		if user, err := w.client.User(t.Context(), token.JWT); err != nil || !user.PlexPass {
			t.Fatalf("user with Plex Pass = %+v, %v", user, err)
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
