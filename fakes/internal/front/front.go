// Package front is a Fake world's fronts: listeners on 127.0.0.1 the
// world owns for a whole run, one for each target as one origin reaches
// it. A front wraps each connection it accepts in the fault Plan's
// connection faults and logs it as front.open and front.close, then hands
// it on. A served front terminates TLS and pushes the plaintext into a
// fake's Inbox; a forwarded front copies it both ways to an upstream
// process.
package front

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
)

// The kinds a front writes.
const (
	Open  = "front.open"
	Close = "front.close"
)

// Why a connection closed, in its front.close record.
const (
	Closed       = "closed"
	Reset        = "reset"
	Cut          = "cut"
	WorldStopped = "world stopped"
)

// Set is a run's fronts and the counter that numbers their connections.
type Set struct {
	plan   *fault.Plan
	record func(events.Record)
	log    *slog.Logger
	conns  atomic.Uint64

	mu     sync.Mutex
	fronts []*Front
}

// NewSet makes fronts that play plan's faults and hand each record to
// record.
func NewSet(plan *fault.Plan, record func(events.Record), log *slog.Logger) *Set {
	return &Set{plan: plan, record: record, log: log}
}

// Handler takes each connection a front accepts, wrapped in its faults.
// ctx ends when the front closes.
type Handler func(ctx context.Context, c net.Conn)

// Front is one listener. It accepts nothing until Serve.
type Front struct {
	set   *Set
	front fault.Front
	addr  netip.AddrPort

	mu     sync.Mutex
	l      net.Listener
	handle Handler
	live   map[*tracked]struct{}
	closed bool
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Listen binds a front for f on 127.0.0.1:0.
func (s *Set) Listen(ctx context.Context, f fault.Front) (*Front, error) {
	l, err := new(net.ListenConfig).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("front %s@%s: %w", f.Target, f.Origin, err)
	}
	fr := &Front{set: s, front: f, addr: netip.MustParseAddrPort(l.Addr().String()), l: l, live: map[*tracked]struct{}{}}
	fr.ctx, fr.cancel = context.WithCancel(context.WithoutCancel(ctx))
	s.mu.Lock()
	s.fronts = append(s.fronts, fr)
	s.mu.Unlock()
	return fr, nil
}

func (f *Front) Addr() netip.AddrPort { return f.addr }

func (f *Front) Fault() fault.Front { return f.front }

// Serve hands each connection to h from now on, and follows the Plan's
// refuse rules: a front a rule refuses stops listening, and binds the same
// port again once the rule clears.
func (f *Front) Serve(h Handler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handle = h
	f.acceptLocked(f.l)
	f.wg.Go(f.follow)
}

func (f *Front) acceptLocked(l net.Listener) {
	f.wg.Go(func() {
		for {
			c, err := l.Accept()
			if err != nil {
				f.mu.Lock()
				current := f.l == l && !f.closed
				f.mu.Unlock()
				if current {
					f.set.log.Error("front stopped accepting", "target", string(f.front.Target), "origin", f.front.Origin, "err", err.Error())
				}
				return
			}
			f.accept(c)
		}
	})
}

// follow closes and binds the listener as refuse rules come and go.
func (f *Front) follow() {
	for {
		refused, changed := f.set.plan.Refused(f.front)
		f.mu.Lock()
		switch {
		case f.closed:
			f.mu.Unlock()
			return
		case refused && f.l != nil:
			_ = f.l.Close()
			f.l = nil
		case !refused && f.l == nil:
			l, err := new(net.ListenConfig).Listen(context.Background(), "tcp", f.addr.String())
			if err != nil {
				f.set.log.Error("front could not bind its port again", "target", string(f.front.Target), "origin", f.front.Origin, "addr", f.addr.String(), "err", err.Error())
			} else {
				f.l = l
				f.acceptLocked(l)
			}
		}
		f.mu.Unlock()
		select {
		case <-changed:
		case <-f.ctx.Done():
			return
		}
	}
}

type opened struct {
	Conn   uint64       `json:"conn"`
	Target fault.Target `json:"target"`
}

type closed struct {
	Conn   uint64       `json:"conn"`
	Target fault.Target `json:"target"`
	// In is the bytes the dialer sent, Out the bytes it was sent.
	In    int64  `json:"in"`
	Out   int64  `json:"out"`
	Cause string `json:"cause"`
}

