package tasks

import (
	"cmp"
	"context"
	"encoding/json"
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

var (
	ChangedEvent  = live.AdminEvent[Changed]("tasks.changed")
	ProgressEvent = live.AdminEvent[Progressed]("tasks.progress")
)

// Status is what a shown task does now: Idle, Running or Waiting.
type Status interface{ status() }

type Idle struct{ Next time.Time }

type Running struct {
	Run         RunID
	Trigger     Trigger
	Since       time.Time
	Done, Total int
}

// Waiting is a task whose every unit is tied to a Node that can't start
// work, since Since.
type Waiting struct {
	Node  string
	Link  nodes.LinkState
	Since time.Time
}

func (Idle) status()    {}
func (Running) status() {}
func (Waiting) status() {}

type View struct {
	Task Task
	// Every and Overdue are the task's overrides, nil at their defaults.
	Every   *int64
	Overdue *Overdue
	// Global is the global overdue rule a FollowGlobal task follows.
	Global Overdue
	Status Status
	// Last is the latest finished run, nil before one has.
	Last *Run
	// Nodes names the Nodes its units run on, in unit order.
	Nodes []string
}

func (v View) CadenceField() setting.DurationField { return v.Task.Setting().Field(v.Every) }

// OverdueSetting is the task's overdue rule as a setting whose default is
// its declared rule, or the global rule when it follows that.
func (v View) OverdueSetting() setting.Choice[Overdue] {
	c := setting.Choice[Overdue]{Key: "tasks." + string(v.Task.ID) + ".overdue", Label: "When overdue", Default: v.Task.Overdue, Options: overdueOptions}
	if c.Default == FollowGlobal {
		c.Default, c.DefaultFrom = v.Global, "Scheduled tasks"
	}
	return c
}

func (v View) OverdueField() setting.ChoiceField { return v.OverdueSetting().Field(v.Overdue) }

// Cadence is the task's cadence in words: "Every 8 h", "Daily in the
// Maintenance window".
func (v View) Cadence() string {
	every := v.Task.Setting().Value(v.Every)
	_, window := v.Task.Cadence.(InWindow)
	switch {
	case window && every == day:
		return "Daily in the Maintenance window"
	case window:
		return fmt.Sprintf("Every %d days in the Maintenance window", every/day)
	case every == day:
		return "Daily"
	}
	return "Every " + setting.Words(every)
}

// Run is one run as its history shows it. Ended is zero and Result empty
// while it is open.
type Run struct {
	ID             RunID
	Trigger        Trigger
	Started, Ended time.Time
	Result         Result
	Summary, Cause string
	// Nodes names the Nodes its units ran on, in unit order.
	Nodes []string
	// Lines is each unit's outcome, written when the run closed.
	Lines []Line
}

type Line struct {
	Key     UnitKey
	Label   string
	Node    string
	Outcome Outcome
	Cause   string
}

// UnitView is one unit a task lists now, with its last outcome. Outcome
// is empty before it has run.
type UnitView struct {
	Key              UnitKey
	Label            string
	Node             string
	Outcome          Outcome
	Cause            string
	Ended, Succeeded time.Time
	DueOnReturn      bool
}

type PartKind string

const (
	PlainPart   PartKind = "plain"
	WarningPart PartKind = "warning"
	FailurePart PartKind = "failure"
)

// Part is one sentence of the Tasks page's status sentence. Href is the
// task's page for a warning or failure.
type Part struct {
	Kind PartKind
	Text string
	Href string
}

// Overview is the Tasks page: every shown task in catalog order, the
// sentence that opens it and the schedule settings.
type Overview struct {
	Tasks     []View
	Multinode bool
	Sentence  []Part
	Schedule  Schedule
}

// Detail is one shown task with its units and its runs, newest first.
type Detail struct {
	View  View
	Units []UnitView
	Runs  []Run
}

type Active struct {
	Run         RunID
	Task        TaskID
	Name        string
	Noun        setting.Noun
	Trigger     Trigger
	Since       time.Time
	Done, Total int
	Nodes       []string
}

type look struct {
	snap   snapshot
	listed map[TaskID][]Unit
	rows   map[TaskID][]db.TasksUnit
	nodes  map[role.NodeID]nodes.Node
	count  int
}

func (s *Scheduler) look(ctx context.Context) (look, error) {
	list, err := s.cfg.Roster.List(ctx)
	if err != nil {
		return look{}, err
	}
	l := look{listed: map[TaskID][]Unit{}, rows: map[TaskID][]db.TasksUnit{}, nodes: map[role.NodeID]nodes.Node{}, count: len(list)}
	for _, n := range list {
		l.nodes[n.ID] = n
	}
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		var err error
		if l.snap, err = readSnapshot(ctx, q); err != nil {
			return err
		}
		for _, t := range s.cfg.Catalog {
			if l.listed[t.ID], err = t.Units(ctx, r); err != nil {
				return fmt.Errorf("list %s's units: %w", t.ID, err)
			}
			if l.rows[t.ID], err = q.ListUnits(ctx, t.ID); err != nil {
				return err
			}
		}
		return nil
	})
	return l, err
}

