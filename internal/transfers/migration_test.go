package transfers_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// TestGroupsAndPathsKeepsEveryFile migrates a Transfer of three files, one
// verified and one placed, through 00002's rebuild: each keeps its group,
// source and marks, none is dropped, and a path source fits beside them.
// 00003 then moves the Grab's Transfer to its Import, every file in group 0.
func TestGroupsAndPathsKeepsEveryFile(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(ledger storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, []storage.Schema[storage.Library]{
			nodes.Schema(), role.LibrarySchema(), works.Schema(), roots.Schema(), health.Schema(), ledger,
		}, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	exec := func(lib *storage.DB[storage.Library], query string, args ...any) {
		t.Helper()
		err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, query, args...)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	old := open(transfers.LibrarySchema().Through(1))
	node, root, id := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	exec(old, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)", node)
	exec(old, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'alpha', ?, '/media', 1)", root, node)
	exec(old, "INSERT INTO transfers_record (id, purpose, job, source_node, destination_kind, destination_root, created_at) VALUES (?, 'grab', 'heat', ?, 'root', ?, 1)", id, node, root)
	exec(old, `INSERT INTO transfers_file (transfer_id, position, item, name, source_kind, info_hash, file_index, size, verified_place, placed) VALUES
		(?, 0, 'e1', 'Episode 1.mkv', 'torrent', zeroblob(20), 0, 10, 1, 1),
		(?, 1, 'e2', 'Episode 2.mkv', 'torrent', zeroblob(20), 1, 20, 1, 0),
		(?, 2, '', 'Episode 3.mkv', 'torrent', zeroblob(20), 2, 30, NULL, 0)`, id, id, id)
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	query := func(lib *storage.DB[storage.Library], query string) []string {
		t.Helper()
		rows, err := lib.QueryContext(ctx, query, id)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatal(err)
			}
			got = append(got, s)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return got
	}

	lib := open(transfers.LibrarySchema().Through(2))
	got := query(lib, `SELECT position || ' ' || group_key || ' ' || source_kind || ' ' || file_index || ' ' || size || ' ' ||
		ifnull(verified_place, '-') || ' ' || placed || ' ' || dropped || ' ' || ifnull(path, '-') FROM transfers_file WHERE transfer_id = ? ORDER BY position`)
	check(t, "files after the migration", got, []string{"0 e1 torrent 0 10 1 1 0 -", "1 e2 torrent 1 20 1 0 0 -", "2  torrent 2 30 - 0 0 -"})
	exec(lib, "INSERT INTO transfers_file (transfer_id, position, group_key, name, source_kind, path, size) VALUES (?, 3, '', 'Heat.mkv', 'path', '/downloads/Heat.mkv', 40)", id)
	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE transfers_file SET dropped = 1 WHERE transfer_id = ? AND position = 0", id)
		return err
	})
	if err == nil {
		t.Error("a placed file was dropped")
	}
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}

	lib = open(transfers.LibrarySchema())
	t.Cleanup(func() { _ = lib.Close() })
	got = query(lib, `SELECT r.purpose || ' ' || f.position || ' ' || f.group_key FROM transfers_file f
		JOIN transfers_record r ON r.id = f.transfer_id WHERE f.transfer_id = ? ORDER BY f.position`)
	check(t, "files after 00003", got, []string{"import 0 0", "import 1 0", "import 2 0", "import 3 0"})
}

// TestBackupPurposeKeepsEveryTransfer migrates an import Transfer through
// 00004's rebuild: it keeps its files, marks and failure, reads Normal,
// and a Backup copy fits beside it with its Node and folder stored, while
// a root Transfer naming a Node, or a copy with no destination id, is
// refused.
func TestBackupPurposeKeepsEveryTransfer(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	open := func(ledger storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, []storage.Schema[storage.Library]{
			nodes.Schema(), role.LibrarySchema(), works.Schema(), roots.Schema(), health.Schema(), ledger,
		}, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	exec := func(lib *storage.DB[storage.Library], query string, args ...any) error {
		t.Helper()
		return lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, query, args...)
			return err
		})
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	old := open(transfers.LibrarySchema().Through(3))
	node, root, id := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	must(exec(old, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)", node))
	must(exec(old, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'alpha', ?, '/media', 1)", root, node))
	must(exec(old, "INSERT INTO transfers_record (id, purpose, job, source_node, destination_kind, destination_root, created_at, failure_code, failure_detail) VALUES (?, 'import', 'heat', ?, 'root', ?, 1, 'damaged', 'read_back')", id, node, root))
	must(exec(old, `INSERT INTO transfers_file (transfer_id, position, group_key, name, source_kind, info_hash, file_index, size, verified_place, placed) VALUES
		(?, 0, '0', 'Heat.mkv', 'torrent', zeroblob(20), 0, 10, 1, 1),
		(?, 1, '0', 'Heat.srt', 'torrent', zeroblob(20), 1, 20, NULL, 0)`, id, id))
	must(old.Close())

	lib := open(transfers.LibrarySchema())
	t.Cleanup(func() { _ = lib.Close() })
	var got []string
	rows, err := lib.QueryContext(ctx, `SELECT r.priority || ' ' || r.failure_code || ' ' || f.position || ' ' || ifnull(f.verified_place, '-') || ' ' || f.placed
		FROM transfers_file f JOIN transfers_record r ON r.id = f.transfer_id WHERE f.transfer_id = ? ORDER BY f.position`, id)
	must(err)
	for rows.Next() {
		var s string
		must(rows.Scan(&s))
		got = append(got, s)
	}
	must(rows.Err())
	check(t, "files after 00004", got, []string{"normal damaged 0 1 1", "normal damaged 1 - 0"})

	copyID := storage.NewUUID()
	must(exec(lib, "INSERT INTO transfers_record (id, purpose, job, source_node, destination_kind, destination_backup, destination_node, destination_folder, priority, created_at) VALUES (?, 'backup', 'copy', ?, 'backup', randomblob(16), ?, '/backups', 'low', 2)", copyID, node, node))
	must(exec(lib, "INSERT INTO transfers_file (transfer_id, position, group_key, name, source_kind, source_folder, size) VALUES (?, 0, '', 'slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age', 'backup', 'backups', 30)", copyID))
	if exec(lib, "INSERT INTO transfers_record (id, purpose, job, source_node, destination_kind, destination_root, destination_node, created_at) VALUES (?, 'import', 'node', ?, 'root', ?, ?, 3)", storage.NewUUID(), node, root, node) == nil {
		t.Error("a root Transfer stored a Node")
	}
	if exec(lib, "INSERT INTO transfers_record (id, purpose, job, source_node, destination_kind, destination_node, destination_folder, created_at) VALUES (?, 'backup', 'no-id', ?, 'backup', ?, '/backups', 4)", storage.NewUUID(), node, node) == nil {
		t.Error("a copy with no destination id was stored")
	}
}
