package tasks_test

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

// gates is a task's units a, b and c. A run of a unit waits until its key is
// let through, and gates records which units ran, where, and how many ran
// at once.
type gates struct {
	mu      sync.Mutex
	open    map[tasks.UnitKey]chan struct{}
	ran     []string
	running map[tasks.UnitKey]int
	twice   bool
}

func newGates(keys ...tasks.UnitKey) *gates {
	g := &gates{open: map[tasks.UnitKey]chan struct{}{}, running: map[tasks.UnitKey]int{}}
	for _, k := range keys {
		g.open[k] = make(chan struct{})
	}
	return g
}

// let lets every run of key through from now on.
func (g *gates) let(key tasks.UnitKey) {
	g.mu.Lock()
	defer g.mu.Unlock()
	close(g.open[key])
	delete(g.open, key)
}

// enter records a run of u starting and answers its gate, nil once let.
func (g *gates) enter(u tasks.Unit) chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.running[u.Key]++
	g.twice = g.twice || g.running[u.Key] > 1
	return g.open[u.Key]
}

func (g *gates) leave(u tasks.Unit) {
	node, _ := u.Place.Node()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.running[u.Key]--
	g.ran = append(g.ran, string(u.Key)+"@"+map[role.NodeID]string{home: "home", seedbox: "seedbox"}[node])
}

func (g *gates) runs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.ran)
}

func gated(g *gates, units ...tasks.Unit) tasks.Task {
	t := tidy(nil)
	t.Units = func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) { return units, nil }
	t.Run = func(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error {
		gate := g.enter(u)
		defer g.leave(u)
		if gate == nil {
			return nil
		}
		select {
		case <-gate:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return t
}

func here(keys ...tasks.UnitKey) []tasks.Unit {
	var out []tasks.Unit
	for _, k := range keys {
		out = append(out, tasks.Unit{Key: k, Label: string(k), Place: tasks.OnController()})
	}
	return out
}

func start(t *testing.T, s *tasks.Scheduler, st tasks.Start) tasks.Started {
	t.Helper()
	started, err := s.Start(t.Context(), st)
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return started
}

func scan(trigger tasks.Trigger, keys ...tasks.UnitKey) tasks.Start {
	st := tasks.Start{Task: "tidy", Units: keys, Trigger: trigger}
	if trigger == tasks.ByRunNow {
		st.By = admin
	}
	return st
}

func TestStartOpensSubset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		g := newGates()
		s := open(t, lib, gated(g, here("a", "b", "c")...))
		clock(t, s)
		started := start(t, s, scan(tasks.ByRunNow, "b"))
		if started.Run != 1 || !slices.Equal(started.Added, []tasks.UnitKey{"b"}) {
			t.Fatalf("Start = %+v, want run 1 with b added", started)
		}
		if got := g.runs(); !equal(got, []string{"b@home"}) {
			t.Fatalf("ran %q, want b alone", got)
		}
		want := []string{`run-now|1|succeeded|1 step|[{"key":"b","label":"b","node":"` + home.String() + `","outcome":"succeeded"}]`}
		if got := rows(t, lib, "SELECT kind, requested_by IS NOT NULL, result, summary, units FROM tasks_run"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
	})
}

func TestStartJoinsOpenRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		g := newGates("b")
		s := open(t, lib, gated(g, here("a", "b", "c")...))
		clock(t, s)
		start(t, s, scan(tasks.ByRunNow, "b"))
		if got := start(t, s, scan(tasks.ByEvent, "c")); got.Run != 1 || !slices.Equal(got.Added, []tasks.UnitKey{"c"}) {
			t.Fatalf("Start of c while b runs = %+v, want c added to run 1", got)
		}
		if got := start(t, s, scan(tasks.ByRunNow, "c")); got.Run != 1 || len(got.Added) != 0 {
			t.Fatalf("Start of c while c waits = %+v, want run 1 joined with nothing added", got)
		}
		g.let("b")
		synctest.Wait()
		if got := g.runs(); !equal(got, []string{"b@home", "c@home"}) {
			t.Fatalf("ran %q, want b then c once each", got)
		}
		if got := rows(t, lib, "SELECT id, kind, result, summary, json_extract(units, '$[1].key') FROM tasks_run"); !equal(got, []string{"1|run-now|succeeded|2 steps|c"}) {
			t.Fatalf("runs = %q, want one run of b and c", got)
		}
	})
}

