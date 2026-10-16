// Package tasksapi is the Scheduled tasks operations: the Tasks page, one
// task with its units and runs, Run now and Cancel, each task's cadence
// and overdue rule, the schedule settings and the runs the header lists.
package tasksapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Clock is the Node's tasks owner as its operations reach it.
type Clock interface {
	// Tasks is the tasks owner, false unless the Node is a Controller.
	Tasks() (*tasks.Scheduler, bool)
}

func open(c Clock) *tasks.Scheduler {
	s, ok := c.Tasks()
	if !ok {
		panic("tasksapi: an Admin operation ran on a Node that is no Controller")
	}
	return s
}

var (
	taskUnknown = api.NewCode(http.StatusNotFound, "task_unknown", "No task shown has this id")
	taskRunning = api.NewCode(http.StatusConflict, "task_running", "The task has a run open")
	taskWaiting = api.NewCode(http.StatusConflict, "task_waiting", "Every unit waits for a Node that can't start work")
	runNotOpen  = api.NewCode(http.StatusNotFound, "run_not_open", "No open run has this id")
)

func answer(err error) error {
	if running, ok := errors.AsType[*tasks.ErrRunning](err); ok {
		return taskRunning.Because("The task has been running since " + running.Since.UTC().Format(time.RFC3339))
	}
	if waiting, ok := errors.AsType[*tasks.ErrWaiting](err); ok {
		return taskWaiting.Because(fmt.Sprintf("Every unit waits for %s, %s since %s", waiting.Node, linkWords[waiting.Link], waiting.Since.UTC().Format(time.RFC3339)))
	}
	switch {
	case errors.Is(err, tasks.ErrUnknownTask):
		return taskUnknown
	case errors.Is(err, tasks.ErrNotOpen):
		return runNotOpen
	}
	return settingapi.Refused(err)
}

var linkWords = map[nodes.LinkState]string{nodes.Down: "offline", nodes.Mismatched: "on another version"}

type part struct{}

func (part) Variants() (discriminator string, variants []any) {
	return "kind", []any{plainPart{}, warningPart{}, failurePart{}}
}

type plainPart struct {
	Kind string `json:"kind" enum:"plain"`
	Text string `json:"text"`
}

type warningPart struct {
	Kind string `json:"kind" enum:"warning"`
	Text string `json:"text"`
	Href string `json:"href" doc:"The task's page"`
}

type failurePart struct {
	Kind string `json:"kind" enum:"failure"`
	Text string `json:"text"`
	Href string `json:"href" doc:"The task's page"`
}

func partOf(p tasks.Part) api.OneOf[part] {
	switch p.Kind {
	case tasks.WarningPart:
		return api.NewOneOf[part](warningPart{Kind: string(p.Kind), Text: p.Text, Href: p.Href})
	case tasks.FailurePart:
		return api.NewOneOf[part](failurePart{Kind: string(p.Kind), Text: p.Text, Href: p.Href})
	case tasks.PlainPart:
		return api.NewOneOf[part](plainPart{Kind: string(p.Kind), Text: p.Text})
	}
	panic(fmt.Sprintf("tasksapi: part kind %q", p.Kind))
}

type state struct{}

func (state) Variants() (discriminator string, variants []any) {
	return "kind", []any{idleState{}, runningState{}, waitingState{}}
}

type idleState struct {
	Kind string    `json:"kind" enum:"idle"`
	Next time.Time `json:"next" doc:"When the next run is due; now when it is due already"`
}

type runningState struct {
	Kind    string    `json:"kind" enum:"running"`
	RunID   int64     `json:"runId"`
	Trigger string    `json:"trigger" enum:"schedule,catch-up,run-now,returned"`
	Since   time.Time `json:"since"`
	Done    int       `json:"done" minimum:"0"`
	Total   int       `json:"total" minimum:"0"`
}

