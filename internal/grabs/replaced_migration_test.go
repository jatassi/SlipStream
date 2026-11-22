package grabs_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestNeedsYouKeepsEveryFailureACellStillReadsFromAcrossTheReplacedMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openSets(t, path, downloadclients.Schema(), grabs.Schema().Through(16), indexers.Schema(), files.Schema())
	node, work, target := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	items := []storage.UUID{storage.NewUUID(), storage.NewUUID(), storage.NewUUID()}
	earlier, lone, pack, later := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	failed := func(id, item storage.UUID, at int) tableRow {
		g := fetchingGrab(id, work, item, node)
		g.r["stage"], g.r["failed_stage"], g.r["failure_code"], g.r["failure_message"] = "failed", "importing", "import_failed", ""
		g.r["created_at"], g.r["updated_at"] = at, at
		return g
	}
	cell := func(grab, item storage.UUID, position int) tableRow {
		return tableRow{"grabs_target", row{"grab_id": grab, "position": position, "item_id": item, "target_id": target}}
	}
	fetching := fetchingGrab(later, work, items[1], node)
	fetching.r["created_at"], fetching.r["updated_at"] = 3, 3
	rows := []tableRow{
		{"nodes_node", row{"id": node, "public_key": hash(32), "name": "home", "paired_at": 0}},
		workRow(work, "Dune"),
		{"targets_target", row{"id": target, "name": "1080p", "created_at": 0}},
	}
	for i, item := range items {
		rows = append(rows, tableRow{"works_item", row{"id": item, "work_id": work, "key": item.String(), "label": "One", "position": i + 1}})
	}
	rows = append(rows,
		failed(earlier, items[0], 0), cell(earlier, items[0], 0),
		failed(lone, items[0], 1), cell(lone, items[0], 0),
		failed(pack, items[1], 2), cell(pack, items[1], 0), cell(pack, items[2], 1),
		fetching, cell(later, items[1], 0),
	)
	run(t, old, insertRows(rows...))
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path)
	t.Cleanup(func() { _ = lib.Close() })
	if got := migratedNeedsYou(t, lib); got != 2 {
		t.Fatalf("Needs you counts %d after the migration, want 2: the lone failure and the pack a cell still reads from", got)
	}
	id := storage.NewUUID()
	newest := fetchingGrab(id, work, items[2], node)
	newest.r["created_at"], newest.r["updated_at"] = 4, 4
	run(t, lib, insertRows(newest, cell(id, items[2], 0)))
	if got := migratedNeedsYou(t, lib); got != 1 {
		t.Fatalf("Needs you counts %d once a later Grab takes the pack's last cell, want 1", got)
	}
}

func migratedNeedsYou(t *testing.T, lib library) int {
	t.Helper()
	var n int
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		n, err = grabs.NeedsYou(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
