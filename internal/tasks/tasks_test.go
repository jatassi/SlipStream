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

func (roster) Standing(role.NodeID) (nodes.Standing, time.Time) { return nodes.InStep, time.Time{} }

func (roster) Route(_ role.Name, home role.NodeID) (role.NodeID, error) { return home, nil }

func (roster) Changed() <-chan struct{} { return nil }

func (roster) Names(context.Context) (map[role.NodeID]nodes.Named, error) {
	return map[role.NodeID]nodes.Named{home: {ID: home, Name: "home"}}, nil
}

type quiet struct{}

func (quiet) Publish(context.Context, tasks.Changed) {}

type still struct{}

func (still) Publish(context.Context, tasks.Progressed) {}

var admin = storage.NewUUID()

func tidy(release <-chan struct{}) tasks.Task {
	return tasks.Task{
		ID:      "tidy",
		Name:    "Tidy",
		Noun:    setting.Noun{One: "step", Many: "steps"},
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
	h := live.NewHub([]live.Event{tasks.ChangedEvent, tasks.ProgressEvent})
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
		if err := tasks.PruneRuns(t.Context(), lib); err != nil {
			t.Fatal(err)
		}
		want := []string{"3|tidy|succeeded", "4|tidy|succeeded", "5|slow|NULL"}
		if got := rows(t, lib, "SELECT id, task_id, result FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs after the prune = %q, want %q", got, want)
		}
	})
}

func equal(a, b []string) bool { return strings.Join(a, "\n") == strings.Join(b, "\n") }

// links is home and seedbox, which lapsed once lapsed is set: its
// portable work routes to home. Once removed is set, seedbox's row is a
// tombstone.
type links struct {
	mu      sync.Mutex
	link    nodes.LinkState
	lapsed  bool
	removed bool
	board   chan struct{}
}

var seedbox = role.NodeID{UUID: storage.NewUUID()}

func (l *links) set(link nodes.LinkState) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.link = link
}

func (*links) Self() role.NodeID { return home }

func (l *links) lapse(lapsed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lapsed = lapsed
}

func (l *links) Standing(node role.NodeID) (nodes.Standing, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch {
	case node == home, l.link == nodes.Up:
		return nodes.InStep, time.Time{}
	case l.lapsed:
		return nodes.LapsedNode, time.Time{}
	case l.link == nodes.Mismatched:
		return nodes.MismatchedNode, time.Time{}
	}
	return nodes.Offline, time.Time{}
}

func (l *links) Route(_ role.Name, h role.NodeID) (role.NodeID, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lapsed {
		return home, nil
	}
	return h, nil
}

func (l *links) remove() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.removed = true
}

func (l *links) Get(_ context.Context, id role.NodeID) (nodes.Node, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch {
	case id == home:
		return nodes.Node{ID: home, Name: "home", Self: true, Link: nodes.Local}, nil
	case id == seedbox && !l.removed:
		return nodes.Node{ID: seedbox, Name: "seedbox", Link: l.link}, nil
	}
	return nodes.Node{}, nodes.ErrNotFound
}

func (l *links) Changed() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.board == nil {
		l.board = make(chan struct{})
	}
	return l.board
}

// changed closes Changed, as the roster does at its next change.
func (l *links) changed() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.board != nil {
		close(l.board)
	}
	l.board = make(chan struct{})
}

func (l *links) Names(context.Context) (map[role.NodeID]nodes.Named, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return map[role.NodeID]nodes.Named{
		home:    {ID: home, Name: "home"},
		seedbox: {ID: seedbox, Name: "seedbox", Removed: l.removed},
	}, nil
}

// TestARunStillNamesARemovedNode runs a unit on seedbox, then removes
// seedbox: the run's line still names it and says it was removed, and the
// install no longer counts as multinode.
func TestARunStillNamesARemovedNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		far := tidy(nil)
		far.Units = func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "s", Label: "S", Place: tasks.TiedTo(seedbox)}}, nil
		}
		far.Run = func(context.Context, tasks.Unit, *tasks.Progress) error { return nil }
		r := &links{link: nodes.Up}
		s := openOn(t, lib, r, newHub(t), far)
		clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		r.remove()

		d, err := s.Task(t.Context(), "tidy")
		if err != nil {
			t.Fatal(err)
		}
		if len(d.Runs) != 1 || len(d.Runs[0].Lines) != 1 {
			t.Fatalf("runs = %+v, want one run with one line", d.Runs)
		}
		if ln := d.Runs[0].Lines[0]; ln.Node != "seedbox" || ln.NodeID != seedbox || !ln.NodeRemoved {
			t.Fatalf("the line reads %+v, want seedbox, removed", ln)
		}
		if got := d.Runs[0].Nodes; !equal(got, []string{"seedbox"}) {
			t.Fatalf("the run names %q, want seedbox", got)
		}
		if u := d.Units[0]; u.Node != "seedbox" || !u.NodeRemoved {
			t.Fatalf("the unit reads %+v, want seedbox, removed", u)
		}
		o, err := s.Overview(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if o.Multinode {
			t.Fatal("the overview is multinode with seedbox removed, want home alone")
		}
	})
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
			t.Fatalf("runs before the roster changed = %q, want only the Run now", got)
		}
		r.changed()
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

// TestAPortableUnitRunsWhereItsHomeRoutesEachRun runs a unit of
// seedbox's Searching work three times: skipped while seedbox is offline
// within the grace, on home once seedbox lapsed, and on seedbox again once
// it is back.
func TestAPortableUnitRunsWhereItsHomeRoutesEachRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		search := tidy(nil)
		search.Units = func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "s", Label: "S", Place: tasks.PortableOn(role.Searching, seedbox)}}, nil
		}
		var mu sync.Mutex
		var ran []string
		search.Run = func(_ context.Context, u tasks.Unit, _ *tasks.Progress) error {
			node, _ := u.Place.Node()
			mu.Lock()
			defer mu.Unlock()
			ran = append(ran, map[role.NodeID]string{home: "home", seedbox: "seedbox"}[node])
			return nil
		}
		r := &links{}
		s := openOn(t, lib, r, newHub(t), search)
		clock(t, s)
		runNow := func() {
			t.Helper()
			if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
		}
		runNow()
		r.lapse(true)
		runNow()
		r.lapse(false)
		r.set(nodes.Up)
		runNow()
		if want := []string{"home", "seedbox"}; !equal(ran, want) {
			t.Fatalf("the unit ran on %q, want %q", ran, want)
		}
		want := []string{"skipped|Waiting for seedbox", "succeeded|1 step", "succeeded|1 step"}
		if got := rows(t, lib, "SELECT result, summary FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
		lines := rows(t, lib, "SELECT units FROM tasks_run ORDER BY id")
		for i, node := range []role.NodeID{seedbox, home, seedbox} {
			if !strings.Contains(lines[i], node.String()) {
				t.Fatalf("run %d's line is %s, want it on %s", i+1, lines[i], node)
			}
		}
	})
}
