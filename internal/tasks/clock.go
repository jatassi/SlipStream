package tasks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks/internal/db"
)

const retry = time.Minute

type ask struct {
	start Start
	reply chan<- asked
}

// asked is how the clock answered a Start. wait is the close of a sealed
// run the Start met, after which it asks again.
type asked struct {
	started Started
	wait    <-chan struct{}
	err     error
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
	s.followRoster(ctx)
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

// followRoster tells Admins every task changed and plans again at each
// change of the roster, since what a view or a plan reads of the Nodes
// changes only with it.
func (s *Scheduler) followRoster(ctx context.Context) {
	changed := s.cfg.Roster.Changed()
	s.runs.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-changed:
				changed = s.cfg.Roster.Changed()
				s.changed(ctx, everyTask)
				s.Wake()
			}
		}
	})
}

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
		a.reply <- s.begin(ctx, a.start)
	}
	return false
}

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
		err := s.planned(ctx, s.byID[o.Task], o.Trigger)
		if errors.Is(err, errNoUnits) && o.Trigger != ByReturn {
			// A hidden task's passed slot leaves the window free for the
			// tasks it held back, so the clock plans again now.
			wake = now
		}
		if err != nil && !errors.Is(err, errNoUnits) {
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
	states, k := s.states(snap, started, atStart)
	return states, k, nil
}

func (s *Scheduler) states(snap snapshot, started time.Time, atStart map[TaskID]bool) ([]state, knobs) {
	k := s.knobs(snap.settings, started)
	returned := map[TaskID]bool{}
	for _, u := range snap.due {
		if !returned[u.TaskID] && s.reaches(u.TiedTo) {
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
		install:       s.self,
	}
}

func (s *Scheduler) reaches(node role.NodeID) bool {
	standing, _ := s.cfg.Roster.Standing(node)
	return node == s.self || standing == nodes.InStep
}

func (s *Scheduler) peerOf(ctx context.Context, node role.NodeID) peer {
	n, err := s.cfg.Roster.Get(ctx, node)
	if err != nil {
		return peer{}
	}
	p := peer{name: n.Name}
	if n.Build != nil {
		p.version = n.Build.Version
	}
	return p
}

var errNoUnits = errors.New("the task lists no units")

// planned opens the run the plan asked for: of every unit t lists, or of
// those due on their Node's return.
func (s *Scheduler) planned(ctx context.Context, t Task, trigger Trigger) error {
	all, err := s.list(ctx, t, trigger)
	if err != nil {
		return err
	}
	run := all
	if trigger == ByReturn && t.Prepare == nil {
		if run, err = s.returning(ctx, t.ID, all); err != nil {
			return err
		}
		if len(run) == 0 {
			return errNoUnits
		}
	}
	_, err = s.open(ctx, t, trigger, storage.UUID{}, all, run)
	return err
}

// open records a run of units and starts it. all is every unit t lists,
// which become the task's units. A task with Prepare lists its units once
// its run is open.
func (s *Scheduler) open(ctx context.Context, t Task, trigger Trigger, by storage.UUID, all, units []Unit) (RunID, error) {
	var r openRun
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if run, err := q.GetOpenRun(ctx, t.ID); err == nil {
			return &ErrRunning{Since: run.StartedAt.Time()}
		} else if !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		if t.Prepare == nil {
			if err := syncUnits(ctx, q, t.ID, all); err != nil {
				return err
			}
		}
		started := storage.Now()
		id, err := q.InsertRun(ctx, db.InsertRunParams{TaskID: t.ID, Kind: trigger, RequestedBy: by, StartedAt: started})
		if err != nil {
			return err
		}
		r = openRun{id: id, task: t, trigger: trigger, started: started}
		return setQueued(ctx, q, id, units)
	})
	if err != nil {
		return 0, err
	}
	s.start(ctx, r, units)
	s.changed(ctx, t.ID)
	return r.id, nil
}

// list is every unit t lists, or none for a task with Prepare: its run
// lists them once Prepare has run. errNoUnits when t isn't shown.
func (s *Scheduler) list(ctx context.Context, t Task, trigger Trigger) ([]Unit, error) {
	var all []Unit
	var shown bool
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if t.Prepare == nil {
			if all, err = t.Units(ctx, r); err != nil {
				return err
			}
		}
		shown, err = shownIn(ctx, r, t, all)
		return err
	})
	switch {
	case err != nil:
		return nil, err
	case !shown:
		return nil, s.passHiddenSlot(ctx, t, trigger)
	}
	return all, nil
}

// shownIn reports whether t is shown, given the units it lists.
func shownIn(ctx context.Context, r storage.Reader[file], t Task, units []Unit) (bool, error) {
	if t.Shown == nil {
		return len(units) > 0, nil
	}
	return t.Shown(ctx, r)
}

func (s *Scheduler) passHiddenSlot(ctx context.Context, t Task, trigger Trigger) error {
	if trigger == BySchedule || trigger == ByCatchUp {
		if err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return db.New(tx).AdvanceSlot(ctx, db.AdvanceSlotParams{ID: t.ID, SlotStartedAt: storage.Now(), JitterFrac: s.drawJitter()})
		}); err != nil {
			return err
		}
	}
	return errNoUnits
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
		node := s.nodeOf(u.Place)
		for _, d := range due {
			if d.TaskID == task && d.UnitKey == u.Key && s.reaches(node) {
				out = append(out, u)
			}
		}
	}
	return out, nil
}

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

