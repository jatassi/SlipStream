package grabs_test

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func at(seconds int) storage.Time { return storage.FromTime(time.Unix(int64(seconds), 0)) }

func keyOf(prefix string, id storage.UUID) string { return prefix + hex.EncodeToString(id.Bytes()) }

func autoRun(id, work, item storage.UUID, started, finished int, outcome string, candidates int, grab any) tableRow {
	r := row{"id": id, "work_id": work, "item_id": item, "started_at": at(started), "candidates": candidates, "grab_id": grab}
	if finished > 0 {
		r["finished_at"], r["outcome"] = at(finished), outcome
	}
	return tableRow{"grabs_search", r}
}

func fetchingGrab(id, work, item, node storage.UUID) tableRow {
	return tableRow{"grabs_grab", row{
		"id": id, "work_id": work, "item_id": item, "edition": "any", "source": "search", "node_id": node,
		"release_title": "Dune.S01E01.1080p.WEB", "release_size": 1 << 30, "release_magnet": "magnet:?xt=urn:btih:00",
		"stage": "fetching", "created_at": 0, "updated_at": 0,
	}}
}

type expectedRun struct {
	name string
	run  storage.UUID
	want string
}

// expectRuns checks that each run reads its want through query.
func expectRuns(t *testing.T, lib library, query string, runs []expectedRun) {
	t.Helper()
	for _, c := range runs {
		if got := scalar(t, lib, query, c.run); got != c.want {
			t.Errorf("%s reads %q, want %q", c.name, got, c.want)
		}
	}
}

func scalar(t *testing.T, lib library, query string, args ...any) string {
	t.Helper()
	var got string
	if err := lib.QueryRowContext(t.Context(), query, args...).Scan(&got); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return got
}

