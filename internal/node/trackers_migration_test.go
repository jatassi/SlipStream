package node

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestTheTrackersSetMigratesALibraryFromBeforeS29(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(schemas []storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, schemas, preMigrationHook, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	beforeS29 := slices.DeleteFunc(LibrarySchemas(noTypes), func(s storage.Schema[storage.Library]) bool { return s.Set() == "trackers" })
	old := open(beforeS29)
	seedpool := storage.NewUUID()
	err := old.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, 'seedpool', 'private', 0)`, seedpool)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := open(LibrarySchemas(noTypes))
	t.Cleanup(func() { _ = lib.Close() })
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO trackers_rule (tracker_id, source, clears, seed_seconds, updated_at) VALUES (?, 'admin', 'time', 3600, 0)`, seedpool)
		return err
	})
	if err != nil {
		t.Fatalf("an Admin rule on a Tracker from before S29: %v", err)
	}
	var settings, unset, violations int
	err = lib.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM trackers_settings),
		(SELECT count(*) FROM trackers_settings WHERE margin_seconds IS NULL AND warn_after_seconds IS NULL AND goal_combine IS NULL),
		(SELECT count(*) FROM pragma_foreign_key_check)`).Scan(&settings, &unset, &violations)
	if err != nil {
		t.Fatal(err)
	}
	if settings != 1 || unset != 1 || violations != 0 {
		t.Errorf("after S29 the library holds %d seeding settings rows, %d at every default, and %d foreign key violations, want 1, 1 and none",
			settings, unset, violations)
	}
}
