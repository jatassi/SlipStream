// Package roletest holds the test-only Probe Role and CheckFresh. The
// composition root never wires the Probe, so production never links it.
package roletest

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	probev1 "github.com/jatassi/slipstream/internal/pb/slipstream/probe/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Name is the Probe's Role name, outside role.Names.
const Name role.Name = "probe"

var Def = role.Def[*probev1.Intent, *probev1.Facts, *probev1.Result]{Name: Name}

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is the probe library set: each Node's intent value and the
// results applied from it, in order.
func Schema() storage.Schema[file] { return storage.NewSchema[file]("probe", schemaFiles) }

// SetIntent changes node's Probe intent, as Controller logic would.
func SetIntent(ctx context.Context, tx *storage.Tx[file], node role.NodeID, value string) error {
	return db.New(tx).SetIntent(ctx, db.SetIntentParams{NodeID: node, Value: value})
}

// Applied is every result applied from node, in order.
func Applied(ctx context.Context, r storage.Reader[file], node role.NodeID) ([]uint64, error) {
	ns, err := db.New(r).ListApplied(ctx, node)
	if err != nil {
		return nil, err
	}
	out := make([]uint64, len(ns))
	for i, n := range ns {
		out[i] = uint64(n) //nolint:gosec // tests report small counts
	}
	return out, nil
}

// Control is the Probe's Controller half.
type Control struct {
	*role.Control[*probev1.Intent, *probev1.Facts, *probev1.Result]

	mu      sync.Mutex
	pauseAt uint64
	pause   chan struct{}
}

// Serve derives a Node's intent from its probe_intent row and applies a
// result by appending it to probe_applied.
func Serve() *Control {
	c := &Control{}
	c.Control = Def.Serve(derive, c.apply)
	return c
}

// PauseAt holds the apply of result n, inside its library Write, until
// resume.
func (c *Control) PauseAt(n uint64) (resume func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := make(chan struct{})
	c.pauseAt, c.pause = n, p
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.pause = nil
		close(p)
	}
}

func derive(ctx context.Context, r storage.Reader[file], node role.NodeID) (*probev1.Intent, error) {
	value, err := db.New(r).GetIntent(ctx, node)
	if errors.Is(err, storage.ErrNoRows) {
		return &probev1.Intent{}, nil
	}
	return &probev1.Intent{Value: value}, err
}

func (c *Control) apply(ctx context.Context, tx *storage.Tx[file], node role.NodeID, r *probev1.Result) error {
	c.mu.Lock()
	p := c.pause
	if c.pauseAt != r.GetN() {
		p = nil
	}
	c.mu.Unlock()
	if p != nil {
		<-p
	}
	return db.New(tx).InsertApplied(ctx, db.InsertAppliedParams{NodeID: node, N: int64(r.GetN())}) //nolint:gosec // tests report small counts
}

// Probe is one Node's Probe holder. Its run function records each
// snapshot, and panics when the test asks.
type Probe struct {
	local *storage.DB[storage.Local]
	crash chan struct{}

	mu       sync.Mutex
	holder   role.Holder[*probev1.Intent, *probev1.Facts, *probev1.Result]
	last     *role.Snapshot[*probev1.Intent]
	starts   int
	releases int
}

func NewProbe(local *storage.DB[storage.Local]) *Probe {
	return &Probe{local: local, crash: make(chan struct{})}
}

func (p *Probe) Held() role.Held { return Def.Hold(p.run, p.release) }

func (p *Probe) release(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.releases++
	return nil
}

func (p *Probe) run(ctx context.Context, h role.Holder[*probev1.Intent, *probev1.Facts, *probev1.Result]) error {
	p.mu.Lock()
	p.holder = h
	p.starts++
	p.mu.Unlock()
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		for snap := range h.Intent(ctx) {
			p.mu.Lock()
			p.last = &snap
			p.mu.Unlock()
		}
	}()
	select {
	case <-p.crash:
		panic("the probe crashed")
	case <-ctx.Done():
		<-watched
		return fmt.Errorf("probe: %w", ctx.Err())
	}
}

// Snapshot is the last snapshot the run function saw, false before one.
func (p *Probe) Snapshot() (role.Snapshot[*probev1.Intent], bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.last == nil {
		return role.Snapshot[*probev1.Intent]{}, false
	}
	return *p.last, true
}

// Starts is how many times the supervisor started the run function.
func (p *Probe) Starts() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.starts
}

// Releases is how many times the supervisor released the Probe.
func (p *Probe) Releases() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.releases
}

// Report commits result n in its own local.db Write.
func (p *Probe) Report(ctx context.Context, n uint64) error {
	p.mu.Lock()
	h := p.holder
	p.mu.Unlock()
	if h == nil {
		return errors.New("the probe has not started")
	}
	return p.local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return h.Report(ctx, tx, &probev1.Result{N: n})
	})
}

// Crash panics the running run function.
func (p *Probe) Crash() { p.crash <- struct{}{} }

// CheckFresh fails t for each scope of a live session whose intent,
// derived now, differs from what its holder has with no Touch pending:
// Controller logic that changed a table and never called Touch. Call it
// while the cluster is idle, as after synctest.Wait. names names each
// Node in the message.
func CheckFresh(t testing.TB, c *role.Controls, names map[role.NodeID]string) {
	t.Helper()
	lags, err := c.Lagging(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(lags, func(a, b role.Lag) int { return a.Node.Compare(b.Node.UUID) })
	for _, l := range lags {
		t.Errorf("%s intent for %s changed without Touch", l.Scope, names[l.Node])
	}
}