// TestStartRequeuesRunningKey asks for b again while it runs, on seedbox,
// after seedbox's portable work has started routing to home, and for a
// again after it ended: b runs once more on seedbox only after its first
// run ended, and a runs once more after it.
func TestStartRequeuesRunningKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		g := newGates("b")
		r := &links{link: nodes.Up}
		units := []tasks.Unit{
			{Key: "a", Label: "a", Place: tasks.OnController()},
			{Key: "b", Label: "b", Place: tasks.PortableOn(role.Searching, seedbox)},
		}
		s := openOn(t, lib, r, newHub(t), gated(g, units...))
		clock(t, s)
		start(t, s, scan(tasks.ByRunNow, "a", "b"))
		r.lapse(true)
		for _, key := range []tasks.UnitKey{"b", "a"} {
			if got := start(t, s, scan(tasks.ByRunNow, key)); got.Run != 1 || !slices.Equal(got.Added, []tasks.UnitKey{key}) {
				t.Fatalf("Start of %s = %+v, want it queued again in run 1", key, got)
			}
		}
		if got := g.runs(); !equal(got, []string{"a@home", "a@home"}) {
			t.Fatalf("before b ended, ran %q, want a twice", got)
		}
		g.let("b")
		synctest.Wait()
		if got := g.runs(); !equal(got, []string{"a@home", "a@home", "b@seedbox", "b@seedbox"}) {
			t.Fatalf("ran %q, want b twice on seedbox", got)
		}
		if g.twice {
			t.Fatal("a unit ran twice at once")
		}
		if got := rows(t, lib, "SELECT count(*), max(summary) FROM tasks_run"); !equal(got, []string{"1|2 steps"}) {
			t.Fatalf("runs = %q, want one run of a and b", got)
		}
	})
}

func TestWholeTaskStartRefusesWhenNothingAdded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		g := newGates("b")
		s := open(t, lib, gated(g, here("a", "b", "c")...))
		clock(t, s)
		start(t, s, scan(tasks.ByEvent, "b"))
		id, err := s.RunNow(t.Context(), "tidy", admin)
		if err != nil || id != 1 {
			t.Fatalf("Run now while b runs = %d, %v, want a and c added to run 1", id, err)
		}
		synctest.Wait()
		if _, err := s.RunNow(t.Context(), "tidy", admin); !errors.As(err, new(*tasks.ErrRunning)) {
			t.Fatalf("Run now with every unit in the run = %v, want ErrRunning", err)
		}
		g.let("b")
		synctest.Wait()
		if got := g.runs(); !equal(got, []string{"b@home", "a@home", "c@home"}) {
			t.Fatalf("ran %q, want b, then a and c once each", got)
		}
		if got := rows(t, lib, "SELECT kind, result, summary FROM tasks_run"); !equal(got, []string{"event|succeeded|3 steps"}) {
			t.Fatalf("runs = %q, want the event run of all three", got)
		}
	})
}

func TestStartNamingOfflineUnitSkipsThenReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		g := newGates()
		r := &links{}
		s := openOn(t, lib, r, newHub(t), gated(g, tasks.Unit{Key: "s", Label: "S", Place: tasks.TiedTo(seedbox)}))
		clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); !errors.As(err, new(*tasks.ErrWaiting)) {
			t.Fatalf("Run now of a task waiting for seedbox = %v, want ErrWaiting", err)
		}
		start(t, s, scan(tasks.ByRunNow, "s"))
		if got := rows(t, lib, "SELECT unit_key, last_outcome, last_cause, due_on_return FROM tasks_unit"); !equal(got, []string{"s|skipped|seedbox is offline|1"}) {
			t.Fatalf("units = %q, want s skipped and due on return", got)
		}
		r.set(nodes.Up)
		r.changed()
		synctest.Wait()
		want := []string{"run-now|skipped|Waiting for seedbox", "returned|succeeded|1 step"}
		if got := rows(t, lib, "SELECT kind, result, summary FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
		if got := g.runs(); !equal(got, []string{"s@seedbox"}) {
			t.Fatalf("ran %q, want s once on seedbox", got)
		}
	})
}

type frames struct {
	mu   sync.Mutex
	seen []tasks.Progressed
}

func (f *frames) Publish(_ context.Context, v tasks.Progressed) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = append(f.seen, v)
}

func (f *frames) all() []tasks.Progressed {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.seen)
}

func TestStepUnknownTotal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		release := make(chan struct{})
		walk := tidy(nil)
		walk.Run = func(_ context.Context, u tasks.Unit, p *tasks.Progress) error {
			if u.Key != "b" {
				return nil
			}
			p.Step(tasks.Step{Phase: "listing", Done: 1204})
			<-release
			return nil
		}
		f := &frames{}
		s, err := tasks.Open(t.Context(), tasks.Config{
			Library: lib, Catalog: []tasks.Task{walk}, Roster: roster{}, Zone: time.UTC, Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
			Hub: newHub(t), Changed: quiet{}, Progressed: f, Log: slog.New(slog.DiscardHandler),
		})
		if err != nil {
			t.Fatal(err)
		}
		clock(t, s)
		run, err := s.RunNow(t.Context(), "tidy", admin)
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		id, steps, ok := s.Steps("tidy")
		want := map[tasks.UnitKey]tasks.Step{"b": {Phase: "listing", Done: 1204}}
		if !ok || id != run || len(steps) != 1 || steps["b"] != want["b"] {
			t.Fatalf("Steps = %d %+v %t, want run %d with b listing 1204 of an unknown total", id, steps, ok, run)
		}
		frame := tasks.Progressed{RunID: run, TaskID: "tidy", Done: 1, Total: 3, Unit: &tasks.UnitStep{Key: "b", Step: want["b"]}}
		if seen := f.all(); !slices.ContainsFunc(seen, func(p tasks.Progressed) bool {
			return p.Unit != nil && *p.Unit == *frame.Unit && p.Done == frame.Done && p.Total == frame.Total && p.RunID == run
		}) {
			t.Fatalf("frames = %+v, want one with b's step and 1 of 3 units done", seen)
		}
		close(release)
		synctest.Wait()
		if _, _, ok := s.Steps("tidy"); ok {
			t.Fatal("Steps answers a run after it closed")
		}
	})
}

