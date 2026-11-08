package grabs_test

import (
	"context"
	"crypto/sha1" //nolint:gosec // the v1 info hash
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/works"
)

func releaseNamed(t *testing.T, name string) release {
	t.Helper()
	info, err := bencode.Encode(map[string]any{"name": name, "piece length": int64(16384), "pieces": "aaaaaaaaaaaaaaaaaaaa", "length": int64(4096)})
	if err != nil {
		t.Fatal(err)
	}
	return release{info: info, link: magnet.Link{InfoHash: sha1.Sum(info), Name: name}} //nolint:gosec // the v1 info hash
}

// trackedGrab is a fetching Grab of rel from a Tracker of privacy.
func (p *pipeline) trackedGrab(rel release, privacy indexers.Privacy) grabs.GrabID {
	p.t.Helper()
	tracker := storage.NewUUID()
	id := p.grab(rel.link.String(), &rel.link.InfoHash)
	run(p.t, p.lib,
		sql(`INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, ?, ?, ?)`, tracker, rel.link.Name, string(privacy), storage.Now()),
		sql(`UPDATE grabs_grab SET tracker_id = ? WHERE id = ?`, tracker, id),
	)
	return id
}

func (p *pipeline) addNode(name string) role.NodeID {
	p.t.Helper()
	id := role.NodeID{UUID: storage.NewUUID()}
	run(p.t, p.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, id, name)
	})
	return id
}

// TestAPrivateGrabWaitsForItsOfflineHomeUntilSentWhileAPublicOneGoesToTheCover
// takes home, the Indexer's home Node and its only Downloading holder, out
// of step while Controller covers it. A public Tracker's Grab downloads on
// Controller. A private Tracker's Grab waits at fetching for home and never
// moves on its own, and the Admin's send to Controller's built-in client
// fetches and downloads it there. Until then its view names home's
// built-in client as the one it waits for.
func TestAPrivateGrabWaitsForItsOfflineHomeUntilSentWhileAPublicOneGoesToTheCover(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		public, private := releaseNamed(t, "Public.Release"), releaseNamed(t, "Private.Release")
		p := newPipeline(t, answer{res: downloadclients.Resolved{Info: public.info}}, answer{res: downloadclients.Resolved{Info: private.info}})
		controller := p.addNode("Controller")
		p.roster.takeAway(p.node, controller)
		pub := p.trackedGrab(public, indexers.Public)
		priv := p.trackedGrab(private, indexers.Private)
		p.running()

		time.Sleep(10 * hold)
		if v := p.view(pub); v.Stage != grabs.Downloading || v.Node != controller {
			t.Fatalf("with home away the public Grab is %s on %v (%+v), want downloading on Controller", v.Stage, v.Node, v.Failed)
		}
		if v := p.view(priv); v.Stage != grabs.Fetching || v.Torrent != nil || v.WaitsFor == nil || v.WaitsFor.Node != p.node || v.WaitsFor.Client != nil {
			t.Fatalf("with home away the private Grab is %s with %+v waiting for %+v, want fetching and waiting for home's built-in client", v.Stage, v.Torrent, v.WaitsFor)
		}
		if v := p.view(pub); v.WaitsFor != nil {
			t.Fatalf("the public Grab waits for %+v, want no client", v.WaitsFor)
		}
		if calls := p.swarm.asked(); len(calls) != 1 || calls[0].link.InfoHash != public.link.InfoHash || calls[0].node != controller {
			t.Fatalf("the Nodes were asked for %+v, want only Controller for the public release", calls)
		}

		if err := p.g.SendTo(t.Context(), identity.Admin{}, priv, downloadclients.Choice{BuiltInOn: controller}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * hold)
		if v := p.view(priv); v.Stage != grabs.Downloading || v.Node != controller || v.Torrent == nil || v.Torrent.InfoHash != private.link.InfoHash || v.WaitsFor != nil {
			t.Fatalf("once sent the private Grab is %s on %v with %+v waiting for %+v, want downloading its release on Controller", v.Stage, v.Node, v.Torrent, v.WaitsFor)
		}
		if calls := p.swarm.asked(); len(calls) != 2 || calls[1].node != controller {
			t.Fatalf("the Nodes were asked for %+v, want Controller for the private release second", calls)
		}
	})
}

