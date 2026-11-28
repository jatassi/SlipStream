package conversions

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/nodes"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// reconciler is what the reconciler keeps in memory: the test runs in
// flight, each holder's last signature, and the Nodes whose facts moved.
// A Controller restart forgets it and asks again.
type reconciler struct {
	mu      sync.Mutex
	testing map[role.NodeID]map[encode.Key]bool
	signed  map[role.NodeID]converting.Signature
	moved   map[role.NodeID]bool
	// applied is the Conversions results moved since the last pass.
	applied []ID
	// unasked is when each test run whose ask failed last failed.
	unasked map[test]time.Time
	// retry is when a check whose probe failed is tried again.
	retry map[ID]time.Time
	noted chan struct{}
	tests sync.WaitGroup
}

func newReconciler() *reconciler {
	return &reconciler{
		testing: map[role.NodeID]map[encode.Key]bool{}, signed: map[role.NodeID]converting.Signature{},
		moved: map[role.NodeID]bool{}, retry: map[ID]time.Time{}, unasked: map[test]time.Time{},
		noted: make(chan struct{}, 1),
	}
}

func (r *reconciler) note() {
	select {
	case r.noted <- struct{}{}:
	default:
	}
}

// Run reconciles until ctx ends: it wakes on a poke, a new Board, a
// holder's facts and its one timer.
func (c *Conversions) Run(ctx context.Context) error {
	defer c.rec.tests.Wait()
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		board := c.cfg.Board()
		select {
		case <-ctx.Done():
			return nil
		case <-c.cfg.Poke.C():
		case <-board.Changed():
		case <-timer.C:
		case <-c.rec.noted:
			c.followFacts(ctx)
			continue
		}
		next := c.cfg.Clock().Add(c.cfg.Timings.Pass)
		if due, err := c.pass(ctx, c.roster()); err != nil {
			c.cfg.Log.Warn("reconcile the Conversions", "err", err)
		} else if !due.IsZero() && due.Before(next) {
			next = due
		}
		timer.Reset(max(next.Sub(c.cfg.Clock()), 0))
	}
}

// Register sets the Importer the hand-off starts. Call it once, before Run.
func (c *Conversions) Register(im Importer) {
	if c.importer != nil {
		panic("conversions: an Importer is registered twice")
	}
	c.importer = im
}

// roster is the Converting Nodes as one pass reads them: the Board's
// holders and each holder's facts.
type roster struct {
	holders holders
	facts   func(role.NodeID) (*cv1.Facts, bool)
}

// holders is nodes.Holders, as a pass reads it.
type holders interface {
	Holds(n nodes.ID) bool
	May(n nodes.ID) bool
	Lapsed(n nodes.ID) bool
	Away(n nodes.ID) (nodes.Absence, bool)
	WrittenOff(n nodes.ID) bool
	Serving() []nodes.ID
}

func (c *Conversions) boardRoster() roster {
	return roster{holders: c.cfg.Board().Who(role.Converting), facts: c.cfg.Control().Facts}
}

// pass is one reconcile: it converges the nudged Works, puts back what
// lapsed Nodes held, places, asks test runs and checks outputs. due is
// the next time a row's wait ends.
func (c *Conversions) pass(ctx context.Context, ros roster) (due time.Time, err error) {
	c.rec.mu.Lock()
	moved := c.rec.applied
	c.rec.applied = nil
	c.rec.mu.Unlock()
	for more := true; more; {
		var ids []ID
		if ids, more, err = c.drain(ctx); err != nil {
			return time.Time{}, err
		}
		moved = append(moved, ids...)
	}
	if blocked, err := c.reconciling(ctx); err != nil || blocked {
		c.publish(ctx, moved, ros)
		return time.Time{}, err
	}
	ids, err := c.settle(ctx, ros)
	moved = append(moved, ids...)
	if err != nil {
		return time.Time{}, err
	}
	step, ids, due, err := c.place(ctx, ros)
	moved = append(moved, ids...)
	if err != nil {
		return time.Time{}, err
	}
	c.ask(ctx, step.Tests)
	ids, retry, err := c.checks(ctx)
	moved = append(moved, ids...)
	if !retry.IsZero() && (due.IsZero() || retry.Before(due)) {
		due = retry
	}
	c.publish(ctx, moved, ros)
	if due.IsZero() {
		due = c.cfg.Clock().Add(c.cfg.Timings.Pass)
	}
	return due, err
}

func (c *Conversions) reconciling(ctx context.Context) (bool, error) {
	var blocked bool
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		blocked, err = c.cfg.Reconciling(ctx, r)
		return err
	})
	return blocked, err
}

// publish tells the holders, Transfers and imports a pass's Writes, and
// publishes the Conversions they moved.
func (c *Conversions) publish(ctx context.Context, moved []ID, ros roster) {
	if len(moved) == 0 {
		return
	}
	c.cfg.Control().Touch(ros.holders.Serving()...)
	c.cfg.Transfers.Touch()
	c.importer.Wake()
	ch := Changed{ConversionIDs: []string{}, WorkIDs: []string{}, GroupIDs: []string{}}
	seen := map[ID]bool{}
	var ws []works.WorkID
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		for _, id := range moved {
			if seen[id] {
				continue
			}
			seen[id] = true
			ch.ConversionIDs = append(ch.ConversionIDs, id.String())
			if row, err := readRow(ctx, db.New(r), id); err == nil && !slices.Contains(ws, row.Work) {
				ws = append(ws, row.Work)
				ch.WorkIDs = append(ch.WorkIDs, row.Work.String())
			}
		}
		return nil
	})
	if err != nil {
		c.cfg.Log.Warn("read the Conversions a pass moved", "err", err)
	}
	c.cfg.Changed.Publish(ctx, ch)
}
