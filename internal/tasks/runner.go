package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
}

var errCanceled = errors.New("an Admin canceled the run")

const progressEvery = 500 * time.Millisecond

// active is an open run this process runs. Each Node with queued units has
// one worker, which runs them in order and exits once its queue is empty.
// The last worker to exit seals the run, which then closes, so a Start that
// meets a sealed run waits for the close and opens another. The fields
// after closed are guarded by Scheduler.mu.
type active struct {
	run    openRun
	ctx    context.Context //nolint:containedctx // the clock's, which outlives every worker the run starts
	runCtx context.Context //nolint:containedctx // canceled by Cancel; each unit's Run gets it
	cancel context.CancelCauseFunc
	closed chan struct{}

	// entries is one per unit key, in the order each was first queued.
	entries []*entry
	byKey   map[UnitKey]*entry
	queues  map[role.NodeID][]*entry
	workers map[role.NodeID]bool
	sealed  bool
	// done and total count units: those ended of those queued, a unit
	// queued again counting again.
	done, total int
	sent        time.Time
}

// entry is one unit of a run. It is queued, running, or both while it waits
// to run again behind itself on node.
type entry struct {
	unit    Unit
	node    role.NodeID
	queued  bool
	running bool
	step    *Step
	sent    time.Time
	res     result
}

