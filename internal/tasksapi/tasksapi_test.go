package tasksapi_test

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/tasksapi"
)

var (
	home    = role.NodeID{UUID: storage.NewUUID()}
	seedbox = role.NodeID{UUID: storage.NewUUID()}
	// offline is when seedbox's link went down.
	offline = time.Date(2026, 10, 16, 21, 31, 0, 0, time.UTC)
)

// roster is home, and seedbox offline since offline.
type roster struct{}

func (roster) Self() role.NodeID { return home }

func (r roster) Get(ctx context.Context, id role.NodeID) (nodes.Node, error) {
	list, err := r.List(ctx)
	for _, n := range list {
		if n.ID == id {
			return n, err
		}
	}
	return nodes.Node{}, nodes.ErrNotFound
}

func (roster) List(context.Context) ([]nodes.Node, error) {
	return []nodes.Node{
		{ID: home, Name: "home", Self: true, Link: nodes.Local},
		{ID: seedbox, Name: "seedbox", Link: nodes.Down, Since: offline},
	}, nil
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type clock struct{ s *tasks.Scheduler }

func (c clock) Tasks() (*tasks.Scheduler, bool) { return c.s, true }

var tidyCadence = setting.Duration{Key: "tasks.tidy.cadence", Label: "Runs every", Default: 8 * time.Hour, Min: time.Hour, Max: 24 * time.Hour, Step: time.Hour}

// task is a task of units, each of whose Run blocks until its run ends.
// Its Wait rule keeps the clock from opening it at start.
func task(id tasks.TaskID, units ...tasks.Unit) tasks.Task {
	return tasks.Task{
		ID:      id,
		Name:    strings.ToUpper(string(id[:1])) + string(id[1:]),
		About:   "Tidies up.",
		Noun:    "step",
		Cadence: tasks.Every{Interval: tidyCadence},
		Overdue: tasks.Wait,
		Units: func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return units, nil
		},
		Run: func(ctx context.Context, _ tasks.Unit, _ *tasks.Progress) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
}

func onController(key tasks.UnitKey) tasks.Unit {
	return tasks.Unit{Key: key, Label: strings.ToUpper(string(key)), Place: tasks.OnController()}
}

// serve serves the tasks operations over a running clock of catalog.
func serve(t *testing.T, catalog ...tasks.Task) *ownertest.Owner {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), tasks.Schema())
	hub := live.NewHub([]live.Event{nodes.ChangedEvent})
	t.Cleanup(hub.Close)
	s, err := tasks.Open(t.Context(), tasks.Config{
		Library: lib, Catalog: catalog, Roster: roster{}, Zone: time.UTC, Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Hub: hub, Changed: discard[tasks.Changed]{}, Progressed: discard[tasks.Progressed]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = s.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return ownertest.Serve(t, lib, func(r *api.Router) { tasksapi.Routes(r, clock{s}) })
}

const overdueOptions = `"options":[{"value":"catch-up","label":"Catch up"},{"value":"wait","label":"Wait"}]`

// tidyTask is the wire task tidy, its cadence every hours, its state,
// last run and Run now as given.
func tidyTask(hours int, state, lastRun, runNow string) string {
	isDefault := hours == 8
	return fmt.Sprintf(`{"id":"tidy","name":"Tidy","about":"Tidies up.","group":"controller",`+
		`"cadence":{"words":"Every %d h","inWindow":false,"field":{"key":"tasks.tidy.cadence","label":"Runs every","source":"database","applies":"live","isDefault":%t,"kind":"duration","value":%d,"default":28800,"min":3600,"max":86400,"step":3600}},`+
		`"overdue":{"key":"tasks.tidy.overdue","label":"When overdue","source":"database","applies":"live","isDefault":true,"kind":"choice","value":"wait","default":"wait",%s},`+
		`"state":%s,%s"nodes":["home"],"runNow":%s}`, hours, isDefault, hours*3600, overdueOptions, state, lastRun, runNow)
}

const (
	idle      = `{"kind":"idle","next":"T"}`
	available = `{"available":true}`
)

func TestRunNowAnswersItsRunThenThatTheTaskRuns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := serve(t, task("tidy", onController("a"), onController("b")))
		synctest.Wait()
		o.Expect(t, http.MethodPost, "/api/v1/tasks/tidy/runs", "", http.StatusAccepted, `{"runId":1}`)
		o.Expect(t, http.MethodPost, "/api/v1/tasks/tidy/runs", "", http.StatusConflict,
			`{"status":409,"code":"task_running","title":"The task has a run open","detail":"The task has been running since 2000-01-01T00:00:00Z"}`)
		o.Expect(t, http.MethodGet, "/api/v1/tasks/runs/open", "", http.StatusOK,
			`{"headerDelaySeconds":5,"runs":[{"runId":1,"taskId":"tidy","name":"Tidy","trigger":"run-now","startedAt":"T","nodes":["home"],"done":0,"total":2}]}`)
		o.Expect(t, http.MethodGet, "/api/v1/tasks/tidy", "", http.StatusOK, `{"task":`+
			tidyTask(8, `{"kind":"running","runId":1,"trigger":"run-now","since":"T","done":0,"total":2}`, "", `{"available":false,"reason":"running"}`)+
			`,"units":[`+
			`{"key":"a","label":"A","node":"home","outcome":null,"cause":null,"endedAt":null,"lastSuccessAt":null,"dueOnReturn":false},`+
			`{"key":"b","label":"B","node":"home","outcome":null,"cause":null,"endedAt":null,"lastSuccessAt":null,"dueOnReturn":false}],`+
			`"runs":[{"id":1,"trigger":"run-now","startedAt":"T","endedAt":null,"result":null,"summary":"","cause":null,"nodes":[],"units":[]}]}`)

		if status, reply := o.Call(t, http.MethodDelete, "/api/v1/tasks/runs/1", ""); status != http.StatusNoContent || reply != "" {
			t.Fatalf("Cancel = %d %q, want 204 and no body", status, reply)
		}
		synctest.Wait()
		o.Expect(t, http.MethodDelete, "/api/v1/tasks/runs/1", "", http.StatusNotFound,
			`{"status":404,"code":"run_not_open","title":"No open run has this id"}`)
		lines := `"units":[{"key":"a","label":"A","node":"home","outcome":"canceled","cause":null},{"key":"b","label":"B","node":"home","outcome":"canceled","cause":null}]`
		last := `{"id":1,"trigger":"run-now","startedAt":"T","endedAt":"T","result":"canceled","summary":"0 of 2 steps","cause":null,"nodes":["home"],` + lines + `}`
		o.Expect(t, http.MethodGet, "/api/v1/tasks/tidy", "", http.StatusOK, `{"task":`+
			tidyTask(8, idle, `"lastRun":`+last+`,`, available)+
			`,"units":[`+
			`{"key":"a","label":"A","node":"home","outcome":"canceled","cause":null,"endedAt":"T","lastSuccessAt":null,"dueOnReturn":false},`+
			`{"key":"b","label":"B","node":"home","outcome":"canceled","cause":null,"endedAt":"T","lastSuccessAt":null,"dueOnReturn":false}],`+
			`"runs":[`+last+`]}`)
		o.Expect(t, http.MethodGet, "/api/v1/tasks/runs/open", "", http.StatusOK, `{"headerDelaySeconds":5,"runs":[]}`)
	})
}

