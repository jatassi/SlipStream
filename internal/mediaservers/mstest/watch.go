package mstest

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// watching is what a holder's watch reads of the Timings.
var watching = link.Timings{Call: 10 * time.Second, KeepAlive: 30 * time.Second, ReconnectMin: time.Second, ReconnectMax: time.Minute, SessionCap: 100}

// Watch runs node's Media Server link holder, as the Role runs it: it
// holds node's intent, sent again after each change, its reports apply
// at once as node's, and its facts are node's. Stop is the holder ending
// as a Node that stops does, with no last report.
func (w *World) Watch(t *testing.T, node role.NodeID) (stop func()) {
	t.Helper()
	l := mediaserverlink.NewLink(mediaserverlink.Config{
		Kinds: mediaserverlink.Kinds, HTTP: w.net.client(), Timings: watching, Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
	})
	touched := make(chan struct{}, 1)
	w.mu.Lock()
	w.watches[node] = touched
	w.mu.Unlock()
	h := &holding{t: t, w: w, node: node, touched: touched}
	control := mediaservers.ControlService{Library: w.Lib, Sealer: w.Sealer, Nodes: caller{node: node, self: w.Home}}
	local := storagetest.Local(t, role.LocalSchema())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := l.Run(local, control)(ctx, h); err != nil {
			t.Errorf("the holder on %s stopped: %v", node, err)
		}
	}()
	stopped := false
	stop = func() {
		if stopped {
			return
		}
		stopped = true
		cancel()
		<-done
		w.mu.Lock()
		delete(w.watches, node)
		delete(w.facts, node)
		w.mu.Unlock()
	}
	t.Cleanup(stop)
	synctest.Wait()
	return stop
}

func (w *World) touch(ns ...role.NodeID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, n := range ns {
		if c, ok := w.watches[n]; ok {
			select {
			case c <- struct{}{}:
			default:
			}
		}
	}
}

func (w *World) factsOf(node role.NodeID) (*msv1.Facts, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	f, ok := w.facts[node]
	return f, ok
}

// holding is the Role's holder half for one Node.
type holding struct {
	t       *testing.T
	w       *World
	node    role.NodeID
	touched chan struct{}
}

func (h *holding) Intent(ctx context.Context) iter.Seq[role.Snapshot[*msv1.Intent]] {
	return func(yield func(role.Snapshot[*msv1.Intent]) bool) {
		for {
			var in *msv1.Intent
			err := h.w.Lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				in, err = h.w.ms.Intent(ctx, r, h.node)
				return err
			})
			if err != nil {
				return
			}
			if !yield(role.Snapshot[*msv1.Intent]{Intent: in, Standing: role.Live}) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-h.touched:
			}
		}
	}
}

func (h *holding) SetFacts(f *msv1.Facts) {
	h.w.mu.Lock()
	h.w.facts[h.node] = f
	h.w.mu.Unlock()
	h.w.ms.Noted(h.node, string(role.MediaServerLink))
}

// Report applies r at once, where the Role would apply it once the Node's
// local Write commits and its outbox reaches the Controller.
func (h *holding) Report(_ context.Context, _ *storage.Tx[storage.Local], r *msv1.Report) error {
	err := h.w.Apply(h.t.Context(), h.node, r)
	if err != nil {
		h.t.Errorf("apply %v from %s: %v", r, h.node, err)
	}
	return err
}

// Apply applies one of node's reports, as the Role's receive Write does.
func (w *World) Apply(ctx context.Context, node role.NodeID, r *msv1.Report) error {
	return w.Lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.ms.Apply(ctx, tx, node, r)
	})
}

// caller is the Node a holder calls the Controller as.
type caller struct{ node, self role.NodeID }

func (c caller) Caller(context.Context) (role.NodeID, bool) { return c.node, true }
func (c caller) Self() role.NodeID                          { return c.self }

// Plays is every play of server, by start, each as "<path> <decision>
// <reasons> by <user> on <player>, open" or "..., ended <n>s in" for one
// that ended that long after it started.
func (w *World) Plays(t *testing.T, server link.ServerID) []string {
	t.Helper()
	var out []string
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := db.New(r).ListPlays(ctx, server)
		for _, p := range rows {
			end := "open"
			if !p.EndedAt.IsZero() {
				end = fmt.Sprintf("ended %ds in", int(p.EndedAt.Time().Sub(p.StartedAt.Time()).Seconds()))
			}
			out = append(out, fmt.Sprintf("%s %s %s by %s on %s, %s", p.ServerPath, p.Decision, p.Reasons, p.UserName, p.Player, end))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// SessionsChanged is every media-servers.sessions-changed published so
// far.
func (w *World) SessionsChanged() []mediaservers.SessionsChanged {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]mediaservers.SessionsChanged(nil), w.sessions...)
}

type sessionsEvents struct{ w *World }

func (e sessionsEvents) Publish(_ context.Context, v mediaservers.SessionsChanged) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	e.w.sessions = append(e.w.sessions, v)
}
