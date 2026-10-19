package tasks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks/internal/db"
)

type openRun struct {
	id      RunID
	task    Task
	trigger Trigger
	started storage.Time
	units   []Unit
}

var errCanceled = errors.New("an Admin canceled the run")

const progressEvery = 500 * time.Millisecond

type active struct {
	run      openRun
	cancel   context.CancelCauseFunc
	progress *progress
}

func (s *Scheduler) start(ctx context.Context, r openRun) {
	runCtx, cancel := context.WithCancelCause(ctx)
	p := newProgress(ctx, s, r)
	s.mu.Lock()
	s.running[r.id] = &active{run: r, cancel: cancel, progress: p}
	s.mu.Unlock()
	s.runs.Go(func() {
		defer cancel(nil)
		s.run(ctx, runCtx, r, p)
		s.mu.Lock()
		delete(s.running, r.id)
		s.mu.Unlock()
		s.Wake()
	})
}

// Cancel cancels the open run id. It addresses a run, not a task, so a
// late Cancel never stops a newer run.
func (s *Scheduler) Cancel(id RunID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.running[id]
	if !ok {
		return ErrNotOpen
	}
	a.cancel(errCanceled)
	return nil
}

func (s *Scheduler) run(nodeCtx, runCtx context.Context, r openRun, p *progress) {
	if r.task.Prepare != nil {
		var err error
		if r, err = s.prepare(nodeCtx, runCtx, r, p); err != nil {
			if nodeCtx.Err() == nil {
				s.cfg.Log.Error("list a run's units", "task", r.task.ID, "run", r.id, "err", err)
				s.finish(nodeCtx, r, RunFailed, "0 "+r.task.Noun.Many, err.Error(), nil)
			}
			return
		}
	}
	results := make([]result, len(r.units))
	placed := make([]role.NodeID, len(r.units))
	byNode := map[role.NodeID][]int{}
	var order []role.NodeID
	for i, u := range r.units {
		node := s.nodeOf(u.Place)
		placed[i] = node
		if _, seen := byNode[node]; !seen {
			order = append(order, node)
		}
		byNode[node] = append(byNode[node], i)
	}
	var wg sync.WaitGroup
	for _, node := range order {
		wg.Go(func() {
			for _, i := range byNode[node] {
				if nodeCtx.Err() != nil {
					return
				}
				results[i] = s.unit(nodeCtx, runCtx, r, r.units[i], node, p)
				p.finished()
			}
		})
	}
	wg.Wait()
	if nodeCtx.Err() != nil {
		return
	}
	s.close(nodeCtx, r, placed, results, errors.Is(context.Cause(runCtx), errCanceled))
}

// prepare runs the task's Prepare, then lists the units the run acts on
// and records them as the task's units.
func (s *Scheduler) prepare(ctx, runCtx context.Context, r openRun, p *progress) (openRun, error) {
	r.task.Prepare(runCtx, r.trigger)
	var all []Unit
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		all, err = r.task.Units(ctx, rd)
		return err
	})
	if err != nil {
		return r, err
	}
	if err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return syncUnits(ctx, db.New(tx), r.task.ID, all)
	}); err != nil {
		return r, err
	}
	r.units = all
	if r.trigger == ByReturn {
		if r.units, err = s.returning(ctx, r.task.ID, all); err != nil {
			return r, err
		}
	}
	s.mu.Lock()
	s.running[r.id].run = r
	s.mu.Unlock()
	p.listed(len(r.units))
	return r, nil
}

// unit runs u on node, the Node its place routed it to for this run.
func (s *Scheduler) unit(ctx, runCtx context.Context, r openRun, u Unit, node role.NodeID, p *progress) result {
	var res result
	standing, _ := s.cfg.Roster.Standing(node)
	switch who := s.peerOf(ctx, node); {
	case node == s.self || standing == nodes.InStep:
		err := r.task.Run(runCtx, placedOn(u, node), &Progress{step: p.step})
		res = classify(runCtx, err, u.Place, who)
	case who.name == "":
		res = unsent(rpc.NotHolder, "", u.Place, peer{name: "the Node"})
	case standing == nodes.MismatchedNode:
		res = unsent(rpc.Mismatched, "", u.Place, who)
	default:
		res = unsent(rpc.Disconnected, "", u.Place, who)
	}
	if ctx.Err() != nil {
		return res
	}
	if err := s.record(ctx, r, u, node, res); err != nil {
		s.cfg.Log.Error("record a unit", "task", r.task.ID, "unit", u.Key, "err", err)
	}
	s.changed(ctx, r.task.ID)
	return res
}

