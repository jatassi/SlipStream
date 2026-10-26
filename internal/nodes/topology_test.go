package nodes_test

import (
	"context"
	"io"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func (c *controller) topology(t *testing.T) nodes.Topology {
	t.Helper()
	topo, err := c.svc.Topology(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return topo
}

func onlyLink(t *testing.T, topo nodes.Topology) nodes.TopologyLink {
	t.Helper()
	if len(topo.Links) != 1 {
		t.Fatalf("the topology has links %+v, want the one to seedbox", topo.Links)
	}
	return topo.Links[0]
}

// expectInSession checks the topology of home downloading and seedbox,
// Node id, searching over a live link that seedbox dialed.
func expectInSession(t *testing.T, topo nodes.Topology, id nodes.ID) {
	t.Helper()
	if topo.Sentence != "Both Nodes are online." {
		t.Errorf("in session the sentence reads %q", topo.Sentence)
	}
	if len(topo.Nodes) != 2 || !topo.Nodes[0].Controller || topo.Nodes[1].Name != "seedbox" || topo.Nodes[1].Standing != nodes.InStep {
		t.Fatalf("the topology's Nodes are %+v, want home then seedbox in step", topo.Nodes)
	}
	if got := roleNames(topo.Nodes[1].Roles); len(got) != 1 || got[0] != role.Searching || topo.Nodes[1].Downloading != nil {
		t.Errorf("seedbox stands in %v with a client %+v, want Searching and no client", got, topo.Nodes[1].Downloading)
	}
	if d := topo.Nodes[0].Downloading; d == nil || len(d.Torrents) != 0 {
		t.Errorf("home, downloading with a client that reports nothing yet, reads %+v; want a client with no torrents", d)
	}
	if l := onlyLink(t, topo); l.Worker != id || l.Dialer != id || l.State != nodes.Up || l.RTT < 80*time.Millisecond {
		t.Errorf("the link reads %+v, want seedbox dialing, up, with at least the path's 80 ms round trip", l)
	}
}

// seedbox searches in session with home over a path that adds 80 ms each
// round trip. The topology reads both Nodes online and seedbox dialing,
// then seedbox offline with home searching in its place once it is gone
// past the grace.
func TestTheTopologyFollowsTheLinkAndTheBoard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "seedbox", "127.0.0.3")
		if _, err := c.net.Path(linktest.WorkerAddr, linktest.ControllerAddr).Set(fault.Rule{Fault: fault.Latency(40 * time.Millisecond)}); err != nil {
			t.Fatal(err)
		}
		_, id := c.pair(t, w, c.mint(t), "seedbox")
		holder(t, role.InProcess(c.svc.Controls()))
		stop := c.hold(t, w)
		c.apply(t, nodes.Roles{id: role.Of(role.Searching), c.svc.Self(): role.Of(role.Downloading)})
		time.Sleep(time.Minute)
		synctest.Wait()

		expectInSession(t, c.topology(t), id)

		stop()
		w.Drop(c.node.Key)
		c.node.Reset(w.Key)
		synctest.Wait()
		time.Sleep(nodes.OfflineGrace.Default)
		synctest.Wait()
		topo := c.topology(t)
		if want := "seedbox is offline. home is searching in its place."; topo.Sentence != want {
			t.Errorf("past the grace the sentence reads %q, want %q", topo.Sentence, want)
		}
		if l := onlyLink(t, topo); l.State != nodes.Down || !l.Dialer.IsZero() {
			t.Errorf("with seedbox gone the link reads %+v, want down with no dialer", l)
		}
		if topo.Nodes[1].Standing != nodes.LapsedNode {
			t.Errorf("seedbox stands %s, want lapsed", topo.Nodes[1].Standing)
		}
	})
}