func (l look) name(node role.NodeID) string { return l.nodes[node].Name }

func (s *Scheduler) Overview(ctx context.Context) (Overview, error) {
	l, err := s.look(ctx)
	if err != nil {
		return Overview{}, err
	}
	now := time.Now()
	views := s.views(l, now)
	return Overview{Tasks: views, Multinode: l.count > 1, Sentence: s.sentence(l, views, now), Schedule: scheduleOf(l.snap.settings)}, nil
}

// Task is one shown task with its units and up to runs_kept of its runs,
// or ErrUnknownTask.
func (s *Scheduler) Task(ctx context.Context, id TaskID) (Detail, error) {
	l, err := s.look(ctx)
	if err != nil {
		return Detail{}, err
	}
	views := s.views(l, time.Now())
	i := slices.IndexFunc(views, func(v View) bool { return v.Task.ID == id })
	if i < 0 {
		return Detail{}, ErrUnknownTask
	}
	d := Detail{View: views[i]}
	rows := map[UnitKey]db.TasksUnit{}
	for _, row := range l.rows[id] {
		rows[row.UnitKey] = row
	}
	for _, u := range l.listed[id] {
		uv := UnitView{Key: u.Key, Label: u.Label, Node: l.name(s.nodeOf(u.Place))}
		if row, ok := rows[u.Key]; ok && row.LastOutcome != nil {
			uv.Outcome, uv.Cause = *row.LastOutcome, deref(row.LastCause)
			uv.Ended, uv.Succeeded = row.LastEndedAt.Time(), row.LastSuccessAt.Time()
			uv.DueOnReturn = row.DueOnReturn != nil && *row.DueOnReturn == 1
		}
		d.Units = append(d.Units, uv)
	}
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		runs, err := db.New(r).ListRuns(ctx, db.ListRunsParams{TaskID: id, Kept: RunsKept.Value(l.snap.settings.RunsKept)})
		for _, run := range runs {
			d.Runs = append(d.Runs, l.runOf(run))
		}
		return err
	})
	return d, err
}

// Running is every open run this process runs, oldest first.
func (s *Scheduler) Running(ctx context.Context) ([]Active, error) {
	list, err := s.cfg.Roster.List(ctx)
	if err != nil {
		return nil, err
	}
	l := look{nodes: map[role.NodeID]nodes.Node{}}
	for _, n := range list {
		l.nodes[n.ID] = n
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Active, 0, len(s.running))
	for _, a := range s.running {
		done, total := a.progress.now()
		out = append(out, Active{
			Run: a.run.id, Task: a.run.task.ID, Name: a.run.task.Name, Noun: a.run.task.Noun, Trigger: a.run.trigger,
			Since: a.run.started.Time(), Done: done, Total: total, Nodes: s.names(l, a.run.units),
		})
	}
	slices.SortFunc(out, func(a, b Active) int { return cmp.Compare(a.Run, b.Run) })
	return out, nil
}

func (s *Scheduler) nodeOf(p Place) role.NodeID {
	if node, ok := p.Node(); ok {
		return node
	}
	return s.self
}

func (s *Scheduler) names(l look, units []Unit) []string {
	var out []string
	for _, u := range units {
		out = appendNew(out, l.name(s.nodeOf(u.Place)))
	}
	return out
}

func (s *Scheduler) views(l look, now time.Time) []View {
	started := now
	s.mu.Lock()
	if !s.started.IsZero() {
		started = s.started
	}
	s.mu.Unlock()
	states, k := s.states(l.snap, started, nil)
	p := plan(now, k, states)
	rows := map[TaskID]db.TasksTask{}
	for _, row := range l.snap.rows {
		rows[row.ID] = row
	}
	latest := map[TaskID]db.TasksRun{}
	for _, run := range l.snap.latest {
		latest[run.TaskID] = run
	}
	var views []View
	for _, t := range s.cfg.Catalog {
		units := l.listed[t.ID]
		if len(units) == 0 {
			continue
		}
		row := rows[t.ID]
		v := View{Task: t, Every: row.EverySeconds, Overdue: row.Overdue, Global: k.overdue, Nodes: s.names(l, units)}
		if run, ok := latest[t.ID]; ok {
			last := l.runOf(run)
			v.Last = &last
		}
		next, ok := p.Next[t.ID]
		if !ok || next.Before(now) {
			next = now
		}
		v.Status = s.status(l, t, units, next)
		views = append(views, v)
	}
	return views
}

