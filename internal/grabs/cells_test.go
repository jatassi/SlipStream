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
	work, item := storage.NewUUID(), storage.NewUUID()
	home, away := storage.NewUUID(), storage.NewUUID()
	hd, uhd, sd := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	fetching, waiting, failed := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
		stage, failure_code, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Orchard.WEB-DL', 1, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', ?, ?, ?, ?)`
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0), (?, randomblob(32), 'seedbox', 0)`, home, away),
		sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0), (?, '2160p', 0), (?, 'SD', 0)`, hd, uhd, sd),
		sql(`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, work),
		sql(`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, item, work),
		sql(grab, fetching, work, item, home, "fetching", nil, 1, 1),
		sql(grab, waiting, work, item, away, "fetching", nil, 2, 2),
		sql(grab, failed, work, item, away, "failed", "fetch_failed", 3, 3),
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

func TestAnUnfinishedGrabWhoseClientIsAwayWaitsOnTheClient(t *testing.T) {
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	work, item := storage.NewUUID(), storage.NewUUID()
	media, away := storage.NewUUID(), storage.NewUUID()
	down, fine, off, offline := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	hd, uhd, sd, dvd := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	const client = `INSERT INTO downloadclients_client (id, node_id, kind, name, address, address_key, label, is_on, settings_version,
		poll_state, poll_since, poll_problem, poll_settings_version, created_at, updated_at)
		VALUES (?, ?, 'transmission', ?, ?, ?, 'tv-sonarr', ?, 1, ?, 0, ?, 1, 0, 0)`
	const torrent = `INSERT INTO downloadclients_torrent (id, node_id, client_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
		added_order, added_at) VALUES (?, ?, ?, randomblob(20), x'00', 'Orchard.WEB-DL', 1, 0, '[]', 'fixture', 'started', ?, 0)`
	const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size,
		stage, torrent_id, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Orchard.WEB-DL', 1, 'downloading', ?, ?, ?)`
	noResponse := `{"cause":"no_response"}`
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'Media server', 0), (?, randomblob(32), 'seedbox', 0)`, media, away),
		sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0), (?, '2160p', 0), (?, 'SD', 0), (?, 'DVD', 0)`, hd, uhd, sd, dvd),
		sql(`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, work),
		sql(`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, item, work),
		sql(client, down, media, "Transmission", "http://192.168.1.20:9091/transmission/rpc", "a", 1, "not_responding", noResponse),
		sql(client, fine, media, "Transmission 4K", "http://192.168.1.20:9092/transmission/rpc", "b", 1, "responding", nil),
		sql(client, off, media, "Transmission old", "http://192.168.1.20:9093/transmission/rpc", "c", 0, "not_responding", noResponse),
		sql(client, offline, away, "Seedbox Transmission", "http://192.168.1.30:9091/transmission/rpc", "d", 1, "not_responding", noResponse),
	)
	grabsIn := map[storage.UUID]storage.UUID{}
	for i, c := range []struct{ client, node, target storage.UUID }{{down, media, hd}, {fine, media, uhd}, {off, media, sd}, {offline, away, dvd}} {
		tid, gid := storage.NewUUID(), storage.NewUUID()
		grabsIn[c.target] = gid
		run(t, lib,
			sql(torrent, tid, c.node, c.client, i+1),
			sql(grab, gid, work, item, c.node, tid, i+1, i+1),
			sql(`INSERT INTO grabs_target VALUES (?, 0, ?)`, gid, c.target),
		)
	}
	inStep := func(n role.NodeID) bool { return n.UUID == media }
	var got map[files.CellKey]files.GrabCell
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = grabs.GrabCells{InStep: inStep}.GrabCells(ctx, r, works.Only([]works.WorkID{{UUID: work}}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		what   string
		target storage.UUID
		want   string
	}{
		{"in a client that isn't responding", hd, grabs.ClientAway},
		{"in a responding client", uhd, ""},
		{"in an off client", sd, ""},
		{"in a client on a Node out of step", dvd, grabs.NodeAway},
	} {
		cell := got[files.CellKey{Item: works.ItemID{UUID: item}, Edition: targets.AnyEdition, Target: targets.TargetID{UUID: c.target}}]
		if cell.Grab != grabsIn[c.target] || cell.State != files.GrabDownloading || cell.Waiting != c.want {
			t.Errorf("the Grab %s reads %+v, want downloading waiting on %q", c.what, cell, c.want)
		}
	}
}
