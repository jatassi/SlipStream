package fault_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
)

// tier runs test once over an in-memory Pipe inside a synctest bubble and
// once over loopback TCP, with the same assertions. dial returns the end
// p wraps for front and its peer.
type tier struct {
	name string
	run  func(t *testing.T, test func(t *testing.T, dial func(p *fault.Plan) (wrapped, peer net.Conn)))
}

var front = fault.Front{Target: "tracker", Origin: "home", Layers: fault.Conn | fault.HTTP}

var tiers = []tier{
	{"pipe", func(t *testing.T, test func(*testing.T, func(*fault.Plan) (net.Conn, net.Conn))) {
		synctest.Test(t, func(t *testing.T) {
			test(t, func(p *fault.Plan) (net.Conn, net.Conn) {
				a, b := fault.Pipe(&net.TCPAddr{}, &net.TCPAddr{})
				wrapped := p.Conn(a, front)
				t.Cleanup(func() { _ = wrapped.Close(); _ = b.Close() })
				return wrapped, b
			})
		})
	}},
	{"tcp", func(t *testing.T, test func(*testing.T, func(*fault.Plan) (net.Conn, net.Conn))) {
		test(t, func(p *fault.Plan) (net.Conn, net.Conn) {
			l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = l.Close() }()
			peer, err := new(net.Dialer).DialContext(t.Context(), "tcp", l.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			wrapped, err := p.Listener(l, front).Accept()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = wrapped.Close(); _ = peer.Close() })
			return wrapped, peer
		})
	}},
}

func eachTier(t *testing.T, test func(t *testing.T, dial func(*fault.Plan) (net.Conn, net.Conn))) {
	for _, tr := range tiers {
		t.Run(tr.name, func(t *testing.T) { tr.run(t, test) })
	}
}

func plan(t *testing.T, f fault.Fault) *fault.Plan {
	t.Helper()
	p := fault.NewPlan([]fault.Front{front}, nil)
	arm(t, p, fault.Rule{ID: "rule", Target: "tracker", Fault: f})
	return p
}

func TestResetPlays(t *testing.T) {
	eachTier(t, func(t *testing.T, dial func(*fault.Plan) (net.Conn, net.Conn)) {
		p := fault.NewPlan([]fault.Front{front}, nil)
		wrapped, peer := dial(p)
		arm(t, p, fault.Rule{ID: "rule", Target: "tracker", Fault: fault.Reset{}})
		if _, err := wrapped.Read(make([]byte, 1)); !errors.Is(err, fault.ErrReset) {
			t.Fatalf("the reset end read %v, want ErrReset", err)
		}
		if n, err := peer.Read(make([]byte, 1)); n != 0 || err == nil {
			t.Fatalf("the peer read %d, %v; want a failure", n, err)
		}
		again, _ := dial(p)
		if _, err := again.Write([]byte("x")); !errors.Is(err, fault.ErrReset) {
			t.Fatalf("a new connection wrote %v, want ErrReset", err)
		}
	})
}

func TestStallPlays(t *testing.T) {
	eachTier(t, func(t *testing.T, dial func(*fault.Plan) (net.Conn, net.Conn)) {
		p := plan(t, fault.Stall{})
		wrapped, peer := dial(p)
		write(t, wrapped, "lost")
		write(t, peer, "held")
		held := readAsync(wrapped)
		if err := peer.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		if n, err := peer.Read(make([]byte, 8)); n != 0 || !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("under Stall the peer read %d, %v; want nothing until its deadline", n, err)
		}
		select {
		case r := <-held:
			t.Fatalf("under Stall a read returned %+v", r)
		default:
		}
		p.Clear("rule")
		if r := <-held; r != (read{"held", nil}) {
			t.Fatalf("after the clear the read returned %+v, want held", r)
		}
		if err := peer.SetReadDeadline(time.Time{}); err != nil {
			t.Fatal(err)
		}
		write(t, wrapped, "after")
		if r := <-readAsync(peer); r != (read{"after", nil}) {
			t.Fatalf("after the clear the peer read %+v, want after", r)
		}
	})
}

func TestLatencyPlays(t *testing.T) {
	eachTier(t, func(t *testing.T, dial func(*fault.Plan) (net.Conn, net.Conn)) {
		wrapped, peer := dial(plan(t, fault.Latency(100*time.Millisecond)))
		for _, way := range [][2]net.Conn{{wrapped, peer}, {peer, wrapped}} {
			start := time.Now()
			got := readAsync(way[1])
			write(t, way[0], "ping")
			if r := <-got; r != (read{"ping", nil}) {
				t.Fatalf("read %+v", r)
			}
			if d := time.Since(start); d < 100*time.Millisecond || d > 2*time.Second {
				t.Fatalf("delivered after %v, want 100ms", d)
			}
		}
	})
}

