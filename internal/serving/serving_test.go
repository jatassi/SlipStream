package serving_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/serving"
)

const drain = 30 * time.Second

var (
	frontAddr  = netip.AddrPortFrom(linktest.WorkerAddr, 8484)
	clientAddr = netip.MustParseAddr("192.0.2.7")
)

// world is a Controller whose HTTP server serves Serve streams and a
// Worker whose Front serves on 127.0.0.3:8484, on one in-memory network.
type world struct {
	pair   *linktest.Pair
	worker *serving.Worker
	client *http.Client
	// events is the Controller's live stream: each value is one event,
	// written and flushed when the test sends it.
	events chan string
	// release stops the Serving run, waits for it to return and releases
	// the Role, as the supervisor does when the Worker lets Serving go.
	release func() error
}

func controllerHandler(events chan string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flush := http.NewResponseController(w).Flush
		if flush() != nil {
			return
		}
		for {
			select {
			case e := <-events:
				if _, err := fmt.Fprintf(w, "event: %s\ndata: {}\n\n", e); err != nil || flush() != nil {
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "controller %s %s host=%s forwarded=%s", r.Method, r.URL.Path, r.Host, r.Header.Get("Forwarded")) //nolint:gosec // the test reads the echo as text
	})
	return mux
}

var workerApp = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "worker %s %s", r.Method, r.URL.Path)
})

func start(t *testing.T, controllerVersion, workerVersion string) *world {
	t.Helper()
	events := make(chan string)
	ln := serving.NewListener()
	p := linktest.New(t,
		linktest.Side{Version: controllerVersion, Streams: map[link.Kind]func(link.Key, link.Stream){
			link.KindServe: func(_ link.Key, s link.Stream) { ln.Offer(s) },
		}},
		linktest.Side{Version: workerVersion},
	)
	controller := &http.Server{Handler: controllerHandler(events), ReadHeaderTimeout: time.Minute}
	go func() { _ = controller.Serve(ln) }()
	t.Cleanup(func() { _ = controller.Close() })

	w := serving.NewWorker(p.Worker.Link, p.Controller.Key, serving.Timings{Drain: drain}, slog.New(slog.DiscardHandler))
	frontLn, err := p.Worker.Host.Listen("tcp", frontAddr.String())
	if err != nil {
		t.Fatal(err)
	}
	front := &http.Server{Handler: serving.Front(workerApp, func() *serving.Worker { return w }), ReadHeaderTimeout: time.Minute}
	go func() { _ = front.Serve(frontLn) }()
	t.Cleanup(func() { _ = front.Close() })

	browser := p.Net.Host(clientAddr)
	ctx, stop := context.WithCancel(context.Background())
	ran := make(chan error, 1)
	go func() { ran <- w.Run(ctx) }()
	wd := &world{
		pair:   p,
		worker: w,
		client: &http.Client{Transport: &http.Transport{DialContext: browser.DialContext, DisableKeepAlives: true}},
		events: events,
		release: sync.OnceValue(func() error {
			stop()
			if err := <-ran; err != nil {
				return err
			}
			return w.Release(context.Background())
		}),
	}
	t.Cleanup(func() { _ = wd.release() })
	synctest.Wait()
	return wd
}

func (w *world) get(t *testing.T, path string, header ...string) (status int, body string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://seedbox.localhost:8484"+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.URL.Host = frontAddr.String()
	req.Host = "seedbox.localhost:8484"
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	res, err := w.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	read, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(read)
}

func (w *world) live(t *testing.T) (events *bufio.Reader, done func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+frontAddr.String()+"/api/v1/live", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.client.Do(req) //nolint:bodyclose // done closes it
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("the live stream through the Worker = %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	return bufio.NewReader(res.Body), func() {
		cancel()
		_ = res.Body.Close()
	}
}

func readEvent(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	var lines []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("reading the live stream after %q: %v", lines, err)
		}
		if line == "\n" {
			return strings.Join(lines, "")
		}
		lines = append(lines, line)
	}
}

// The first-PR fact "SSE through a reverse proxy over a yamux stream":
// each event reaches the browser before the Controller writes the next,
// so nothing between them buffers. The Controller sends the second event
// only once the first was read; were the proxy to hold the first, the
// bubble would deadlock.
func TestALiveStreamFlowsThroughAServingWorkerUnbuffered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t, "", "")
		w.pair.Connect(t)
		events, done := w.live(t)
		defer done()
		for _, e := range []string{"works.changed", "grabs.changed", "tasks.progress"} {
			w.events <- e
			if got, want := readEvent(t, events), "event: "+e+"\ndata: {}\n"; got != want {
				t.Fatalf("the live stream read %q, want %q", got, want)
			}
		}
	})
}

