// Package serving is the Serving Role on a Worker: while the Worker holds
// it, every path of its HTTP handler is the Controller's, proxied over
// Serve streams on the link, and the Controller serves those streams with
// its own root handler.
package serving

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/link"
)

type Timings struct {
	// Drain is how long open requests may finish once a Worker stops
	// serving, before its streams close.
	Drain time.Duration `yaml:"drain"`
}

func DefaultTimings() Timings { return Timings{Drain: 30 * time.Second} }

// StatusPath is getServing's path, which a Worker always answers itself.
const StatusPath = api.Prefix + "/serving"

// ControllerUnreachable answers an API request a Serving Worker could not
// send to its Controller. It is no operation's code.
var ControllerUnreachable = api.NewCode(http.StatusServiceUnavailable, "controller_unreachable", "The Controller can't be reached")

// Link is the Worker's link to its Controller.
type Link interface {
	OpenStream(ctx context.Context, key link.Key, kind link.Kind) (link.Stream, error)
	State(key link.Key) link.State
	Subscribe(ctx context.Context) <-chan link.Change
}

// Worker is a Worker's Serving: whether it serves, the requests it has
// open, and the Serve streams they use.
type Worker struct {
	link       Link
	controller link.Key
	timings    Timings
	transport  *http.Transport
	log        *slog.Logger

	mu      sync.Mutex
	serving bool
	active  int
	// idle is closed once active drops to zero, while a release waits.
	idle    chan struct{}
	streams map[link.Stream]struct{}
	// down is when the link to the Controller went down, zero while live.
	down time.Time
}

func NewWorker(l Link, controller link.Key, t Timings, log *slog.Logger) *Worker {
	w := &Worker{link: l, controller: controller, timings: t, log: log, streams: map[link.Stream]struct{}{}}
	w.transport = &http.Transport{
		DialContext:         w.dial,
		DisableCompression:  true,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     90 * time.Second,
	}
	return w
}

// Run is the Serving Role's run: the Worker serves through its Controller
// until ctx ends.
func (w *Worker) Run(ctx context.Context) error {
	w.serve(true)
	<-ctx.Done()
	w.serve(false)
	return nil
}

// Release lets the open requests finish for serving.drain, then closes
// every Serve stream. Live clients reconnect to the Worker itself.
func (w *Worker) Release(ctx context.Context) error {
	w.mu.Lock()
	var idle chan struct{}
	if w.active > 0 {
		w.idle = make(chan struct{})
		idle = w.idle
	}
	w.mu.Unlock()
	if idle != nil {
		drained := time.NewTimer(w.timings.Drain)
		defer drained.Stop()
		select {
		case <-idle:
		case <-drained.C:
		case <-ctx.Done():
		}
	}
	w.closeStreams()
	return nil
}

// Close stops serving and closes every Serve stream at once, as the Node
// stops.
func (w *Worker) Close() {
	w.serve(false)
	w.closeStreams()
}

// Follow notes when the link to the Controller goes down, until ctx ends.
func (w *Worker) Follow(ctx context.Context) error {
	changes := w.link.Subscribe(ctx)
	w.note(w.link.State(w.controller))
	for c := range changes {
		if c.Key == w.controller {
			w.note(c.State)
		}
	}
	return nil
}

// Unreachable is since when the Controller can't be reached, false while
// the link to it is live.
func (w *Worker) Unreachable() (time.Time, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.down, !w.down.IsZero()
}

func (w *Worker) note(s link.State) {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch s.(type) {
	case link.Live:
		w.down = time.Time{}
	case link.Down:
		if w.down.IsZero() {
			w.down = time.Now()
		}
	}
}

func (w *Worker) serve(on bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.serving = on
}

func (w *Worker) begin() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.serving {
		return false
	}
	w.active++
	return true
}

func (w *Worker) end() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.active--
	if w.active == 0 && w.idle != nil {
		close(w.idle)
		w.idle = nil
	}
}

func (w *Worker) dial(ctx context.Context, _, _ string) (net.Conn, error) {
	s, err := w.link.OpenStream(ctx, w.controller, link.KindServe)
	if err != nil {
		return nil, err
	}
	w.mu.Lock()
	w.streams[s] = struct{}{}
	w.mu.Unlock()
	return stream{Stream: s, w: w}, nil
}

func (w *Worker) closeStreams() {
	w.transport.CloseIdleConnections()
	w.mu.Lock()
	streams := w.streams
	w.streams = map[link.Stream]struct{}{}
	w.mu.Unlock()
	for s := range streams {
		_ = s.Reset()
	}
}

// stream is a Serve stream the Worker forgets once the transport closes
// it.
type stream struct {
	link.Stream
	w *Worker
}

func (s stream) Close() error {
	s.w.mu.Lock()
	delete(s.w.streams, s.Stream)
	s.w.mu.Unlock()
	return s.Stream.Close() //nolint:wrapcheck // the transport closes its own conn
}

// Front serves every request through the Controller while the Worker that
// worker returns serves, and with local otherwise. getServing is always
// local. When the Controller can't be reached, an API request answers 503
// controller_unreachable and any other gets local, the Worker's own app,
// whose boot shows the status card.
func Front(local http.Handler, worker func() *Worker) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		w := worker()
		if w == nil || req.URL.Path == StatusPath || !w.begin() {
			local.ServeHTTP(rw, req)
			return
		}
		defer w.end()
		proxy := &httputil.ReverseProxy{
			Rewrite:       forward,
			Transport:     w.transport,
			FlushInterval: -1,
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				w.log.Debug("serve through the Controller", "method", req.Method, "path", req.URL.Path, "err", err)
				if strings.HasPrefix(req.URL.Path, "/api/") {
					ControllerUnreachable.ServeHTTP(rw, req)
					return
				}
				local.ServeHTTP(rw, req)
			},
		}
		proxy.ServeHTTP(rw, req)
	})
}

// forward keeps the request's Host, so the Controller's origin check and
// passkeys see the address the browser used, and names the client in
// Forwarded.
func forward(r *httputil.ProxyRequest) {
	r.Out.URL.Scheme = "http"
	r.Out.URL.Host = "controller"
	if peer, err := netip.ParseAddrPort(r.In.RemoteAddr); err == nil {
		r.Out.Header.Set("Forwarded", api.Forwarded(peer.Addr().Unmap()))
	}
}

// Listener is the Controller's end of Serve streams: each one is a
// connection its HTTP server accepts.
type Listener struct {
	conns  chan net.Conn
	done   chan struct{}
	closed sync.Once
}

func NewListener() *Listener {
	return &Listener{conns: make(chan net.Conn), done: make(chan struct{})}
}

// Offer hands s to the server, or resets it once the listener closed.
func (l *Listener) Offer(s link.Stream) {
	select {
	case l.conns <- s:
	case <-l.done:
		_ = s.Reset()
	}
}

func (l *Listener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *Listener) Close() error {
	l.closed.Do(func() { close(l.done) })
	return nil
}

func (l *Listener) Addr() net.Addr { return linkAddr{} }

type linkAddr struct{}

func (linkAddr) Network() string { return "link" }
func (linkAddr) String() string  { return "link" }