// TestSendToRefusesAFetchingGrabWhoseClientIsInStep refuses to send a
// Grab whose fetch may go ahead: it waits for no client's Node.
func TestSendToRefusesAFetchingGrabWhoseClientIsInStep(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		id := p.trackedGrab(releaseNamed(t, "Private.Release"), indexers.Private)
		err := p.g.SendTo(t.Context(), identity.Admin{}, id, downloadclients.Choice{BuiltInOn: p.node})
		if !errors.Is(err, grabs.ErrNotSendable) {
			t.Fatalf("a send of a Grab whose home is in step answered %v, want ErrNotSendable", err)
		}
	})
}

// waitingByTarget is what cells reads each of p's Work's cells as waiting
// on, by Target.
func (p *pipeline) waitingByTarget(cells grabs.GrabCells) map[storage.UUID]string {
	p.t.Helper()
	var got map[files.CellKey]files.GrabCell
	err := p.lib.Read(p.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = cells.GrabCells(ctx, r, works.Only([]works.WorkID{p.work}))
		return err
	})
	if err != nil {
		p.t.Fatal(err)
	}
	out := map[storage.UUID]string{}
	for key, cell := range got {
		out[key.Target.UUID] = cell.Waiting
	}
	return out
}

// TestAFetchingGrabWaitsOnTheOfflineNodeItsRouteNames keeps home, the
// Indexer's home Node, in step while seedbox is out of step. A private
// Tracker's Grab routed from home to seedbox's built-in client, and a
// public one the Admin sent there, each wait at fetching on seedbox, while
// a public one that goes by its Work's root waits on nothing.
func TestAFetchingGrabWaitsOnTheOfflineNodeItsRouteNames(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		seedbox := p.addNode("seedbox")
		p.roster.takeAway(seedbox, p.node)
		routed := p.trackedGrab(releaseNamed(t, "Private.Release"), indexers.Private)
		sent := p.trackedGrab(releaseNamed(t, "Sent.Release"), indexers.Public)
		rooted := p.trackedGrab(releaseNamed(t, "Public.Release"), indexers.Public)
		goals := map[grabs.GrabID]storage.UUID{routed: storage.NewUUID(), sent: storage.NewUUID(), rooted: storage.NewUUID()}
		for id, target := range goals {
			run(t, p.lib,
				sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, ?, 0)`, target, target.String()),
				sql(`INSERT INTO grabs_target VALUES (?, 0, ?)`, id, target),
			)
		}
		run(t, p.lib,
			sql(`INSERT INTO trackers_route (tracker_id, home_node_id, built_in_on) SELECT tracker_id, ?, ? FROM grabs_grab WHERE id = ?`, p.node, seedbox, routed),
			sql(`UPDATE grabs_grab SET sent_built_in_on = ? WHERE id = ?`, seedbox, sent),
		)

		waiting := p.waitingByTarget(grabs.GrabCells{InStep: p.roster.InStep, Router: downloadclients.Router{Roster: p.roster, Routes: trackers.Routing{Roster: p.roster}}})
		for _, c := range []struct {
			what string
			id   grabs.GrabID
			want string
		}{
			{"routed to seedbox", routed, grabs.NodeAway},
			{"sent to seedbox", sent, grabs.NodeAway},
			{"going by its root", rooted, ""},
		} {
			if w, ok := waiting[goals[c.id]]; !ok || w != c.want {
				t.Errorf("the Grab %s waits on %q (listed %t), want %q", c.what, w, ok, c.want)
			}
			if v := p.view(c.id); (v.WaitsFor != nil) != (c.want != "") || (v.WaitsFor != nil && v.WaitsFor.Node != seedbox) {
				t.Errorf("the view of the Grab %s waits for %+v, want seedbox's built-in client only while its cell waits", c.what, v.WaitsFor)
			}
		}
	})
}