func (s *Scheduler) start(ctx context.Context, r openRun, units []Unit) {
	runCtx, cancel := context.WithCancelCause(ctx)
	a := &active{
		run: r, ctx: ctx, runCtx: runCtx, cancel: cancel, closed: make(chan struct{}),
		byKey: map[UnitKey]*entry{}, queues: map[role.NodeID][]*entry{}, workers: map[role.NodeID]bool{},
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running[r.id] = a
	if r.task.Prepare != nil {
		s.runs.Go(func() { s.prepare(a) })
		return
	}
	s.add(a, units, true)
}

// add queues each of units in a and starts a worker for each Node that has
// none. A whole-task add skips every unit already in the run. Otherwise a
// queued unit is skipped and one running or ended is queued again, on the
// Node it runs on while it runs, so a key never runs twice at once. A run
// left with no worker is sealed and ends. It answers the keys it queued.
// The caller holds s.mu.
func (s *Scheduler) add(a *active, units []Unit, whole bool) []UnitKey {
	var added []UnitKey
	for _, u := range units {
		e := a.byKey[u.Key]
		if e != nil && (whole || e.queued) {
			continue
		}
		node := s.nodeOf(u.Place)
		switch {
		case e == nil:
			e = &entry{unit: u}
			a.byKey[u.Key] = e
			a.entries = append(a.entries, e)
		case e.running:
			node = e.node
		}
		e.unit, e.queued = u, true
		a.queues[node] = append(a.queues[node], e)
		a.total++
		added = append(added, u.Key)
		if !a.workers[node] {
			a.workers[node] = true
			s.runs.Go(func() { s.work(a, node) })
		}
	}
	if len(a.workers) == 0 {
		a.sealed = true
		s.runs.Go(func() { s.end(a) })
	}
	return added
}

// units is each unit of a. The caller holds s.mu.
func (a *active) units() []Unit {
	units := make([]Unit, len(a.entries))
	for i, e := range a.entries {
		units[i] = e.unit
	}
	return units
}

// queuedLines is units as lines with no outcome yet: what a run cut short by
// a stop closes as canceled.
func queuedLines(units []Unit) []line {
	lines := make([]line, len(units))
	for i, u := range units {
		lines[i] = line{Key: u.Key, Label: u.Label}
	}
	return lines
}

func setQueued(ctx context.Context, q *db.Queries, id RunID, units []Unit) error {
	lines, err := json.Marshal(queuedLines(units))
	if err != nil {
		return fmt.Errorf("run %d lines: %w", id, err)
	}
	return q.SetOpenRunUnits(ctx, db.SetOpenRunUnitsParams{ID: id, Units: new(string(lines))})
}

// work runs node's queue of a until it is empty.
func (s *Scheduler) work(a *active, node role.NodeID) {
	for {
		e, ok := s.next(a, node)
		if !ok {
			return
		}
		res := s.unit(a, e.unit, node)
		s.ended(a, e, res)
	}
}

// next takes node's next unit of a, false once its queue is empty or the
// clock stopped. The last worker to find none seals the run and ends it.
func (s *Scheduler) next(a *active, node role.NodeID) (*entry, bool) {
	s.mu.Lock()
	q := a.queues[node]
	if len(q) > 0 && a.ctx.Err() == nil {
		e := q[0]
		a.queues[node] = q[1:]
		e.queued, e.running, e.node = false, true, node
		s.mu.Unlock()
		return e, true
	}
	delete(a.queues, node)
	delete(a.workers, node)
	a.sealed = len(a.workers) == 0
	sealed := a.sealed
	s.mu.Unlock()
	if sealed {
		s.end(a)
	}
	return nil, false
}

func (s *Scheduler) ended(a *active, e *entry, res result) {
	s.mu.Lock()
	e.running, e.step, e.res = false, nil, res
	a.done++
	frame := Progressed{RunID: a.run.id, TaskID: a.run.task.ID, Done: a.done, Total: a.total}
	now := time.Now()
	send := now.Sub(a.sent) >= progressEvery
	if send {
		a.sent = now
	}
	s.mu.Unlock()
	if send {
		s.cfg.Progressed.Publish(a.ctx, frame)
	}
}

func (s *Scheduler) stepped(a *active, key UnitKey, st Step) {
	s.mu.Lock()
	e := a.byKey[key]
	e.step = &st
	frame := Progressed{RunID: a.run.id, TaskID: a.run.task.ID, Done: a.done, Total: a.total, Unit: &UnitStep{Key: key, Step: st}}
	now := time.Now()
	send := now.Sub(e.sent) >= progressEvery
	if send {
		e.sent = now
	}
	s.mu.Unlock()
	if send {
		s.cfg.Progressed.Publish(a.ctx, frame)
	}
}

// Steps is task's open run and the latest step of each of its running units
// that has reported one, false while the task has no open run.
func (s *Scheduler) Steps(task TaskID) (run RunID, steps map[UnitKey]Step, open bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.openOf(task)
	if a == nil {
		return 0, nil, false
	}
	steps = map[UnitKey]Step{}
	for _, e := range a.entries {
		if e.step != nil {
			steps[e.unit.Key] = *e.step
		}
	}
	return a.run.id, steps, true
}

// openOf is task's open run, nil when it has none. The caller holds s.mu.
func (s *Scheduler) openOf(task TaskID) *active {
	for _, a := range s.running {
		if a.run.task.ID == task {
			return a
		}
	}
	return nil
}

// end closes a sealed run, unless the clock stopped: the next Open closes it
// as cut short.
func (s *Scheduler) end(a *active) {
	if a.ctx.Err() == nil {
		s.close(a)
	}
	s.release(a)
}

func (s *Scheduler) release(a *active) {
	a.cancel(nil)
	s.mu.Lock()
	delete(s.running, a.run.id)
	s.mu.Unlock()
	close(a.closed)
	s.Wake()
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

// prepare runs the task's Prepare, then lists the units the run acts on,
// records them as the task's units and queues them.
func (s *Scheduler) prepare(a *active) {
	units, err := s.prepared(a)
	if err != nil {
		if a.ctx.Err() == nil {
			s.cfg.Log.Error("list a run's units", "task", a.run.task.ID, "run", a.run.id, "err", err)
			s.finish(a.ctx, a.run, RunFailed, "0 "+a.run.task.Noun.Many, err.Error(), nil)
		}
		s.mu.Lock()
		a.sealed = true
		s.mu.Unlock()
		s.release(a)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.add(a, units, true)
}

func (s *Scheduler) prepared(a *active) ([]Unit, error) {
	r := a.run
	r.task.Prepare(a.runCtx, r.trigger)
	var all []Unit
	err := s.cfg.Library.Read(a.ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		all, err = r.task.Units(ctx, rd)
		return err
	})
	if err != nil {
		return nil, err
	}
	run := all
	if r.trigger == ByReturn {
		if run, err = s.returning(a.ctx, r.task.ID, all); err != nil {
			return nil, err
		}
	}
	return run, s.cfg.Library.Write(a.ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if err := syncUnits(ctx, q, r.task.ID, all); err != nil {
			return err
		}
		return setQueued(ctx, q, r.id, run)
	})
}

// unit runs u of a on node, the Node its place routed it to for this run.
func (s *Scheduler) unit(a *active, u Unit, node role.NodeID) result {
	ctx, r := a.ctx, a.run
	var res result
	standing, _ := s.cfg.Roster.Standing(node)
	switch who := s.peerOf(ctx, node); {
	case node == s.self || standing == nodes.InStep:
		err := r.task.Run(a.runCtx, placedOn(u, node), &Progress{step: func(st Step) { s.stepped(a, u.Key, st) }})
		res = classify(a.runCtx, err, u.Place, who)
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
		LastEndedAt: now, LastOwned: res.owned, TaskID: r.task.ID, UnitKey: u.Key,
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

// close rolls a sealed run's units up. Every unit has ended by then and
// nothing changes them, so it reads them without s.mu.
func (s *Scheduler) close(a *active) {
	results := make([]result, len(a.entries))
	lines := make([]line, len(a.entries))
	for i, e := range a.entries {
		results[i] = e.res
		lines[i] = lineOf(e.unit.Key, e.unit.Label, e.node, e.res)
	}
	outcome, summary, cause := rollUp(a.run.task.Noun, results, errors.Is(context.Cause(a.runCtx), errCanceled))
	s.finish(a.ctx, a.run, outcome, summary, cause, lines)
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
