package tasks

import (
	"context"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks/internal/db"
)

type (
	TaskID  = db.TaskID
	RunID   = db.RunID
	UnitKey = db.UnitKey
	Overdue = db.Overdue
	Trigger = db.Trigger
	Outcome = db.Outcome
	Result  = db.Result
)

const (
	// FollowGlobal declares that a task does what tasks.overdue says.
	FollowGlobal Overdue = ""
	// CatchUp runs an overdue task once now, however many slots it missed.
	CatchUp Overdue = "catch-up"
	// Wait records the missed slots as passed and waits for the next one.
	Wait Overdue = "wait"
)

const (
	BySchedule Trigger = "schedule"
	ByCatchUp  Trigger = "catch-up"
	ByRunNow   Trigger = "run-now"
	ByReturn   Trigger = "returned"
)

const (
	Succeeded Outcome = "succeeded"
	Failed    Outcome = "failed"
	Skipped   Outcome = "skipped"
	Canceled  Outcome = "canceled"
)

const (
	RunSucceeded Result = "succeeded"
	RunPartial   Result = "partial"
	RunFailed    Result = "failed"
	RunSkipped   Result = "skipped"
	RunCanceled  Result = "canceled"
)

// Task is one Scheduled task as its owner declares it beside the code that
// does the work. A task is shown exactly while Units lists one or more.
type Task struct {
	ID      TaskID
	Name    string
	About   string
	Group   Group
	Cadence Cadence
	// Overdue is the task's rule when its override is null.
	Overdue Overdue
	// Noun counts units in a run's summary and its progress: "5 of 6 steps".
	Noun setting.Noun
	// Units lists what a run acts on, in the order the units run.
	Units func(ctx context.Context, r storage.Reader[storage.Library]) ([]Unit, error)
	// Run does one unit's work. The runner classifies its error.
	Run func(ctx context.Context, u Unit, p *Progress) error
	// Follows is the events after which Units may answer differently, so
	// open pages refetch the task.
	Follows []live.Event
	// Health is the warning category; "" is scheduled-tasks.
	Health string
}

type Group struct{ roles role.Group }

// ControllerGroup holds tasks no Role does.
var ControllerGroup = Group{}

func GroupOf(g role.Group) Group { return Group{roles: g} }

func (g Group) String() string {
	if g == ControllerGroup {
		return "controller"
	}
	return string(g.roles)
}

// Cadence is Every or InWindow.
type Cadence interface{ cadence() }

// Every runs a task every Interval, plus jitter. AtStart also runs it
// each time the clock starts.
type Every struct {
	Interval setting.Duration
	AtStart  bool
}

// InWindow starts a task in the Maintenance window every Days.
type InWindow struct {
	Days setting.Duration
}

func (Every) cadence()    {}
func (InWindow) cadence() {}

func (t Task) Setting() setting.Duration {
	switch c := t.Cadence.(type) {
	case Every:
		return c.Interval
	case InWindow:
		return c.Days
	}
	panic("tasks: a task without a cadence")
}

type Unit struct {
	Key   UnitKey
	Label string
	Place Place
}

type Place struct {
	kind placeKind
	node role.NodeID
	role role.Name
}

type placeKind uint8

const (
	onController placeKind = iota
	tiedTo
	portableOn
)

func OnController() Place { return Place{} }

// TiedTo runs a unit on node and never elsewhere; one that couldn't run is
// due when node returns.
func TiedTo(node role.NodeID) Place { return Place{kind: tiedTo, node: node} }

// PortableOn runs a unit of r's work homed on home on the Node the roster
// routes it to at each slot: home while it may, another holder of r once
// home lapses. One that couldn't run waits for the next slot.
func PortableOn(r role.Name, home role.NodeID) Place {
	return Place{kind: portableOn, node: home, role: r}
}

// Node is the Node the unit runs on, false on the Controller.
func (p Place) Node() (role.NodeID, bool) { return p.node, p.kind != onController }

type Progress struct {
	step func(done, total int)
}

func (p *Progress) Step(done, total int) { p.step(done, total) }
