package node

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// beforeS26 is the library's sets as 6468ad4 ran them, the release before
// S26 (Indexers and Trackers): indexers before grabs.
func beforeS26() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(), identity.Schema(), mail.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), metadata.Schema(),
		targets.Schema(), roots.Schema(), indexers.Schema().Through(2), downloadclients.Schema().Through(3), grabs.Schema().Through(6),
		files.Schema(), tasks.Schema(), health.Schema(), firstrun.Schema(),
	}
}

func TestS26DropsTheIndexersAndFailsGrabsStillFetching(t *testing.T) {
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
	old := open(beforeS26())
	node, indexer, work, item := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	fetching, downloading, torrent, search := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	err := old.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
			release_link, stage, torrent_id, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, ?, 'Orchard.1080p', 1, 'https://site.test/dl', ?, ?, 0, 0)`
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, []any{node}},
			{`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, []any{indexer, node}},
			{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
			{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, []any{item, work}},
			{`INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted, added_order, added_at)
				VALUES (?, ?, randomblob(20), x'00', 'Orchard.1080p', 1, 1, '[]', 'fixture', 'started', 1, 0)`, []any{torrent, node}},
			{grab, []any{fetching, work, item, indexer, node, "fetching", nil}},
			{grab, []any{downloading, work, item, indexer, node, "downloading", torrent}},
			{`INSERT INTO grabs_stage (grab_id, stage, entered_at) VALUES (?, 'fetching', 0)`, []any{fetching}},
			{`INSERT INTO grabs_search (id, work_id, item_id, indexer_id, started_at, finished_at, outcome, grab_id) VALUES (?, ?, ?, ?, 0, 1, 'grabbed', ?)`,
				[]any{search, work, item, indexer, downloading}},
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
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := open(LibrarySchemas(noTypes))
	t.Cleanup(func() { _ = lib.Close() })
	var indexersLeft, searches, violations int
	err = lib.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM indexers_indexer), (SELECT count(*) FROM grabs_search),
		(SELECT count(*) FROM pragma_foreign_key_check)`).Scan(&indexersLeft, &searches, &violations)
	if err != nil {
		t.Fatal(err)
	}
	if indexersLeft != 0 || searches != 0 || violations != 0 {
		t.Errorf("after S26 the library holds %d Indexers, %d search runs and %d foreign key violations, want none of each", indexersLeft, searches, violations)
	}
	for id, want := range map[storage.UUID]string{fetching: "failed fetching indexer_removed 2", downloading: "downloading   0"} {
		var stage, failedStage, code *string
		var stages int
		var ref *storage.UUID
		err := lib.QueryRowContext(ctx, `SELECT stage, failed_stage, failure_code, indexer_id, (SELECT count(*) FROM grabs_stage WHERE grab_id = g.id)
			FROM grabs_grab g WHERE id = ?`, id).Scan(&stage, &failedStage, &code, &ref, &stages)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%s %s %s %d", deref(stage), deref(failedStage), deref(code), stages); got != want || ref != nil {
			t.Errorf("Grab %s after S26 is %q naming Indexer %v, want %q naming none", id, got, ref, want)
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
