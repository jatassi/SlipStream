package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api/apitest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// idleEngine is a torrent engine with no torrents and no goroutine. The
// libtorrent engine polls its alerts in cgo, which a synctest bubble never
// counts as blocked, so fake time would never advance.
type idleEngine struct {
	listen netip.AddrPort
	events chan engine.Event
}

var errIdleEngine = errors.New("the idle engine holds no torrents")

func (e *idleEngine) Add(engine.AddParams) (engine.Handle, error)     { return 0, errIdleEngine }
func (e *idleEngine) ConnectPeer(engine.Handle, netip.AddrPort) error { return errIdleEngine }
func (e *idleEngine) Status(engine.Handle) (engine.Status, error) {
	return engine.Status{}, errIdleEngine
}
func (e *idleEngine) Remove(engine.Handle) error          { return errIdleEngine }
func (e *idleEngine) ListenAddr() (netip.AddrPort, error) { return e.listen, nil }
func (e *idleEngine) Events() <-chan engine.Event         { return e.events }

func (e *idleEngine) Close() error {
	close(e.events)
	return nil
}

// inMemory binds every listener in memory, the control socket included,
// runs an idle torrent engine and logs to a file that never rotates.
func inMemory() (machine, *apitest.Listener) {
	httpLn := apitest.NewListener(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8484})
	return machine{
		listen: func(_ context.Context, network, address string) (net.Listener, error) {
			if network == "unix" {
				return apitest.NewListener(&net.UnixAddr{Name: address, Net: network}), nil
			}
			return httpLn, nil
		},
		openTorrent: func(c engine.Config) (engine.Engine, error) {
			return &idleEngine{listen: netip.AddrPortFrom(c.ListenInterface.Addr(), 6881), events: make(chan engine.Event)}, nil
		},
		openLog: func(dir string, stderr io.Writer) (*logs.Log, error) {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return nil, fmt.Errorf("create logs dir: %w", err)
			}
			f, err := os.OpenFile(filepath.Join(dir, logs.File), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
			if err != nil {
				return nil, fmt.Errorf("open log file: %w", err)
			}
			return logs.New(stderr, f), nil
		},
	}, httpLn
}

type client struct {
	t    *testing.T
	c    *http.Client
	base string
}

type call struct {
	method, path string
	body         io.Reader
	headers      map[string]string
	session      string
}

type reply struct {
	status int
	header http.Header
	body   string
}

func (c client) send(k call) *http.Response {
	c.t.Helper()
	body := k.body
	if body == nil {
		body = http.NoBody
	}
	req, err := http.NewRequestWithContext(c.t.Context(), k.method, c.base+k.path, body)
	if err != nil {
		c.t.Fatal(err)
	}
	if k.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for name, v := range k.headers {
		req.Header.Set(name, v)
	}
	if k.session != "" {
		req.Header.Set("Cookie", identity.CookieName+"="+k.session)
	}
	res, err := c.c.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	return res
}

func (c client) do(k call) reply {
	c.t.Helper()
	res := c.send(k)
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	return reply{status: res.StatusCode, header: res.Header, body: string(b)}
}

func (c client) claim(code, pin string) reply {
	c.t.Helper()
	return c.do(call{method: http.MethodPost, path: "/api/v1/setup/controller", body: strings.NewReader(claimBody(pin)), headers: map[string]string{"X-Setup-Code": code}})
}

func (c client) signIn(email, pin string) reply {
	c.t.Helper()
	return c.do(call{method: http.MethodPost, path: "/api/v1/session", body: strings.NewReader(`{"email":"` + email + `","pin":"` + pin + `"}`)})
}

func (c client) getSession(session string) reply {
	c.t.Helper()
	return c.do(call{method: http.MethodGet, path: "/api/v1/session", session: session})
}

func (c client) stream(session, lastEventID string) *apitest.Stream {
	c.t.Helper()
	k := call{method: http.MethodGet, path: "/api/v1/events", session: session}
	if lastEventID != "" {
		k.headers = map[string]string{"Last-Event-ID": lastEventID}
	}
	res := c.send(k) //nolint:bodyclose // the Stream closes it
	if res.StatusCode != http.StatusOK {
		c.t.Fatalf("GET /api/v1/events = %d", res.StatusCode)
	}
	s := apitest.NewStream(res.Body)
	c.t.Cleanup(func() { _ = s.Close() })
	return s
}

