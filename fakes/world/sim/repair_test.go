package sim_test

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/pairing"
)

// TestARunningWorkerRepairsLiveAndRestarts sends office's Join Code to
// seedbox while it runs as home's Worker. The join op answers joined, the
// Node restarts, and seedbox comes back with home superseded and its
// Roles following office.
func TestARunningWorkerRepairsLiveAndRestarts(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := simtest.Home(t)
		s.Name = "repair-running"
		s.Nodes = append(s.Nodes, scenario.Node{Name: "office"})
		w := sim.Launch(t, s)
		officeAPIKey := office(t, w)
		worker := w.Node("seedbox")

		reply, err := worker.JoinOp(t.Context(), mint(t, w, "office", officeAPIKey))
		if err != nil || reply.Refused != nil || !strings.HasPrefix(reply.Joined, "joined office as seedbox") {
			t.Fatalf("the join op with office's code = %+v, %v; want joined office", reply, err)
		}
		worker.AwaitRestart()
		synctest.Wait()

		homeKey, officeKey := key(t, w, "home"), key(t, w, "office")
		if r := readRecord(t, worker); r.Controller.Key != officeKey.String() || len(r.Superseded) != 1 || r.Superseded[0].Key != homeKey.String() {
			t.Fatalf("seedbox's record is %+v, want office %s with home %s superseded", r, officeKey, homeKey)
		}
		rows := officeRows(t, w, officeAPIKey)
		i := slices.IndexFunc(rows, func(n nodeapi.Node) bool { return n.Name == "seedbox" })
		if i < 0 || rows[i].Link.State != "up" || rows[i].Facts == nil || rows[i].Facts.AppliedVersion != rows[i].IntentVersion {
			t.Fatalf("office lists %+v, want seedbox up with its intent applied", rows)
		}
		reply, err = worker.JoinOp(t.Context(), mint(t, w, "home", w.Manifest().APIKey))
		if err != nil || reply.Refused == nil || reply.Refused.Kind != pairing.SupersededController {
			t.Fatalf("the join op with home's code = %+v, %v; want superseded Controller", reply, err)
		}
	})
}

func officeRows(t *testing.T, w *sim.World, apiKey string) []nodeapi.Node {
	t.Helper()
	rows, err := nodeapi.ListNodes(t.Context(), w.Client(), "http://"+w.Node("office").Ready().Listeners.HTTP.String(), apiKey)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