// Start names a run of a task's units.
type Start struct {
	Task TaskID
	// Units is the keys of the units to run, nil for every unit the task
	// lists.
	Units   []UnitKey
	Trigger Trigger
	// By is the Admin who asked, for a run-now Start only.
	By storage.UUID
}

// Started is the run a Start's units are in and the keys it queued, empty
// when it queued none. A task with Prepare lists its units once its run is
// open, so a Start that opens its run queued none yet.
type Started struct {
	Run   RunID
	Added []UnitKey
}

// Start queues st's units. With no run of the task open it opens one of
// them. With one open it adds them to it: a whole-task Start adds each unit
// not in the run yet, and answers *ErrRunning when that is none. A Start
// naming units skips each one still queued and queues each one running or
// ended again behind itself on its Node, so a unit never runs twice at
// once and a scan asked for mid-scan runs once more after. A task with
// Prepare is never added to: it answers *ErrRunning while it runs. A
// whole-task run-now Start that opens a run refuses with *ErrWaiting while
// every unit is tied to a Node that can't start work. A Start naming units
// queues them all the same, and each is Skipped and due when its Node
// returns. ErrUnknownTask when the task isn't shown or lists none of the
// units named.
func (s *Scheduler) Start(ctx context.Context, st Start) (Started, error) {
	for {
		reply := make(chan asked, 1)
		select {
		case s.asks <- ask{start: st, reply: reply}:
		case <-ctx.Done():
			return Started{}, fmt.Errorf("start %s: %w", st.Task, ctx.Err())
		}
		var a asked
		select {
		case a = <-reply:
		case <-ctx.Done():
			return Started{}, fmt.Errorf("start %s: %w", st.Task, ctx.Err())
		}
		if a.wait == nil {
			return a.started, a.err
		}
		select {
		case <-a.wait:
		case <-ctx.Done():
			return Started{}, fmt.Errorf("start %s: %w", st.Task, ctx.Err())
		}
	}
}

// RunNow opens a run-now run of every unit of id for the Admin by, or adds
// the units not in its open run yet. It never moves the task's slot.
func (s *Scheduler) RunNow(ctx context.Context, id TaskID, by storage.UUID) (RunID, error) {
	st, err := s.Start(ctx, Start{Task: id, Trigger: ByRunNow, By: by})
	return st.Run, err
}

// begin answers st on the clock, so it never races the plan's opening.
func (s *Scheduler) begin(ctx context.Context, st Start) asked {
	t, ok := s.byID[st.Task]
	if !ok {
		return asked{err: ErrUnknownTask}
	}
	all, err := s.list(ctx, t, st.Trigger)
	if errors.Is(err, errNoUnits) {
		return asked{err: ErrUnknownTask}
	}
	if err != nil {
		return asked{err: err}
	}
	units := all
	if st.Units != nil {
		units = slices.DeleteFunc(slices.Clone(all), func(u Unit) bool { return !slices.Contains(st.Units, u.Key) })
		if len(units) == 0 {
			return asked{err: ErrUnknownTask}
		}
	}
	if a, ok := s.merge(ctx, t, st, units); ok {
		return a
	}
	if st.Trigger == ByRunNow && st.Units == nil {
		if err := s.waiting(ctx, units); err != nil {
			return asked{err: err}
		}
	}
	id, err := s.open(ctx, t, st.Trigger, st.By, all, units)
	if err != nil {
		return asked{err: err}
	}
	started := Started{Run: id}
	if t.Prepare == nil {
		for _, u := range units {
			started.Added = append(started.Added, u.Key)
		}
	}
	return asked{started: started}
}

// merge adds units to t's open run, false when it has none.
func (s *Scheduler) merge(ctx context.Context, t Task, st Start, units []Unit) (asked, bool) {
	s.mu.Lock()
	a := s.openOf(t.ID)
	switch {
	case a == nil:
		s.mu.Unlock()
		return asked{}, false
	case a.sealed || a.runCtx.Err() != nil:
		s.mu.Unlock()
		return asked{wait: a.closed}, true
	case t.Prepare != nil:
		s.mu.Unlock()
		return asked{err: &ErrRunning{Since: a.run.started.Time()}}, true
	}
	added := s.add(a, units, st.Units == nil)
	queued := a.units()
	s.mu.Unlock()
	if len(added) == 0 {
		if st.Units == nil {
			return asked{err: &ErrRunning{Since: a.run.started.Time()}}, true
		}
		return asked{started: Started{Run: a.run.id}}, true
	}
	if err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return setQueued(ctx, db.New(tx), a.run.id, queued)
	}); err != nil {
		s.cfg.Log.Error("record a run's units", "task", t.ID, "run", a.run.id, "err", err)
	}
	s.changed(ctx, t.ID)
	return asked{started: Started{Run: a.run.id, Added: added}}, true
}

// waiting refuses a run-now run whose every unit is tied to a Node that
// can't start work.
func (s *Scheduler) waiting(ctx context.Context, units []Unit) error {
	if len(units) == 0 {
		return nil
	}
	for _, u := range units {
		if u.Place.kind != tiedTo || s.reaches(u.Place.node) {
			return nil
		}
	}
	node := units[0].Place.node
	standing, since := s.cfg.Roster.Standing(node)
	return &ErrWaiting{Node: s.peerOf(ctx, node).name, Standing: standing, Since: since}
}
