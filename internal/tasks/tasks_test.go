package tasks_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

var home = role.NodeID{UUID: storage.NewUUID()}

type roster struct{}

func (roster) Self() role.NodeID { return home }

func (roster) Get(_ context.Context, id role.NodeID) (nodes.Node, error) {
	if id != home {
		return nodes.Node{}, nodes.ErrNotFound
	}
	return nodes.Node{ID: home, Name: "home", Self: true, Link: nodes.Local}, nil
}

func (roster) Reach(role.NodeID) (nodes.LinkState, time.Time) { return nodes.Local, time.Time{} }

func (r roster) List(ctx context.Context) ([]nodes.Node, error) {
	n, err := r.Get(ctx, home)
	return []nodes.Node{n}, err
}

type quiet struct{}

func (quiet) Publish(context.Context, tasks.Changed) {}

type still struct{}

func (still) Publish(context.Context, tasks.Progressed) {}

var admin = storage.NewUUID()

// tidy is a task of three Controller units whose second blocks until
// release closes or its run ends. Its Wait rule keeps the clock from
// opening it at start.
func tidy(release <-chan struct{}) tasks.Task {
	return tasks.Task{
		ID:      "tidy",
		Name:    "Tidy",
		Noun:    "step",
		Cadence: tasks.Every{Interval: setting.Duration{Key: "tasks.tidy.cadence", Label: "Tidy", Default: 8 * time.Hour, Min: time.Hour, Max: 24 * time.Hour, Step: time.Hour}},
		Overdue: tasks.Wait,
		Units: func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{
				{Key: "a", Label: "A", Place: tasks.OnController()},
				{Key: "b", Label: "B", Place: tasks.OnController()},
				{Key: "c", Label: "C", Place: tasks.OnController()},
			}, nil
		},
		Run: func(ctx context.Context, u tasks.Unit, _ *tasks.Progress) error {
			if u.Key != "b" {
				return nil
			}
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}

func open(t *testing.T, lib *storage.DB[storage.Library], catalog ...tasks.Task) *tasks.Scheduler {
	t.Helper()
	return openOn(t, lib, roster{}, newHub(t), catalog...)
}

func newHub(t *testing.T) *live.Hub {
	h := live.NewHub([]live.Event{nodes.ChangedEvent, tasks.ChangedEvent, tasks.ProgressEvent})
	t.Cleanup(h.Close)
	return h
}

func openOn(t *testing.T, lib *storage.DB[storage.Library], r tasks.Roster, hub *live.Hub, catalog ...tasks.Task) *tasks.Scheduler {
	t.Helper()
	s, err := tasks.Open(t.Context(), tasks.Config{
		Library: lib, Catalog: catalog, Roster: r, Zone: time.UTC, Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Hub: hub, Changed: quiet{}, Progressed: still{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// clock runs s's clock until the returned stop is called or the test ends.
func clock(t *testing.T, s *tasks.Scheduler) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = s.Run(ctx)
	}()
	stop = func() {
		cancel()
		<-done
	}
	t.Cleanup(stop)
	return stop
}

// rows is every row query returns, each as its values joined by "|".
func rows(t *testing.T, lib *storage.DB[storage.Library], query string) []string {
	t.Helper()
	r, err := lib.QueryContext(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	cols, err := r.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for r.Next() {
		values := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := r.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		cells := make([]string, len(values))
		for i, v := range values {
			cells[i] = "NULL"
			if v.Valid {
				cells[i] = v.String
			}
		}
		out = append(out, strings.Join(cells, "|"))
	}
	if err := r.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestASecondOpenRunIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		release := make(chan struct{})
		s := open(t, lib, tidy(release))
		clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		started := time.Now()
		time.Sleep(time.Minute)
		synctest.Wait()
		_, err := s.RunNow(t.Context(), "tidy", admin)
		if running, ok := errors.AsType[*tasks.ErrRunning](err); !ok || !running.Since.Equal(started) {
			t.Fatalf("Run now while running = %v, want ErrRunning since %s", err, started)
		}
		err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO tasks_run (task_id, kind, started_at) VALUES ('tidy', 'schedule', 1)")
			return err
		})
		if err == nil || !strings.Contains(err.Error(), "UNIQUE constraint failed: tasks_run.task_id") {
			t.Fatalf("a second open run = %v, want tasks_run_open to refuse it", err)
		}
		close(release)
		synctest.Wait()
		want := []string{"run-now|succeeded|3 steps|NULL"}
		if got := rows(t, lib, "SELECT kind, result, summary, cause FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
	})
}

func TestOpenClosesARunLeftOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		s := open(t, lib, tidy(nil))
		stop := clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		stop()
		if got := rows(t, lib, "SELECT result FROM tasks_run"); !equal(got, []string{"NULL"}) {
			t.Fatalf("after a stop mid-run the run's result is %q, want it open", got)
		}
		slot := rows(t, lib, "SELECT slot_started_at FROM tasks_task")

		open(t, lib, tidy(nil))
		units := `[{"key":"a","label":"A","node":"` + home.String() + `","outcome":"succeeded"},{"key":"b","label":"B","outcome":"canceled"},{"key":"c","label":"C","outcome":"canceled"}]`
		wantRuns := []string{"run-now|canceled|1 of 3 steps|SlipStream stopped mid-run|" + units}
		if got := rows(t, lib, "SELECT kind, result, summary, cause, units FROM tasks_run"); !equal(got, wantRuns) {
			t.Fatalf("runs = %q, want %q", got, wantRuns)
		}
		if got := rows(t, lib, "SELECT slot_started_at FROM tasks_task"); !equal(got, slot) {
			t.Fatalf("slot_started_at = %q, want %q as before the run", got, slot)
		}

		before := dump(t, lib)
		open(t, lib, tidy(nil))
		if after := dump(t, lib); after != before {
			t.Fatalf("a second Open changed the tasks set:\n%s\nwant\n%s", after, before)
		}
	})
}

func TestARunCutShortRerunsOnceAtStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		eager := tidy(nil)
		eager.Overdue = tasks.CatchUp
		stop := clock(t, open(t, lib, eager))
		synctest.Wait()
		stop()

		release := make(chan struct{})
		close(release)
		clock(t, open(t, lib, tidy(release)))
		synctest.Wait()
		want := []string{
			"catch-up|canceled|1 of 3 steps|SlipStream stopped mid-run",
			"catch-up|succeeded|3 steps|NULL",
		}
		if got := rows(t, lib, "SELECT kind, result, summary, cause FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
		if got := rows(t, lib, "SELECT slot_started_at = (SELECT started_at FROM tasks_run WHERE id = 2) FROM tasks_task"); !equal(got, []string{"1"}) {
			t.Fatalf("the slot is not the rerun's start: %q", got)
		}
	})
}

func dump(t *testing.T, lib *storage.DB[storage.Library]) string {
	var b bytes.Buffer
	for _, table := range []string{"tasks_task", "tasks_unit", "tasks_run", "tasks_settings"} {
		fmt.Fprintln(&b, table, rows(t, lib, "SELECT * FROM "+table+" ORDER BY 1"))
	}
	return b.String()
}

func TestOpenDropsTasksTheCatalogNoLongerLists(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		release := make(chan struct{})
		close(release)
		s := open(t, lib, tidy(release))
		clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		open(t, lib)
		if got := rows(t, lib, "SELECT (SELECT count(*) FROM tasks_task) + (SELECT count(*) FROM tasks_unit) + (SELECT count(*) FROM tasks_run)"); !equal(got, []string{"0"}) {
			t.Fatalf("rows left of an unlisted task: %q", got)
		}
	})
}

func TestPruneRunsKeepsRunsKeptAndEveryOpenRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		release := make(chan struct{})
		close(release)
		quick := tidy(release)
		slow := tidy(nil)
		slow.ID = "slow"
		s := open(t, lib, quick, slow)
		clock(t, s)
		two, err := tasks.RunsKept.Parse(new(int64(2)))
		if err != nil {
			t.Fatal(err)
		}
		if err := s.SetSchedule(t.Context(), tasks.ScheduleChange{RunsKept: &two}); err != nil {
			t.Fatal(err)
		}
		for range 4 {
			if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
		}
		if _, err := s.RunNow(t.Context(), "slow", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if err := s.PruneRuns(t.Context()); err != nil {
			t.Fatal(err)
		}
		want := []string{"3|tidy|succeeded", "4|tidy|succeeded", "5|slow|NULL"}
		if got := rows(t, lib, "SELECT id, task_id, result FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs after the prune = %q, want %q", got, want)
		}
	})
}

func equal(a, b []string) bool { return strings.Join(a, "\n") == strings.Join(b, "\n") }

// links is home and seedbox, whose link a test sets.
type links struct {
	mu   sync.Mutex
	link nodes.LinkState
}

var seedbox = role.NodeID{UUID: storage.NewUUID()}

func (l *links) set(link nodes.LinkState) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.link = link
}

func (*links) Self() role.NodeID { return home }

func (l *links) Reach(node role.NodeID) (nodes.LinkState, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if node == home {
		return nodes.Local, time.Time{}
	}
	return l.link, time.Time{}
}

func (l *links) Get(ctx context.Context, id role.NodeID) (nodes.Node, error) {
	list, err := l.List(ctx)
	for _, n := range list {
		if n.ID == id {
			return n, err
		}
	}
	return nodes.Node{}, nodes.ErrNotFound
}

func (l *links) List(context.Context) ([]nodes.Node, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return []nodes.Node{{ID: home, Name: "home", Self: true, Link: nodes.Local}, {ID: seedbox, Name: "seedbox", Link: l.link}}, nil
}

func TestANodeReturningRunsItsDueUnitsOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		far := tidy(nil)
		far.Units = func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "a", Label: "A", Place: tasks.OnController()}, {Key: "s", Label: "S", Place: tasks.TiedTo(seedbox)}}, nil
		}
		far.Run = func(context.Context, tasks.Unit, *tasks.Progress) error { return nil }
		r := &links{}
		hub := newHub(t)
		s := openOn(t, lib, r, hub, far)
		clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		r.set(nodes.Up)
		synctest.Wait()
		if got := rows(t, lib, "SELECT count(*) FROM tasks_run"); !equal(got, []string{"1"}) {
			t.Fatalf("runs before nodes.changed = %q, want only the Run now", got)
		}
		nodes.ChangedEvent.Bind(hub).Publish(t.Context(), nodes.Changed{NodeID: seedbox.String()})
		synctest.Wait()
		want := []string{"run-now|partial|1 of 2 steps, seedbox offline", "returned|succeeded|1 step"}
		if got := rows(t, lib, "SELECT kind, result, summary FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
		if got := rows(t, lib, "SELECT unit_key, last_outcome, due_on_return FROM tasks_unit ORDER BY position"); !equal(got, []string{"a|succeeded|0", "s|succeeded|0"}) {
			t.Fatalf("units = %q, want both succeeded and none due", got)
		}
	})
}
