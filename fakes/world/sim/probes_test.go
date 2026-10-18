package sim_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
)

var fitting = []worldtest.Probe{{Probe: "engine-starts", Fit: "fits"}, {Probe: "data-writable", Fit: "fits"}, {Probe: "listener-up", Fit: "fits"}}

// suggested is what home's role board ticks first for the Node named name.
func suggested(t *testing.T, a *worldtest.API, name string) []string {
	t.Helper()
	r := a.Admin("home", http.MethodGet, "/api/v1/nodes/roles", "")
	var board struct {
		Nodes []struct {
			Name      string   `json:"name"`
			Suggested []string `json:"suggested"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(r.Body), &board); err != nil || r.Status != http.StatusOK {
		t.Fatalf("getRoleBoard = %d %s", r.Status, r.Body)
	}
	for _, n := range board.Nodes {
		if n.Name == name {
			return n.Suggested
		}
	}
	t.Fatalf("the role board has no node %s", name)
	return nil
}

// topologyProbes is the probes home's topology draws for the Node named
// name.
func topologyProbes(t *testing.T, a *worldtest.API, name string) []worldtest.Probe {
	t.Helper()
	topo := decode[topologyView](t, "getTopology", a.Admin("home", http.MethodGet, "/api/v1/nodes/topology", ""))
	for _, n := range topo.Nodes {
		if n.Name == name {
			return n.Probes
		}
	}
	t.Fatalf("the topology has no node %s", name)
	return nil
}

// awaitSeedboxProbes waits for seedbox's link to be up and listNodes to
// carry want as its probes. The probes reach home through sync after the
// link is up, so the listing can carry earlier probes for a moment.
func awaitSeedboxProbes(t *testing.T, a *worldtest.API, want []worldtest.Probe) {
	t.Helper()
	for deadline := time.Now().Add(time.Minute); ; {
		r := seedbox(t, a)
		if r.Link.State == "up" && slices.Equal(r.Probes, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("listNodes has seedbox's link %s and probes %+v, want up and %+v", r.Link.State, r.Probes, want)
		}
		time.Sleep(time.Second)
		synctest.Wait()
	}
}

// TestADownloadsFolderSeedboxCantWriteBlocksDownloadingThere lists every
// probe fitting on both Nodes, and the board suggests Searching and
// Downloading for seedbox. Restarted with its downloads folder on a file,
// seedbox's engine-starts probe fails on listNodes and in the topology,
// the board suggests Searching alone, and giving seedbox Downloading back
// is blocked in those words.
func TestADownloadsFolderSeedboxCantWriteBlocksDownloadingThere(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t))
		a := w.API()
		for _, r := range a.Nodes("home") {
			if got := r.Probes; !slices.Equal(got, fitting) {
				t.Fatalf("listNodes has %s's probes %+v, want each fitting", r.Name, got)
			}
		}
		if got := suggested(t, a, "seedbox"); !slices.Equal(got, []string{"searching", "downloading"}) {
			t.Fatalf("the board suggests %v for seedbox, want Searching and Downloading", got)
		}
		id := seedbox(t, a).ID
		a.SetRoles("home", id, "searching")

		file := filepath.Join(t.TempDir(), "downloads")
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		box := w.Node("seedbox")
		box.Stop()
		box.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_TORRENT_DOWNLOADS="+file) })
		want := slices.Clone(fitting)
		want[0] = worldtest.Probe{Probe: "engine-starts", Fit: "fails", Reason: "downloads_not_writable"}
		awaitSeedboxProbes(t, a, want)
		if got := topologyProbes(t, a, "seedbox"); !slices.Equal(got, want) {
			t.Fatalf("the topology has seedbox's probes %+v, want %+v", got, want)
		}
		if got := suggested(t, a, "seedbox"); !slices.Equal(got, []string{"searching"}) {
			t.Fatalf("the board suggests %v for seedbox, want Searching alone", got)
		}

		checks := blockedChecks(t, a.Assign("home", nodeapi.Assignment{NodeID: id, Roles: []string{"searching", "downloading"}}))
		if len(checks) != 1 || checks[0].Words != "Downloading can't run on seedbox. Its downloads folder can't be written to." {
			t.Fatalf("the blocked set checks %+v, want seedbox's failing engine-starts probe", checks)
		}
	})
}
