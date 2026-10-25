package sim_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/nodes"
	rpctestv1 "github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/rpctest/v1/rpctestv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

type taskDetail struct {
	Task struct {
		State struct {
			Kind string `json:"kind"`
		} `json:"state"`
	} `json:"task"`
	Units []struct {
		Label       string `json:"label"`
		DueOnReturn bool   `json:"dueOnReturn"`
	} `json:"units"`
	Runs []struct {
		ID        int64     `json:"id"`
		Trigger   string    `json:"trigger"`
		Result    string    `json:"result"`
		Summary   string    `json:"summary"`
		StartedAt time.Time `json:"startedAt"`
		Units     []struct {
			Label   string  `json:"label"`
			Node    string  `json:"node"`
			Outcome string  `json:"outcome"`
			Cause   *string `json:"cause"`
		} `json:"units"`
	} `json:"runs"`
}

func (d taskDetail) runs() []string {
	out := []string{}
	for _, r := range d.Runs {
		out = append(out, r.Trigger+"|"+r.Result+"|"+r.Summary)
	}
	return out
}

func (d taskDetail) units() []string {
	out := []string{}
	for _, u := range d.Runs[0].Units {
		cause := ""
		if u.Cause != nil {
			cause = *u.Cause
		}
		out = append(out, strings.Join([]string{u.Label, u.Node, u.Outcome, cause}, "|"))
	}
	return out
}

func (d taskDetail) dueOnReturn() []string {
	out := []string{}
	for _, u := range d.Units {
		if u.DueOnReturn {
			out = append(out, u.Label)
		}
	}
	return out
}

func getTask(t *testing.T, a *worldtest.API, id tasks.TaskID) taskDetail {
	t.Helper()
	return decode[taskDetail](t, "getTask", a.Admin("home", http.MethodGet, "/api/v1/tasks/"+string(id), ""))
}

func runNow(t *testing.T, a *worldtest.API, id tasks.TaskID) int64 {
	t.Helper()
	r := a.Admin("home", http.MethodPost, "/api/v1/tasks/"+string(id)+"/runs", "")
	var started struct {
		RunID int64 `json:"runId"`
	}
	if err := json.Unmarshal([]byte(r.Body), &started); r.Status != http.StatusAccepted || err != nil {
		t.Fatalf("runTask = %d %s, want 202", r.Status, r.Body)
	}
	return started.RunID
}

func sentence(t *testing.T, a *worldtest.API) []string {
	t.Helper()
	overview := decode[struct {
		Sentence []struct {
			Text string `json:"text"`
		} `json:"sentence"`
	}](t, "listTasks", a.Admin("home", http.MethodGet, "/api/v1/tasks", ""))
	out := []string{}
	for _, p := range overview.Sentence {
		out = append(out, p.Text)
	}
	return out
}

func expect(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s:\n got %q\nwant %q", what, got, want)
	}
}

var inUTC = sim.Edit(func(_ string, e *node.Env) { e.Zone = time.UTC })

func partition(t *testing.T, w *sim.World) time.Time {
	t.Helper()
	flip(t, w.Path("home", "seedbox"), fault.Stall{})
	link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
	time.Sleep(link.Keepalive + link.WriteTimeout)
	synctest.Wait()
	return time.Now()
}

func heal(w *sim.World, down time.Time) {
	healed := down.Add(time.Minute)
	time.Sleep(time.Until(healed))
	w.Path("home", "seedbox").Clear("")
	time.Sleep(time.Until(redialAt("home/seedbox", down, healed)))
	synctest.Wait()
}

func at(clock string) time.Time {
	t, err := time.Parse(time.RFC3339, clock)
	if err != nil {
		panic(err)
	}
	return t
}