// TestTheSearchQueueMigrationMakesGroupRunsAndKeepsEachItemsLastResult
// migrates per-Item runs of a two-Item level and of an Item with no
// level through grabs 00014.
func TestTheSearchQueueMigrationMakesGroupRunsAndKeepsEachItemsLastResult(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openSets(t, path, downloadclients.Schema(), grabs.Schema().Through(13), indexers.Schema(), files.Schema())
	node, work, level, first, second, lone := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	fetching, joined := storage.NewUUID(), storage.NewUUID()
	older, grabbed, other, made, joining, open, manual, listed := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(),
		storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	sealer := secrettest.Sealer(t)
	keptAD := secret.NewColumn("grabs_release.link_sealed").Row(grabbed.String() + "/1")
	const link = "https://site.test/dl/2"
	release := func(search storage.UUID, position int, outcome string, rank int, kept bool) tableRow {
		r := row{
			"search_id": search, "position": position, "title": fmt.Sprint("Dune.S01E01.1080p.WEB-", position), "indexer_name": "TorrentLeech",
			"size": 1 << 30, "seeders": 9, "outcome": outcome, "rank": rank, "reasons": "[]", "scores": "[]", "formats": "[]", "covers": "[]",
		}
		if kept {
			r["release"], r["link_sealed"] = "{}", sealer.Seal([]byte(link), keptAD)
		}
		return tableRow{"grabs_release", r}
	}
	run(t, old, insertRows(
		tableRow{"nodes_node", row{"id": node, "public_key": hash(32), "name": "home", "paired_at": 0}},
		workRow(work, "Dune"),
		tableRow{"works_level", row{"id": level, "work_id": work, "depth": 1, "key": "l1", "label": "Season 1", "position": 1}},
		tableRow{"works_item", row{"id": first, "work_id": work, "level_id": level, "key": "i1", "label": "One", "position": 1}},
		tableRow{"works_item", row{"id": second, "work_id": work, "level_id": level, "key": "i2", "label": "Two", "position": 2}},
		tableRow{"works_item", row{"id": lone, "work_id": work, "key": "i3", "label": "Three", "position": 3}},
		fetchingGrab(fetching, work, first, node),
		fetchingGrab(joined, work, lone, node),
		autoRun(older, work, first, 1, 2, "nothing_acceptable", 3, nil),
		release(older, 0, "kept", 1, false),
		tableRow{"grabs_search_indexer", row{"search_id": older, "position": 0, "indexer_name": "TorrentLeech", "query_id": storage.NewUUID(), "outcome": "found", "releases": 3}},
		autoRun(grabbed, work, first, 10, 11, "grabbed", 2, fetching),
		release(grabbed, 0, "grabbed", 1, false),
		release(grabbed, 1, "not_taken", 2, true),
		autoRun(other, work, second, 3, 4, "nothing_acceptable", 0, nil),
		autoRun(made, work, lone, 20, 21, "grabbed", 1, joined),
		autoRun(joining, work, lone, 30, 31, "grabbed", 1, joined),
		autoRun(open, work, second, 40, 0, "", 0, nil),
		tableRow{"grabs_search_indexer", row{"search_id": open, "position": 0, "indexer_name": "TorrentLeech", "query_id": storage.NewUUID()}},
		tableRow{"grabs_manual", row{"id": manual, "work_id": work, "scope": "{}", "started_at": 0}},
		tableRow{"grabs_search", row{"id": listed, "work_id": work, "item_id": first, "manual_id": manual, "started_at": at(5), "finished_at": at(6), "outcome": "listed"}},
	))
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path)
	t.Cleanup(func() { _ = lib.Close() })
	levelKey, loneKey := keyOf("l:", level), keyOf("i:", lone)
	if got, want := string(indexers.LevelGroup(works.LevelID{UUID: level})), levelKey; got != want {
		t.Fatalf("LevelGroup = %q, want the key the migration builds, %q", got, want)
	}
	if got, want := string(indexers.ItemGroup(works.ItemID{UUID: lone})), loneKey; got != want {
		t.Fatalf("ItemGroup = %q, want the key the migration builds, %q", got, want)
	}
	runRow := `SELECT coalesce(group_key, '-') || ' ' || coalesce(lane, '-') || ' ' || coalesce(trigger, '-') || ' ' || include_gone || ' ' ||
		coalesce(outcome, '-') || ' ' || grabs || ' ' ||
		(SELECT group_concat(hex(item_id)) FROM grabs_search_item WHERE search_id = s.id) FROM grabs_search s WHERE id = ?`
	hexOf := func(id storage.UUID) string { return strings.ToUpper(hex.EncodeToString(id.Bytes())) }
	expectRuns(t, lib, runRow, []expectedRun{
		{"the grabbed press of the first Item", grabbed, levelKey + " 10 search 1 grabbed 1 " + hexOf(first)},
		{"the second Item's press with no candidates", other, levelKey + " 10 search 1 nothing_found 0 " + hexOf(second)},
		{"the lone Item's press that made the Grab", made, loneKey + " 10 search 1 grabbed 1 " + hexOf(lone)},
		{"the lone Item's press that joined it", joining, loneKey + " 10 search 1 grabbed 1 " + hexOf(lone)},
		{"the open press", open, levelKey + " 10 search 1 - 0 " + hexOf(second)},
		{"the Manual run", listed, "- - - 0 listed 0 " + hexOf(first)},
	})
	if got := scalar(t, lib, `SELECT count(*) || ' ' || (SELECT count(*) FROM grabs_release WHERE search_id = ?) || ' ' ||
		(SELECT count(*) FROM grabs_search_indexer WHERE search_id = ?) FROM grabs_search WHERE id = ?`, older, older, older); got != "0 0 0" {
		t.Errorf("the first Item's older press, its Releases and its lines number %q, want all gone behind its later press", got)
	}
	joins := `SELECT group_concat(hex(search_id) || '=' || made, ' ') FROM (SELECT * FROM grabs_search_grab WHERE grab_id = ? ORDER BY made DESC)`
	if got, want := scalar(t, lib, joins, fetching), hexOf(grabbed)+"=1"; got != want {
		t.Errorf("the fetching Grab's runs read %q, want %q", got, want)
	}
	if got, want := scalar(t, lib, joins, joined), hexOf(made)+"=1 "+hexOf(joining)+"=0"; got != want {
		t.Errorf("the joined Grab's runs read %q, want the earlier run making it and the later joining it, %q", got, want)
	}
	var sealed secret.Sealed
	if err := lib.QueryRowContext(ctx, `SELECT link_sealed FROM grabs_release WHERE search_id = ? AND position = 1`, grabbed).Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	if raw, err := sealer.Open(sealed, keptAD); err != nil || string(raw) != link {
		t.Errorf("the grabbed press's kept Release opens to %q (%v), want %q under its run and position", raw, err, link)
	}
	var dispatched, deadline storage.Time
	if err := lib.QueryRowContext(ctx, `SELECT dispatched_at, deadline_at FROM grabs_search_indexer WHERE search_id = ?`, open).Scan(&dispatched, &deadline); err != nil {
		t.Fatal(err)
	}
	if want := at(40); dispatched.Compare(want) != 0 || deadline.Compare(storage.FromTime(want.Time().Add(2*time.Minute))) != 0 {
		t.Errorf("the open press's line was dispatched at %s with its deadline at %s, want at its start %s and two minutes after", dispatched, deadline, want)
	}
	firsts := `SELECT group_concat(hex(item_id) || '=' || first_searched_at, ' ') FROM (SELECT * FROM grabs_item_search ORDER BY first_searched_at)`
	want := fmt.Sprintf("%s=%d %s=%d %s=%d", hexOf(first), at(1).Time().UnixMicro(), hexOf(second), at(3).Time().UnixMicro(), hexOf(lone), at(20).Time().UnixMicro())
	if got := scalar(t, lib, firsts); got != want {
		t.Errorf("grabs_item_search reads %q, want each Item's first press %q", got, want)
	}
	if got := scalar(t, lib, `SELECT count(*) FROM pragma_foreign_key_check`); got != "0" {
		t.Errorf("after the upgrade %s rows break a foreign key, want none", got)
	}

	run(t, lib, sql(`DELETE FROM grabs_grab WHERE id = ?`, fetching))
	if got, want := scalar(t, lib, runRow, grabbed), levelKey+" 10 search 1 grabbed 1 "+hexOf(first); got != want {
		t.Errorf("with its Grab deleted the grabbed press reads %q, want %q", got, want)
	}
	if got := scalar(t, lib, `SELECT count(*) FROM grabs_search_grab WHERE search_id = ?`, grabbed); got != "0" {
		t.Errorf("with its Grab deleted the grabbed press keeps %s join rows, want none", got)
	}

	eachQueueRuleRefusesItsRow(t, lib, work, first, joined)
}

