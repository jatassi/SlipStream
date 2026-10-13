package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jatassi/slipstream/internal/api/apitest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type noEvents struct{}

func (noEvents) Publish(context.Context, storage.UUID, identity.SessionsChanged) {}
func (noEvents) EndSession(storage.UUID)                                         {}

func newRouter(t *testing.T, log *bytes.Buffer) *Router {
	t.Helper()
	r, _ := newRouterOn(t, log)
	return r
}

func newRouterOn(t *testing.T, log *bytes.Buffer) (*Router, *storage.DB[storage.Library]) {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	svc, err := identity.New(t.Context(), lib, make([]byte, 32), noEvents{}, noEvents{})
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(Services{
		Node:     identitytest.Controller{Node: identitytest.Node{Service: svc}},
		Web:      fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}},
		Log:      slog.New(slog.NewTextHandler(log, nil)),
	}), lib
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody))
	return w
}

const internalBody = `{"status":500,"code":"internal","title":"Something went wrong"}` + "\n"

func TestTheAccessWalkFlagsAPlantedOperation(t *testing.T) {
	r := NewRouter(Services{})
	huma.Register(r.huma, huma.Operation{OperationID: "plantedGet", Method: http.MethodGet, Path: "/planted"},
		func(context.Context, *struct{}) (*struct{}, error) { return &struct{}{}, nil })
	got := apitest.CheckAccess(r.Handler(), r.Document(), apitest.Credentials{})
	want := "GET /api/v1/planted: operation plantedGet has no x-slipstream-access"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("CheckAccess = %q, want [%q]", got, want)
	}
}

func TestRegisterRefusesALevelBelowItsOwnersFloor(t *testing.T) {
	o := NewRouter(Services{}).Owner("nodes", Admin)
	got := panicOf(func() {
		Register(o, Account, Op{ID: "listNodes", Method: http.MethodGet, Path: "/nodes"},
			func(context.Context, identity.Account, *struct{}) (*struct{}, error) { return nil, nil })
	})
	if want := "listNodes: owner nodes admits admin operations only; level is account"; got != want {
		t.Fatalf("panic = %q, want %q", got, want)
	}
}

func TestRegisterRefusesWhatWouldCollideOrHaveNoLimit(t *testing.T) {
	r := NewRouter(Services{})
	o := r.Owner("fixture", Public)
	ok := func(context.Context, identity.Anyone, *struct{}) (*struct{}, error) { return nil, nil }
	withBody := func(context.Context, identity.Anyone, *struct{ Body struct{} }) (*struct{}, error) { return nil, nil }
	Register(o, Public, Op{ID: "getFixture", Method: http.MethodGet, Path: "/fixture"}, ok)
	for _, tc := range []struct {
		register func()
		want     string
	}{
		{func() { r.Owner("fixture", Public) }, "api: owner tag fixture is used by two owners"},
		{func() { Register(o, Public, Op{ID: "getFixture", Method: http.MethodGet, Path: "/other"}, ok) },
			"api: operation id getFixture is registered twice: GET /api/v1/fixture by owner fixture and GET /api/v1/other by owner fixture"},
		{func() { Register(o, Public, Op{ID: "fixture-get", Method: http.MethodGet, Path: "/x"}, ok) },
			`api: operation id "fixture-get" is not lowerCamelCase and verb-first`},
		{func() {
			Register(o, Public, Op{ID: "postFixture", Method: http.MethodPost, Path: "/x", MaxBody: -1}, withBody)
		},
			"postFixture: an operation's body limit is never unlimited"},
		{func() {
			Register(o, Public, Op{ID: "uploadFixture", Method: http.MethodPost, Path: "/z", MaxBody: math.MaxInt64}, withBody)
		},
			"uploadFixture: a body limit of math.MaxInt64 overflows; raise it only as far as the operation needs"},
		{func() { Register(o, Public, Op{ID: "readFixture", Method: http.MethodGet, Path: "/y"}, withBody) },
			"readFixture: a GET operation takes no request body"},
	} {
		if got := panicOf(tc.register); got != tc.want {
			t.Errorf("panic = %q, want %q", got, tc.want)
		}
	}
}

var fixtureGone = NewCode(http.StatusGone, "fixture_gone", "The fixture is gone")

