package identityapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/apitest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/identityapi"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type recorder struct {
	mu        sync.Mutex
	published []string
	ended     []string
}

func (r *recorder) Publish(_ context.Context, account storage.UUID, v identity.SessionsChanged) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.Account != account.String() {
		panic("SessionsChanged names another Account than it is published to")
	}
	r.published = append(r.published, account.String())
}

func (r *recorder) EndSession(session storage.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ended = append(r.ended, session.String())
}

type node struct {
	t      *testing.T
	lib    *storage.DB[storage.Library]
	h      http.Handler
	doc    []byte
	code   string
	events *recorder
}

var fixtureAdmin = struct{ ID, Path string }{"getFixture", "/fixture"}

func newNode(t *testing.T, origins ...api.Origin) *node {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	keys, err := secret.Open(ctx, lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	events := &recorder{}
	svc, err := identity.New(ctx, lib, keys.PINPepper(), events, events)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(t.Output(), nil))
	gate, code, err := identity.NewGate(ctx, svc.HasAdmin, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{
		Identity:       svc,
		Gate:           gate,
		Web:            fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		TrustedOrigins: origins,
		Log:            log,
	})
	identityapi.Routes(r, svc)
	api.Register(r.Owner("fixture", api.Admin), api.Admin, api.Op{ID: fixtureAdmin.ID, Method: http.MethodGet, Path: fixtureAdmin.Path},
		func(context.Context, identity.Admin, *struct{}) (*struct{}, error) { return &struct{}{}, nil })
	return &node{t: t, lib: lib, h: r.Handler(), doc: r.Document(), code: code, events: events}
}

type call struct {
	method, target, body string
	headers              map[string]string
	session              string
}

func (n *node) do(c call) *httptest.ResponseRecorder {
	n.t.Helper()
	req := httptest.NewRequestWithContext(n.t.Context(), c.method, c.target, strings.NewReader(c.body))
	if c.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	if c.session != "" {
		req.Header.Set("Cookie", identity.CookieName+"="+c.session)
	}
	w := httptest.NewRecorder()
	n.h.ServeHTTP(w, req)
	return w
}

func claimBody(pin string) string {
	return `{"email":"admin@example.com","displayName":"Admin","pin":"` + pin + `"}`
}

func (n *node) claim(code, pin string) *httptest.ResponseRecorder {
	return n.do(call{method: http.MethodPost, target: "/api/v1/setup/controller", body: claimBody(pin), headers: map[string]string{"X-Setup-Code": code}})
}

func (n *node) signIn(email, pin string) *httptest.ResponseRecorder {
	return n.do(call{method: http.MethodPost, target: "/api/v1/session", body: `{"email":"` + email + `","pin":"` + pin + `"}`})
}

func (n *node) getSession(session string) *httptest.ResponseRecorder {
	return n.do(call{method: http.MethodGet, target: "/api/v1/session", session: session})
}

type claimResponse struct {
	Account struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		Admin       bool   `json:"admin"`
	} `json:"account"`
	APIKey string `json:"apiKey"`
}

// claimed claims n and returns the response and the session token.
func (n *node) claimed() (res claimResponse, session string) {
	n.t.Helper()
	w := n.claim(n.code, "1234")
	if w.Code != http.StatusCreated {
		n.t.Fatalf("claimController = %d %s", w.Code, w.Body)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		n.t.Fatal(err)
	}
	return res, sessionOf(n.t, w)
}

func sessionOf(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == identity.CookieName && c.MaxAge > 0 {
			return c.Value
		}
	}
	t.Fatalf("no session cookie in %q", w.Header().Values("Set-Cookie"))
	return ""
}

func expect(t *testing.T, w *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if w.Code != status || w.Body.String() != body {
		t.Fatalf("got %d %q, want %d %q", w.Code, w.Body, status, body)
	}
}

func problem(status int, code, title string) string {
	return fmt.Sprintf(`{"status":%d,"code":%q,"title":%q}`, status, code, title) + "\n"
}

var (
	signedOut      = problem(401, "signed_out", "Sign in to continue")
	pinWrong       = problem(401, "pin_wrong", "The email or PIN is wrong")
	alreadyClaimed = problem(409, "already_claimed", "This Node is already set up")
	clearingCookie = "slipstream_session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax"
	sessionCookie  = regexp.MustCompile(`^slipstream_session=[A-Za-z0-9_-]{43}; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax$`)
)