// TestTopologyCarriesStorage has seedbox stand in Storage: the topology
// carries its Storage figures and none for home, and keeps them once
// seedbox lapses.
func TestTopologyCarriesStorage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		figures := func(_ context.Context, _ storage.Reader[storage.Library], node nodes.ID) (nodes.StorageFigures, error) {
			return nodes.StorageFigures{Roots: []nodes.StorageRoot{{ID: node.UUID, ContentType: "alpha", Path: "/media/alpha", State: "ready", VolumeID: "v1"}}}, nil
		}
		c := startController(t, func(cfg *nodes.Config) { cfg.Storage = figures })
		w := c.worker(t, "seedbox", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "seedbox")
		holder(t, role.InProcess(c.svc.Controls()))
		stop := c.hold(t, w)
		c.apply(t, nodes.Roles{id: role.Of(role.Storage), c.svc.Self(): role.Of(role.Downloading)})
		time.Sleep(time.Minute)
		synctest.Wait()
		stored := func(n nodes.TopologyNode) bool {
			return n.Storage != nil && len(n.Storage.Roots) == 1 && n.Storage.Roots[0].ID == n.ID.UUID
		}
		topo := c.topology(t)
		if home, seedbox := topo.Nodes[0], topo.Nodes[1]; home.Storage != nil || !stored(seedbox) {
			t.Fatalf("in session home carries %+v and seedbox %+v; want seedbox's figures alone", home.Storage, seedbox.Storage)
		}

		stop()
		w.Drop(c.node.Key)
		c.node.Reset(w.Key)
		synctest.Wait()
		time.Sleep(nodes.OfflineGrace.Default)
		synctest.Wait()
		topo = c.topology(t)
		if home, seedbox := topo.Nodes[0], topo.Nodes[1]; home.Storage != nil || !stored(seedbox) {
			t.Fatalf("past the grace home carries %+v and seedbox %+v; want seedbox's figures alone", home.Storage, seedbox.Storage)
		}
	})
}

// While seedbox is in session the Sync stream moves bytes over the link.
// Each sample that shows them is published, and so is the first that
// shows none, so the canvas comes to rest; nothing is published while
// nothing moves.
func TestTrafficIsPublishedWhileBytesMoveAndOnceAtRest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "seedbox", "127.0.0.3")
		_, id := c.pair(t, w, c.mint(t), "seedbox")
		stop := c.hold(t, w)
		time.Sleep(2 * time.Minute)
		stop()
		w.Drop(c.node.Key)
		synctest.Wait()
		time.Sleep(5 * time.Second)
		synctest.Wait()

		samples := c.traffic.all()
		moving := 0
		for i, s := range samples {
			if len(s.Links) > 1 || len(s.Links) == 1 && (s.Links[0].A != c.svc.Self().String() || s.Links[0].B != id.String()) {
				t.Fatalf("sample %d has links %+v, want at most the one between home and seedbox", i, s.Links)
			}
			if !still(s) {
				moving++
				continue
			}
			if i == 0 || still(samples[i-1]) {
				t.Fatalf("sample %d at rest follows another at rest or none: %+v", i, samples[:i+1])
			}
		}
		if moving == 0 || len(samples) >= 120 {
			t.Fatalf("over two minutes %d samples were published, %d moving; want some, and fewer than one a second", len(samples), moving)
		}
		if last := samples[len(samples)-1]; !still(last) {
			t.Fatalf("with seedbox gone the last sample shows %+v, want the link at rest", last.Links)
		}
	})
}