func TestAnUndeclaredCodeAnswersInternal(t *testing.T) {
	var log bytes.Buffer
	r := newRouter(t, &log)
	Register(r.Owner("fixture", Public), Public, Op{ID: "fixtureGet", Method: http.MethodGet, Path: "/fixture"},
		func(context.Context, identity.Anyone, *struct{}) (*struct{}, error) { return nil, fixtureGone })
	w := get(t, r.Handler(), "/api/v1/fixture")
	if w.Code != http.StatusInternalServerError || w.Body.String() != internalBody {
		t.Fatalf("got %d %q, want 500 %q", w.Code, w.Body, internalBody)
	}
	if !strings.Contains(log.String(), `level=ERROR msg="operation fixtureGet returned undeclared code fixture_gone" method=GET path=/api/v1/fixture`) {
		t.Fatalf("log = %q", log.String())
	}
}

func TestADeclaredCodeAnswersAsItself(t *testing.T) {
	r := newRouter(t, &bytes.Buffer{})
	Register(r.Owner("fixture", Public), Public, Op{ID: "fixtureGet", Method: http.MethodGet, Path: "/fixture", Errors: []*Code{fixtureGone}},
		func(context.Context, identity.Anyone, *struct{}) (*struct{}, error) { return nil, fixtureGone })
	w := get(t, r.Handler(), "/api/v1/fixture")
	want := `{"status":410,"code":"fixture_gone","title":"The fixture is gone"}` + "\n"
	if w.Code != http.StatusGone || w.Body.String() != want || w.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("got %d %s %q, want 410 application/problem+json %q", w.Code, w.Header().Get("Content-Type"), w.Body, want)
	}
}

func TestAPlainErrorIsScrubbedAndLogged(t *testing.T) {
	var log bytes.Buffer
	r := newRouter(t, &log)
	Register(r.Owner("fixture", Public), Public, Op{ID: "fixtureGet", Method: http.MethodGet, Path: "/fixture"},
		func(context.Context, identity.Anyone, *struct{}) (*struct{}, error) {
			return nil, errors.New("secret /etc/path")
		})
	w := get(t, r.Handler(), "/api/v1/fixture")
	if w.Code != http.StatusInternalServerError || w.Body.String() != internalBody {
		t.Fatalf("got %d %q, want 500 %q", w.Code, w.Body, internalBody)
	}
	if !strings.Contains(log.String(), `level=ERROR msg="operation failed" method=GET path=/api/v1/fixture operation=fixtureGet error="secret /etc/path"`) {
		t.Fatalf("log = %q", log.String())
	}
}

func TestAnOperationWhoseClientLeftLogsAtDebug(t *testing.T) {
	var log bytes.Buffer
	r := newRouter(t, &log)
	r.services.Log = slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: noTime}))
	ctx, leave := context.WithCancel(t.Context())
	Register(r.Owner("fixture", Public), Public, Op{ID: "fixtureGet", Method: http.MethodGet, Path: "/fixture"},
		func(ctx context.Context, _ identity.Anyone, _ *struct{}) (*struct{}, error) {
			leave()
			return nil, fmt.Errorf("read: %w", ctx.Err())
		})
	r.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/fixture", http.NoBody))
	if want := "level=DEBUG msg=\"operation failed\" method=GET path=/api/v1/fixture operation=fixtureGet error=\"read: context canceled\"\n"; log.String() != want {
		t.Fatalf("log = %q, want %q", log.String(), want)
	}
}

func TestAPanicFailsOnlyItsRequest(t *testing.T) {
	var log bytes.Buffer
	r := newRouter(t, &log)
	Register(r.Owner("fixture", Public), Public, Op{ID: "fixtureGet", Method: http.MethodGet, Path: "/fixture"},
		func(context.Context, identity.Anyone, *struct{}) (*struct{}, error) { panic("fixture bug") })
	w := get(t, r.Handler(), "/api/v1/fixture")
	if w.Code != http.StatusInternalServerError || w.Body.String() != internalBody {
		t.Fatalf("got %d %q, want 500 %q", w.Code, w.Body, internalBody)
	}
	if !strings.Contains(log.String(), `level=ERROR msg="operation panicked" operation=fixtureGet method=GET path=/api/v1/fixture panic="fixture bug" stack=`) {
		t.Fatalf("log = %q", log.String())
	}
}

type fixtureInput struct {
	ID    string `path:"id" format:"uuid"`
	Count int    `query:"count" minimum:"1"`
	Tag   string `header:"X-Fixture" pattern:"^[a-z]+$"`
	Body  struct {
		Name string `json:"name" minLength:"1"`
	}
}