func claimBody(pin string) string {
	return `{"email":"admin@example.com","displayName":"Admin","pin":"` + pin + `"}`
}

func problem(status int, code, title string) string {
	return fmt.Sprintf(`{"status":%d,"code":%q,"title":%q}`, status, code, title) + "\n"
}

var (
	sessionCookie  = regexp.MustCompile(`^slipstream_session=[A-Za-z0-9_-]{43}; Path=/; Max-Age=2592000; HttpOnly; SameSite=Lax$`)
	clearingCookie = "slipstream_session=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax"
	signedOut      = problem(401, "signed_out", "Sign in to continue")
	pinWrong       = problem(401, "pin_wrong", "The email or PIN is wrong")
)

func expect(t *testing.T, r reply, status int, body string) {
	t.Helper()
	if r.status != status || r.body != body {
		t.Fatalf("got %d %q, want %d %q", r.status, r.body, status, body)
	}
}

func sessionOf(t *testing.T, r reply) string {
	t.Helper()
	cookie := r.header.Get("Set-Cookie")
	if !sessionCookie.MatchString(cookie) {
		t.Fatalf("Set-Cookie = %q, want a session cookie", cookie)
	}
	return strings.TrimPrefix(strings.Split(cookie, ";")[0], identity.CookieName+"=")
}

func next(t *testing.T, s *apitest.Stream) string {
	t.Helper()
	msg, err := s.Next()
	if err != nil {
		t.Fatalf("the stream ended: %v", err)
	}
	return msg
}

// claimedNode is what the steps of the stream proof share.
type claimedNode struct {
	c       client
	account string
	apiKey  string
	// signedIn is createSession's body for the Admin.
	signedIn string
}

// TestTheEventStreamThroughAStartedNode is the F4 stream proof: a whole Node
// started in a synctest bubble, called over its HTTP listener, through the
// spec's eleven steps in order.
func TestTheEventStreamThroughAStartedNode(t *testing.T) {
	env, _ := testEnv(t)
	synctest.Test(t, func(t *testing.T) {
		m, ln := inMemory()
		n, err := startOn(t.Context(), env, m)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			n.Stop()
			<-n.Done()
			if n.Err() != nil {
				t.Errorf("the Node stopped with %v", n.Err())
			}
		})
		cn := claimThroughTheGate(t, client{t: t, c: ln.Client(), base: "http://slipstream"}, n.Ready().SetupCode)
		a := signInAsTheAdmin(t, cn)
		streamAndSignOut(t, cn, a)
		expireAndSlide(t, cn)
		refuseCrossSiteAndOversized(t, cn)
	})
}