// The Controller's origin check and passkeys read the Host the browser
// used, and its limits the address the Worker names: a client's own
// Forwarded never passes.
func TestTheControllerSeesTheBrowsersHostAndAddress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t, "", "")
		w.pair.Connect(t)
		status, body := w.get(t, "/api/v1/system", "Forwarded", "for=203.0.113.9")
		if want := "controller GET /api/v1/system host=seedbox.localhost:8484 forwarded=for=192.0.2.7"; status != http.StatusOK || body != want {
			t.Fatalf("GET through the Worker = %d %q, want 200 %q", status, body, want)
		}
	})
}

// During an update the Controller and the Worker run different versions,
// and the Worker keeps serving the Controller's app.
func TestAMismatchedWorkerStillServes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t, "0.0.0-new", "0.0.0-old")
		w.pair.Connect(t)
		if status, body := w.get(t, "/movies"); status != http.StatusOK || body != "controller GET /movies host=seedbox.localhost:8484 forwarded=for=192.0.2.7" {
			t.Fatalf("a navigation through a mismatched Worker = %d %q", status, body)
		}
	})
}

func TestAnUnreachableControllerLeavesTheWorkersOwnApp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t, "", "")
		for _, c := range []struct{ path, want string }{
			{"/movies", "200 worker GET /movies"},
			{"/assets/app.js", "200 worker GET /assets/app.js"},
			{"/api/v1/system", `503 {"status":503,"code":"controller_unreachable","title":"The Controller can't be reached"}` + "\n"},
			{"/api/v1/serving", "200 worker GET /api/v1/serving"},
		} {
			if status, body := w.get(t, c.path); fmt.Sprintf("%d %s", status, body) != c.want {
				t.Errorf("GET %s with no link = %d %q, want %q", c.path, status, body, c.want)
			}
		}
		w.pair.Connect(t)
		if status, body := w.get(t, "/api/v1/serving"); status != http.StatusOK || body != "worker GET /api/v1/serving" {
			t.Errorf("getServing with the link live = %d %q, want the Worker's own", status, body)
		}
	})
}

// Unticking Serving sends new requests to the Worker's own app at once,
// lets the open ones run for serving.drain, then closes their streams.
func TestReleasingDrainsThenClosesTheStreams(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t, "", "")
		w.pair.Connect(t)
		events, done := w.live(t)
		defer done()
		begun := time.Now()
		released := make(chan error, 1)
		go func() { released <- w.release() }()
		synctest.Wait()
		if status, body := w.get(t, "/movies"); status != http.StatusOK || body != "worker GET /movies" {
			t.Fatalf("a navigation once released = %d %q, want the Worker's own app", status, body)
		}
		w.events <- "works.changed"
		if got := readEvent(t, events); got != "event: works.changed\ndata: {}\n" {
			t.Fatalf("the open stream while draining read %q", got)
		}
		var mu sync.Mutex
		var closedAt time.Time
		go func() {
			_, err := events.ReadString('\n')
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				closedAt = time.Now()
			}
		}()
		if err := <-released; err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		mu.Lock()
		defer mu.Unlock()
		if got := closedAt.Sub(begun); got != drain {
			t.Fatalf("the open stream closed %v after release, want %v", got, drain)
		}
	})
}

func TestReleasingWithNothingOpenEndsAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t, "", "")
		w.pair.Connect(t)
		if status, _ := w.get(t, "/movies"); status != http.StatusOK {
			t.Fatalf("a navigation through the Worker = %d", status)
		}
		begun := time.Now()
		if err := w.release(); err != nil {
			t.Fatal(err)
		}
		if waited := time.Since(begun); waited != 0 {
			t.Fatalf("a release with no open request waited %v", waited)
		}
	})
}

func TestTheWorkerNotesWhenItsControllerWentDown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t, "", "")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go func() { _ = w.worker.Follow(ctx) }()
		synctest.Wait()
		if since, down := w.worker.Unreachable(); !down || !since.Equal(time.Now()) {
			t.Fatalf("before the link is up, Unreachable = %v %v", since, down)
		}
		w.pair.Connect(t)
		synctest.Wait()
		if _, down := w.worker.Unreachable(); down {
			t.Fatal("with the link live the Controller reads unreachable")
		}
		time.Sleep(time.Minute)
		lost := time.Now()
		w.pair.Worker.Drop(w.pair.Controller.Key)
		synctest.Wait()
		time.Sleep(time.Minute)
		if since, down := w.worker.Unreachable(); !down || !since.Equal(lost) {
			t.Fatalf("a minute after the link went down, Unreachable = %v %v, want %v", since, down, lost)
		}
	})
}