// The access check runs before Huma reads parameters or the body, for
// every input shape: a caller the level refuses gets its 401 or 403, never
// a 400 or 422, and its body is never read.
func TestTheGateAnswersBeforeParsing(t *testing.T) {
	r, lib := newRouterOn(t, &bytes.Buffer{})
	Register(r.Owner("fixture", Account), Account, Op{ID: "putFixture", Method: http.MethodPut, Path: "/fixture/{id}"},
		func(context.Context, identity.Account, *fixtureInput) (*struct{}, error) { return nil, nil })
	h := r.Handler()
	send := func(session string) (*httptest.ResponseRecorder, bool) {
		read := false
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/api/v1/fixture/not-a-uuid?count=0", readerFunc(func(p []byte) (int, error) {
			read = true
			return copy(p, `{"name":""}`), io.EOF
		}))
		req.Header.Set("X-Fixture", "NOT LOWER")
		req.Header.Set("Content-Type", "application/json")
		if session != "" {
			req.Header.Set("Cookie", identity.CookieName+"="+session)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w, read
	}
	w, read := send("")
	want := `{"status":401,"code":"signed_out","title":"Sign in to continue"}` + "\n"
	if w.Code != http.StatusUnauthorized || w.Body.String() != want || read {
		t.Fatalf("nobody got %d %q, body read %v; want 401 %q, unread", w.Code, w.Body, read, want)
	}
	w, read = send(identitytest.Session(t, lib, identitytest.Member(t, lib, "member@example.com")))
	if w.Code != http.StatusUnprocessableEntity || !read {
		t.Fatalf("a Member got %d %q, body read %v; want 422 after reading the body", w.Code, w.Body, read)
	}
}

func TestAPublicOperationLeavesASessionUntouched(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, lib := newRouterOn(t, &bytes.Buffer{})
		o := r.Owner("fixture", Public)
		Register(o, Public, Op{ID: "getFixture", Method: http.MethodGet, Path: "/fixture"},
			func(context.Context, identity.Anyone, *struct{}) (*struct{}, error) { return &struct{}{}, nil })
		Register(o, Account, Op{ID: "getAccountFixture", Method: http.MethodGet, Path: "/account-fixture"},
			func(context.Context, identity.Account, *struct{}) (*struct{}, error) { return &struct{}{}, nil })
		h := r.Handler()
		session := identitytest.Session(t, lib, identitytest.Member(t, lib, "member@example.com"))
		time.Sleep(2 * time.Hour)
		send := func(target string) []string {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody)
			req.Header.Set("Cookie", identity.CookieName+"="+session)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			return w.Header().Values("Set-Cookie")
		}
		if got := send("/api/v1/fixture"); got != nil {
			t.Fatalf("the public operation set %q, want no cookie", got)
		}
		want := identity.CookieName + "=" + session + "; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax"
		if got := send("/api/v1/account-fixture"); len(got) != 1 || got[0] != want {
			t.Fatalf("the Account operation set %q, want [%q]: the public one slid the session", got, want)
		}
	})
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func TestUnknownAPIPathsAnswerNotFound(t *testing.T) {
	h := newRouter(t, &bytes.Buffer{}).Handler()
	want := `{"status":404,"code":"not_found","title":"Nothing is here"}` + "\n"
	for _, target := range []string{"/api/v1/nothing", "/api/v2/health", "/api/"} {
		w := get(t, h, target)
		if w.Code != http.StatusNotFound || w.Body.String() != want {
			t.Errorf("GET %s = %d %q, want 404 %q", target, w.Code, w.Body, want)
		}
	}
}

func TestTheWebAppFallsBackToIndexAndCachesHashedAssets(t *testing.T) {
	var log bytes.Buffer
	r := newRouter(t, &log)
	r.services.Web = fstest.MapFS{
		"index.html":       {Data: []byte("index")},
		"assets/app-1a.js": {Data: []byte("app")},
	}
	h := r.Handler()
	for _, tc := range []struct{ target, body, cache string }{
		{"/", "index", "no-cache"},
		{"/library/works", "index", "no-cache"},
		{"/assets/app-1a.js", "app", "public, max-age=31536000, immutable"},
	} {
		w := get(t, h, tc.target)
		if w.Code != http.StatusOK || w.Body.String() != tc.body || w.Header().Get("Cache-Control") != tc.cache {
			t.Errorf("GET %s = %d %q Cache-Control %q, want 200 %q %q", tc.target, w.Code, w.Body, w.Header().Get("Cache-Control"), tc.body, tc.cache)
		}
	}
}