type waitingState struct {
	Kind  string    `json:"kind" enum:"waiting"`
	Node  string    `json:"node" doc:"The Node every unit waits for"`
	Link  string    `json:"link" enum:"down,mismatched"`
	Since time.Time `json:"since" doc:"When the Node's link entered that state"`
}

func stateOf(s tasks.Status) api.OneOf[state] {
	switch s := s.(type) {
	case tasks.Running:
		return api.NewOneOf[state](runningState{Kind: "running", RunID: int64(s.Run), Trigger: string(s.Trigger), Since: s.Since, Done: s.Done, Total: s.Total})
	case tasks.Waiting:
		return api.NewOneOf[state](waitingState{Kind: "waiting", Node: s.Node, Link: s.Link.String(), Since: s.Since})
	case tasks.Idle:
		return api.NewOneOf[state](idleState{Kind: "idle", Next: s.Next})
	}
	panic(fmt.Sprintf("tasksapi: status %T", s))
}

type cadence struct {
	Words    string                `json:"words" doc:"The cadence as the Tasks page says it, such as Every 8 h"`
	InWindow bool                  `json:"inWindow" doc:"The task starts in the Maintenance window, every field.value seconds in whole days"`
	Field    setting.DurationField `json:"field"`
}

type runNow struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty" enum:"running,waiting" doc:"Why Run now is unavailable; absent when it is available"`
}

type task struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	About   string              `json:"about"`
	Group   string              `json:"group" enum:"controller,acquire,store,process,connect,reach"`
	Noun    setting.Noun        `json:"noun" doc:"What its progress counts: 3 of 12 steps"`
	Cadence cadence             `json:"cadence"`
	Overdue setting.ChoiceField `json:"overdue"`
	State   api.OneOf[state]    `json:"state"`
	LastRun *run                `json:"lastRun,omitempty" doc:"The latest finished run, absent before one has"`
	Nodes   []string            `json:"nodes" nullable:"false" doc:"The Nodes its units run on"`
	RunNow  runNow              `json:"runNow"`
}

func taskOf(v tasks.View) task {
	_, inWindow := v.Task.Cadence.(tasks.InWindow)
	out := task{
		ID: string(v.Task.ID), Name: v.Task.Name, About: v.Task.About, Group: v.Task.Group.String(), Noun: v.Task.Noun,
		Cadence: cadence{Words: v.Cadence(), InWindow: inWindow, Field: v.CadenceField()},
		Overdue: v.OverdueField(), State: stateOf(v.Status), Nodes: nonNil(v.Nodes), RunNow: runNow{Available: true},
	}
	if v.Last != nil {
		last := runOf(*v.Last)
		out.LastRun = &last
	}
	switch v.Status.(type) {
	case tasks.Running:
		out.RunNow = runNow{Reason: "running"}
	case tasks.Waiting:
		out.RunNow = runNow{Reason: "waiting"}
	}
	return out
}

type run struct {
	ID        int64      `json:"id"`
	Trigger   string     `json:"trigger" enum:"schedule,catch-up,run-now,returned"`
	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt" doc:"Null while it runs"`
	Result    *string    `json:"result" enum:"succeeded,partial,failed,skipped,canceled" doc:"Null while it runs"`
	Summary   string     `json:"summary" doc:"Such as 5 of 6 steps, seedbox offline; empty while it runs"`
	Cause     *string    `json:"cause" doc:"The first failure's cause, null when none"`
	Nodes     []string   `json:"nodes" nullable:"false" doc:"The Nodes its units ran on"`
	Units     []runUnit  `json:"units" nullable:"false" doc:"Each unit's outcome as the run ended, kept after the unit is gone; empty while it runs"`
}

type runUnit struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Node    string  `json:"node" doc:"Empty when the Node is no longer paired"`
	Outcome string  `json:"outcome" enum:"succeeded,failed,skipped,canceled"`
	Cause   *string `json:"cause" doc:"Why it failed or was skipped, null when it succeeded"`
}