func (s *Scheduler) status(l look, t Task, units []Unit, next time.Time) Status {
	s.mu.Lock()
	for _, a := range s.running {
		if a.run.task.ID == t.ID {
			done, total := a.progress.now()
			s.mu.Unlock()
			return Running{Run: a.run.id, Trigger: a.run.trigger, Since: a.run.started.Time(), Done: done, Total: total}
		}
	}
	s.mu.Unlock()
	for _, u := range units {
		if u.Place.kind != tiedTo || s.reaches(u.Place.node) {
			return Idle{Next: next}
		}
	}
	node := units[0].Place.node
	link, since := s.cfg.Roster.Reach(node)
	return Waiting{Node: l.name(node), Link: link, Since: since}
}

func (l look) runOf(row db.TasksRun) Run {
	r := Run{ID: row.ID, Trigger: row.Kind, Started: row.StartedAt.Time(), Ended: row.EndedAt.Time(), Summary: deref(row.Summary), Cause: deref(row.Cause)}
	if row.Result != nil {
		r.Result = *row.Result
	}
	if row.Units == nil {
		return r
	}
	var lines []line
	if err := json.Unmarshal([]byte(*row.Units), &lines); err != nil {
		panic(fmt.Sprintf("tasks: run %d's units: %v", row.ID, err))
	}
	for _, ln := range lines {
		name := ""
		if id, err := storage.ParseUUID(ln.Node); err == nil {
			name = l.name(role.NodeID{UUID: id})
		}
		r.Lines = append(r.Lines, Line{Key: ln.Key, Label: ln.Label, Node: name, Outcome: ln.Outcome, Cause: ln.Cause})
		if name != "" {
			r.Nodes = appendNew(r.Nodes, name)
		}
	}
	return r
}

func (s *Scheduler) sentence(l look, views []View, now time.Time) []Part {
	var parts []Part
	for _, v := range views {
		if w, ok := s.warning(l, v, now); ok {
			kind, text := WarningPart, w.Text+"."
			if v.Last.Result == RunFailed {
				kind, text = FailurePart, fmt.Sprintf("%s failed at %s.", v.Task.Name, v.Last.Ended.In(s.cfg.Zone).Format("15:04"))
			}
			parts = append(parts, Part{Kind: kind, Text: text, Href: href(v.Task.ID)})
		}
	}
	var order []string
	waiting := map[string][]TaskID{}
	for _, v := range views {
		if w, ok := v.Status.(Waiting); ok {
			order = appendNew(order, w.Node)
			waiting[w.Node] = append(waiting[w.Node], v.Task.ID)
		}
	}
	for _, node := range order {
		text := "1 task is waiting for " + node + "."
		if n := len(waiting[node]); n > 1 {
			text = fmt.Sprintf("%d tasks are waiting for %s.", n, node)
		}
		parts = append(parts, Part{Kind: WarningPart, Text: text, Href: href(waiting[node][0])})
	}
	if len(parts) == 0 {
		return []Part{{Kind: PlainPart, Text: "Every task ran as scheduled."}}
	}
	return parts
}

func href(id TaskID) string { return "/tasks/" + string(id) }

func (s *Scheduler) warning(l look, v View, now time.Time) (Warning, bool) {
	if v.Last == nil {
		return Warning{}, false
	}
	l2 := latest{result: v.Last.Result, ended: v.Last.Ended, cause: v.Last.Cause}
	intervals := PartialWarnAfter.Value(l.snap.settings.PartialWarnIntervals)
	return warn(v.Task, l2, sinceOf(l.rows[v.Task.ID]), v.Task.Setting().Value(v.Every), intervals, now, s.cfg.Zone)
}

func (s *Scheduler) Warnings(ctx context.Context) ([]Warning, error) {
	l, err := s.look(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var out []Warning
	for _, v := range s.views(l, now) {
		if w, ok := s.warning(l, v, now); ok {
			out = append(out, w)
		}
	}
	return out, nil
}

// Zone names the time zone the Maintenance window is kept in: its IANA
// name, or its abbreviation where the process only knows it as Local.
func (s *Scheduler) Zone() string {
	if name := s.cfg.Zone.String(); name != "Local" {
		return name
	}
	abbreviation, _ := time.Now().In(s.cfg.Zone).Zone()
	return abbreviation
}