// skipping lists a and b. a succeeds and b ends with skip's error.
func skipping(id tasks.TaskID, name string, skip func(string) error) tasks.Task {
	t := tidy(nil)
	t.ID, t.Name = id, name
	t.Units = func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
		return here("a", "b"), nil
	}
	t.Run = func(_ context.Context, u tasks.Unit, _ *tasks.Progress) error {
		if u.Key == "b" {
			return skip("Movies is unmounted.")
		}
		return nil
	}
	return t
}

func TestSkipRollUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		s := open(t, lib, skipping("tidy", "Tidy", tasks.Skip))
		clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		units := `[{"key":"a","label":"a","node":"` + home.String() + `","outcome":"succeeded"},{"key":"b","label":"b","node":"` + home.String() + `","outcome":"skipped","cause":"Movies is unmounted."}]`
		if got := rows(t, lib, "SELECT result, summary, units FROM tasks_run"); !equal(got, []string{"partial|1 of 2 steps, 1 skipped|" + units}) {
			t.Fatalf("runs = %q, want b skipped with its cause", got)
		}
		if got := rows(t, lib, "SELECT unit_key, last_outcome, due_on_return, last_owned FROM tasks_unit ORDER BY position"); !equal(got, []string{"a|succeeded|0|0", "b|skipped|0|0"}) {
			t.Fatalf("units = %q, want b skipped, not due on any return", got)
		}
	})
}

func TestOwnedSkipRaisesNoWarning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		s := open(t, lib, skipping("tidy", "Tidy", tasks.Skip), skipping("scan", "Scan", tasks.SkipOwned))
		clock(t, s)
		time.Sleep(33 * time.Hour)
		synctest.Wait()
		want := []string{"tidy|warn|Tidy|Tidy hasn't finished b since Jan 1 08:00. Run it now to try b again.|run tidy|Not finished,Unfinished since"}
		if got := answer(t, s, health.ScheduledTasksID); !equal(got, want) {
			t.Fatalf("scheduled tasks = %q, want only Tidy's plain skip", got)
		}
		if got := rows(t, lib, "SELECT task_id, unit_key, last_outcome, last_owned FROM tasks_unit WHERE unit_key = 'b' ORDER BY task_id"); !equal(got, []string{"scan|b|skipped|1", "tidy|b|skipped|0"}) {
			t.Fatalf("units = %q, want Scan's skip owned", got)
		}
	})
}

func TestEventTrigger(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		nightly := gated(newGates(), here("a", "b")...)
		nightly.Cadence = tasks.InWindow{Days: setting.Duration{Key: "tasks.tidy.cadence", Label: "Tidy", Default: 24 * time.Hour, Min: 24 * time.Hour, Max: 7 * 24 * time.Hour, Step: 24 * time.Hour}}
		s := open(t, lib, nightly)
		clock(t, s)
		synctest.Wait()
		slot := rows(t, lib, "SELECT slot_started_at FROM tasks_task")
		start(t, s, scan(tasks.ByEvent, "a"))
		if got := rows(t, lib, "SELECT kind, requested_by, result, summary FROM tasks_run"); !equal(got, []string{"event|NULL|succeeded|1 step"}) {
			t.Fatalf("runs = %q, want an event run of a", got)
		}
		if got := rows(t, lib, "SELECT slot_started_at FROM tasks_task"); !equal(got, slot) {
			t.Fatalf("slot_started_at = %q, want %q as before the event run", got, slot)
		}
		d, err := s.Task(t.Context(), "tidy")
		if err != nil {
			t.Fatal(err)
		}
		if d.View.Last == nil || d.View.Last.Trigger != tasks.ByEvent {
			t.Fatalf("the last run is %+v, want the event run", d.View.Last)
		}
	})
}

func TestARunCutShortClosesWithOnlyTheUnitsItQueued(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		g := newGates("b", "c")
		s := open(t, lib, gated(g, here("a", "b", "c")...))
		stop := clock(t, s)
		start(t, s, scan(tasks.ByRunNow, "b"))
		start(t, s, scan(tasks.ByRunNow, "c"))
		stop()

		open(t, lib, gated(g, here("a", "b", "c")...))
		want := []string{`canceled|0 of 2 steps|[{"key":"b","label":"b","outcome":"canceled"},{"key":"c","label":"c","outcome":"canceled"}]`}
		if got := rows(t, lib, "SELECT result, summary, units FROM tasks_run"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
	})
}
