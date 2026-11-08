package node

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/storage"
)

// externalColumns is every column of downloadclients_torrent as S30
// (External Download Clients) left it, blobs in hex.
var externalColumns = []string{
	"hex(id)", "hex(node_id)", "hex(client_id)", "attached", "hex(info_hash)", "hex(metainfo)", "name", "size", "private", "trackers",
	"content_type", "hex(tracker_id)", "save_folder", "wanted", "added_order", "added_at", "removed_at", "removed_reason", "delete_files",
	"parted_at", "freed", "uploaded", "downloaded", "active_seconds", "seeding_seconds", "totals_at", "raw_uploaded", "raw_downloaded",
	"raw_active_seconds", "raw_seeding_seconds", "report", "reported_at", "completion", "completed_at", "fault", "fault_at",
}

// every is each row of query's single text column, in order.
func every(t *testing.T, lib *storage.DB[storage.Library], query string) []string {
	t.Helper()
	rows, err := lib.QueryContext(t.Context(), query)
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

func externalRows(t *testing.T, lib *storage.DB[storage.Library]) []string {
	t.Helper()
	return every(t, lib, "SELECT json_array("+strings.Join(externalColumns, ", ")+") FROM downloadclients_torrent ORDER BY added_order")
}

func torrentIndexes(t *testing.T, lib *storage.DB[storage.Library]) []string {
	t.Helper()
	return every(t, lib, `SELECT name || ': ' || coalesce(sql, '') FROM sqlite_schema
		WHERE type = 'index' AND tbl_name = 'downloadclients_torrent' ORDER BY name`)
}

// TestTheSeedingRebuildKeepsEveryTorrent migrates a library from the head
// before downloadclients 00006, holding built-in and external Torrents,
// Grabs and a goal run on them, and keeps every row, reference and index.
func TestTheSeedingRebuildKeepsEveryTorrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	node, work, item, tracker, client := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	seeding, external, parted, writtenOff := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	const torrent = `INSERT INTO downloadclients_torrent (id, node_id, client_id, attached, info_hash, metainfo, name, size, private, trackers,
		content_type, tracker_id, save_folder, wanted, added_order, added_at, removed_at, removed_reason, delete_files, parted_at, freed,
		uploaded, downloaded, active_seconds, seeding_seconds, totals_at, raw_uploaded, raw_downloaded, raw_active_seconds,
		raw_seeding_seconds, report, reported_at, completion, completed_at, fault, fault_at)
		VALUES (?, ?, ?, ?, randomblob(20), x'64', ?, 1000, 1, '["tracker.test"]', 'fixture', ?, ?, ?, ?, 10, ?, ?, ?, ?, ?,
		400, 1000, 3600, 60, 20, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	grabbed := map[storage.UUID]storage.UUID{storage.NewUUID(): seeding, storage.NewUUID(): external, storage.NewUUID(): parted}
	stmts := []statement{
		{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Media server', 0)`, []any{node}},
		{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
		{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, []any{item, work}},
		{`INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, 'seedpool', 'private', 0)`, []any{tracker}},
		{`INSERT INTO downloadclients_client (id, node_id, kind, name, address, address_key, label, is_on, settings_version, created_at, updated_at)
			VALUES (?, ?, 'transmission', 'Transmission', 'http://192.168.1.20:9091', '192.168.1.20:9091', 'slipstream', 1, 1, 0, 0)`, []any{client, node}},
		{torrent, []any{seeding, node, nil, nil, "Orchard", tracker, "/downloads/fixture", "paused", 1, nil, nil, 0, nil, nil,
			nil, nil, nil, nil, `{"at":"2026-10-19T21:00:00Z","engine":"seeding","done":1000}`, 21, `{"files":[]}`, 21, "unregistered", 15}},
		{torrent, []any{external, node, client, 1, "Harbor", nil, nil, "started", 2, nil, nil, 0, nil, nil,
			9000, 4000, 7200, 3600, nil, nil, nil, nil, nil, nil}},
		{torrent, []any{parted, node, nil, nil, "Attic", tracker, nil, "started", 3, 30, "admin", 1, 40, 900,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}},
		{torrent, []any{writtenOff, node, client, 0, "Meadow", nil, nil, "started", 4, 30, "client", 0, 30, nil,
			10, 10, 10, 10, nil, nil, nil, nil, nil, nil}},
		{`INSERT INTO trackers_goal_run (torrent_id, acted_at, action) VALUES (?, 50, 'pause')`, []any{seeding}},
	}
	for g, tor := range grabbed {
		stmts = append(stmts, statement{`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, stage,
			torrent_id, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Orchard.1080p', 1000, 'downloading', ?, 0, 0)`,
			[]any{g, work, item, node, tor}})
	}
	beforeLaneB := with(LibrarySchemas(noTypes), downloadclients.Schema().Through(5))
	old := openAt(t, path, beforeLaneB)
	err := old.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, s := range stmts {
			if _, err := tx.ExecContext(ctx, s.sql, s.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, indexes := externalRows(t, old), torrentIndexes(t, old)
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path, LibrarySchemas(noTypes))
	t.Cleanup(func() { _ = lib.Close() })
	if after := externalRows(t, lib); !slices.Equal(after, rows) {
		t.Errorf("the rebuild changed the Torrents:\nbefore %v\nafter  %v", rows, after)
	}
	if after := torrentIndexes(t, lib); !slices.Equal(after, indexes) {
		t.Errorf("the rebuild changed the indexes:\nbefore %v\nafter  %v", indexes, after)
	}
	expectReferencesKept(t, lib, grabbed, seeding)
	if got := every(t, lib, `SELECT json_array(announced_seconds, retry) FROM downloadclients_torrent ORDER BY added_order`); !slices.Equal(got, slices.Repeat([]string{"[0,null]"}, 4)) {
		t.Errorf("the rebuilt Torrents' announced seconds and retries are %v, want 0 and none each", got)
	}

	refused := func(sql string, args ...any) error {
		return lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, sql, args...)
			return err
		})
	}
	if err := refused(`UPDATE downloadclients_torrent SET raw_uploaded = 1, raw_downloaded = 1, raw_active_seconds = 1, raw_seeding_seconds = 1 WHERE id = ?`, seeding); err == nil {
		t.Error("the rebuilt table took raw counters on a built-in Torrent, want S30's client_id CHECK to refuse them")
	}
	if err := refused(`UPDATE downloadclients_torrent SET removed_at = 60, removed_reason = 'goal_met' WHERE id = ?`, seeding); err != nil {
		t.Errorf("a seeding goal's remove: %v", err)
	}
}

// expectReferencesKept checks every Grab still names the Torrent grabbed
// maps it to, the goal run still names ran, and the library passes the
// foreign key check.
func expectReferencesKept(t *testing.T, lib *storage.DB[storage.Library], grabbed map[storage.UUID]storage.UUID, ran storage.UUID) {
	t.Helper()
	for grab, torrent := range grabbed {
		var got *storage.UUID
		if err := lib.QueryRowContext(t.Context(), "SELECT torrent_id FROM grabs_grab WHERE id = ?", grab).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got == nil || *got != torrent {
			t.Errorf("Grab %s names Torrent %v after the rebuild, want %s", grab, got, torrent)
		}
	}
	var runs, violations int
	err := lib.QueryRowContext(t.Context(), `SELECT
		(SELECT count(*) FROM trackers_goal_run r JOIN downloadclients_torrent t ON t.id = r.torrent_id WHERE t.id = ?),
		(SELECT count(*) FROM pragma_foreign_key_check)`, ran).Scan(&runs, &violations)
	if err != nil {
		t.Fatal(err)
	}
	if runs != 1 || violations != 0 {
		t.Errorf("after the rebuild %d goal runs name their Torrent and the library has %d foreign key violations, want 1 and none", runs, violations)
	}
}
