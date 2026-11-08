package node

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
)

// with is schemas with each of sets in place of the set of its name.
func with(schemas []storage.Schema[storage.Library], sets ...storage.Schema[storage.Library]) []storage.Schema[storage.Library] {
	out := slices.Clone(schemas)
	for _, s := range sets {
		i := slices.IndexFunc(out, func(o storage.Schema[storage.Library]) bool { return o.Set() == s.Set() })
		out[i] = s
	}
	return out
}

type statement struct {
	sql  string
	args []any
}

func openAt(t *testing.T, path string, schemas []storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	t.Helper()
	lib, err := storage.OpenLibrary(t.Context(), path, schemas, preMigrationHook, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

// seed opens a library at schemas, runs stmts in one Write and closes it,
// answering its Torrents as torrentRows reads them.
func seed(t *testing.T, path string, schemas []storage.Schema[storage.Library], stmts []statement) []string {
	t.Helper()
	lib := openAt(t, path, schemas)
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
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
	rows := torrentRows(t, lib)
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}
	return rows
}

// torrentRows is every Torrent's columns as S28 (Built-in torrent client)
// kept them, one JSON array a Torrent, in added order.
func torrentRows(t *testing.T, lib *storage.DB[storage.Library]) []string {
	t.Helper()
	rows, err := lib.QueryContext(t.Context(), `SELECT json_array(hex(id), hex(node_id), hex(info_hash), hex(metainfo), name, size, private,
		trackers, content_type, hex(tracker_id), save_folder, wanted, added_order, added_at, removed_at, removed_reason, delete_files,
		parted_at, freed, uploaded, downloaded, active_seconds, seeding_seconds, totals_at, report, reported_at, completion, completed_at,
		fault, fault_at) FROM downloadclients_torrent ORDER BY added_order`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			t.Fatal(err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// expectEveryGrabsTorrentKept checks every Grab still names the Torrent
// it named, as want maps them, no Torrent names a client and the library
// passes the foreign key check.
func expectEveryGrabsTorrentKept(t *testing.T, lib *storage.DB[storage.Library], want map[storage.UUID]storage.UUID) {
	t.Helper()
	for grab, torrent := range want {
		var got *storage.UUID
		if err := lib.QueryRowContext(t.Context(), "SELECT torrent_id FROM grabs_grab WHERE id = ?", grab).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got == nil || *got != torrent {
			t.Errorf("Grab %s names Torrent %v after the rebuild, want %s", grab, got, torrent)
		}
	}
	var named, violations int
	err := lib.QueryRowContext(t.Context(), `SELECT
		(SELECT count(*) FROM downloadclients_torrent WHERE client_id IS NOT NULL OR attached IS NOT NULL OR raw_uploaded IS NOT NULL),
		(SELECT count(*) FROM pragma_foreign_key_check)`).Scan(&named, &violations)
	if err != nil {
		t.Fatal(err)
	}
	if named != 0 || violations != 0 {
		t.Errorf("after the rebuild %d Torrents name a client and the library has %d foreign key violations, want none of each", named, violations)
	}
}

func TestTheTorrentRebuildKeepsEveryGrabsTorrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	node, work, item := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	live, removing, parted, writtenOff := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	bound := []struct {
		id, torrent storage.UUID
		stage       string
		failure     any
	}{
		{storage.NewUUID(), live, "downloading", nil},
		{storage.NewUUID(), live, "downloading", nil},
		{storage.NewUUID(), removing, "downloading", nil},
		{storage.NewUUID(), parted, "failed", "removed"},
		{storage.NewUUID(), writtenOff, "failed", "node_written_off"},
	}
	const torrent = `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type,
		save_folder, wanted, added_order, added_at, removed_at, removed_reason, delete_files, parted_at, freed, uploaded, downloaded,
		active_seconds, seeding_seconds, totals_at, report, reported_at, completion, completed_at, fault, fault_at)
		VALUES (?, ?, randomblob(20), x'64', ?, 1000, 1, '["tracker.test"]', 'fixture', ?, ?, ?, 10, ?, ?, ?, ?, ?, 400, 1000, 3600, 60, 20,
		?, ?, ?, ?, ?, ?)`
	stmts := []statement{
		{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Media server', 0)`, []any{node}},
		{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
		{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, []any{item, work}},
		{torrent, []any{live, node, "Orchard", "/downloads/fixture", "paused", 1, nil, nil, 0, nil, nil,
			`{"at":"2026-10-19T21:00:00Z","engine":"seeding","done":1000}`, 21, `{"files":[]}`, 21, "unregistered", 15}},
		{torrent, []any{removing, node, "Attic", nil, "started", 2, 30, "admin", 1, nil, nil, nil, nil, nil, nil, nil, nil}},
		{torrent, []any{parted, node, "Harbor", nil, "started", 3, 30, "mark_failed", 1, 40, 900, nil, nil, nil, nil, nil, nil}},
		{torrent, []any{writtenOff, node, "Meadow", nil, "started", 4, 30, "node_written_off", 0, 30, nil, nil, nil, nil, nil, nil, nil}},
	}
	grabbed := map[storage.UUID]storage.UUID{}
	for _, g := range bound {
		stmts = append(stmts, statement{`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, stage,
			torrent_id, failure_code, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Orchard.1080p', 1000, ?, ?, ?, 0, 0)`,
			[]any{g.id, work, item, node, g.stage, g.torrent, g.failure}})
		grabbed[g.id] = g.torrent
	}
	before := seed(t, path, with(LibrarySchemas(noTypes), downloadclients.Schema().Through(4), grabs.Schema().Through(7)), stmts)

	lib := openAt(t, path, LibrarySchemas(noTypes))
	t.Cleanup(func() { _ = lib.Close() })
	if after := torrentRows(t, lib); !slices.Equal(after, before) {
		t.Errorf("the rebuild changed the Torrents:\nbefore %v\nafter  %v", before, after)
	}
	expectEveryGrabsTorrentKept(t, lib, grabbed)
}

// TestGrabsTorrentsMigrateIntoTheRebuiltTable runs grabs 00005, which
// makes each Grab's Torrent, after the rebuild: the order a library from
// before S28 (Built-in torrent client) migrates in.
func TestGrabsTorrentsMigrateIntoTheRebuiltTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	node, indexer, work, item := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	seeding, dropped := storage.NewUUID(), storage.NewUUID()
	const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
		release_link, stage, info_hash, metainfo, path, seed_dropped_at, created_at, updated_at)
		VALUES (?, ?, ?, 'any', 'search', ?, ?, ?, 1000, 'https://site.test/dl', 'imported', randomblob(20), x'64', '/library/Orchard.mkv', ?, ?, 0)`
	seed(t, path, with(beforeS26(), downloadclients.Schema().Through(4), grabs.Schema().Through(4)), []statement{
		{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Media server', 0)`, []any{node}},
		{`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, []any{indexer, node}},
		{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
		{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, []any{item, work}},
		{grab, []any{seeding, work, item, indexer, node, "Orchard.1080p", nil, 1}},
		{grab, []any{dropped, work, item, indexer, node, "Attic.1080p", 50, 2}},
	})

	lib := openAt(t, path, LibrarySchemas(noTypes))
	t.Cleanup(func() { _ = lib.Close() })
	expectEveryGrabsTorrentKept(t, lib, map[storage.UUID]storage.UUID{seeding: seeding, dropped: dropped})
	var reason *string
	var freed *int64
	err := lib.QueryRowContext(t.Context(), "SELECT removed_reason, freed FROM downloadclients_torrent WHERE id = ? AND parted_at IS NOT NULL", dropped).Scan(&reason, &freed)
	if err != nil {
		t.Fatal(err)
	}
	if reason == nil || downloadclients.RemoveReason(*reason) != downloadclients.NodeWrittenOff || freed != nil {
		t.Errorf("the dropped Grab's Torrent parted %v with %v freed, want node_written_off with none", reason, freed)
	}
}