func runOf(r tasks.Run) run {
	out := run{ID: int64(r.ID), Trigger: string(r.Trigger), StartedAt: r.Started, Summary: r.Summary, Cause: optional(r.Cause), Nodes: nonNil(r.Nodes), Units: []runUnit{}}
	if !r.Ended.IsZero() {
		ended, result := r.Ended, string(r.Result)
		out.EndedAt, out.Result = &ended, &result
	}
	for _, l := range r.Lines {
		out.Units = append(out.Units, runUnit{Key: string(l.Key), Label: l.Label, Node: l.Node, Outcome: string(l.Outcome), Cause: optional(l.Cause)})
	}
	return out
}

type unit struct {
	Key           string     `json:"key"`
	Label         string     `json:"label"`
	Node          string     `json:"node" doc:"The Node it runs on"`
	Outcome       *string    `json:"outcome" enum:"succeeded,failed,skipped,canceled" doc:"Its last outcome, null before it has run"`
	Cause         *string    `json:"cause" doc:"Why it last failed or was skipped, null otherwise"`
	EndedAt       *time.Time `json:"endedAt" doc:"When it last ended, null before it has run"`
	LastSuccessAt *time.Time `json:"lastSuccessAt" doc:"Null before it has succeeded"`
	DueOnReturn   bool       `json:"dueOnReturn" doc:"It runs once when its Node returns"`
}

func unitOf(u tasks.UnitView) unit {
	out := unit{Key: string(u.Key), Label: u.Label, Node: u.Node, Cause: optional(u.Cause), DueOnReturn: u.DueOnReturn}
	if u.Outcome != "" {
		outcome := string(u.Outcome)
		out.Outcome, out.EndedAt = &outcome, &u.Ended
	}
	if !u.Succeeded.IsZero() {
		out.LastSuccessAt = &u.Succeeded
	}
	return out
}

type group struct {
	Group string `json:"group" enum:"controller,acquire,store,process,connect,reach"`
	Tasks []task `json:"tasks" nullable:"false" minItems:"1"`
}

// groupOrder is the Tasks page's order: the Controller's tasks, then each
// Role Group's.
var groupOrder = append([]tasks.Group{tasks.ControllerGroup}, groupsOf(role.Groups())...)

func groupsOf(gs []role.Group) []tasks.Group {
	out := make([]tasks.Group, len(gs))
	for i, g := range gs {
		out[i] = tasks.GroupOf(g)
	}
	return out
}

func groupsFrom(views []tasks.View) []group {
	out := []group{}
	for _, g := range groupOrder {
		var listed []task
		for _, v := range views {
			if v.Task.Group == g {
				listed = append(listed, taskOf(v))
			}
		}
		if len(listed) > 0 {
			out = append(out, group{Group: g.String(), Tasks: listed})
		}
	}
	return out
}

type schedule struct {
	Window               setting.WindowField   `json:"window"`
	JitterPercent        setting.CountField    `json:"jitterPercent"`
	JitterCap            setting.DurationField `json:"jitterCap"`
	RunsKept             setting.CountField    `json:"runsKept"`
	HeaderDelay          setting.DurationField `json:"headerDelay"`
	PartialWarnIntervals setting.CountField    `json:"partialWarnIntervals"`
	Overdue              setting.ChoiceField   `json:"overdue"`
}

func scheduleOf(s tasks.Schedule, zone string) schedule {
	return schedule{
		Window:               tasks.MaintenanceWindow.Field(s.Window, zone),
		JitterPercent:        tasks.JitterPercent.Field(s.JitterPercent),
		JitterCap:            tasks.JitterCap.Field(s.JitterCap),
		RunsKept:             tasks.RunsKept.Field(s.RunsKept),
		HeaderDelay:          tasks.HeaderDelay.Field(s.HeaderDelay),
		PartialWarnIntervals: tasks.PartialWarnAfter.Field(s.PartialWarnAfter),
		Overdue:              tasks.OverdueRule.Field(s.Overdue),
	}
}

