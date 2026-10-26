package grabs_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAnUnfinishedGrabOnANodeOutOfStepWaitsOnIt(t *testing.T) {
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	work, item, indexer := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	home, away := storage.NewUUID(), storage.NewUUID()
	hd, uhd, sd := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	fetching, waiting, failed := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size, release_link,
		stage, failure_code, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, ?, 'Orchard.WEB-DL', 1, 'https://site.test/dl', ?, ?, ?, ?)`
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0), (?, randomblob(32), 'seedbox', 0)`, home, away),
		sql(`INSERT INTO indexers_indexer VALUES (?, 'Fixture Site', 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, home),
		sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0), (?, '2160p', 0), (?, 'SD', 0)`, hd, uhd, sd),
		sql(`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, work),
		sql(`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, item, work),
		sql(grab, fetching, work, item, indexer, home, "fetching", nil, 1, 1),
		sql(grab, waiting, work, item, indexer, away, "fetching", nil, 2, 2),
		sql(grab, failed, work, item, indexer, away, "failed", "fetch_failed", 3, 3),
		sql(`INSERT INTO grabs_target VALUES (?, 0, ?), (?, 0, ?), (?, 0, ?)`, fetching, hd, waiting, uhd, failed, sd),
	)
	inStep := func(n role.NodeID) bool { return n.UUID == home }
	var got map[files.CellKey]files.GrabCell
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = grabs.GrabCells{InStep: inStep}.GrabCells(ctx, r, works.Only([]works.WorkID{{UUID: work}}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	cell := func(target storage.UUID) files.GrabCell {
		return got[files.CellKey{Item: works.ItemID{UUID: item}, Edition: targets.AnyEdition, Target: targets.TargetID{UUID: target}}]
	}
	for _, c := range []struct {
		what   string
		target storage.UUID
		want   files.GrabCell
	}{
		{"fetching on home", hd, files.GrabCell{Grab: fetching}},
		{"fetching on the seedbox out of step", uhd, files.GrabCell{Grab: waiting, Waiting: grabs.NodeAway}},
		{"failed on the seedbox", sd, files.GrabCell{State: files.GrabFailed, Grab: failed}},
	} {
		if g := cell(c.target); g != c.want {
			t.Errorf("the Grab %s reads %+v, want %+v", c.what, g, c.want)
		}
	}
}