func TestHousekeepingTidiesEveryNodeAndCatchesUpAReturningWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), inUTC)
		a := w.API()
		time.Sleep(time.Until(at("2000-01-01T04:30:00Z")))
		synctest.Wait()
		d := getTask(t, a, "housekeeping")
		expect(t, "runs by 04:30", d.runs(), []string{"schedule|succeeded|6 steps"})
		expect(t, "units by 04:30", d.units(), []string{
			"Run history|home|succeeded|",
			"Join Codes|home|succeeded|",
			"Sessions|home|succeeded|",
			"Local database on home|home|succeeded|",
			"Local database on seedbox|seedbox|succeeded|",
			"Library database|home|succeeded|",
		})

		down := partition(t, w)
		runNow(t, a, "housekeeping")
		synctest.Wait()
		d = getTask(t, a, "housekeeping")
		expect(t, "runs with seedbox down", d.runs(), []string{"run-now|partial|5 of 6 steps, seedbox offline", "schedule|succeeded|6 steps"})
		expect(t, "due on return", d.dueOnReturn(), []string{"Local database on seedbox"})
		expect(t, "sentence with seedbox down", sentence(t, a), []string{"Every task ran as scheduled."})

		heal(w, down)
		d = getTask(t, a, "housekeeping")
		expect(t, "runs after seedbox returns", d.runs(), []string{"returned|succeeded|1 step", "run-now|partial|5 of 6 steps, seedbox offline", "schedule|succeeded|6 steps"})
		expect(t, "the returned run's units", d.units(), []string{"Local database on seedbox|seedbox|succeeded|"})
		expect(t, "due on return after it ran", d.dueOnReturn(), []string{})
	})
}

func TestHousekeepingSkipsAWorkerOnAnotherBuild(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), inUTC)
		worker := w.Node("seedbox")
		worker.Stop()
		worker.Start(func(e *node.Env) { e.Version = "0.0.1-sim" })
		synctest.Wait()
		a := w.API()
		runNow(t, a, "housekeeping")
		synctest.Wait()
		d := getTask(t, a, "housekeeping")
		expect(t, "runs", d.runs(), []string{"run-now|partial|5 of 6 steps, seedbox runs 0.0.1-sim"})
		if got := d.units()[4]; got != "Local database on seedbox|seedbox|skipped|seedbox runs 0.0.1-sim" {
			t.Fatalf("seedbox's unit is %q, want Skipped as seedbox runs 0.0.1-sim", got)
		}
	})
}

func TestHousekeepingWarnsOnlyOnceAUnitMissedThreeIntervals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), inUTC)
		a := w.API()
		time.Sleep(time.Until(at("2000-01-01T04:30:00Z")))
		synctest.Wait()
		tidied := getTask(t, a, "housekeeping").Runs[0].StartedAt
		partition(t, w)
		time.Sleep(time.Until(at("2000-01-04T03:59:59Z")))
		synctest.Wait()
		d := getTask(t, a, "housekeeping")
		expect(t, "runs after two days", d.runs(), []string{"schedule|partial|5 of 6 steps, seedbox offline", "schedule|partial|5 of 6 steps, seedbox offline", "schedule|succeeded|6 steps"})
		expect(t, "sentence after two days", sentence(t, a), []string{"Every task ran as scheduled."})

		time.Sleep(time.Until(at("2000-01-04T04:30:01Z")))
		synctest.Wait()
		expect(t, "sentence after the third day", sentence(t, a), []string{"Housekeeping hasn't finished Local database on seedbox since " + tidied.Format("Jan 2 15:04") + "."})
	})
}

func held(release <-chan struct{}) tasks.Task {
	return tasks.Task{
		ID:      "held",
		Name:    "Held",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.Every{Interval: setting.Duration{Key: "tasks.held.cadence", Default: time.Hour, Min: time.Hour, Max: 24 * time.Hour, Step: time.Hour}},
		Overdue: tasks.Wait,
		Noun:    setting.Noun{One: "step", Many: "steps"},
		Units: func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "held", Label: "Held", Place: tasks.OnController()}}, nil
		},
		Run: func(ctx context.Context, _ tasks.Unit, _ *tasks.Progress) error {
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}

func TestAControllerRestartMidRunCatchesUpOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		w := sim.Launch(t, home(t), inUTC, sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Test.Tasks = func(node.Half) []tasks.Task { return []tasks.Task{held(release)} }
			}
		}))
		time.Sleep(2 * time.Hour)
		synctest.Wait()
		if d := getTask(t, w.API(), "held"); d.Task.State.Kind != "running" {
			t.Fatalf("held is %s at 02:00, want running", d.Task.State.Kind)
		}
		controller := w.Node("home")
		controller.Stop()
		close(release)
		controller.Start()
		synctest.Wait()
		d := getTask(t, w.API(), "held")
		expect(t, "runs after the restart", d.runs(), []string{"catch-up|succeeded|1 step", "schedule|canceled|0 of 1 step"})
	})
}

type searches struct{ ended chan error }

func (s searches) Search(ctx context.Context, _ *rpctestv1.SearchRequest) (*rpctestv1.SearchResponse, error) {
	<-ctx.Done()
	err := context.Cause(ctx)
	s.ended <- err
	return nil, fmt.Errorf("search: %w", err)
}

func searching(h node.Half) tasks.Task {
	return tasks.Task{
		ID:      "searching",
		Name:    "Searching",
		Group:   tasks.ControllerGroup,
		Cadence: tasks.Every{Interval: setting.Duration{Key: "tasks.searching.cadence", Default: 24 * time.Hour, Min: time.Hour, Max: 24 * time.Hour, Step: time.Hour}},
		Overdue: tasks.Wait,
		Noun:    setting.Noun{One: "search", Many: "searches"},
		Units: func(ctx context.Context, r storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			named, err := nodes.Names(ctx, r)
			if err != nil {
				return nil, err
			}
			i := slices.IndexFunc(named, func(n nodes.Named) bool { return n.Name == "seedbox" })
			if i < 0 {
				return nil, nil
			}
			return []tasks.Unit{{Key: "search", Label: "Search", Place: tasks.PortableOn(role.Searching, named[i].ID)}}, nil
		},
		Run: func(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error {
			box, _ := u.Place.Node()
			_, err := rpctestv1connect.SearchService.On(h.Cluster, box).Search(ctx, &rpctestv1.SearchRequest{})
			return err
		},
	}
}

func TestAPortableUnitWaitsForItsNextSlotAndCancelReachesTheWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), inUTC, sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Test.Tasks = func(h node.Half) []tasks.Task { return []tasks.Task{searching(h)} }
			}
		}))
		handler := searches{ended: make(chan error, 1)}
		rpctestv1connect.SearchService.Mount(w.Node("seedbox").Half().Dispatch, handler)
		a := w.API()
		run := runNow(t, a, "searching")
		synctest.Wait()
		worldtest.Expect(t, "cancel the run", a.Admin("home", http.MethodDelete, fmt.Sprintf("/api/v1/tasks/runs/%d", run), ""), worldtest.Reply{Status: http.StatusNoContent})
		synctest.Wait()
		select {
		case err := <-handler.ended:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("seedbox's handler ended with %v, want context.Canceled", err)
			}
		default:
			t.Fatal("the Cancel never reached seedbox's handler")
		}
		d := getTask(t, a, "searching")
		expect(t, "runs after the Cancel", d.runs(), []string{"run-now|canceled|0 of 1 search"})

		down := partition(t, w)
		runNow(t, a, "searching")
		synctest.Wait()
		d = getTask(t, a, "searching")
		expect(t, "units with seedbox down", d.units(), []string{"Search|seedbox|skipped|seedbox is offline"})
		expect(t, "due on return", d.dueOnReturn(), []string{})
		heal(w, down)
		d = getTask(t, a, "searching")
		expect(t, "runs after seedbox returns", d.runs(), []string{"run-now|skipped|Waiting for seedbox", "run-now|canceled|0 of 1 search"})
	})
}