type overview struct {
	Multinode bool              `json:"multinode" doc:"The install has more than one Node, so the Tasks page shows a Node column"`
	Sentence  []api.OneOf[part] `json:"sentence" nullable:"false" minItems:"1" doc:"The status sentence's parts, in order"`
	Groups    []group           `json:"groups" nullable:"false" doc:"Each group with a shown task, in the page's fixed order"`
	Settings  schedule          `json:"settings"`
}

type overviewOutput struct{ Body overview }

type detail struct {
	Task  task   `json:"task"`
	Units []unit `json:"units" nullable:"false" doc:"Each unit it lists now, in run order"`
	Runs  []run  `json:"runs" nullable:"false" doc:"Its runs, newest first, up to the runs kept"`
}

type detailOutput struct{ Body detail }

type taskOutput struct{ Body task }

type taskInput struct {
	TaskID string `path:"taskId"`
}

type runInput struct {
	RunID int64 `path:"runId"`
}

type runStarted struct {
	RunID int64 `json:"runId"`
}

type runStartedOutput struct{ Body runStarted }

type taskPatch struct {
	Cadence api.Patch[int64]  `json:"cadence,omitempty" doc:"Seconds; null resets it to its default"`
	Overdue api.Patch[string] `json:"overdue,omitempty" doc:"catch-up or wait; null resets it to its default"`
}

type taskPatchInput struct {
	TaskID string `path:"taskId"`
	Body   taskPatch
}

type scheduleOutput struct{ Body schedule }

type schedulePatch struct {
	Window               api.Patch[setting.Times] `json:"window,omitempty" doc:"null resets it to its default"`
	JitterPercent        api.Patch[int64]         `json:"jitterPercent,omitempty" doc:"null resets it to its default"`
	JitterCap            api.Patch[int64]         `json:"jitterCap,omitempty" doc:"Seconds; null resets it to its default"`
	RunsKept             api.Patch[int64]         `json:"runsKept,omitempty" doc:"null resets it to its default"`
	HeaderDelay          api.Patch[int64]         `json:"headerDelay,omitempty" doc:"Seconds; null resets it to its default"`
	PartialWarnIntervals api.Patch[int64]         `json:"partialWarnIntervals,omitempty" doc:"null resets it to its default"`
	Overdue              api.Patch[string]        `json:"overdue,omitempty" doc:"catch-up or wait; null resets it to its default"`
}

type schedulePatchInput struct {
	Body schedulePatch
}

type openRun struct {
	RunID     int64        `json:"runId"`
	TaskID    string       `json:"taskId"`
	Name      string       `json:"name" doc:"The task's name"`
	Noun      setting.Noun `json:"noun" doc:"What its progress counts: 3 of 12 steps"`
	Trigger   string       `json:"trigger" enum:"schedule,catch-up,run-now,returned"`
	StartedAt time.Time    `json:"startedAt"`
	Nodes     []string     `json:"nodes" nullable:"false" doc:"The Nodes its units run on"`
	Done      int          `json:"done" minimum:"0"`
	Total     int          `json:"total" minimum:"0"`
}

type openRuns struct {
	HeaderDelaySeconds int64     `json:"headerDelaySeconds" doc:"How long a scheduled run is open before the header shows it; a run-now run shows at once"`
	Runs               []openRun `json:"runs" nullable:"false" doc:"Oldest first"`
}

type openRunsOutput struct{ Body openRuns }