// eachQueueRuleRefusesItsRow writes one row per rule grabs 00014 adds to
// lib: each good row lands and each bad one is refused.
func eachQueueRuleRefusesItsRow(t *testing.T, lib library, work, item, grab storage.UUID) {
	ctx := t.Context()
	target := storage.NewUUID()
	run(t, lib, insertRows(tableRow{"targets_target", row{"id": target, "name": "HD", "created_at": 0}}))
	waitOf := func() row {
		return row{"work_id": work, "item_id": item, "edition": "any", "target_id": target, "until_at": at(20), "started_at": at(10)}
	}
	queueRun := func() row {
		return row{"id": storage.NewUUID(), "work_id": work, "group_key": "n:aired-1", "lane": 40, "trigger": "backlog", "started_at": 0}
	}
	position := 100
	tables := map[string]func() row{
		"grabs_search": queueRun,
		"grabs_search_grab": func() row {
			search := storage.NewUUID()
			run(t, lib, insertRows(tableRow{"grabs_search", with(queueRun(), row{"id": search})}))
			return row{"search_id": search, "grab_id": grab, "made": 0}
		},
		"grabs_search_indexer": func() row {
			search := storage.NewUUID()
			run(t, lib, insertRows(tableRow{"grabs_search", with(queueRun(), row{"id": search})}))
			return row{"search_id": search, "position": 0, "indexer_name": "TorrentLeech"}
		},
		"grabs_release": func() row {
			search := storage.NewUUID()
			run(t, lib, insertRows(tableRow{"grabs_search", with(queueRun(), row{"id": search})}))
			return row{
				"search_id": search, "position": 0, "title": "Dune.S01.1080p.WEB", "indexer_name": "TorrentLeech", "size": 1 << 30, "seeders": 9,
				"outcome": "heard", "reasons": "[]", "scores": "[]", "formats": "[]", "covers": "[]", "release": "{}", "link_sealed": hash(48),
				"heard_at": at(1),
			}
		},
		"grabs_grab": func() row {
			position++
			r := fetchingGrab(storage.NewUUID(), work, item, storage.NewUUID()).r
			run(t, lib, insertRows(tableRow{"nodes_node", row{"id": r["node_id"], "public_key": hash(32), "name": fmt.Sprint("n", position), "paired_at": 0}}))
			return with(r, row{"source": "rss", "release_guid": "https://site.test/t/1", "chosen_fills": 2})
		},
		"grabs_nudge": func() row {
			return row{"work_id": work, "trigger": "retarget", "created_at": 0}
		},
		"grabs_due": func() row {
			position++
			return row{"work_id": work, "group_key": fmt.Sprint("n:aired-", position), "due_at": 0, "gap_seconds": 3600, "changed_at": 0}
		},
		"grabs_failure": func() row {
			return row{
				"grab_id": storage.NewUUID(), "work_id": work, "item_id": item, "edition": "any", "target_id": target, "code": "torrent_invalid",
				"failed_at": 0, "counted": 1, "refuses": 0, "stopped_at": at(1),
			}
		},
		"grabs_wait":     waitOf,
		"grabs_settings": func() row { return row{"id": 1} },
	}
	for _, c := range []struct {
		name   string
		table  string
		change row
		ok     bool
	}{
		{"a Backlog run of a named group", "grabs_search", nil, true},
		{"a press in the now lane", "grabs_search", row{"lane": 10, "trigger": "search", "include_gone": 1}, true},
		{"an RSS run", "grabs_search", row{"group_key": nil, "lane": nil, "trigger": "rss", "finished_at": at(1), "outcome": "grabbed", "grabs": 1}, true},
		{"a queue run with no lane", "grabs_search", row{"lane": nil}, false},
		{"a queue run in an unknown lane", "grabs_search", row{"lane": 20}, false},
		{"a run of a gap reset", "grabs_search", row{"trigger": "retarget"}, false},
		{"a key under three characters", "grabs_search", row{"group_key": "n:"}, false},
		{"an RSS run of a group", "grabs_search", row{"lane": nil, "trigger": "rss", "finished_at": at(1), "outcome": "nothing_found"}, false},
		{"an RSS run with no root", "grabs_search", row{"group_key": nil, "lane": nil, "trigger": "rss", "finished_at": at(1), "outcome": "no_root"}, false},
		{"a grabbed run with no Grab", "grabs_search", row{"finished_at": at(1), "outcome": "grabbed"}, false},
		{"a run that took Grabs and found nothing", "grabs_search", row{"finished_at": at(1), "outcome": "nothing_found", "grabs": 1}, false},
		{"a third phase", "grabs_search", row{"phase": 3}, false},
		{"a run joining a Grab", "grabs_search_grab", nil, true},
		{"a run making a Grab half", "grabs_search_grab", row{"made": 2}, false},
		{"a second run making one Grab", "grabs_search_grab", row{"made": 1}, false},
		{"a planned line", "grabs_search_indexer", nil, true},
		{"a single-Item line in flight", "grabs_search_indexer", row{"item_id": item, "query_id": storage.NewUUID(), "dispatched_at": at(1), "deadline_at": at(121)}, true},
		{"a line in flight with no dispatch", "grabs_search_indexer", row{"query_id": storage.NewUUID()}, false},
		{"a dispatch with no deadline", "grabs_search_indexer", row{"query_id": storage.NewUUID(), "dispatched_at": at(1)}, false},
		{"a dispatch with no query", "grabs_search_indexer", row{"dispatched_at": at(1), "deadline_at": at(121)}, false},
		{"a heard Release", "grabs_release", nil, true},
		{"a waiting candidate", "grabs_release", row{"outcome": "waiting", "rank": 1, "heard_at": nil}, true},
		{"a heard Release with a rank", "grabs_release", row{"rank": 1}, false},
		{"a heard Release with no time", "grabs_release", row{"heard_at": nil}, false},
		{"a kept Release with a heard time", "grabs_release", row{"outcome": "kept", "rank": 1}, false},
		{"a heard Release with no link", "grabs_release", row{"link_sealed": nil}, false},
		{"a waiting candidate that is a copy", "grabs_release", row{"outcome": "waiting", "rank": 1, "heard_at": nil, "copy_of": 0}, false},
		{"an RSS Grab with its GUID", "grabs_grab", nil, true},
		{"a Grab chosen to fill one Item", "grabs_grab", row{"chosen_fills": 1}, false},
		{"a GUID over 512 bytes", "grabs_grab", row{"release_guid": strings.Repeat("é", 257)}, false},
		{"an empty GUID", "grabs_grab", row{"release_guid": ""}, false},
		{"a gap reset", "grabs_nudge", nil, true},
		{"a failure's nudge into the now lane", "grabs_nudge", row{"trigger": "failed_grab", "lane": 10, "items": "[]"}, true},
		{"a nudge of RSS sync", "grabs_nudge", row{"trigger": "rss"}, false},
		{"a nudge into an unknown lane", "grabs_nudge", row{"trigger": "search", "lane": 20}, false},
		{"Items that aren't a list", "grabs_nudge", row{"trigger": "search", "lane": 10, "items": `{"a":1}`}, false},
		{"a group at its interval", "grabs_due", nil, true},
		{"a group queued for Backlog search", "grabs_due", row{"lane": 40, "queued_at": at(1), "trigger": "backlog"}, true},
		{"a queued press with gone cells", "grabs_due", row{"lane": 10, "queued_at": at(1), "trigger": "search", "include_gone": 1}, true},
		{"a lane with no trigger", "grabs_due", row{"lane": 40, "queued_at": at(1)}, false},
		{"a lane with no queued time", "grabs_due", row{"lane": 40, "trigger": "backlog"}, false},
		{"gone cells not queued", "grabs_due", row{"include_gone": 1}, false},
		{"a gap reset queued", "grabs_due", row{"lane": 40, "queued_at": at(1), "trigger": "retarget"}, false},
		{"a gap under the shortest interval", "grabs_due", row{"gap_seconds": 3599}, false},
		{"a gap over the longest gap", "grabs_due", row{"gap_seconds": 7776001}, false},
		{"a stopped cell", "grabs_failure", nil, true},
		{"a refusal by Indexer and GUID", "grabs_failure", row{"refuses": 1, "indexer_id": storage.NewUUID(), "guid": "a", "stopped_at": nil}, true},
		{"a refusal with nothing to refuse by", "grabs_failure", row{"refuses": 1, "stopped_at": nil}, true},
		{"a stop that isn't counted", "grabs_failure", row{"counted": 0}, false},
		{"a 19-byte info hash", "grabs_failure", row{"info_hash": hash(19)}, false},
		{"a wait", "grabs_wait", nil, true},
		{"a wait that ends as it starts", "grabs_wait", row{"until_at": at(10), "edition": "standard"}, false},
		{"a second settings row", "grabs_settings", row{"id": 2}, false},
	} {
		err := lib.Write(ctx, insertRows(tableRow{c.table, with(tables[c.table](), c.change)}))
		if c.ok && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
		if !c.ok && !refusedByConstraint(err) {
			t.Errorf("%s: want a constraint to refuse it, got %v", c.name, err)
		}
	}
	for _, c := range []struct {
		name   string
		change string
		ok     bool
	}{
		{"every setting at its default", "backlog_on = NULL", true},
		{"settings at their bounds", "longest_gap_seconds = 86400, full_pace_seconds = 0, stop_count = 20, stop_window_seconds = 3600", true},
		{"a longest gap under a day", "longest_gap_seconds = 86399", false},
		{"a full pace over 90 days", "full_pace_seconds = 7776001", false},
		{"a stop after no failures", "stop_count = 0", false},
		{"a stop after 21 failures", "stop_count = 21", false},
		{"a stop window over 30 days", "stop_window_seconds = 2592001", false},
		{"a switch neither on nor off", "rss_on = 2", false},
	} {
		err := lib.Write(ctx, sql(`UPDATE grabs_settings SET `+c.change+` WHERE id = 1`))
		if c.ok != (err == nil) || (!c.ok && !refusedByCheck(err)) {
			t.Errorf("%s: got %v", c.name, err)
		}
	}
}

func refusedByConstraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "constraint failed")
}