// placedOn is u as its task's Run sees it, on node.
func placedOn(u Unit, node role.NodeID) Unit {
	u.Place.node = node
	return u
}

func (s *Scheduler) record(ctx context.Context, r openRun, u Unit, node role.NodeID, res result) error {
	now := storage.Now()
	params := db.RecordUnitParams{
		LastRunID: &r.id, LastOutcome: &res.outcome, LastCause: optional(res.cause), LastNode: node,
		LastEndedAt: now, TaskID: r.task.ID, UnitKey: u.Key,
	}
	switch res.outcome {
	case Succeeded:
		params.LastSuccessAt = now
		params.DueOnReturn = new(int64(0))
	case Failed:
		params.DueOnReturn = new(int64(0))
	case Skipped:
		due := int64(0)
		if res.dueOnReturn {
			due = 1
		}
		params.DueOnReturn = &due
	}
	return s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).RecordUnit(ctx, params)
	})
}

func (s *Scheduler) close(ctx context.Context, r openRun, placed []role.NodeID, results []result, canceled bool) {
	outcome, summary, cause := rollUp(r.task.Noun, results, canceled)
	lines := make([]line, len(r.units))
	for i, u := range r.units {
		lines[i] = lineOf(u.Key, u.Label, placed[i], results[i])
	}
	s.finish(ctx, r, outcome, summary, cause, lines)
}

func (s *Scheduler) finish(ctx context.Context, r openRun, outcome Result, summary, cause string, lines []line) {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if err := closeRun(ctx, q, r.id, storage.Now(), outcome, summary, cause, lines); err != nil {
			return err
		}
		if r.trigger != BySchedule && r.trigger != ByCatchUp {
			return nil
		}
		return q.AdvanceSlot(ctx, db.AdvanceSlotParams{ID: r.task.ID, SlotStartedAt: r.started, JitterFrac: s.drawJitter()})
	})
	if err != nil {
		s.cfg.Log.Error("close a run", "task", r.task.ID, "run", r.id, "err", err)
	}
	s.changed(ctx, r.task.ID)
}

func (s *Scheduler) drawJitter() *float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return new(s.cfg.Rand.Float64())
}

type progress struct {
	ctx   context.Context
	s     *Scheduler
	units int
	mu    sync.Mutex
	done  int
	sent  time.Time
	frame Progressed
}

func newProgress(ctx context.Context, s *Scheduler, r openRun) *progress {
	return &progress{ctx: ctx, s: s, units: len(r.units), frame: Progressed{RunID: r.id, TaskID: r.task.ID, Total: len(r.units)}}
}

func (p *progress) listed(units int) {
	p.mu.Lock()
	p.units, p.frame.Total = units, units
	p.mu.Unlock()
}

func (p *progress) finished() {
	p.mu.Lock()
	p.done++
	p.frame.Done, p.frame.Total = p.done, p.units
	p.mu.Unlock()
	p.publish()
}

func (p *progress) step(done, total int) {
	p.mu.Lock()
	p.frame.Done, p.frame.Total = p.done*total+done, p.units*total
	p.mu.Unlock()
	p.publish()
}

func (p *progress) now() (done, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.frame.Done, p.frame.Total
}

func (p *progress) publish() {
	p.mu.Lock()
	now := time.Now()
	if now.Sub(p.sent) < progressEvery {
		p.mu.Unlock()
		return
	}
	p.sent = now
	frame := p.frame
	p.mu.Unlock()
	p.s.cfg.Progressed.Publish(p.ctx, frame)
}
