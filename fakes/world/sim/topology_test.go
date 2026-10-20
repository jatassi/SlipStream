package sim_test

import (
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type topologyView struct {
	Sentence string `json:"sentence"`
	Nodes    []struct {
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		State     string            `json:"state"`
		Probes    []worldtest.Probe `json:"probes"`
		Resources struct {
			Downloading *struct {
				Torrents []struct {
					GrabID string `json:"grabId"`
					Title  string `json:"title"`
					Done   int64  `json:"done"`
					Wanted int64  `json:"wanted"`
				} `json:"torrents"`
			} `json:"downloading"`
		} `json:"resources"`
	} `json:"nodes"`
	Links []struct {
		A      string `json:"a"`
		B      string `json:"b"`
		Dialer string `json:"dialer"`
		State  string `json:"state"`
	} `json:"links"`
}

// TestTheTopologyDrawsSeedboxsTorrent walks a Grab to imported on
// seedbox. The topology lists the torrent under seedbox's Downloading
// with the release's title, done, and the link seedbox dialed.
func TestTheTopologyDrawsSeedboxsTorrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithWalk())
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectImported(t, g)
		a := w.API()
		box := seedbox(t, a)

		topo := decode[topologyView](t, "getTopology", a.Admin("home", http.MethodGet, "/api/v1/nodes/topology", ""))
		if topo.Sentence != "Both Nodes are online." || len(topo.Nodes) != 2 || topo.Nodes[1].ID != box.ID || topo.Nodes[1].State != "online" {
			t.Fatalf("the topology reads %+v, want home and seedbox online", topo)
		}
		dl := topo.Nodes[1].Resources.Downloading
		if dl == nil || len(dl.Torrents) != 1 || dl.Torrents[0].GrabID != g.ID || dl.Torrents[0].Title != release ||
			dl.Torrents[0].Wanted == 0 || dl.Torrents[0].Done != dl.Torrents[0].Wanted {
			t.Fatalf("seedbox's Downloading reads %+v, want the Grab's torrent, titled %s and done", dl, release)
		}
		if l := topo.Links; len(l) != 1 || l[0].A != topo.Nodes[0].ID || l[0].B != box.ID || l[0].Dialer != box.ID || l[0].State != "up" {
			t.Fatalf("the topology's links are %+v, want home's link to seedbox, up and dialed by seedbox", l)
		}
	})
}
