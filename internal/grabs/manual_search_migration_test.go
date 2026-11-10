package grabs_test

import (
	"context"
	"crypto/rand"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

type row = map[string]any

func insertRow(ctx context.Context, tx *storage.Tx[storage.Library], table string, r row) error {
	cols := slices.Sorted(maps.Keys(r))
	args := make([]any, len(cols))
	for i, c := range cols {
		args[i] = r[c]
	}
	marks := strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ")
	_, err := tx.ExecContext(ctx, "INSERT INTO "+table+" ("+strings.Join(cols, ", ")+") VALUES ("+marks+")", args...)
	return err
}

type tableRow struct {
	table string
	r     row
}

func insertRows(rows ...tableRow) func(context.Context, *storage.Tx[storage.Library]) error {
	return func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, tr := range rows {
			if err := insertRow(ctx, tx, tr.table, tr.r); err != nil {
				return err
			}
		}
		return nil
	}
}

func workRow(id storage.UUID, label string) tableRow {
	return tableRow{"works_work", row{"id": id, "content_type": "fixture", "key": id.String(), "label": label, "sort_label": label, "added_at": 0, "refreshed_at": 0}}
}

func refusedByCheck(err error) bool {
	return err != nil && strings.Contains(err.Error(), "CHECK constraint failed")
}

func with(base, change row) row {
	out := maps.Clone(base)
	maps.Copy(out, change)
	return out
}