// A Transfer lane between home and seedbox carries 1 MiB each way and
// closes before the next sample. Its bytes still count on their cable.
func TestTrafficCountsTransferLanes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startController(t)
		w := c.worker(t, "seedbox", "127.0.0.3")
		c.pair(t, w, c.mint(t), "seedbox")
		c.hold(t, w)
		time.Sleep(time.Minute)
		synctest.Wait()
		before := len(c.traffic.all())

		lane, err := w.DialLane(t.Context(), c.node.Key, c.node.Target().Addresses)
		if err != nil {
			t.Fatal(err)
		}
		const size = 1 << 20
		go func() { _, _ = lane.Write(make([]byte, size)) }()
		if _, err := io.ReadFull(lane, make([]byte, size)); err != nil {
			t.Fatal(err)
		}
		_ = lane.Close()
		time.Sleep(5 * time.Second)
		synctest.Wait()

		var toSeedbox, toHome float64
		for _, s := range c.traffic.all()[before:] {
			for _, l := range s.Links {
				toSeedbox += float64(l.ABBps) * nodes.DefaultTimings().Traffic.Seconds()
				toHome += float64(l.BABps) * nodes.DefaultTimings().Traffic.Seconds()
			}
		}
		if toSeedbox < size || toHome < size {
			t.Fatalf("the cable counted %.0f bytes to seedbox and %.0f to home, want the lane's %d each way", toSeedbox, toHome, size)
		}
	})
}

func still(s nodes.Traffic) bool {
	for _, l := range s.Links {
		if l.ABBps != 0 || l.BABps != 0 {
			return false
		}
	}
	return true
}

// sampled is a torrent client an Admin watches: each Node's moves one
// Torrent, half done.
type sampled struct{ torrent storage.UUID }

func (s sampled) Live(nodes.ID) (nodes.LiveClient, bool) {
	return nodes.LiveClient{
		DownBps: 4096, UpBps: 1024,
		Torrents: map[storage.UUID]nodes.LiveTorrent{s.torrent: {Done: 500, Wanted: 1000, Peers: 7}},
	}, true
}

// TestTrafficAndTheTopologyReadLiveSamples has home's client keep two
// Torrents, one of them sampled. nodes.traffic carries home's rates and
// the sampled Torrent; the topology draws both Torrents, the sampled one
// as sampled and the other as last reported.
func TestTrafficAndTheTopologyReadLiveSamples(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		moving, resting := storage.NewUUID(), storage.NewUUID()
		kept := func(context.Context, storage.Reader[storage.Library], nodes.ID) ([]nodes.Kept, error) { //nolint:unparam // Config.Kept reads the Library
			return []nodes.Kept{
				{Torrent: moving, Title: "Orchard", Done: 100, Wanted: 1000},
				{Torrent: resting, Title: "Meadow", Done: 2000, Wanted: 2000},
			}, nil
		}
		c := startController(t, func(cfg *nodes.Config) { cfg.Kept, cfg.Torrents = kept, sampled{moving} })
		holder(t, role.InProcess(c.svc.Controls()))
		c.apply(t, nodes.Roles{c.svc.Self(): role.Of(role.Downloading)})
		time.Sleep(time.Minute)
		synctest.Wait()

		want := []nodes.Torrent{
			{ID: moving, Title: "Orchard", Done: 500, Wanted: 1000, Peers: 7},
			{ID: resting, Title: "Meadow", Done: 2000, Wanted: 2000},
		}
		if d := c.topology(t).Nodes[0].Downloading; d == nil || !slices.Equal(d.Torrents, want) {
			t.Fatalf("home's Downloading reads %+v, want %+v", d, want)
		}
		home := nodes.NodeTraffic{
			ID: c.svc.Self().String(), DownBps: 4096, UpBps: 1024,
			Torrents: []nodes.TorrentTraffic{{TorrentID: moving.String(), Done: 500, Wanted: 1000, Peers: 7}},
		}
		samples := c.traffic.all()
		if len(samples) == 0 || !slices.EqualFunc(samples[len(samples)-1].Nodes, []nodes.NodeTraffic{home}, func(a, b nodes.NodeTraffic) bool {
			return a.ID == b.ID && a.DownBps == b.DownBps && a.UpBps == b.UpBps && slices.Equal(a.Torrents, b.Torrents)
		}) {
			t.Fatalf("nodes.traffic carried %+v, want home's client as %+v", samples, home)
		}
	})
}