// claimThroughTheGate is steps 1 and 2.
func claimThroughTheGate(t *testing.T, c client, code string) claimedNode {
	expect(t, c.claim("0000-0000-0000", "1234"), 403, problem(403, "setup_code_wrong", "The Setup code is wrong"))
	expect(t, c.claim(code, "12a4"), 422,
		`{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.pin","message":"expected string to match pattern ^[0-9]{4}$"}]}`+"\n")
	typed := strings.ToLower(strings.ReplaceAll(code, "-", " "))
	body, held := io.Pipe()
	first := make(chan reply)
	go func() {
		first <- c.do(call{method: http.MethodPost, path: "/api/v1/setup/controller", body: body, headers: map[string]string{"X-Setup-Code": typed}})
	}()
	synctest.Wait()
	expect(t, c.claim(code, "1234"), 409, problem(409, "setup_busy", "Another setup is running"))
	if _, err := io.WriteString(held, claimBody("1234")); err != nil {
		t.Fatal(err)
	}
	_ = held.Close()
	claimed := <-first
	var res struct {
		Account struct {
			ID    string `json:"id"`
			Admin bool   `json:"admin"`
		} `json:"account"`
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal([]byte(claimed.body), &res); err != nil || claimed.status != http.StatusCreated || !res.Account.Admin {
		t.Fatalf("claimController with the typed code = %d %s, want 201 with an Admin", claimed.status, claimed.body)
	}
	expect(t, c.claim(code, "1234"), 409, problem(409, "already_claimed", "This Node is already set up"))
	return claimedNode{
		c:        c,
		account:  res.Account.ID,
		apiKey:   res.APIKey,
		signedIn: `{"account":{"id":"` + res.Account.ID + `","email":"admin@example.com","displayName":"Admin","admin":true}}` + "\n",
	}
}

// signInAsTheAdmin is step 3. It returns session A.
func signInAsTheAdmin(t *testing.T, cn claimedNode) string {
	expect(t, cn.c.signIn("admin@example.com", "1235"), 401, pinWrong)
	expect(t, cn.c.signIn("nobody@example.com", "1234"), 401, pinWrong)
	r := cn.c.signIn(" Admin@Example.COM ", "1234")
	expect(t, r, 200, cn.signedIn)
	return sessionOf(t, r)
}

// streamAndSignOut is steps 4 to 9 on session A's streams.
func streamAndSignOut(t *testing.T, cn claimedNode, a string) {
	c := cn.c
	streamA := c.stream(a, "")
	first := next(t, streamA)
	h, _, _ := strings.Cut(strings.TrimPrefix(first, "id: "), "\n")
	head, err := strconv.ParseInt(h, 10, 64)
	if err != nil || first != resync(head) {
		t.Fatalf("first message = %q, want id, retry 3000 and live.resync", first)
	}
	changed := func(id int64, retry string) string {
		return fmt.Sprintf("id: %d\n%sevent: identity.sessions-changed\ndata: {\"account\":%q}\n", id, retry, cn.account)
	}

	expect(t, c.signIn("admin@example.com", "1234"), 200, cn.signedIn)
	if got := next(t, streamA); got != changed(head+1, "") {
		t.Fatalf("after a second sign-in stream A read %q, want %q", got, changed(head+1, ""))
	}
	_ = streamA.Close()
	expect(t, c.signIn("admin@example.com", "1234"), 200, cn.signedIn)
	streamA = c.stream(a, strconv.FormatInt(head+1, 10))
	if got := next(t, streamA); got != changed(head+2, "retry: 3000\n") {
		t.Fatalf("the reconnected stream read %q, want %q", got, changed(head+2, "retry: 3000\n"))
	}

	for _, last := range []string{"5", "abc"} {
		s := c.stream(a, last)
		if got := next(t, s); got != resync(head+2) {
			t.Fatalf("Last-Event-ID %s: first message %q, want %q", last, got, resync(head+2))
		}
		_ = s.Close()
	}

	quiet := time.Now()
	if got := next(t, streamA); got != ": keepalive\n" || time.Since(quiet) != 25*time.Second {
		t.Fatalf("after %v of quiet stream A read %q, want a keepalive after 25s", time.Since(quiet), got)
	}

	expect(t, c.do(call{method: http.MethodGet, path: "/api/v1/events", headers: map[string]string{"X-Api-Key": cn.apiKey}}), 403,
		problem(403, "needs_account", "Only a signed-in person can do this"))

	expect(t, c.do(call{method: http.MethodDelete, path: "/api/v1/session", session: a}), 204, "")
	if got, want := next(t, streamA), "event: live.session-ended\ndata: {\"reason\":\"signed-out\"}\n"; got != want {
		t.Fatalf("after sign-out stream A read %q, want %q", got, want)
	}
	if msg, err := streamA.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("stream A read %q, %v after session-ended; want it closed", msg, err)
	}
	gone := c.getSession(a)
	expect(t, gone, 401, signedOut)
	if got := gone.header.Get("Set-Cookie"); got != clearingCookie {
		t.Fatalf("Set-Cookie = %q, want %q", got, clearingCookie)
	}
}

func resync(head int64) string {
	return fmt.Sprintf("id: %d\nretry: 3000\nevent: live.resync\ndata: {}\n", head)
}