func hash(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// TestManualSearchMigrationKeepsRowsAndEachCheckRefusesItsRow migrates a
// library holding a finished automatic run, its line, its Release and its
// Grab through S32 (Manual search and grab)'s rebuilds, then writes one
// row per new rule: each good row lands and each bad one is refused.
func TestManualSearchMigrationKeepsRowsAndEachCheckRefusesItsRow(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openSets(t, path, downloadclients.Schema(), grabs.Schema().Through(9), indexers.Schema(), files.Schema())
	node, work, item, grab, auto := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	run(t, old, insertRows(
		tableRow{"nodes_node", row{"id": node, "public_key": hash(32), "name": "home", "paired_at": 0}},
		workRow(work, "Heat"),
		tableRow{"works_item", row{"id": item, "work_id": work, "key": "i1", "label": "Heat", "position": 1}},
		tableRow{"grabs_grab", row{
			"id": grab, "work_id": work, "item_id": item, "edition": "any", "source": "search", "node_id": node,
			"release_title": "Heat.1995.1080p.BluRay", "release_size": 1 << 30, "release_magnet": "magnet:?xt=urn:btih:00",
			"stage": "fetching", "created_at": 0, "updated_at": 0,
		}},
		tableRow{"grabs_search", row{"id": auto, "work_id": work, "item_id": item, "started_at": 0, "finished_at": 1, "outcome": "grabbed", "grab_id": grab}},
		tableRow{"grabs_search_indexer", row{"search_id": auto, "position": 0, "indexer_name": "TorrentLeech", "outcome": "found", "releases": 1}},
		tableRow{"grabs_release", row{
			"search_id": auto, "position": 0, "title": "Heat.1995.1080p.BluRay", "indexer_name": "TorrentLeech", "size": 1 << 30,
			"seeders": 9, "outcome": "grabbed", "rank": 1, "reasons": "[]", "scores": "[]", "formats": "[]", "covers": "[]",
		}},
	))
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path)
	t.Cleanup(func() { _ = lib.Close() })
	for _, c := range []struct {
		query string
		want  string
	}{
		{"SELECT source || ' ' || overrides || ' ' || (manual_id IS NULL) || ' ' || stage FROM grabs_grab", "search {} 1 fetching"},
		{"SELECT outcome || ' ' || (manual_id IS NULL) || ' ' || (grab_id IS NOT NULL) FROM grabs_search", "grabbed 1 1"},
		{"SELECT indexer_name || ' ' || outcome || ' ' || releases FROM grabs_search_indexer", "TorrentLeech found 1"},
		{"SELECT title || ' ' || outcome || ' ' || rank || ' ' || (read IS NULL) FROM grabs_release", "Heat.1995.1080p.BluRay grabbed 1 1"},
	} {
		var got string
		if err := lib.QueryRowContext(ctx, c.query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", c.query, err)
		}
		if got != c.want {
			t.Errorf("after the migration %s reads %q, want %q", c.query, got, c.want)
		}
	}

	manual, listed := storage.NewUUID(), storage.NewUUID()
	run(t, lib, insertRows(
		tableRow{"grabs_manual", row{"id": manual, "work_id": work, "scope": "{}", "started_at": 0}},
		tableRow{"grabs_search", row{"id": listed, "work_id": work, "item_id": item, "manual_id": manual, "started_at": 0}},
	))

	position, another := 0, storage.UUID{}
	tables := map[string]func() row{
		"grabs_manual": func() row {
			another = storage.NewUUID()
			return row{"id": storage.NewUUID(), "work_id": another, "scope": "{}", "started_at": 0, "query": "heat"}
		},
		"grabs_search": func() row {
			return row{"id": storage.NewUUID(), "work_id": work, "item_id": item, "started_at": 0, "finished_at": 1, "outcome": "listed", "manual_id": manual}
		},
		"grabs_search_indexer": func() row {
			position++
			return row{"search_id": listed, "position": position, "indexer_name": "TorrentLeech", "outcome": "rss_only"}
		},
		"grabs_release": func() row {
			position++
			return row{
				"search_id": listed, "position": position, "title": "Heat.1995.2160p.WEB-DL", "indexer_name": "TorrentLeech",
				"size": 1 << 30, "seeders": 4, "outcome": "listed", "reasons": "[]", "scores": "[]", "formats": "[]", "covers": "[]",
				"release": "{}", "link_sealed": hash(48), "read": "{}",
			}
		},
		"grabs_grab": func() row {
			return row{
				"id": storage.NewUUID(), "work_id": work, "item_id": item, "edition": "any", "source": "manual", "manual_id": manual,
				"grabbed_by_name": "Jackson", "overrides": `{"edition":true}`, "node_id": node, "release_title": "Heat.1995.2160p.WEB-DL",
				"release_size": 1 << 30, "release_magnet": "magnet:?xt=urn:btih:01", "stage": "fetching", "created_at": 0, "updated_at": 0,
			}
		},
		"grabs_blocklist": func() row {
			return row{
				"info_hash": hash(20), "release_title": "Heat.1995.2160p.WEB-DL", "release_size": 1 << 30, "work_id": work,
				"cause": "by_hand", "added_by_name": "Jackson", "added_at": 0,
			}
		},
	}
	for _, c := range []struct {
		name   string
		table  string
		change row
		ok     bool
	}{
		{"a Manual search with a typed query", "grabs_manual", nil, true},
		{"a Manual search with no query", "grabs_manual", row{"query": nil}, true},
		{"an empty query", "grabs_manual", row{"query": ""}, false},
		{"a query over 200 characters", "grabs_manual", row{"query": strings.Repeat("h", 201)}, false},
		{"a scope that isn't JSON", "grabs_manual", row{"scope": "season 2"}, false},
		{"a finished Manual run", "grabs_search", nil, true},
		{"a Manual run that grabbed", "grabs_search", row{"outcome": "grabbed", "grab_id": grab}, false},
		{"a Manual run with nothing acceptable", "grabs_search", row{"outcome": "nothing_acceptable"}, false},
		{"an automatic run listed", "grabs_search", row{"manual_id": nil}, false},
		{"an RSS feed skipped", "grabs_search_indexer", nil, true},
		{"a listed Release with its reading", "grabs_release", nil, true},
		{"a listed Release with a rank", "grabs_release", row{"rank": 1}, false},
		{"a listed Release that is a copy", "grabs_release", row{"copy_of": 0}, false},
		{"a listed Release with no reading", "grabs_release", row{"read": nil}, false},
		{"a listed Release with no sealed link", "grabs_release", row{"link_sealed": nil}, false},
		{"a refused Release with a reading", "grabs_release", row{"outcome": "refused"}, false},
		{"a Manual Grab with its record", "grabs_grab", nil, true},
		{"a search Grab with an Admin", "grabs_grab", row{"source": "search", "manual_id": nil, "overrides": "{}"}, false},
		{"a search Grab with overrides", "grabs_grab", row{"source": "search", "manual_id": nil, "grabbed_by_name": nil}, false},
		{"a search Grab of a Manual search", "grabs_grab", row{"source": "search", "grabbed_by_name": nil, "overrides": "{}"}, false},
		{"a Grab from another source", "grabs_grab", row{"source": "rss"}, false},
		{"an entry blocklisted by hand", "grabs_blocklist", nil, true},
		{"an engine error naming its client", "grabs_blocklist", row{"cause": "engine_error", "client_name": "qBittorrent on Den", "added_by_name": nil}, true},
		{"a 19-byte info hash", "grabs_blocklist", row{"info_hash": hash(19)}, false},
		{"a negative size", "grabs_blocklist", row{"release_size": -1}, false},
		{"an unknown cause", "grabs_blocklist", row{"cause": "stalled"}, false},
		{"a release fault with an Admin", "grabs_blocklist", row{"cause": "unregistered"}, false},
		{"an engine error naming no client", "grabs_blocklist", row{"cause": "engine_error", "added_by_name": nil}, false},
	} {
		rows := []tableRow{{c.table, with(tables[c.table](), c.change)}}
		if c.table == "grabs_manual" {
			rows = append([]tableRow{workRow(another, "Ronin")}, rows...)
		}
		err := lib.Write(ctx, insertRows(rows...))
		if c.ok && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
		if !c.ok && !refusedByCheck(err) {
			t.Errorf("%s: want a CHECK to refuse it, got %v", c.name, err)
		}
	}
}
