package grabs_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestThePacksMigrationGivesEveryGrabItsItemAndSettlesOnlyImportedCells
// migrates a Grab at every stage through grabs 00011, then writes one row
// per new rule: each good row lands and each bad one is refused.
func TestThePacksMigrationGivesEveryGrabItsItemAndSettlesOnlyImportedCells(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openSets(t, path, downloadclients.Schema(), grabs.Schema().Through(10), indexers.Schema(), files.Schema())
	node, client, torrent, work, item, target := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	rows := []tableRow{
		{"nodes_node", row{"id": node, "public_key": hash(32), "name": "home", "paired_at": 0}},
		{"downloadclients_client", row{
			"id": client, "node_id": node, "kind": "transmission", "name": "Transmission", "address": "http://192.168.1.20:9091/transmission/rpc",
			"address_key": "a", "label": "tv-sonarr", "is_on": 1, "settings_version": 1, "poll_state": "responding", "poll_since": 0,
			"poll_settings_version": 1, "created_at": 0, "updated_at": 0,
		}},
		{"downloadclients_torrent", row{
			"id": torrent, "node_id": node, "client_id": client, "info_hash": hash(20), "metainfo": []byte{0}, "name": "Heat.1995.1080p.BluRay",
			"size": 1, "private": 0, "trackers": "[]", "content_type": "fixture", "wanted": "started", "added_order": 1, "added_at": 0,
		}},
		workRow(work, "Heat"),
		{"works_item", row{"id": item, "work_id": work, "key": "i1", "label": "Heat", "position": 1}},
		{"targets_target", row{"id": target, "name": "1080p", "created_at": 0}},
	}
	byStage := map[grabs.Stage]storage.UUID{}
	for i, stage := range grabs.Stages {
		id := storage.NewUUID()
		byStage[stage] = id
		g := row{
			"id": id, "work_id": work, "item_id": item, "edition": "any", "source": "search", "node_id": node,
			"release_title": "Heat.1995.1080p.BluRay", "release_size": 1 << 30, "release_magnet": "magnet:?xt=urn:btih:00",
			"stage": string(stage), "created_at": i, "updated_at": 100 + i,
		}
		if stage != grabs.Fetching {
			g["torrent_id"] = torrent
		}
		if stage == grabs.Transferring || stage == grabs.Importing || stage == grabs.Imported {
			g["path"], g["file_index"], g["probe"] = "Heat (1995)/Heat (1995).mkv", 0, "{}"
		}
		if stage == grabs.Failed {
			g["failed_stage"], g["failure_code"], g["failure_message"] = "downloading", "download_failed", "The client stopped it."
		}
		rows = append(rows, tableRow{"grabs_grab", g}, tableRow{"grabs_target", row{"grab_id": id, "position": 0, "target_id": target}})
	}
	run(t, old, insertRows(rows...))
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path)
	t.Cleanup(func() { _ = lib.Close() })
	for _, stage := range grabs.Stages {
		want := "0 1 1 0 null"
		if stage == grabs.Imported {
			want = "0 1 1 0 placed 105"
		}
		var got string
		q := `SELECT i.position || ' ' || (i.item_id = ?) || ' ' || (t.item_id = i.item_id) || ' ' || t.forced || ' ' ||
			coalesce(t.settled || ' ' || t.settled_at, 'null')
			FROM grabs_grab_item i JOIN grabs_target t ON t.grab_id = i.grab_id WHERE i.grab_id = ?`
		if err := lib.QueryRowContext(ctx, q, item, byStage[stage]).Scan(&got); err != nil {
			t.Fatalf("a %s Grab: %v", stage, err)
		}
		if got != want {
			t.Errorf("after the migration a %s Grab's Item and cell read %q, want %q", stage, got, want)
		}
	}

	grab := byStage[grabs.Downloading]
	position := 0
	tables := map[string]func() row{
		"grabs_target": func() row {
			position++
			return row{"grab_id": grab, "position": position, "item_id": item, "target_id": storage.NewUUID(), "forced": 0}
		},
		"grabs_grab_item": func() row {
			position++
			return row{"grab_id": grab, "item_id": storage.NewUUID(), "position": position}
		},
		"grabs_blocklist": func() row {
			return row{"info_hash": hash(20), "release_title": "Heat.1995.1080p.BluRay", "release_size": 1 << 30, "cause": "no_video", "added_at": 0}
		},
	}
	for _, c := range []struct {
		name   string
		table  string
		change row
		ok     bool
	}{
		{"a cell held", "grabs_target", nil, true},
		{"a forced cell", "grabs_target", row{"forced": 1}, true},
		{"a cell placed", "grabs_target", row{"settled": "placed", "settled_at": 1}, true},
		{"a cell nothing filled", "grabs_target", row{"settled": "none", "settled_at": 1}, true},
		{"a cell declined", "grabs_target", row{"settled": "declined", "settled_at": 1}, true},
		{"a cell settled some other way", "grabs_target", row{"settled": "lost", "settled_at": 1}, false},
		{"a cell settled at no time", "grabs_target", row{"settled": "placed"}, false},
		{"a cell unsettled with a time", "grabs_target", row{"settled_at": 1}, false},
		{"a cell forced twice over", "grabs_target", row{"forced": 2}, false},
		{"an Item at a negative position", "grabs_grab_item", row{"position": -1}, false},
		{"a download with no video", "grabs_blocklist", nil, true},
		{"an unreadable download", "grabs_blocklist", row{"cause": "unreadable"}, true},
		{"a misdescribed download", "grabs_blocklist", row{"cause": "misdescribed"}, true},
		{"a download with no video and an Admin", "grabs_blocklist", row{"added_by_name": "Jackson"}, false},
		{"an unreadable download with an Admin", "grabs_blocklist", row{"cause": "unreadable", "added_by_name": "Jackson"}, false},
		{"a misdescribed download with an Admin", "grabs_blocklist", row{"cause": "misdescribed", "added_by_name": "Jackson"}, false},
	} {
		r := with(tables[c.table](), c.change)
		var before []tableRow
		switch c.table {
		case "grabs_target":
			before = []tableRow{{"targets_target", row{"id": r["target_id"], "name": fmt.Sprint("T", position), "created_at": 0}}}
		case "grabs_grab_item":
			before = []tableRow{{"works_item", row{"id": r["item_id"], "work_id": work, "key": fmt.Sprint("i", position+1), "label": "Heat", "position": position + 1}}}
		}
		err := lib.Write(ctx, insertRows(append(before, tableRow{c.table, r})...))
		if c.ok && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
		if !c.ok && !refusedByCheck(err) {
			t.Errorf("%s: want a CHECK to refuse it, got %v", c.name, err)
		}
	}
	for _, step := range []string{"chosen_by_another", "fewer_unwanted", "lost_by_luck"} {
		search := storage.NewUUID()
		err := lib.Write(ctx, insertRows(
			tableRow{"grabs_search", row{"id": search, "work_id": work, "item_id": item, "started_at": 0, "finished_at": 1, "outcome": "nothing_acceptable"}},
			tableRow{"grabs_release", row{
				"search_id": search, "position": 0, "title": "Heat.1995.S01.1080p.BluRay", "indexer_name": "TorrentLeech", "size": 1 << 30,
				"seeders": 9, "outcome": "outranked", "rank": 2, "reasons": "[]", "scores": "[]", "formats": "[]", "covers": "[]",
				"lost_step": step, "lost_own": "1", "lost_ref": "0",
			}},
		))
		if ok := step != "lost_by_luck"; ok != (err == nil) || (!ok && !refusedByCheck(err)) {
			t.Errorf("a Release lost at %s: got %v", step, err)
		}
	}
}
