package sim_test

import (
	"net/http"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

// TestTheMovesScenarioBoots launches the moves Scenario and sees its five
// roots on seedbox and den and each Node's external clients responding.
func TestTheMovesScenarioBoots(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := movestest.Scenario(t)
		w := sim.Launch(t, s, sim.WithWalk())
		time.Sleep(time.Minute)
		nodes := map[string]string{}
		for _, n := range simtest.Decode[struct {
			Nodes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"nodes"`
		}](t, "listNodes", w.API().Admin("home", http.MethodGet, "/api/v1/nodes", "")).Nodes {
			nodes[n.ID] = n.Name
		}
		roots := simtest.Decode[struct {
			Roots []simtest.RootRow `json:"roots"`
		}](t, "listRoots", w.API().Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots
		var got []string
		for _, r := range roots {
			got = append(got, nodes[r.NodeID]+" "+filepath.Base(r.Path))
		}
		slices.Sort(got)
		if want := []string{"den movies-e", "den movies-f", "den series-e", "seedbox movies-d", "seedbox series-d"}; !slices.Equal(got, want) {
			t.Fatalf("the roots are %q, want %q", got, want)
		}
		list := simtest.Decode[struct {
			Cards []struct {
				NodeName  string        `json:"nodeName"`
				Externals []savedClient `json:"externals"`
			} `json:"cards"`
		}](t, "listDownloadClients", w.API().Admin("home", http.MethodGet, "/api/v1/download-clients", ""))
		var clients []string
		for _, c := range list.Cards {
			for _, e := range c.Externals {
				clients = append(clients, c.NodeName+" "+e.Name+" "+e.Reach)
			}
		}
		slices.Sort(clients)
		if want := []string{"den Transmission responding", "den rTorrent responding", "seedbox Box responding", "seedbox Deluge responding"}; !slices.Equal(clients, want) {
			t.Fatalf("the external clients read %q, want %q", clients, want)
		}
	})
}