func Routes(r *api.Router, c Clock) {
	o := r.Owner("tasks", api.Admin)
	h := handlers{c}

	api.Register(o, api.Admin, api.Op{
		ID:      "listTasks",
		Method:  http.MethodGet,
		Path:    "/tasks",
		Summary: "List every shown task by group, with the status sentence and the schedule settings",
		Status:  http.StatusOK,
	}, h.listTasks)

	api.Register(o, api.Admin, api.Op{
		ID:      "getTaskSettings",
		Method:  http.MethodGet,
		Path:    "/tasks/settings",
		Summary: "Get the schedule settings",
		Status:  http.StatusOK,
	}, h.getTaskSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateTaskSettings",
		Method:  http.MethodPatch,
		Path:    "/tasks/settings",
		Summary: "Change the schedule settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange},
	}, h.updateTaskSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "listOpenTaskRuns",
		Method:  http.MethodGet,
		Path:    "/tasks/runs/open",
		Summary: "List every open run, with how long a scheduled one waits before the header shows it",
		Status:  http.StatusOK,
	}, h.listOpenTaskRuns)

	api.Register(o, api.Admin, api.Op{
		ID:      "cancelTaskRun",
		Method:  http.MethodDelete,
		Path:    "/tasks/runs/{runId}",
		Summary: "Cancel an open run",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{runNotOpen},
	}, h.cancelTaskRun)

	api.Register(o, api.Admin, api.Op{
		ID:      "getTask",
		Method:  http.MethodGet,
		Path:    "/tasks/{taskId}",
		Summary: "Get one shown task with its units and its runs",
		Status:  http.StatusOK,
		Errors:  []*api.Code{taskUnknown},
	}, h.getTask)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateTask",
		Method:  http.MethodPatch,
		Path:    "/tasks/{taskId}",
		Summary: "Change a task's cadence or overdue rule",
		Status:  http.StatusOK,
		Errors:  []*api.Code{taskUnknown, settingapi.ErrOutOfRange},
	}, h.updateTask)

	api.Register(o, api.Admin, api.Op{
		ID:      "runTask",
		Method:  http.MethodPost,
		Path:    "/tasks/{taskId}/runs",
		Summary: "Run a task now over every unit; its schedule stays as it was",
		Status:  http.StatusAccepted,
		Errors:  []*api.Code{taskUnknown, taskRunning, taskWaiting},
	}, h.runTask)
}

type handlers struct{ clock Clock }

func (h handlers) listTasks(ctx context.Context, _ identity.Admin, _ *struct{}) (*overviewOutput, error) {
	s := open(h.clock)
	o, err := s.Overview(ctx)
	if err != nil {
		return nil, err
	}
	out := overview{Multinode: o.Multinode, Groups: groupsFrom(o.Tasks), Settings: scheduleOf(o.Schedule, s.Zone())}
	for _, p := range o.Sentence {
		out.Sentence = append(out.Sentence, partOf(p))
	}
	return &overviewOutput{Body: out}, nil
}

func (h handlers) getTask(ctx context.Context, _ identity.Admin, in *taskInput) (*detailOutput, error) {
	d, err := open(h.clock).Task(ctx, tasks.TaskID(in.TaskID))
	if err != nil {
		return nil, answer(err)
	}
	out := detail{Task: taskOf(d.View), Units: []unit{}, Runs: []run{}}
	for _, u := range d.Units {
		out.Units = append(out.Units, unitOf(u))
	}
	for _, r := range d.Runs {
		out.Runs = append(out.Runs, runOf(r))
	}
	return &detailOutput{Body: out}, nil
}

func (h handlers) runTask(ctx context.Context, admin identity.Admin, in *taskInput) (*runStartedOutput, error) {
	var by storage.UUID
	if account, ok := admin.Account(); ok {
		by = account.Profile().ID.UUID
	}
	id, err := open(h.clock).RunNow(ctx, tasks.TaskID(in.TaskID), by)
	if err != nil {
		return nil, answer(err)
	}
	return &runStartedOutput{Body: runStarted{RunID: int64(id)}}, nil
}

func (h handlers) cancelTaskRun(_ context.Context, _ identity.Admin, in *runInput) (*struct{}, error) {
	if err := open(h.clock).Cancel(tasks.RunID(in.RunID)); err != nil {
		return nil, answer(err)
	}
	return &struct{}{}, nil
}