func TestACadenceOutsideItsRangeAnswersItsSentenceAndNullResets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := serve(t, task("tidy", onController("a")))
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/tidy", `{"cadence":60}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 1 h and 1 day."}`)
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/tidy", `{"overdue":"later"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Choose Catch up or Wait."}`)
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/tidy", `{"cadence":7200}`, http.StatusOK, tidyTask(2, idle, "", available))
		o.Expect(t, http.MethodGet, "/api/v1/tasks/tidy", "", http.StatusOK, `{"task":`+tidyTask(2, idle, "", available)+
			`,"units":[{"key":"a","label":"A","node":"home","outcome":null,"cause":null,"endedAt":null,"lastSuccessAt":null,"dueOnReturn":false}],"runs":[]}`)
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/tidy", `{"cadence":null}`, http.StatusOK, tidyTask(8, idle, "", available))
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/nothing", `{"cadence":null}`, http.StatusNotFound,
			`{"status":404,"code":"task_unknown","title":"No task shown has this id"}`)
	})
}

func TestTheScheduleSettingsSaveAndReset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		o := serve(t)
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/settings", `{"jitterPercent":51,"runsKept":20}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 0 and 50%."}`)
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/settings", `{"window":{"start":"01:00","end":"03:00"},"runsKept":20,"overdue":"wait"}`, http.StatusOK,
			schedule(`"isDefault":false,"kind":"window","value":{"start":"01:00","end":"03:00"}`, `"isDefault":false,"kind":"count","value":20`, `"isDefault":false,"kind":"choice","value":"wait"`))
		o.Expect(t, http.MethodPatch, "/api/v1/tasks/settings", `{"window":null,"runsKept":null,"overdue":null}`, http.StatusOK,
			schedule(`"isDefault":true,"kind":"window","value":{"start":"04:00","end":"06:00"}`, `"isDefault":true,"kind":"count","value":50`, `"isDefault":true,"kind":"choice","value":"catch-up"`))
		o.Expect(t, http.MethodGet, "/api/v1/tasks/settings", "", http.StatusOK,
			schedule(`"isDefault":true,"kind":"window","value":{"start":"04:00","end":"06:00"}`, `"isDefault":true,"kind":"count","value":50`, `"isDefault":true,"kind":"choice","value":"catch-up"`))
	})
}

// schedule is the schedule settings with the window, the runs kept and the
// overdue rule as given.
func schedule(window, runsKept, overdue string) string {
	about := `"source":"database","applies":"live",`
	return `{"window":{"key":"tasks.window","label":"Maintenance window",` + about + window + `,"default":{"start":"04:00","end":"06:00"},"zone":"UTC"},` +
		`"jitterPercent":{"key":"tasks.jitterPercent","label":"Jitter",` + about + `"isDefault":true,"kind":"count","value":10,"default":10,"min":0,"max":50,"unit":"%"},` +
		`"jitterCap":{"key":"tasks.jitterCap","label":"Longest jitter",` + about + `"isDefault":true,"kind":"duration","value":300,"default":300,"min":0,"max":3600,"step":1},` +
		`"runsKept":{"key":"tasks.runsKept","label":"Runs kept per task",` + about + runsKept + `,"default":50,"min":1,"max":1000,"unit":"runs"},` +
		`"headerDelay":{"key":"tasks.headerDelay","label":"Show a scheduled run in the header after",` + about + `"isDefault":true,"kind":"duration","value":5,"default":5,"min":0,"max":300,"step":1},` +
		`"partialWarnIntervals":{"key":"tasks.partialWarnIntervals","label":"Warn when a unit has not succeeded for",` + about + `"isDefault":true,"kind":"count","value":3,"default":3,"min":1,"max":100,"unit":"intervals"},` +
		`"overdue":{"key":"tasks.overdue","label":"Overdue tasks",` + about + overdue + `,"default":"catch-up",` + overdueOptions + `}}`
}

func TestTheTasksPageListsShownTasksAndWhatWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		far := task("far", tasks.Unit{Key: "node:seedbox", Label: "seedbox", Place: tasks.TiedTo(seedbox)})
		far.Group = tasks.GroupOf(role.Acquire)
		o := serve(t, task("tidy", onController("a")), task("hidden"), far)
		synctest.Wait()
		o.Expect(t, http.MethodPost, "/api/v1/tasks/far/runs", "", http.StatusConflict,
			`{"status":409,"code":"task_waiting","title":"Every unit waits for a Node that can't start work","detail":"Every unit waits for seedbox, offline since 2026-10-16T21:31:00Z"}`)
		o.Expect(t, http.MethodPost, "/api/v1/tasks/hidden/runs", "", http.StatusNotFound,
			`{"status":404,"code":"task_unknown","title":"No task shown has this id"}`)
		o.Expect(t, http.MethodGet, "/api/v1/tasks/hidden", "", http.StatusNotFound,
			`{"status":404,"code":"task_unknown","title":"No task shown has this id"}`)
		status, body := o.Call(t, http.MethodGet, "/api/v1/tasks", "")
		farTask := `{"id":"far","name":"Far","about":"Tidies up.","group":"acquire",` +
			`"cadence":{"words":"Every 8 h","inWindow":false,"field":{"key":"tasks.tidy.cadence","label":"Runs every","source":"database","applies":"live","isDefault":true,"kind":"duration","value":28800,"default":28800,"min":3600,"max":86400,"step":3600}},` +
			`"overdue":{"key":"tasks.far.overdue","label":"When overdue","source":"database","applies":"live","isDefault":true,"kind":"choice","value":"wait","default":"wait",` + overdueOptions + `},` +
			`"state":{"kind":"waiting","node":"seedbox","link":"down","since":"T"},"nodes":["seedbox"],"runNow":{"available":false,"reason":"waiting"}}`
		want := `{"multinode":true,"sentence":[{"kind":"warning","text":"1 task is waiting for seedbox.","href":"/tasks/far"}],` +
			`"groups":[{"group":"controller","tasks":[` + tidyTask(8, idle, "", available) + `]},{"group":"acquire","tasks":[` + farTask + `]}],` +
			`"settings":` + schedule(`"isDefault":true,"kind":"window","value":{"start":"04:00","end":"06:00"}`, `"isDefault":true,"kind":"count","value":50`, `"isDefault":true,"kind":"choice","value":"catch-up"`) + "}\n"
		if got := ownertest.Masked(body); status != http.StatusOK || got != want {
			t.Fatalf("GET /tasks = %d\n%s\nwant\n%s", status, got, want)
		}
	})
}