func TestClaimController(t *testing.T) {
	n := newNode(t)
	expect(t, n.claim("0000-0000-0000", "1234"), 403, problem(403, "setup_code_wrong", "The Setup code is wrong"))

	w := n.claim(n.code, "12a4")
	expect(t, w, 422, `{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.pin","message":"expected string to match pattern ^[0-9]{4}$"}]}`+"\n")

	typed := strings.ToLower(strings.ReplaceAll(n.code, "-", " "))
	w = n.claim(typed, "1234")
	if w.Code != http.StatusCreated {
		t.Fatalf("claimController = %d %s, want 201", w.Code, w.Body)
	}
	var res claimResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if !res.Account.Admin || res.Account.Email != "admin@example.com" || res.Account.DisplayName != "Admin" {
		t.Fatalf("account = %+v, want the Admin admin@example.com", res.Account)
	}
	if !regexp.MustCompile(`^ssk_[A-Za-z0-9_-]{43}$`).MatchString(res.APIKey) {
		t.Fatalf("apiKey = %q", res.APIKey)
	}
	if got := w.Header().Get("Set-Cookie"); !sessionCookie.MatchString(got) {
		t.Fatalf("Set-Cookie = %q", got)
	}
	if strings.Join(n.events.published, ",") != res.Account.ID {
		t.Fatalf("published to %v, want [%s]", n.events.published, res.Account.ID)
	}
	expect(t, n.claim(n.code, "1234"), 409, alreadyClaimed)
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/session", session: sessionOf(t, w)}), 200,
		`{"account":{"id":"`+res.Account.ID+`","email":"admin@example.com","displayName":"Admin","admin":true}}`+"\n")
}

// The gate turns busy before the body is read, so a second setup request
// while the first still sends its body gets setup_busy.
func TestConcurrentSetupIsBusy(t *testing.T) {
	n := newNode(t)
	body, send := io.Pipe()
	reading := make(chan struct{})
	first := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/setup/controller", &signalOnRead{r: body, first: reading})
	first.Header.Set("X-Setup-Code", n.code)
	first.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		n.h.ServeHTTP(w, first)
	}()
	<-reading
	expect(t, n.claim(n.code, "1234"), 409, problem(409, "setup_busy", "Another setup is running"))
	if _, err := io.WriteString(send, claimBody("1234")); err != nil {
		t.Fatal(err)
	}
	_ = send.Close()
	<-done
	if w.Code != http.StatusCreated {
		t.Fatalf("the first request = %d %s, want 201", w.Code, w.Body)
	}
	expect(t, n.claim(n.code, "1234"), 409, alreadyClaimed)
}

type signalOnRead struct {
	r     io.Reader
	first chan struct{}
	once  sync.Once
}

func (s *signalOnRead) Read(p []byte) (int, error) {
	s.once.Do(func() { close(s.first) })
	return s.r.Read(p) //nolint:wrapcheck // a Reader returns io.EOF as itself
}

func TestCreateSession(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	wrong := n.signIn("admin@example.com", "1235")
	expect(t, wrong, 401, pinWrong)
	unknown := n.signIn("nobody@example.com", "1234")
	expect(t, unknown, 401, pinWrong)
	if !reflect.DeepEqual(wrong.Header(), unknown.Header()) {
		t.Fatalf("a wrong PIN and an unknown email differ: %v and %v", wrong.Header(), unknown.Header())
	}
	w := n.signIn(" Admin@Example.COM ", "1234")
	expect(t, w, 200, `{"account":{"id":"`+res.Account.ID+`","email":"admin@example.com","displayName":"Admin","admin":true}}`+"\n")
	if got := w.Header().Get("Set-Cookie"); !sessionCookie.MatchString(got) {
		t.Fatalf("Set-Cookie = %q", got)
	}
	if len(n.events.published) != 2 {
		t.Fatalf("published %v, want one per sign-in", n.events.published)
	}
}

func TestTheGatesCodes(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	member := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
	expect(t, n.getSession(""), 401, signedOut)
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/session", headers: map[string]string{"X-Api-Key": res.APIKey}}), 403,
		problem(403, "needs_account", "Only a signed-in person can do this"))
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/fixture", session: member}), 403,
		problem(403, "needs_admin", "Only an Admin can do this"))
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/fixture", headers: map[string]string{"X-Api-Key": res.APIKey}}), 204, "")
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/session", session: member, headers: map[string]string{"X-Api-Key": "ssk_wrong"}}), 401,
		problem(401, "api_key_wrong", "The API key is wrong"))
}

func TestDeleteSession(t *testing.T) {
	n := newNode(t)
	_, a := n.claimed()
	w := n.do(call{method: http.MethodDelete, target: "/api/v1/session", session: a})
	expect(t, w, 204, "")
	if got := w.Header().Get("Set-Cookie"); got != clearingCookie {
		t.Fatalf("Set-Cookie = %q, want %q", got, clearingCookie)
	}
	if len(n.events.ended) != 1 || len(n.events.published) != 2 {
		t.Fatalf("ended %v and published %v, want one session ended and two changes", n.events.ended, n.events.published)
	}
	w = n.getSession(a)
	expect(t, w, 401, signedOut)
	if got := w.Header().Get("Set-Cookie"); got != clearingCookie {
		t.Fatalf("Set-Cookie = %q, want %q", got, clearingCookie)
	}
}

