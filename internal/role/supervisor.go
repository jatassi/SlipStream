package role

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
)

// run is one assigned Role's goroutine, guarded by Holding.mu.
type run struct {
	cancel     context.CancelFunc
	restarting bool
	stopping   bool
}

var errReturned = errors.New("returned while the Role was still assigned")

// supervise runs each held Role while it is assigned and, for a scoped
// Role, its intent is cached, and reports the node scope's facts, until ctx ends.
func (h *Holding) supervise(ctx context.Context) {
	var roles sync.WaitGroup
	defer roles.Wait()
	var reported *nodev1.NodeFacts
	for {
		h.mu.Lock()
		changed := h.changed
		h.reconcile(ctx, &roles)
		facts := h.nodeFacts()
		h.mu.Unlock()
		if !proto.Equal(facts, reported) {
			h.setFacts(nodeScope, mustAny(facts))
			reported = facts
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return
		}
	}
}

// reconcile starts and stops Roles to match the intent. The caller holds
// h.mu.
func (h *Holding) reconcile(ctx context.Context, roles *sync.WaitGroup) {
	for _, held := range h.cfg.Held {
		r := h.runs[held.name]
		_, cached := h.cache[string(held.name)]
		want := (cached || !held.scoped) && slices.Contains(h.assigned, held.name)
		switch {
		case want && r == nil:
			rctx, cancel := context.WithCancel(ctx)
			r = &run{cancel: cancel}
			h.runs[held.name] = r
			roles.Go(func() {
				defer cancel()
				h.keep(rctx, held, r)
				h.mu.Lock()
				delete(h.runs, held.name)
				h.wake()
				h.mu.Unlock()
			})
		case !want && r != nil && !r.stopping:
			r.stopping = true
			r.cancel()
		}
	}
}

// nodeFacts is the node scope's facts. The caller holds h.mu.
func (h *Holding) nodeFacts() *nodev1.NodeFacts {
	f := &nodev1.NodeFacts{Assigned: nodeIntent(h.assigned).GetRoles(), AppliedVersion: h.cache[nodeScope].version, SettingsRevision: h.host.SettingsRevision}
	if h.host.HTTPAddress.IsValid() {
		f.HttpAddress = h.host.HTTPAddress.String()
	}
	for _, held := range h.cfg.Held {
		switch r := h.runs[held.name]; {
		case r == nil:
		case r.restarting:
			f.Restarting = append(f.Restarting, string(held.name))
		default:
			f.Running = append(f.Running, string(held.name))
		}
	}
	return f
}

// keep runs held until ctx ends, restarting it with backoff each time it
// fails, panics or returns.
func (h *Holding) keep(ctx context.Context, held Held, r *run) {
	n := 0
	for {
		h.setRestarting(r, false)
		started := time.Now()
		err := h.runOnce(ctx, held)
		if ctx.Err() != nil {
			return
		}
		if pe, ok := errors.AsType[*panicError](err); ok {
			h.cfg.Log.Error("role failed", "role", string(held.name), "err", err, "stack", string(pe.stack))
		} else {
			h.cfg.Log.Error("role failed", "role", string(held.name), "err", err)
		}
		if time.Since(started) >= h.cfg.Timings.RestartMax {
			n = 0
		}
		h.setRestarting(r, true)
		if !sleep(ctx, h.restartDelay(n)) {
			return
		}
		n++
	}
}

func (h *Holding) setRestarting(r *run, restarting bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r.restarting = restarting
	h.wake()
}

// runOnce runs held once on its own context, so whatever it started ends
// with it, and recovers a panic as an error.
func (h *Holding) runOnce(ctx context.Context, held Held) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer func() {
		if v := recover(); v != nil {
			err = &panicError{value: v, stack: debug.Stack()}
		}
	}()
	if err := held.run(ctx, h); err != nil {
		return err
	}
	return errReturned
}

type panicError struct {
	value any
	stack []byte
}

func (e *panicError) Error() string { return fmt.Sprintf("panic: %v", e.value) }