func TestThrottlePlaysAndAClearSpeedsUpALiveConnection(t *testing.T) {
	const rate = 16 << 10
	eachTier(t, func(t *testing.T, dial func(*fault.Plan) (net.Conn, net.Conn)) {
		p := plan(t, fault.Throttle(rate))
		wrapped, peer := dial(p)
		for _, way := range [][2]net.Conn{{wrapped, peer}, {peer, wrapped}} {
			start := time.Now()
			go func() { _, _ = way[0].Write(make([]byte, rate/2)) }()
			if _, err := io.ReadFull(way[1], make([]byte, rate/2)); err != nil {
				t.Fatal(err)
			}
			if d := time.Since(start); d < 7*time.Second/16 || d > 3*time.Second {
				t.Fatalf("half a second of bytes at the rate arrived after %v", d)
			}
		}

		start := time.Now()
		go func() { _, _ = wrapped.Write(make([]byte, 4*rate)) }()
		if _, err := io.ReadFull(peer, make([]byte, rate/4)); err != nil {
			t.Fatal(err)
		}
		p.Clear("rule")
		if _, err := io.ReadFull(peer, make([]byte, 4*rate-rate/4)); err != nil {
			t.Fatal(err)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Fatalf("four seconds of bytes took %v with the throttle cleared after a quarter second", d)
		}
	})
}

func TestCutPlaysEachWay(t *testing.T) {
	eachTier(t, func(t *testing.T, dial func(*fault.Plan) (net.Conn, net.Conn)) {
		p := plan(t, fault.Cut(5))
		wrapped, peer := dial(p)
		got := make(chan string, 1)
		go func() {
			b, _ := io.ReadAll(peer)
			got <- string(b)
		}()
		write(t, wrapped, "hel")
		if n, err := wrapped.Write([]byte("lo world")); n != 2 || !errors.Is(err, fault.ErrCut) {
			t.Fatalf("Write = %d, %v; want 2, ErrCut", n, err)
		}
		if s := <-got; s != "hello" {
			t.Fatalf("the peer read %q, want hello", s)
		}

		wrapped, peer = dial(p)
		write(t, peer, "hello world")
		b, err := io.ReadAll(wrapped)
		if string(b) != "hello" || !errors.Is(err, fault.ErrCut) {
			t.Fatalf("the cut end read %q, %v; want hello, ErrCut", b, err)
		}
	})
}

// genuine answers 510 bytes of JSON and counts the requests it served.
type genuine struct{ served atomic.Int32 }

var genuineBody = `{"pad":"` + strings.Repeat("a", 500) + `"}`

func (h *genuine) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.served.Add(1)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(genuineBody))
}

// reply is one answer as the client saw it, its body read to the end.
type reply struct {
	status        string
	header        http.Header
	contentLength int64
	body          string
	bodyErr       error
	headersAfter  time.Duration
	after         time.Duration
}

// httpTiers serves h over an in-memory Network inside a synctest bubble
// and over loopback TCP, and gives test a get that reaches it.
func httpTiers(t *testing.T, h http.Handler, test func(t *testing.T, get func() (reply, error))) {
	start := func(t *testing.T, l net.Listener, dial func(ctx context.Context, network, addr string) (net.Conn, error)) func() (reply, error) {
		srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
		done := make(chan error, 1)
		go func() { done <- srv.Serve(l) }()
		tr := &http.Transport{DialContext: dial}
		t.Cleanup(func() {
			tr.CloseIdleConnections()
			_ = srv.Close()
			if err := <-done; !errors.Is(err, http.ErrServerClosed) {
				t.Errorf("serve: %v", err)
			}
		})
		client := &http.Client{Transport: tr}
		return func() (reply, error) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+l.Addr().String()+"/announce", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			began := time.Now()
			res, err := client.Do(req)
			if err != nil {
				return reply{}, fmt.Errorf("GET /announce: %w", err)
			}
			defer func() { _ = res.Body.Close() }()
			r := reply{status: res.Status, header: res.Header, contentLength: res.ContentLength, headersAfter: time.Since(began)}
			body, err := io.ReadAll(res.Body)
			r.body, r.bodyErr, r.after = string(body), err, time.Since(began)
			return r, nil
		}
	}
	t.Run("pipe", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			n := fault.NewNetwork()
			l, err := n.Host(seedbox).Listen("tcp", "127.0.0.3:443")
			if err != nil {
				t.Fatal(err)
			}
			test(t, start(t, l, n.Host(home).DialContext))
		})
	})
	t.Run("tcp", func(t *testing.T) {
		l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		test(t, start(t, l, new(net.Dialer).DialContext))
	})
}

// withRetry is the reply's status, its retry headers and its body.
func (r reply) withRetry() string {
	var header []string
	for _, k := range []string{"Retry-After", "X-RateLimit-Reset"} {
		if v := r.header.Get(k); v != "" {
			header = append(header, k+": "+v)
		}
	}
	return fmt.Sprintf("%s %v %s", r.status, header, r.body)
}

func faulted(t *testing.T, f fault.Fault, h http.Handler) http.Handler {
	return plan(t, f).HTTP("tracker", fault.Route{Name: "announce"}, func(*http.Request) string { return "home" }, h)
}