func TestSessionsExpireAndSlide(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		_, a := n.claimed()
		slid := func(want bool) {
			t.Helper()
			w := n.getSession(a)
			if w.Code != http.StatusOK {
				t.Fatalf("getSession = %d %s, want 200", w.Code, w.Body)
			}
			if got := w.Header().Get("Set-Cookie"); (got != "") != want || want && !sessionCookie.MatchString(got) {
				t.Fatalf("Set-Cookie = %q, want a re-sent cookie: %v", got, want)
			}
		}
		time.Sleep(time.Hour)
		slid(false)
		time.Sleep(time.Second)
		slid(true)
		slid(false)
		time.Sleep(29 * 24 * time.Hour)
		slid(true)
		time.Sleep(29 * 24 * time.Hour)
		slid(true)
		time.Sleep(30*24*time.Hour + time.Second)
		w := n.getSession(a)
		expect(t, w, 401, signedOut)
		if got := w.Header().Get("Set-Cookie"); got != clearingCookie {
			t.Fatalf("Set-Cookie = %q, want %q", got, clearingCookie)
		}
	})
}

func TestASessionIdleThirtyDaysAndASecondIsSignedOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		_, a := n.claimed()
		time.Sleep(30*24*time.Hour + time.Second)
		expect(t, n.getSession(a), 401, signedOut)
	})
}

func TestCrossSiteRequestsAreRefused(t *testing.T) {
	n := newNode(t, origin(t, "https://media.example.com"))
	res, _ := n.claimed()
	crossSite := map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}
	body := `{"email":"admin@example.com","pin":"1235"}`
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/session", body: body, headers: crossSite}), 403,
		problem(403, "cross_origin", "Another site cannot do this"))
	crossSite["X-Api-Key"] = res.APIKey
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/session", body: body, headers: crossSite}), 401, pinWrong)
	proxied := map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://media.example.com"}
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/session", body: body, headers: proxied}), 401, pinWrong)
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/session", body: body}), 401, pinWrong)

	w := n.do(call{method: http.MethodOptions, target: "/api/v1/session", headers: map[string]string{
		"Origin": "https://evil.example", "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "x-api-key",
	}})
	for name := range w.Header() {
		if strings.HasPrefix(name, "Access-Control-") {
			t.Errorf("OPTIONS answered %s: %s", name, w.Header().Get(name))
		}
	}
}

func origin(t *testing.T, s string) api.Origin {
	t.Helper()
	var o api.Origin
	if err := o.UnmarshalText([]byte(s)); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestBodiesAreCappedAtTwoMiB(t *testing.T) {
	n := newNode(t)
	padded := func(size int) string {
		s := `{"email":"admin@example.com","pin":"1234","x":"`
		return s + strings.Repeat("a", size-len(s)-2) + `"}`
	}
	expect(t, n.do(call{method: http.MethodPost, target: "/api/v1/session", body: padded(2<<20 + 1)}), 413,
		problem(413, "too_large", "The request body is too large"))
	if w := n.do(call{method: http.MethodPost, target: "/api/v1/session", body: padded(2 << 20)}); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("a body of exactly 2 MiB = %d %s, want 422 for its unknown field", w.Code, w.Body)
	}
}

func TestEveryResponseCarriesTheSecurityHeaders(t *testing.T) {
	n := newNode(t)
	want := map[string]string{
		"Content-Security-Policy":      "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), microphone=(), geolocation=()",
	}
	for target, cache := range map[string]string{"/": "no-cache", "/api/v1/session": "no-store", "/api/nothing": "no-store"} {
		w := n.do(call{method: http.MethodGet, target: target})
		for name, value := range want {
			if got := w.Header().Get(name); got != value {
				t.Errorf("GET %s %s = %q, want %q", target, name, got, value)
			}
		}
		if got := w.Header().Get("Cache-Control"); got != cache {
			t.Errorf("GET %s Cache-Control = %q, want %q", target, got, cache)
		}
		if got := w.Header().Get("Strict-Transport-Security"); got != "" {
			t.Errorf("GET %s sent HSTS %q", target, got)
		}
	}
}

func TestTheAccessWalkPassesOnAClaimedNode(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	admin := identity.AccountID{UUID: mustUUID(t, res.Account.ID)}
	member := identitytest.Member(t, n.lib, "member@example.com")
	breaches := apitest.CheckAccess(n.h, n.doc, apitest.Credentials{
		Member: func() string { return identitytest.Session(t, n.lib, member) },
		Admin:  func() string { return identitytest.Session(t, n.lib, admin) },
		APIKey: res.APIKey,
	})
	if len(breaches) != 0 {
		t.Fatalf("breaches:\n%s", strings.Join(breaches, "\n"))
	}
}

func mustUUID(t *testing.T, s string) storage.UUID {
	t.Helper()
	u, err := storage.ParseUUID(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
