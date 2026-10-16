package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks/internal/db"
)

// retry is how soon the clock tries again after it couldn't read or open.
const retry = time.Minute

// ask is a Run now the clock answers, as the only opener of runs.
type ask struct {
	task  TaskID
	by    storage.UUID
	reply chan<- asked
}

type asked struct {
	run RunID
	err error
}

// Run is the clock: it plans, opens what is due and sleeps until the next
// slot or a wake. It logs what fails and tries again, returning only once
// ctx ends and every run it started has stopped.
func (s *Scheduler) Run(ctx context.Context) error {
	defer s.runs.Wait()
	started := time.Now()
	s.mu.Lock()
	s.started = started
	s.mu.Unlock()
	s.follow(ctx, "", nodes.ChangedEvent)
	for _, t := range s.cfg.Catalog {
		if len(t.Follows) > 0 {
			s.follow(ctx, t.ID, t.Follows...)
		}
	}
	atStart := map[TaskID]bool{}
	for _, t := range s.cfg.Catalog {
		if c, ok := t.Cadence.(Every); ok && c.AtStart {
			atStart[t.ID] = true
		}
	}
	for {
		if s.sleep(ctx, s.tick(ctx, started, atStart)) {
			return nil
		}
	}
}

// follow plans again and tells open pages that task changed after each
// of events, until ctx ends. An empty task is every task: a Node changing
// changes which wait for it.
func (s *Scheduler) follow(ctx context.Context, task TaskID, events ...live.Event) {
	woken := s.cfg.Hub.Watch(ctx, events...)
	s.runs.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-woken:
				s.changed(ctx, task)
				s.Wake()
			}
		}
	})
}

// sleep waits for wake, a zero wake waiting for an event alone, and
// answers whether ctx ended.
func (s *Scheduler) sleep(ctx context.Context, wake time.Time) bool {
	var timer <-chan time.Time
	if !wake.IsZero() {
		t := time.NewTimer(time.Until(wake))
		defer t.Stop()
		timer = t.C
	}
	select {
	case <-ctx.Done():
		return true
	case <-timer:
	case <-s.wake:
	case a := <-s.asks:
		id, err := s.runNow(ctx, a)
		a.reply <- asked{run: id, err: err}
	}
	return false
}

// tick opens every run the plan says is due and answers when to plan next.
func (s *Scheduler) tick(ctx context.Context, started time.Time, atStart map[TaskID]bool) time.Time {
	now := time.Now()
	states, k, err := s.read(ctx, started, atStart)
	if err != nil {
		s.cfg.Log.Error("read the tasks", "err", err)
		return now.Add(retry)
	}
	p := plan(now, k, states)
	if err := s.pass(ctx, p.Pass); err != nil {
		s.cfg.Log.Error("pass missed slots", "err", err)
		return now.Add(retry)
	}
	wake := p.Wake
	for _, o := range p.Open {
		if _, err := s.open(ctx, s.byID[o.Task], o.Trigger, storage.UUID{}); err != nil && !errors.Is(err, errNoUnits) {
			s.cfg.Log.Error("open a run", "task", o.Task, "trigger", o.Trigger, "err", err)
			if wake.IsZero() || now.Add(retry).Before(wake) {
				wake = now.Add(retry)
			}
			continue
		}
		delete(atStart, o.Task)
		s.mu.Lock()
		delete(s.cut, o.Task)
		s.mu.Unlock()
	}
	return wake
}