// expireAndSlide is step 10: a session expires after 30 idle days, slides
// past lifetime/720 of use, and re-sends its cookie only when it slides.
func expireAndSlide(t *testing.T, cn claimedNode) {
	c := cn.c
	b := sessionOf(t, c.signIn("admin@example.com", "1234"))
	idle := sessionOf(t, c.signIn("admin@example.com", "1234"))
	used := func(at string, wantCookie bool) {
		t.Helper()
		r := c.getSession(b)
		if r.status != http.StatusOK || (r.header.Get("Set-Cookie") != "") != wantCookie {
			t.Fatalf("%s: getSession = %d with Set-Cookie %q, want 200 and a cookie: %v", at, r.status, r.header.Get("Set-Cookie"), wantCookie)
		}
	}
	time.Sleep(time.Hour)
	used("at the slide step", false)
	time.Sleep(time.Second)
	used("past the slide step", true)
	time.Sleep(29*24*time.Hour - time.Hour - time.Second)
	used("on day 29", true)
	time.Sleep(24*time.Hour + time.Second)
	expect(t, c.getSession(idle), 401, signedOut)
	time.Sleep(28*24*time.Hour - time.Second)
	used("on day 58", true)
}

// refuseCrossSiteAndOversized is step 11.
func refuseCrossSiteAndOversized(t *testing.T, cn claimedNode) {
	c := cn.c
	crossSite := map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}
	signIn := `{"email":"admin@example.com","pin":"1235"}`
	expect(t, c.do(call{method: http.MethodPost, path: "/api/v1/session", body: strings.NewReader(signIn), headers: crossSite}), 403,
		problem(403, "cross_origin", "Another site cannot do this"))
	crossSite["X-Api-Key"] = cn.apiKey
	expect(t, c.do(call{method: http.MethodPost, path: "/api/v1/session", body: strings.NewReader(signIn), headers: crossSite}), 401, pinWrong)
	padding := `{"email":"admin@example.com","pin":"1234","x":"`
	oversized := padding + strings.Repeat("a", 2<<20+1-len(padding)-2) + `"}`
	expect(t, c.do(call{method: http.MethodPost, path: "/api/v1/session", body: strings.NewReader(oversized)}), 413,
		problem(413, "too_large", "The request body is too large"))
	preflight := c.do(call{method: http.MethodOptions, path: "/api/v1/session", headers: map[string]string{
		"Origin": "https://evil.example", "Access-Control-Request-Method": "POST", "Access-Control-Request-Headers": "x-api-key",
	}})
	if got := preflight.header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("OPTIONS answered Access-Control-Allow-Origin %q", got)
	}
	for path, cache := range map[string]string{"/": "no-cache", "/api/v1/health": "no-store"} {
		r := c.do(call{method: http.MethodGet, path: path})
		for name, value := range securityHeaders(cache) {
			if got := r.header.Get(name); got != value {
				t.Errorf("GET %s %s = %q, want %q", path, name, got, value)
			}
		}
	}
}

func securityHeaders(cache string) map[string]string {
	return map[string]string{
		"Content-Security-Policy":      "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "no-referrer",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), microphone=(), geolocation=()",
		"Cache-Control":                cache,
	}
}

// TestTheAccessWalkPassesOnAStartedNode walks every operation in the
// document over a started, claimed Node's real listener, so every later
// spec's operations are covered with no new test.
func TestTheAccessWalkPassesOnAStartedNode(t *testing.T) {
	env, _ := testEnv(t)
	n := startTest(t, env)
	base := &url.URL{Scheme: "http", Host: n.Ready().Listeners.HTTP.String()}
	proxy := httputil.NewSingleHostReverseProxy(base)
	proxy.ErrorLog = log.New(io.Discard, "", 0)
	c := client{t: t, c: &http.Client{}, base: base.String()}
	r := c.claim(n.Ready().SetupCode, "1234")
	var res struct {
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal([]byte(r.body), &res); err != nil || r.status != http.StatusCreated {
		t.Fatalf("claimController = %d %s", r.status, r.body)
	}
	lib, err := storage.OpenLibrary(t.Context(), env.DataDir.LibraryDB(), librarySchemas(), preMigrationHook, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	admin, err := storage.ParseUUID(res.Account.ID)
	if err != nil {
		t.Fatal(err)
	}
	member := identitytest.Member(t, lib, "member@example.com")
	breaches := apitest.CheckAccess(proxy, surface.Document(), apitest.Credentials{
		Member: func() string { return identitytest.Session(t, lib, member) },
		Admin:  func() string { return identitytest.Session(t, lib, identity.AccountID{UUID: admin}) },
		APIKey: res.APIKey,
	})
	if len(breaches) != 0 {
		t.Fatalf("breaches:\n%s", strings.Join(breaches, "\n"))
	}
}