func (h handlers) updateTask(ctx context.Context, _ identity.Admin, in *taskPatchInput) (*taskOutput, error) {
	s := open(h.clock)
	id := tasks.TaskID(in.TaskID)
	d, err := s.Task(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	if p := in.Body.Cadence; p.Sent() {
		v, err := d.View.Task.Setting().Parse(p.Value())
		if err != nil {
			return nil, answer(err)
		}
		if err := s.SetCadence(ctx, id, v); err != nil {
			return nil, answer(err)
		}
	}
	if p := in.Body.Overdue; p.Sent() {
		v, err := d.View.OverdueSetting().Parse(p.Value())
		if err != nil {
			return nil, answer(err)
		}
		if err := s.SetOverdue(ctx, id, v); err != nil {
			return nil, answer(err)
		}
	}
	if d, err = s.Task(ctx, id); err != nil {
		return nil, answer(err)
	}
	return &taskOutput{Body: taskOf(d.View)}, nil
}

func (h handlers) getTaskSettings(ctx context.Context, _ identity.Admin, _ *struct{}) (*scheduleOutput, error) {
	return currentSchedule(ctx, open(h.clock))
}

func (h handlers) updateTaskSettings(ctx context.Context, _ identity.Admin, in *schedulePatchInput) (*scheduleOutput, error) {
	s := open(h.clock)
	c, err := scheduleChange(in.Body)
	if err != nil {
		return nil, answer(err)
	}
	if err := s.SetSchedule(ctx, c); err != nil {
		return nil, err
	}
	return currentSchedule(ctx, s)
}

// scheduleChange parses each sent member: all or none are stored.
func scheduleChange(b schedulePatch) (tasks.ScheduleChange, error) {
	var c tasks.ScheduleChange
	if p := b.Window; p.Sent() {
		v, err := tasks.MaintenanceWindow.Parse(p.Value())
		if err != nil {
			return c, err
		}
		c.Window = &v
	}
	for _, f := range []struct {
		patch api.Patch[int64]
		parse func(*int64) (setting.Stored[int64], error)
		into  **setting.Stored[int64]
	}{
		{b.JitterPercent, tasks.JitterPercent.Parse, &c.JitterPercent},
		{b.JitterCap, tasks.JitterCap.Parse, &c.JitterCap},
		{b.RunsKept, tasks.RunsKept.Parse, &c.RunsKept},
		{b.HeaderDelay, tasks.HeaderDelay.Parse, &c.HeaderDelay},
		{b.PartialWarnIntervals, tasks.PartialWarnAfter.Parse, &c.PartialWarnAfter},
	} {
		if !f.patch.Sent() {
			continue
		}
		v, err := f.parse(f.patch.Value())
		if err != nil {
			return c, err
		}
		*f.into = &v
	}
	if p := b.Overdue; p.Sent() {
		v, err := tasks.OverdueRule.Parse(p.Value())
		if err != nil {
			return c, err
		}
		c.Overdue = &v
	}
	return c, nil
}

func currentSchedule(ctx context.Context, s *tasks.Scheduler) (*scheduleOutput, error) {
	sched, err := s.Schedule(ctx)
	if err != nil {
		return nil, err
	}
	return &scheduleOutput{Body: scheduleOf(sched, s.Zone())}, nil
}

func (h handlers) listOpenTaskRuns(ctx context.Context, _ identity.Admin, _ *struct{}) (*openRunsOutput, error) {
	s := open(h.clock)
	sched, err := s.Schedule(ctx)
	if err != nil {
		return nil, err
	}
	running, err := s.Running(ctx)
	if err != nil {
		return nil, err
	}
	out := openRuns{HeaderDelaySeconds: int64(tasks.HeaderDelay.Value(sched.HeaderDelay) / time.Second), Runs: []openRun{}}
	for _, a := range running {
		out.Runs = append(out.Runs, openRun{
			RunID: int64(a.Run), TaskID: string(a.Task), Name: a.Name, Noun: a.Noun, Trigger: string(a.Trigger),
			StartedAt: a.Since, Nodes: nonNil(a.Nodes), Done: a.Done, Total: a.Total,
		})
	}
	return &openRunsOutput{Body: out}, nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