func (s *Scheduler) pass(ctx context.Context, passes []Passed) error {
	if len(passes) == 0 {
		return nil
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		for _, passed := range passes {
			if err := q.PassSlot(ctx, db.PassSlotParams{ID: passed.Task, SlotStartedAt: storage.FromTime(passed.Slot)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, passed := range passes {
		s.changed(ctx, passed.Task)
	}
	return nil
}

type snapshot struct {
	settings     db.TasksSetting
	rows         []db.TasksTask
	latest, open []db.TasksRun
	due          []db.TasksUnit
}

func (s *Scheduler) snapshot(ctx context.Context) (snapshot, error) {
	var snap snapshot
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		snap, err = readSnapshot(ctx, db.New(r))
		return err
	})
	return snap, err
}

func readSnapshot(ctx context.Context, q *db.Queries) (snapshot, error) {
	var snap snapshot
	var err error
	if snap.settings, err = q.GetSettings(ctx); err != nil {
		return snap, err
	}
	if snap.rows, err = q.ListTasks(ctx); err != nil {
		return snap, err
	}
	if snap.latest, err = q.ListLatestRuns(ctx); err != nil {
		return snap, err
	}
	if snap.open, err = q.ListOpenRuns(ctx); err != nil {
		return snap, err
	}
	snap.due, err = q.ListDueUnits(ctx)
	return snap, err
}

func (s *Scheduler) read(ctx context.Context, started time.Time, atStart map[TaskID]bool) ([]state, knobs, error) {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return nil, knobs{}, err
	}
	states, k := s.states(snap, started, atStart, func(node role.NodeID) bool { return s.reaches(ctx, node) })
	return states, k, nil
}

// states is each catalog task's state for plan. reaches says whether a
// Node can start work now.
func (s *Scheduler) states(snap snapshot, started time.Time, atStart map[TaskID]bool, reaches func(role.NodeID) bool) ([]state, knobs) {
	k := s.knobs(snap.settings, started)
	returned := map[TaskID]bool{}
	for _, u := range snap.due {
		if !returned[u.TaskID] && reaches(u.TiedTo) {
			returned[u.TaskID] = true
		}
	}
	byID := map[TaskID]db.TasksTask{}
	for _, row := range snap.rows {
		byID[row.ID] = row
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	states := make([]state, 0, len(s.cfg.Catalog))
	for _, t := range s.cfg.Catalog {
		row := byID[t.ID]
		st := state{
			task: t, every: row.EverySeconds, override: row.Overdue,
			firstSeen: row.FirstSeenAt.Time(), slot: row.SlotStartedAt.Time(),
			cut: s.cut[t.ID], atStart: atStart[t.ID], returned: returned[t.ID],
		}
		if row.JitterFrac != nil {
			st.jitter = *row.JitterFrac
		}
		for _, r := range snap.latest {
			if r.TaskID == t.ID {
				st.lastEnded = r.EndedAt.Time()
			}
		}
		for _, r := range snap.open {
			st.open = st.open || r.TaskID == t.ID
		}
		states = append(states, st)
	}
	return states, k
}

func (s *Scheduler) knobs(row db.TasksSetting, started time.Time) knobs {
	var window *setting.Span
	if row.WindowStart != nil && row.WindowEnd != nil {
		window = &setting.Span{Start: int(*row.WindowStart), End: int(*row.WindowEnd)}
	}
	return knobs{
		zone:          s.cfg.Zone,
		window:        MaintenanceWindow.Value(window),
		jitterPercent: JitterPercent.Value(row.JitterPercent),
		jitterCap:     JitterCap.Value(row.JitterCapSeconds),
		overdue:       OverdueRule.Value(row.Overdue),
		started:       started,
	}
}

// reaches is whether node can start work now: this Node, or a Node whose
// link is up and on this build.
func (s *Scheduler) reaches(ctx context.Context, node role.NodeID) bool {
	_, ok := s.preflight(ctx, node)
	return ok
}

// preflight is node as a unit's summary names it, and whether it can
// start work now.
func (s *Scheduler) preflight(ctx context.Context, node role.NodeID) (nodes.Node, bool) {
	n, err := s.cfg.Roster.Get(ctx, node)
	if err != nil {
		return nodes.Node{}, false
	}
	return n, n.Link == nodes.Local || n.Link == nodes.Up
}

var errNoUnits = errors.New("the task lists no units")

// open opens a run of t: it lists t's units, syncs tasks_unit to them and
// inserts the run in one Write, which tasks_run_open refuses while one is
// open, then starts the run. An empty list opens nothing and passes a
// slot run's slot, so a hidden task keeps its cadence.
func (s *Scheduler) open(ctx context.Context, t Task, trigger Trigger, by storage.UUID) (RunID, error) {
	all, units, err := s.units(ctx, t, trigger)
	if err != nil {
		return 0, err
	}
	var r openRun
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if run, err := q.GetOpenRun(ctx, t.ID); err == nil {
			return &ErrRunning{Since: run.StartedAt.Time()}
		} else if !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		if err := syncUnits(ctx, q, t.ID, all); err != nil {
			return err
		}
		started := storage.Now()
		id, err := q.InsertRun(ctx, db.InsertRunParams{TaskID: t.ID, Kind: trigger, RequestedBy: by, StartedAt: started})
		r = openRun{id: id, task: t, trigger: trigger, started: started, units: units}
		return err
	})
	if err != nil {
		return 0, err
	}
	s.start(ctx, r)
	s.changed(ctx, t.ID)
	return r.id, nil
}

