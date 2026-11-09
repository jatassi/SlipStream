package imports

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheUpgradeLeavesRootsAlreadySeenReadyForTheNightlyScan(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(t.Context(), path, schemas, func(context.Context, *storage.Unmigrated, storage.Plan) error { return nil },
			storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	seen, unseen := roots.ID{UUID: storage.NewUUID()}, roots.ID{UUID: storage.NewUUID()}
	before := open(works.Schema(), roots.Schema())
	err := before.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		node := storage.NewUUID()
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO roots_root (id, content_type, node_id, path, is_default, place) VALUES (?, 'movie', ?, '/media/movies', 1, 2)`, []any{seen, node}},
			{`INSERT INTO roots_root (id, content_type, node_id, path, is_default, place) VALUES (?, 'series', ?, '/media/series', 1, 1)`, []any{unseen, node}},
			{`INSERT INTO roots_seen (root_id, place, volume_id) VALUES (?, 2, 'vol')`, []any{seen}},
		} {
			if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
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
	lib := open(works.Schema(), roots.Schema(), Schema())
	t.Cleanup(func() { _ = lib.Close() })

	rows, err := db.New(lib).ListCompared(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RootID != seen || rows[0].Place != 2 || !rows[0].ComparedAt.IsZero() {
		t.Fatalf("after the upgrade the scan records %+v, want the root seen ready at place 2, never compared, and none for the root never seen", rows)
	}
}
