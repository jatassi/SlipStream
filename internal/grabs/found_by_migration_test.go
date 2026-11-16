package grabs_test

import (
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestTheFoundByMigrationReadsEachGrabsMakingRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := openSets(t, path, downloadclients.Schema(), grabs.Schema().Through(14), indexers.Schema(), files.Schema())
	node, work, item := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	backlog, failed, joined, rss, manual, pressed := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	backlogRun, failedRun := storage.NewUUID(), storage.NewUUID()
	grab := func(id storage.UUID, source string) tableRow {
		g := fetchingGrab(id, work, item, node)
		g.r["source"] = source
		return g
	}
	queueRun := func(id storage.UUID, trigger string) tableRow {
		return tableRow{"grabs_search", row{
			"id": id, "work_id": work, "group_key": keyOf("i:", item), "lane": 40, "trigger": trigger,
			"started_at": at(1), "finished_at": at(2), "outcome": "grabbed", "grabs": 1,
		}}
	}
	run(t, old, insertRows(
		tableRow{"nodes_node", row{"id": node, "public_key": hash(32), "name": "home", "paired_at": 0}},
		workRow(work, "Dune"),
		tableRow{"works_item", row{"id": item, "work_id": work, "key": "i1", "label": "One", "position": 1}},
		grab(backlog, "search"), grab(failed, "search"), grab(joined, "search"), grab(rss, "rss"), grab(manual, "manual"),
		grab(pressed, "search"),
		queueRun(backlogRun, "backlog"), queueRun(failedRun, "failed_grab"),
		tableRow{"grabs_search_grab", row{"search_id": backlogRun, "grab_id": backlog, "made": 1}},
		tableRow{"grabs_search_grab", row{"search_id": backlogRun, "grab_id": joined, "made": 0}},
		tableRow{"grabs_search_grab", row{"search_id": failedRun, "grab_id": failed, "made": 1}},
	))
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openAt(t, path)
	t.Cleanup(func() { _ = lib.Close() })
	for _, c := range []struct {
		name string
		grab storage.UUID
		want string
	}{
		{"a Grab a Backlog search made", backlog, "backlog"},
		{"a Grab a search after a failed Grab made", failed, "failed_grab"},
		{"a Grab a Backlog search only joined", joined, "search"},
		{"an RSS Grab", rss, "rss"},
		{"a Manual Grab", manual, "manual"},
		{"a Grab with no kept run", pressed, "search"},
	} {
		if got := scalar(t, lib, "SELECT found_by FROM grabs_grab WHERE id = ?", c.grab); got != c.want {
			t.Errorf("%s reads found_by %q, want %q", c.name, got, c.want)
		}
	}
}
