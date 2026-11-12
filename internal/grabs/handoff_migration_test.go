package grabs_test

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

// TestTheHandoffMigrationCarriesEveryGrabPastItsDownload migrates a Grab
// at each of main's Stages, each with one fill and a history up to its
// Stage, through grabs 00012. Probing and transferring become importing,
// their history entries fold into one importing entry at the first of
// them, and each Grab past its download is carried with what it chose.
func TestTheHandoffMigrationCarriesEveryGrabPastItsDownload(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openSets(t, path, downloadclients.Schema(), grabs.Schema().Through(10), indexers.Schema(), files.Schema())
	node, torrent, work, item, target := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	rows := []tableRow{
		{"nodes_node", row{"id": node, "public_key": hash(32), "name": "home", "paired_at": 0}},
		{"downloadclients_torrent", row{
			"id": torrent, "node_id": node, "info_hash": hash(20), "metainfo": []byte{0}, "name": "Heat.1995.1080p.BluRay",
			"size": 1, "private": 0, "trackers": "[]", "content_type": "fixture", "wanted": "started", "added_order": 1, "added_at": 0,
		}},
		workRow(work, "Heat"),
		{"works_item", row{"id": item, "work_id": work, "key": "i1", "label": "Heat", "position": 1}},
		{"targets_target", row{"id": target, "name": "1080p", "created_at": 0}},
	}
	byStage, grabRows := grabAtEveryStage(work, item, node, torrent, target)
	rows = append(rows, grabRows...)
	for _, stage := range oldStages {
		id := byStage[stage]
		rows = append(rows, tableRow{"grabs_fill", row{"grab_id": id, "position": 0, "target_id": target}})
		for i, s := range oldStages[:slices.Index(oldStages, stage)+1] {
			if stage == "failed" && (s == "transferring" || s == "importing" || s == "imported") {
				continue
			}
			rows = append(rows, tableRow{"grabs_stage", row{"grab_id": id, "stage": s, "entered_at": i + 1}})
		}
	}
	run(t, old, insertRows(rows...))
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path)
	t.Cleanup(func() { _ = lib.Close() })
	for stage, want := range map[string]string{
		"fetching":     "fetching - [fetching 1]",
		"downloading":  "downloading - [fetching 1 downloading 2]",
		"probing":      "importing - [fetching 1 downloading 2 importing 3]",
		"transferring": "importing - [fetching 1 downloading 2 importing 3]",
		"importing":    "importing - [fetching 1 downloading 2 importing 3]",
		"imported":     "imported - [fetching 1 downloading 2 importing 3 imported 6]",
		"failed":       "failed importing [fetching 1 downloading 2 importing 3 failed 7]",
	} {
		var now, failedAt string
		q := `SELECT stage, coalesce(failed_stage, '-') FROM grabs_grab WHERE id = ?`
		if err := lib.QueryRowContext(ctx, q, byStage[stage]).Scan(&now, &failedAt); err != nil {
			t.Fatalf("a %s Grab: %v", stage, err)
		}
		if got := fmt.Sprintf("%s %s %v", now, failedAt, history(t, lib, byStage[stage])); got != want {
			t.Errorf("after the migration a %s Grab reads %q, want %q", stage, got, want)
		}
	}

	var carried []grabs.Carry
	err := lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		carried, err = grabs.Carried(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range carried {
		got[c.Was] = fmt.Sprintf("%v %d %s %v", c.Grab.UUID == byStage[c.Was], c.FileIndex, c.Path, slices.Equal(c.Fills, []targets.TargetID{{UUID: target}}))
	}
	want := map[string]string{
		"probing":      "true 0  true",
		"transferring": "true 0 Heat (1995)/Heat (1995).mkv true",
		"importing":    "true 0 Heat (1995)/Heat (1995).mkv true",
	}
	if len(carried) != 3 || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("the carried Grabs read %q, want %q", got, want)
	}
}

// history is grab's Stage entries in lib, each "stage entered_at", in
// order.
func history(t *testing.T, lib library, grab storage.UUID) []string {
	t.Helper()
	rows, err := lib.QueryContext(t.Context(), `SELECT stage || ' ' || entered_at FROM grabs_stage WHERE grab_id = ? ORDER BY entered_at`, grab)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
