package grabs_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// byRoot routes every Grab to the built-in client of its root's Node.
type byRoot struct{}

func (byRoot) Route(_ context.Context, _ storage.Reader[storage.Library], a downloadclients.Ask) (downloadclients.Target, error) {
	return downloadclients.Target{Node: a.Root}, nil
}

func TestAnUnfinishedGrabOnANodeOutOfStepWaitsOnIt(t *testing.T) {
	lib := storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	work, item := storage.NewUUID(), storage.NewUUID()
	home, away := storage.NewUUID(), storage.NewUUID()
	hd, uhd, sd := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	fetching, waiting, failed := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
		stage, failure_code, failed_stage, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Orchard.WEB-DL', 1, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', ?, ?, ?, ?, ?)`
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0), (?, randomblob(32), 'seedbox', 0)`, home, away),
		sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0), (?, '2160p', 0), (?, 'SD', 0)`, hd, uhd, sd),
		sql(`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, work),
		sql(`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, item, work),
		sql(grab, fetching, work, item, home, "fetching", nil, nil, 1, 1),
		sql(grab, waiting, work, item, away, "fetching", nil, nil, 2, 2),
		sql(grab, failed, work, item, away, "failed", "fetch_failed", "fetching", 3, 3),
		sql(`INSERT INTO grabs_target VALUES (?, 0, ?), (?, 0, ?), (?, 0, ?)`, fetching, hd, waiting, uhd, failed, sd),
	)
	rootstest.Root(t, lib, role.NodeID{UUID: home}, contenttypetest.ID, t.TempDir())
	inStep := func(n role.NodeID) bool { return n.UUID == home }
	var got map[files.CellKey]files.GrabCell
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = grabs.GrabCells{InStep: inStep, Router: byRoot{}}.GrabCells(ctx, r, works.Only([]works.WorkID{{UUID: work}}))
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
		{"fetching on home", hd, files.GrabCell{State: files.GrabInFlight, Grab: fetching}},
		{"fetching on the seedbox out of step", uhd, files.GrabCell{State: files.GrabInFlight, Grab: waiting, Waiting: grabs.NodeAway}},
		{"failed on the seedbox", sd, files.GrabCell{State: files.GrabFailed, Grab: failed, FailedAt: "fetching"}},
	} {
		if g := cell(c.target); g != c.want {
			t.Errorf("the Grab %s reads %+v, want %+v", c.what, g, c.want)
		}
	}
}

func TestAnUnfinishedGrabHoldsItsCellWhateverFinishedAfterIt(t *testing.T) {
	lib := storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	work, item, home, client, torrent := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
	const grab = `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, release_magnet,
		stage, failure_code, failed_stage, torrent_id, path, file_index, probe, created_at, updated_at)
		VALUES (?, ?, ?, 'any', 'search', ?, 'Orchard.WEB-DL', 1, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', ?, ?, iif(? IS NULL, NULL, 'downloading'), ?, ?, 0, '{}', ?, ?)`
	run(t, lib,
		sql(`INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, home),
		sql(`INSERT INTO downloadclients_client (id, node_id, kind, name, address, address_key, label, is_on, settings_version,
			poll_state, poll_since, poll_problem, poll_settings_version, created_at, updated_at)
			VALUES (?, ?, 'transmission', 'Transmission', 'http://192.168.1.20:9091/transmission/rpc', 'a', 'tv-sonarr', 1, 1, 'responding', 0, NULL, 1, 0, 0)`, client, home),
		sql(`INSERT INTO downloadclients_torrent (id, node_id, client_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
			added_order, added_at) VALUES (?, ?, ?, randomblob(20), x'00', 'Orchard.WEB-DL', 1, 0, '[]', 'fixture', 'started', 1, 0)`, torrent, home, client),
		sql(`INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'fixture', 'w1', 'Orchard', 'Orchard', 0, 0)`, work),
		sql(`INSERT INTO works_item (id, work_id, key, label, position) VALUES (?, ?, 'i1', 'Orchard', 1)`, item, work),
	)
	rootstest.Root(t, lib, role.NodeID{UUID: home}, contenttypetest.ID, t.TempDir())
	type step struct{ stage, failure, path any }
	fetching, downloading := step{"fetching", nil, nil}, step{"downloading", nil, nil}
	imported, failed, canceled := step{"imported", nil, "Orchard.mkv"}, step{"failed", "fetch_failed", nil}, step{"failed", "canceled", nil}
	cases := []struct {
		what  string
		steps []step
		want  files.GrabState
		from  int
	}{
		{"a fetching Grab", []step{fetching}, files.GrabInFlight, 0},
		{"a fetching Grab then an imported one", []step{fetching, imported}, files.GrabInFlight, 0},
		{"a downloading Grab then a failed one", []step{downloading, failed}, files.GrabInFlight, 0},
		{"a failed Grab then a fetching one", []step{failed, fetching}, files.GrabInFlight, 1},
		{"two unfinished Grabs", []step{downloading, fetching}, files.GrabInFlight, 1},
		{"an imported Grab then a failed one", []step{imported, failed}, files.GrabFailed, 1},
		{"a failed Grab then an imported one", []step{failed, imported}, files.GrabNone, -1},
		{"a failed Grab then a canceled one", []step{failed, canceled}, files.GrabNone, -1},
	}
	ids, cellTargets := make([][]storage.UUID, len(cases)), make([]storage.UUID, len(cases))
	at := 0
	for i, c := range cases {
		cellTargets[i] = storage.NewUUID()
		run(t, lib, sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, ?, 0)`, cellTargets[i], fmt.Sprint("T", i)))
		for _, st := range c.steps {
			id := storage.NewUUID()
			ids[i] = append(ids[i], id)
			at++
			run(t, lib,
				sql(grab, id, work, item, home, st.stage, st.failure, st.failure, torrent, st.path, at, at),
				sql(`INSERT INTO grabs_target VALUES (?, 0, ?)`, id, cellTargets[i]),
			)
		}
	}
	var got map[files.CellKey]files.GrabCell
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = grabs.GrabCells{InStep: func(role.NodeID) bool { return true }, Router: byRoot{}}.GrabCells(ctx, r, works.Only([]works.WorkID{{UUID: work}}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		cell, ok := got[files.CellKey{Item: works.ItemID{UUID: item}, Edition: targets.AnyEdition, Target: targets.TargetID{UUID: cellTargets[i]}}]
		want := files.GrabCell{State: c.want}
		if c.want == files.GrabFailed {
			want.FailedAt = "downloading"
		}
		if c.from >= 0 {
			want.Grab = ids[i][c.from]
		}
		if ok != (c.from >= 0) || cell != want {
			t.Errorf("%s: the cell reads %+v (present %t), want %+v", c.what, cell, ok, want)
		}
	}
}

func TestAnUnfinishedGrabWhoseClientIsAwayWaitsOnTheClient(t *testing.T) {
	lib := storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), indexers.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
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
		if cell.Grab != grabsIn[c.target] || cell.State != files.GrabInFlight || cell.Waiting != c.want {
			t.Errorf("the Grab %s reads %+v, want downloading waiting on %q", c.what, cell, c.want)
		}
	}
}