func TestAnUnembeddedWebAppPointsAtVite(t *testing.T) {
	r := newRouter(t, &bytes.Buffer{})
	r.services.Web = fstest.MapFS{}
	w := get(t, r.Handler(), "/")
	want := "the web app is not embedded in this build; use the Vite dev server\n"
	if w.Code != http.StatusNotFound || w.Body.String() != want {
		t.Fatalf("GET / = %d %q, want 404 %q", w.Code, w.Body, want)
	}
}

func TestOriginAcceptsOnlyTheCanonicalSchemeHostAndPort(t *testing.T) {
	const shape = "not scheme://host[:port]"
	for text, want := range map[string]string{
		"https://media.example.com":       "",
		"http://192.0.2.1:8484":           "",
		"http://[2001:db8::1]:8484":       "",
		"https://[2001:db8::1]":           "",
		"https://media.example.com/":      "not a canonical origin; write https://media.example.com",
		"https://media.example.com/path":  "not a canonical origin; write https://media.example.com",
		"HTTPS://Media.Example.com":       "not a canonical origin; write https://media.example.com",
		"https://media.example.com:443":   "not a canonical origin; write https://media.example.com",
		"http://192.0.2.1:80":             "not a canonical origin; write http://192.0.2.1",
		"http://192.0.2.1:":               "not a canonical origin; write http://192.0.2.1",
		"https://[2001:DB8::1]:443":       "not a canonical origin; write https://[2001:db8::1]",
		"https://media.example.com:8443/": "not a canonical origin; write https://media.example.com:8443",
		"media.example.com":               shape,
		"ftp://media.example.com":         shape,
		"https://user@media.example.com":  shape,
		"https://media.example.com?x=1":   shape,
		"https://media.example.com#frag":  shape,
		"https://":                        shape,
	} {
		var o Origin
		err := o.UnmarshalText([]byte(text))
		var got string
		if err != nil {
			got = err.Error()
		}
		if got != want {
			t.Errorf("UnmarshalText(%q) = %q, want %q", text, got, want)
		}
		if want == "" && o.String() != text {
			t.Errorf("Origin(%q).String() = %q", text, o)
		}
	}
}

func TestTheDocumentIsTheSameBytesEveryTime(t *testing.T) {
	render := func() []byte {
		r := NewRouter(Services{})
		Register(r.Owner("fixture", Public), Public, Op{ID: "getFixture", Method: http.MethodGet, Path: "/fixture"},
			func(context.Context, identity.Anyone, *struct{}) (*struct{}, error) { return nil, nil })
		return r.Document()
	}
	first := render()
	if !bytes.Equal(first, render()) || !bytes.HasSuffix(first, []byte("}\n")) {
		t.Fatal("two renders differ or the document lacks its trailing newline")
	}
	if bytes.Contains(first, []byte(`"$schema"`)) {
		t.Error("the document holds a $schema link")
	}
}

func TestHumaServesNoDocumentDocsOrSchemas(t *testing.T) {
	h := newRouter(t, &bytes.Buffer{}).Handler()
	for _, target := range []string{"/api/v1/openapi.json", "/api/v1/openapi.yaml", "/api/v1/docs", "/api/v1/schemas/Problem.json"} {
		if w := get(t, h, target); w.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, w.Code)
		}
	}
}

func panicOf(f func()) (msg string) {
	defer func() { msg = fmt.Sprint(recover()) }()
	f()
	return "<nil>"
}

func TestTheServerTakesItsTimingsAndLogsAtWarn(t *testing.T) {
	var log bytes.Buffer
	s := NewServer(http.NotFoundHandler(), DefaultTimings(), slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{ReplaceAttr: noTime})))
	if s.ReadHeaderTimeout != 10*time.Second || s.IdleTimeout != 2*time.Minute || s.WriteTimeout != 0 {
		t.Fatalf("timeouts = %v, %v, %v; want 10s, 2m0s, 0s", s.ReadHeaderTimeout, s.IdleTimeout, s.WriteTimeout)
	}
	s.ErrorLog.Print("http: TLS handshake error")
	if want := "level=WARN msg=\"http: TLS handshake error\"\n"; log.String() != want {
		t.Fatalf("log = %q, want %q", log.String(), want)
	}
}

func noTime(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}
