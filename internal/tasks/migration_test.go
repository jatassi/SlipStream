package tasks_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

func TestTheRebuildKeepsRunsAndListsWhatAnOpenRunQueued(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(schema storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{schema}, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
			storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	before := open(tasks.Schema().Through(1))
	err := before.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, stmt := range []string{
			`INSERT INTO tasks_task (id, first_seen_at) VALUES ('tidy', 1)`,
			`INSERT INTO tasks_run (id, task_id, kind, started_at, ended_at, result, summary, units) VALUES (7, 'tidy', 'schedule', 1, 2, 'succeeded', '2 steps', '[]')`,
			`INSERT INTO tasks_run (id, task_id, kind, started_at) VALUES (8, 'tidy', 'catch-up', 3)`,
			`INSERT INTO tasks_unit (task_id, unit_key, position, label, last_run_id, last_outcome, first_seen_at) VALUES ('tidy', 'b', 1, 'B', 7, 'succeeded', 1)`,
			`INSERT INTO tasks_unit (task_id, unit_key, position, label, last_run_id, last_outcome, first_seen_at) VALUES ('tidy', 'a', 0, 'A', 7, 'succeeded', 1)`,
		} {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}
	lib := open(tasks.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	want := []string{`7|succeeded|[]`, `8|NULL|[{"key":"a","label":"A"},{"key":"b","label":"B"}]`}
	if got := rows(t, lib, "SELECT id, result, units FROM tasks_run ORDER BY id"); !equal(got, want) {
		t.Fatalf("runs = %q, want %q", got, want)
	}
	if got := rows(t, lib, "SELECT unit_key, last_run_id, last_owned FROM tasks_unit ORDER BY position"); !equal(got, []string{"a|7|0", "b|7|0"}) {
		t.Fatalf("units = %q, want each still on run 7 and none owned", got)
	}
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO tasks_run (task_id, kind, started_at, ended_at, result) VALUES ('tidy', 'event', 4, 5, 'succeeded')`)
		return err
	})
	if err != nil {
		t.Fatalf("an event run = %v, want the rebuilt CHECK to take it", err)
	}
}