func (f *Front) accept(c net.Conn) {
	t := &tracked{front: f, id: f.set.conns.Add(1)}
	f.set.record(events.Record{Kind: Open, Node: f.front.Origin, Data: data(opened{Conn: t.id, Target: f.front.Target})})
	t.app = &app{Conn: f.set.plan.Conn(&wire{Conn: c, t: t}, f.front), t: t}
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		t.end(WorldStopped)
		_ = t.app.Close()
		return
	}
	f.live[t] = struct{}{}
	f.wg.Go(func() { f.handle(f.ctx, t.app) })
	f.mu.Unlock()
}

// Close stops the front and closes each connection it still carries.
func (f *Front) Close() {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	f.cancel()
	if f.l != nil {
		_ = f.l.Close()
	}
	live := make([]*tracked, 0, len(f.live))
	for t := range f.live {
		live = append(live, t)
	}
	f.mu.Unlock()
	for _, t := range live {
		t.end(WorldStopped)
		_ = t.app.Close()
	}
	f.wg.Wait()
}

func (s *Set) Close() {
	s.mu.Lock()
	fronts := s.fronts
	s.mu.Unlock()
	for _, f := range fronts {
		f.Close()
	}
}

// tracked is one connection: the bytes that crossed its socket and why it
// closed.
type tracked struct {
	front   *Front
	id      uint64
	in, out atomic.Int64
	app     *app

	once     sync.Once
	cause    string
	recorded sync.Once
}

// end notes cause as the reason the connection closes, unless one is
// already noted.
func (t *tracked) end(cause string) {
	t.once.Do(func() { t.cause = cause })
}

func (t *tracked) closed() {
	t.end(Closed)
	f := t.front
	f.mu.Lock()
	delete(f.live, t)
	f.mu.Unlock()
	t.recorded.Do(func() {
		f.set.record(events.Record{Kind: Close, Node: f.front.Origin, Data: data(closed{Conn: t.id, Target: f.front.Target, In: t.in.Load(), Out: t.out.Load(), Cause: t.cause})})
	})
}

// wire counts the bytes on the socket, beneath the faults.
type wire struct {
	net.Conn
	t *tracked
}

func (w *wire) Read(b []byte) (int, error) {
	n, err := w.Conn.Read(b)
	w.t.in.Add(int64(n))
	return n, err //nolint:wrapcheck // the socket's own error, as a net.Conn's caller expects
}

func (w *wire) Write(b []byte) (int, error) {
	n, err := w.Conn.Write(b)
	w.t.out.Add(int64(n))
	return n, err //nolint:wrapcheck // the socket's own error, as a net.Conn's caller expects
}

// SetLinger lets the fault Plan reset the socket beneath this wrapper.
func (w *wire) SetLinger(sec int) error {
	if tcp, ok := w.Conn.(*net.TCPConn); ok {
		return tcp.SetLinger(sec) //nolint:wrapcheck // the socket's own error
	}
	return nil
}

// app is the connection the handler holds, above the faults. It notes the
// first way the connection failed as its cause.
type app struct {
	net.Conn
	t *tracked
}

func (a *app) NetConn() net.Conn { return a.Conn }

func (a *app) Read(b []byte) (int, error) {
	n, err := a.Conn.Read(b)
	a.fail(err)
	return n, err //nolint:wrapcheck // the fault wrapper's own error, as a net.Conn's caller expects
}

func (a *app) Write(b []byte) (int, error) {
	n, err := a.Conn.Write(b)
	a.fail(err)
	return n, err //nolint:wrapcheck // the fault wrapper's own error, as a net.Conn's caller expects
}

func (a *app) fail(err error) {
	switch {
	case err == nil:
	case errors.Is(err, io.EOF):
		a.t.end(Closed)
	case errors.Is(err, fault.ErrReset):
		a.t.end(Reset)
	case errors.Is(err, fault.ErrCut):
		a.t.end(Cut)
	case errors.Is(err, net.ErrClosed), errors.Is(err, os.ErrDeadlineExceeded):
	default:
		a.t.end(err.Error())
	}
}

func (a *app) Close() error {
	err := a.Conn.Close()
	a.t.closed()
	return err //nolint:wrapcheck // the fault wrapper's own error
}

func data(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("front: " + err.Error())
	}
	return b
}
