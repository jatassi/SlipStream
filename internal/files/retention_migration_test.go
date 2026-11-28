package files_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// retentionSeeds are the rows a library holds before files 00008.
type retentionSeeds struct {
	node, work, item, root, target storage.UUID
	// grabbed is a file a Grab brought, landed twice: copied by an older
	// Import, then moved by a newer one, which forced its fill.
	grabbed, grab, older, newer storage.UUID
	// found is a file a scan found, filling unforced.
	found storage.UUID
}

// stmt is one statement and its arguments.
type stmt struct {
	query string
	args  []any
}

func st(query string, args ...any) stmt { return stmt{query, args} }

func exec(ctx context.Context, tx *storage.Tx[storage.Library], stmts ...stmt) error {
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s.query, s.args...); err != nil {
			return fmt.Errorf("%s: %w", s.query, err)
		}
	}
	return nil
}

const seedGrab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
	stage, failure_code, failed_stage, created_at, updated_at)
	VALUES (?, ?, ?, 'standard', 'search', ?, 'Orchard.WEB-DL', 1, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567',
	'failed', 'fetch_failed', 'fetching', 0, 0)`

func seedRetention(t *testing.T, lib *storage.DB[storage.Library]) retentionSeeds {
	t.Helper()
	s := retentionSeeds{
		node: storage.NewUUID(), work: storage.NewUUID(), item: storage.NewUUID(), target: storage.NewUUID(),
		grabbed: storage.NewUUID(), grab: storage.NewUUID(), older: storage.NewUUID(), newer: storage.NewUUID(), found: storage.NewUUID(),
	}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return exec(ctx, tx,
			st(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, s.node),
			st(`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, s.work),
			st(`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, s.item, s.work),
			st(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0)`, s.target),
		)
	})
	if err != nil {
		t.Fatal(err)
	}
	s.root = rootstest.Root(t, lib, role.NodeID{UUID: s.node}, contenttypetest.ID, t.TempDir()).ID.UUID
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		const file = `INSERT INTO files_file (id, work_id, cut, root_id, path, size, probe, grab_id) VALUES (?, ?, 'standard', ?, ?, 1, '{}', ?)`
		const record = `INSERT INTO files_imported (id, file_id, import_id, grab_id, work_id, placed_by, items, cells, replaced, at)
			VALUES (?, ?, ?, ?, ?, ?, '[]', '[]', '[]', ?)`
		return exec(ctx, tx,
			st(seedGrab, s.grab, s.work, s.item, s.node),
			st(file, s.grabbed, s.work, s.root, "Orchard/grabbed.mkv", s.grab),
			st(file, s.found, s.work, s.root, "Orchard/found.mkv", nil),
			st(`INSERT INTO files_file_item (file_id, item_id, position) VALUES (?, ?, 0), (?, ?, 0)`, s.grabbed, s.item, s.found, s.item),
			st(record, storage.NewUUID(), s.grabbed, s.older, s.grab, s.work, "copy", 1),
			st(record, storage.NewUUID(), s.grabbed, s.newer, s.grab, s.work, "move", 2),
			st(record, storage.NewUUID(), s.grabbed, nil, nil, s.work, "link", 0),
			st(`INSERT INTO files_fill (file_id, target_id, item_id, edition, forced_at) VALUES (?, ?, ?, 'standard', 5)`, s.grabbed, s.target, s.item),
			st(`INSERT INTO files_fill (file_id, target_id, item_id, edition) VALUES (?, ?, ?, 'cinema')`, s.found, s.target, s.item),
			st(`INSERT INTO files_renamed (file_id, root_id, from_path, to_path, renamed_at) VALUES (?, ?, 'a.mkv', 'b.mkv', 0)`, s.grabbed, s.root),
		)
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func openRetention(t *testing.T) (*storage.DB[storage.Library], retentionSeeds) {
	t.Helper()
	open := opener(t)
	before := open(identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema().Through(7))
	s := seedRetention(t, before)
	if err := before.Close(); err != nil {
		t.Fatal(err)
	}
	lib := open(identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), conversions.Schema(), imports.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	return lib, s
}

func TestAGrabBehindAFileASearchRunABlocklistEntryAndAnImportRecordDeletes(t *testing.T) {
	lib, s := openRetention(t)
	run := storage.NewUUID()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return exec(ctx, tx,
			st(`INSERT INTO grabs_search (id, work_id, group_key, lane, trigger, started_at, finished_at, outcome, grabs)
				VALUES (?, ?, 'grp', 10, 'search', 0, 1, 'grabbed', 1)`, run, s.work),
			st(`INSERT INTO grabs_search_grab (search_id, grab_id, made) VALUES (?, ?, 1)`, run, s.grab),
			st(`INSERT INTO grabs_blocklist (info_hash, release_title, release_size, work_id, grab_id, cause, added_at)
				VALUES (randomblob(20), 'Orchard.WEB-DL', 1, ?, ?, 'marked_failed', 0)`, s.work, s.grab),
			st(`DELETE FROM grabs_grab WHERE id = ?`, s.grab),
		)
	})
	if err != nil {
		t.Fatalf("deleting the Grab = %v, want it deleted", err)
	}
	distinct(t, lib, `SELECT DISTINCT iif(grab_id IS NULL, 'none', 'set') FROM files_file`, "none")
	distinct(t, lib, `SELECT DISTINCT iif(grab_id IS NULL, 'none', 'set') FROM files_imported`, "none")
	distinct(t, lib, `SELECT DISTINCT iif(grab_id IS NULL, 'none', 'set') FROM grabs_blocklist`, "none")
	distinct(t, lib, `SELECT CAST(count(*) AS TEXT) FROM grabs_search`, "1")
	distinct(t, lib, `SELECT CAST(count(*) AS TEXT) FROM grabs_search_grab`, "0")
}

func TestEveryTableNamingAGrabFindsItsRowsByIndex(t *testing.T) {
	lib, _ := openRetention(t)
	rows, err := lib.QueryContext(t.Context(), `SELECT m.name, f."from" FROM sqlite_schema m, pragma_foreign_key_list(m.name) f
		WHERE m.type = 'table' AND f."table" = 'grabs_grab' ORDER BY m.name`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var children []string
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatal(err)
		}
		children = append(children, table+"."+column)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, c := range children {
		table, column, _ := strings.Cut(c, ".")
		var n int
		err := lib.QueryRowContext(t.Context(), `SELECT count(*) FROM pragma_index_list(?) l, pragma_index_info(l.name) i
			WHERE l.partial = 0 AND i.seqno = 0 AND i.name = ?`, table, column).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			t.Errorf("%s has no full index leading with its Grab, so deleting a Grab scans it", c)
		}
	}
	if len(children) < 9 {
		t.Fatalf("%d tables name a Grab, want at least the 9 of grabs, files and imports: %v", len(children), children)
	}
}
