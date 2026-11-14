package indexers_test

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheSearchWaitKeepsToTenSecondsThroughTenMinutes(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema())
	for _, c := range []struct {
		seconds any
		ok      bool
	}{{nil, true}, {10, true}, {600, true}, {9, false}, {601, false}} {
		err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "UPDATE indexers_settings SET search_wait_seconds = ?", c.seconds)
			return err
		})
		switch {
		case c.ok && err != nil:
			t.Errorf("a search wait of %v seconds: refused: %v", c.seconds, err)
		case !c.ok && (err == nil || !strings.Contains(err.Error(), "CHECK constraint failed")):
			t.Errorf("a search wait of %v seconds: want a CHECK to refuse it, got %v", c.seconds, err)
		}
	}
}

// openRSS opens the library at path with indexers migrated by idx.
func openRSS(t *testing.T, path string, idx storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	t.Helper()
	schemas := []storage.Schema[storage.Library]{secret.Schema(), nodes.Schema(), works.Schema(), idx}
	lib, err := storage.OpenLibrary(t.Context(), path, schemas, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	return lib
}

func exec(t *testing.T, lib *storage.DB[storage.Library], query string, args ...any) error {
	t.Helper()
	return lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
}

func refused(err error) bool {
	return err != nil && strings.Contains(err.Error(), "CHECK constraint failed")
}

const indexerRow = `INSERT INTO indexers_indexer (id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority,
	search_limit, search_window, grab_limit, grab_window, plain, secrets_sealed, feed_host, allow_no_size, caps, notes, revision,
	created_at, updated_at) VALUES (?, ?, 'torznab', ?, 1, 1, 1, 0, 25, 0, 'hour', 0, 'hour', '{}', x'00', 'site.test', 0, '{}', '[]', 1, 0, 0)`

func TestTheRSSSyncMigrationClaimsEveryEarlierSearchForAPress(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openRSS(t, path, indexers.Schema().Through(5))
	node, indexer := storage.NewUUID(), storage.NewUUID()
	for _, q := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'home', 0)`, []any{node, make([]byte, 32)}},
		{indexerRow, []any{indexer, "Site", node}},
		{`INSERT INTO indexers_query (id, indexer_id, use, reserved_at, revision)
			VALUES (x'01', ?, 'search', 1, 1), (x'02', ?, 'recent', 2, 1), (x'03', ?, 'fetch', 3, 1), (x'04', ?, 'test', 4, 1)`,
			[]any{indexer, indexer, indexer, indexer}},
	} {
		if err := exec(t, old, q.query, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openRSS(t, path, indexers.Schema())
	var got string
	err := lib.QueryRowContext(t.Context(), `SELECT group_concat(use || '=' || coalesce(claim, 'none'), ' ') || ' ' ||
		(SELECT searches_at_once FROM indexers_indexer) || ' ' || (SELECT count(*) FROM pragma_foreign_key_check)
		FROM (SELECT use, claim FROM indexers_query ORDER BY reserved_at)`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if want := "search=press recent=none fetch=none test=none 2 0"; got != want {
		t.Errorf("after the migration the queries, Searches at once and foreign key violations read %q, want %q", got, want)
	}

	for _, c := range []struct {
		name, use string
		claim     any
		ok        bool
	}{
		{"a background search", "search", "background", true},
		{"a search with no claim", "search", nil, false},
		{"a fetch with a claim", "fetch", "press", false},
		{"an unknown claim", "search", "now", false},
	} {
		err := exec(t, lib, `INSERT INTO indexers_query (id, indexer_id, use, claim, reserved_at, revision) VALUES (?, ?, ?, ?, 5, 1)`,
			storage.NewUUID(), indexer, c.use, c.claim)
		if c.ok != (err == nil) || !c.ok && !refused(err) {
			t.Errorf("%s: written with %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestSearchesAtOnceKeepsToOneThroughTen(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema())
	node, indexer := storage.NewUUID(), storage.NewUUID()
	if err := exec(t, lib, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'home', 0)`, node, make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	if err := exec(t, lib, indexerRow, indexer, "Site", node); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		n  int
		ok bool
	}{{1, true}, {10, true}, {0, false}, {11, false}} {
		err := exec(t, lib, `UPDATE indexers_indexer SET searches_at_once = ?`, c.n)
		if c.ok != (err == nil) || !c.ok && !refused(err) {
			t.Errorf("Searches at once of %d: written with %v, want ok=%v", c.n, err, c.ok)
		}
	}
}

func TestAFeedPositionKeepsAtMostAHundredGUIDsAndOnePageAtLeast(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema())
	node, indexer := storage.NewUUID(), storage.NewUUID()
	if err := exec(t, lib, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'home', 0)`, node, make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	if err := exec(t, lib, indexerRow, indexer, "Site", node); err != nil {
		t.Fatal(err)
	}
	guids := func(n int) string {
		g := make([]string, n)
		for i := range g {
			g[i] = fmt.Sprintf("%q", fmt.Sprint("guid-", i))
		}
		return "[" + strings.Join(g, ",") + "]"
	}
	for _, c := range []struct {
		name    string
		guids   string
		pages   int
		gapAt   any
		gapKind any
		ok      bool
	}{
		{"a hundred GUIDs over one page", guids(100), 1, nil, nil, true},
		{"a gap with its kind", "[]", 10, 7, "page_cap", true},
		{"a hundred and one GUIDs", guids(101), 1, nil, nil, false},
		{"an object for GUIDs", `{"a":1}`, 1, nil, nil, false},
		{"no page", "[]", 0, nil, nil, false},
		{"a gap with no kind", "[]", 1, 7, nil, false},
		{"an unknown gap kind", "[]", 1, 7, "slow", false},
	} {
		err := exec(t, lib, `INSERT INTO indexers_rss (indexer_id, last_guids, last_fetch_at, last_pages, gap_at, gap_kind)
			VALUES (?, ?, 0, ?, ?, ?) ON CONFLICT (indexer_id) DO UPDATE SET last_guids = excluded.last_guids,
			last_pages = excluded.last_pages, gap_at = excluded.gap_at, gap_kind = excluded.gap_kind`,
			indexer, c.guids, c.pages, c.gapAt, c.gapKind)
		if c.ok != (err == nil) || !c.ok && !refused(err) {
			t.Errorf("%s: written with %v, want ok=%v", c.name, err, c.ok)
		}
	}
}