func TestStatusAnswersWithoutTheHandler(t *testing.T) {
	for _, c := range []struct {
		fault, want string
	}{
		{"status=404", "404 Not Found [] Not Found\n"},
		{"status=503,retry-after=1s", "503 Service Unavailable [Retry-After: 1] Service Unavailable\n"},
		{"status=429,retry-after=30s,reset-header=X-RateLimit-Reset", "429 Too Many Requests [Retry-After: 30 X-RateLimit-Reset: 30] Too Many Requests\n"},
	} {
		t.Run(c.fault, func(t *testing.T) {
			f, err := fault.Parse(c.fault)
			if err != nil {
				t.Fatal(err)
			}
			h := &genuine{}
			httpTiers(t, faulted(t, f, h), func(t *testing.T, get func() (reply, error)) {
				r, err := get()
				if err != nil {
					t.Fatal(err)
				}
				if got := r.withRetry(); got != c.want || h.served.Load() != 0 {
					t.Fatalf("answered %q after %d calls to the handler, want %q and none", got, h.served.Load(), c.want)
				}
			})
		})
	}
}

func TestMalformedKeepsTheStatusAndType(t *testing.T) {
	h := &genuine{}
	httpTiers(t, faulted(t, fault.Malformed{}, h), func(t *testing.T, get func() (reply, error)) {
		r, err := get()
		if err != nil {
			t.Fatal(err)
		}
		if got := r.status + " " + r.header.Get("Content-Type") + " " + r.body; got != "200 OK application/json \x00malformed" || json.Valid([]byte(r.body)) {
			t.Fatalf("answered %q", got)
		}
	})
}

func TestContentTypeKeepsTheStatusAndBody(t *testing.T) {
	httpTiers(t, faulted(t, fault.ContentType("text/html"), &genuine{}), func(t *testing.T, get func() (reply, error)) {
		r, err := get()
		if err != nil {
			t.Fatal(err)
		}
		if got := r.status + " " + r.header.Get("Content-Type"); got != "200 OK text/html" || r.body != genuineBody || r.bodyErr != nil {
			t.Fatalf("answered %q with %d bytes and %v", got, len(r.body), r.bodyErr)
		}
	})
}

func TestHangNeverAnswers(t *testing.T) {
	h := &genuine{}
	httpTiers(t, faulted(t, fault.Hang{}, h), func(t *testing.T, get func() (reply, error)) {
		start := time.Now()
		if _, err := get(); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("a hung request ended with %v, want the client's deadline", err)
		}
		if d := time.Since(start); d < time.Second || h.served.Load() != 0 {
			t.Fatalf("a hung request gave up after %v, reaching the handler %d times", d, h.served.Load())
		}
	})
}

func TestTruncateSendsHalfTheDeclaredBody(t *testing.T) {
	httpTiers(t, faulted(t, fault.Truncate{}, &genuine{}), func(t *testing.T, get func() (reply, error)) {
		r, err := get()
		if err != nil {
			t.Fatal(err)
		}
		if r.contentLength != int64(len(genuineBody)) || r.body != genuineBody[:len(genuineBody)/2] || !errors.Is(r.bodyErr, io.ErrUnexpectedEOF) {
			t.Fatalf("declared %d, read %d bytes and %v; want %d declared, half sent, then an unexpected EOF", r.contentLength, len(r.body), r.bodyErr, len(genuineBody))
		}
	})
}

func TestSlowBodySendsHeadersAtOnce(t *testing.T) {
	httpTiers(t, faulted(t, fault.SlowBody(1<<10), &genuine{}), func(t *testing.T, get func() (reply, error)) {
		r, err := get()
		if err != nil || r.body != genuineBody || r.bodyErr != nil {
			t.Fatalf("read %d bytes, %v, %v", len(r.body), err, r.bodyErr)
		}
		if r.headersAfter > 300*time.Millisecond || r.after < 450*time.Millisecond {
			t.Fatalf("headers after %v and 510 bytes at 1KiB/s after %v", r.headersAfter, r.after)
		}
	})
}

func TestAnHTTPRuleMatchesItsRouteParamsAndOrigin(t *testing.T) {
	p := fault.NewPlan([]fault.Front{front, {Target: "tracker", Origin: "seedbox", Layers: fault.HTTP}}, nil)
	var origin string
	mux := http.NewServeMux()
	mux.Handle("GET /artist/{mbid}", p.HTTP("tracker", fault.Route{Name: "artist", Params: []string{"mbid"}}, func(*http.Request) string { return origin }, ok))
	mux.Handle("GET /artist", p.HTTP("tracker", fault.Route{Name: "search"}, func(*http.Request) string { return origin }, ok))
	arm(t, p, fault.Rule{ID: "one", Target: "tracker", Origin: "home", Route: "artist", Params: map[string]string{"mbid": "a1"}, Fault: fault.Status{Code: 503}})
	for _, c := range []struct {
		origin, route, path string
		want                int
	}{
		{"home", "artist", "/artist/a1", 503},
		{"home", "artist", "/artist/b2", 200},
		{"home", "search", "/artist", 200},
		{"seedbox", "artist", "/artist/a1", 200},
	} {
		origin = c.origin
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, c.path, http.NoBody))
		if rec.Code != c.want {
			t.Errorf("%s from %s answered %d, want %d", c.path, c.origin, rec.Code, c.want)
		}
	}
}
