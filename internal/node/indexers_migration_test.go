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

// beforeS27 is the library's sets as S26 (Indexers and Trackers) left
// them, before the native kind.
func beforeS27() []storage.Schema[storage.Library] {
	schemas := LibrarySchemas(noTypes)
	for i, s := range schemas {
		if s.Set() == "indexers" {
			schemas[i] = s.Through(3)
		}
	}
	return schemas
}

func TestS27KeepsEveryIndexerAndWhatNamesIt(t *testing.T) {
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
	old := open(beforeS27())
	node, catalog, torznab, rss := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	tracker, work, item, grab, query := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	err := old.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		const indexer = `INSERT INTO indexers_indexer (id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority, media,
			search_limit, search_window, grab_limit, grab_window, definition_id, definition_sha, base_address, plain, secrets_sealed,
			feed_host, allow_no_size, caps, notes, revision, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, 1, 1, 0, 25, ?, 0, 'day', 0, 'day', ?, ?, ?, '{}', x'00', ?, 0, '{}', '[]', 1, 0, 0)`
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, []any{node}},
			{`INSERT INTO indexers_definition (id, sha, file, origin, yaml, refusal, fetched_at) VALUES ('orchard', 'abc', 'orchard.yml', 'catalog', x'00', 'old', 0)`, nil},
			{indexer, []any{catalog, "Orchard", "catalog", node, nil, "orchard", "abc", "https://orchard.test/", nil}},
			{indexer, []any{torznab, "Feed", "torznab", node, nil, nil, nil, nil, "feed.test"}},
			{indexer, []any{rss, "Reader", "rss", node, `["fixture"]`, nil, nil, nil, "rss.test"}},
			{`INSERT INTO indexers_status (indexer_id, state, rung, since_at, cause, solver_tried, revision) VALUES (?, 'stopped', 0, 0, 'captcha_needed', 0, 1)`,
				[]any{catalog}},
			{`INSERT INTO indexers_session (indexer_id, node_id, cookies_sealed, user_agent, pasted, earned_at) VALUES (?, ?, x'00', 'ua', 0, 0)`,
				[]any{catalog, node}},
			{`INSERT INTO indexers_query (id, indexer_id, use, reserved_at, revision) VALUES (?, ?, 'search', 0, 1)`, []any{query, torznab}},
			{`INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, 'Orchard', 'private', 0)`, []any{tracker}},
			{`INSERT INTO indexers_reach (indexer_id, tracker_id) VALUES (?, ?)`, []any{catalog, tracker}},
			{`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, []any{work}},
			{`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, []any{item, work}},
			{`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, tracker_id, node_id, release_title, release_size,
				stage, failed_stage, failure_code, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, ?, ?, 'Orchard.1080p', 1, 'failed', 'fetching', 'fetch_failed', 0, 0)`,
				[]any{grab, work, item, catalog, tracker, node}},
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
	var kinds, cause string
	var statuses, sessions, queries, reaches, violations int
	var grabIndexer *storage.UUID
	err = lib.QueryRowContext(ctx, `SELECT (SELECT group_concat(kind, ',') FROM (SELECT kind FROM indexers_indexer ORDER BY kind)),
		(SELECT count(*) FROM indexers_status), (SELECT cause FROM indexers_status), (SELECT count(*) FROM indexers_session WHERE token_sealed IS NULL),
		(SELECT count(*) FROM indexers_query), (SELECT count(*) FROM indexers_reach), (SELECT indexer_id FROM grabs_grab),
		(SELECT count(*) FROM pragma_foreign_key_check)`).Scan(&kinds, &statuses, &cause, &sessions, &queries, &reaches, &grabIndexer, &violations)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%s %d %s %d %d %d %d", kinds, statuses, cause, sessions, queries, reaches, violations); got != "catalog,rss,torznab 1 captcha_needed 1 1 1 0" {
		t.Errorf("after S27 the library holds %q, want every S26 Indexer with its status, session, query and reach, and no foreign key violation", got)
	}
	if grabIndexer == nil || *grabIndexer != catalog {
		t.Errorf("after S27 the Grab names Indexer %v, want %v", grabIndexer, catalog)
	}
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO indexers_indexer (id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority,
		search_limit, search_window, grab_limit, grab_window, port_id, base_address, plain, secrets_sealed, allow_no_size, caps, notes, revision,
		created_at, updated_at) VALUES (?, 'Ported', 'native', ?, 1, 1, 1, 0, 25, 0, 'day', 0, 'day', 'norbits', 'https://norbits.test/', '{}', x'00',
		0, '{}', '[]', 1, 0, 0)`, storage.NewUUID(), node)
		return err
	})
	if err != nil {
		t.Errorf("a native Indexer with a port and an address doesn't fit S27's table: %v", err)
	}
}
