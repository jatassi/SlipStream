package tasks_test

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

func openReconciling(t *testing.T, lib *storage.DB[storage.Library], reconciling *atomic.Bool, catalog ...tasks.Task) *tasks.Scheduler {
	t.Helper()
	s, err := tasks.Open(t.Context(), tasks.Config{
		Library: lib, Catalog: catalog, Roster: roster{}, Zone: time.UTC, Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Hub: newHub(t), Changed: quiet{}, Progressed: still{}, Log: slog.New(slog.DiscardHandler),
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return reconciling.Load(), nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAPausedRunCatchesUpAtTheEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		var reconciling atomic.Bool
		reconciling.Store(true)
		var ran, keptRunning atomic.Int32
		paused := tidy(released())
		paused.ID, paused.Name, paused.WhileReconciling = "backlog", "Backlog search", tasks.Pauses
		paused.Run = func(context.Context, tasks.Unit, *tasks.Progress) error { ran.Add(1); return nil }
		running := tidy(released())
		running.Run = func(context.Context, tasks.Unit, *tasks.Progress) error { keptRunning.Add(1); return nil }
		s := openReconciling(t, lib, &reconciling, paused, running)
		clock(t, s)
		for _, id := range []tasks.TaskID{"backlog", "tidy"} {
			if _, err := s.RunNow(t.Context(), id, admin); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		if ran.Load() != 0 || keptRunning.Load() != 3 {
			t.Fatalf("while Reconciling the paused task ran %d units and the other %d, want 0 and 3", ran.Load(), keptRunning.Load())
		}
		want := []string{"backlog|a|skipped|Skipped because SlipStream is Reconciling after a restore. Backlog search starts again once Reconciling finishes.|1"}
		if got := rows(t, lib, "SELECT task_id, unit_key, last_outcome, last_cause, last_owned FROM tasks_unit WHERE task_id = 'backlog' AND unit_key = 'a'"); !equal(got, want) {
			t.Fatalf("the paused unit = %q, want %q", got, want)
		}
		reconciling.Store(false)
		s.Reconciled()
		synctest.Wait()
		if ran.Load() != 3 {
			t.Fatalf("after Reconciling the paused task ran %d units, want 3", ran.Load())
		}
		if got := rows(t, lib, "SELECT kind, result FROM tasks_run WHERE task_id = 'backlog' ORDER BY id"); !equal(got, []string{"run-now|skipped", "schedule|succeeded"}) {
			t.Fatalf("the paused task's runs = %q, want a skipped Run now and a scheduled run that succeeded", got)
		}
	})
}

func TestATaskThatDeclaresNoWhileReconcilingIsRefused(t *testing.T) {
	lib := storagetest.Library(t, tasks.Schema())
	bare := tidy(nil)
	bare.WhileReconciling = 0
	defer func() {
		if recover() == nil {
			t.Fatal("Open took a task that declares no WhileReconciling")
		}
	}()
	open(t, lib, bare)
}