// units is every unit t lists, and those the run acts on: all of them, or
// for a returned run those due whose Node can start work again.
func (s *Scheduler) units(ctx context.Context, t Task, trigger Trigger) (all, run []Unit, err error) {
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		all, err = t.Units(ctx, r)
		return err
	})
	switch {
	case err != nil:
		return nil, nil, err
	case len(all) == 0:
		if trigger == BySchedule || trigger == ByCatchUp {
			if err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
				return db.New(tx).AdvanceSlot(ctx, db.AdvanceSlotParams{ID: t.ID, SlotStartedAt: storage.Now(), JitterFrac: s.drawJitter()})
			}); err != nil {
				return nil, nil, err
			}
		}
		return nil, nil, errNoUnits
	case trigger == ByReturn:
		run, err = s.returning(ctx, t.ID, all)
		if err == nil && len(run) == 0 {
			err = errNoUnits
		}
		return all, run, err
	case trigger == ByRunNow:
		return all, all, s.runnable(ctx, t.ID, all)
	}
	return all, all, nil
}

func (s *Scheduler) returning(ctx context.Context, task TaskID, units []Unit) ([]Unit, error) {
	var due []db.TasksUnit
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		due, err = db.New(r).ListDueUnits(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	var out []Unit
	for _, u := range units {
		node, _ := u.Place.Node()
		for _, d := range due {
			if d.TaskID == task && d.UnitKey == u.Key && s.reaches(ctx, node) {
				out = append(out, u)
			}
		}
	}
	return out, nil
}

// syncUnits makes task's unit rows the units listed: new ones are added,
// vanished ones deleted, and labels and Nodes updated.
func syncUnits(ctx context.Context, q *db.Queries, task TaskID, units []Unit) error {
	rows, err := q.ListUnits(ctx, task)
	if err != nil {
		return err
	}
	listed := map[UnitKey]bool{}
	now := storage.Now()
	for i, u := range units {
		listed[u.Key] = true
		tied, _ := u.Place.Node()
		if u.Place.kind != tiedTo {
			tied = role.NodeID{}
		}
		if err := q.UpsertUnit(ctx, db.UpsertUnitParams{TaskID: task, UnitKey: u.Key, Position: int64(i), Label: u.Label, TiedTo: tied, FirstSeenAt: now}); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if !listed[row.UnitKey] {
			if err := q.DeleteUnit(ctx, db.DeleteUnitParams{TaskID: task, UnitKey: row.UnitKey}); err != nil {
				return err
			}
		}
	}
	return nil
}

// RunNow opens a run-now run of every unit of id for the Admin by. It
// never moves the task's slot.
func (s *Scheduler) RunNow(ctx context.Context, id TaskID, by storage.UUID) (RunID, error) {
	reply := make(chan asked, 1)
	select {
	case s.asks <- ask{task: id, by: by, reply: reply}:
	case <-ctx.Done():
		return 0, fmt.Errorf("run now %s: %w", id, ctx.Err())
	}
	select {
	case a := <-reply:
		return a.run, a.err
	case <-ctx.Done():
		return 0, fmt.Errorf("run now %s: %w", id, ctx.Err())
	}
}

func (s *Scheduler) runNow(ctx context.Context, a ask) (RunID, error) {
	t, ok := s.byID[a.task]
	if !ok {
		return 0, ErrUnknownTask
	}
	id, err := s.open(ctx, t, ByRunNow, a.by)
	if errors.Is(err, errNoUnits) {
		return 0, ErrUnknownTask
	}
	return id, err
}

// runnable refuses Run now while the task has a run open, and while
// every unit is tied to a Node that can't start work, so no hidden run
// fires once it returns.
func (s *Scheduler) runnable(ctx context.Context, task TaskID, units []Unit) error {
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		run, err := db.New(r).GetOpenRun(ctx, task)
		if err == nil {
			return &ErrRunning{Since: run.StartedAt.Time()}
		}
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}
	for _, u := range units {
		if u.Place.kind != tiedTo {
			return nil
		}
		if _, ok := s.preflight(ctx, u.Place.node); ok {
			return nil
		}
	}
	n, _ := s.preflight(ctx, units[0].Place.node)
	return &ErrWaiting{Node: n.Name, Link: n.Link, Since: n.Since}
}
