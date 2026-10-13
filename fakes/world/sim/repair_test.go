package sim_test

import (
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/pairing"
)

// TestARunningWorkerRepairsOnlyAtStart sends office's Join Code to seedbox
// while it runs as home's Worker. The join op refuses it and nothing
// changes; a restart with the code in SLIPSTREAM_JOIN_CODE re-pairs, and
// seedbox's Roles then follow office.
func TestARunningWorkerRepairsOnlyAtStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := home(t)
		s.Name = "repair-running"
		s.Nodes = append(s.Nodes, scenario.Node{Name: "office"})
		w := sim.Launch(t, s)
		officeAPIKey := office(t, w)
		worker := w.Node("seedbox")
		code := mint(t, w, "office", officeAPIKey)

		reply, err := worker.JoinOp(t.Context(), code)
		want := pairing.Error{
			Kind:   "restart_required",
			Reason: "a running Worker re-pairs only at start: stop the Node and run slipstream join with this code, or restart it with the code in SLIPSTREAM_JOIN_CODE",
		}
		if err != nil || reply.Refused == nil || *reply.Refused != want {
			t.Fatalf("the join op with office's code = %+v, %v; want refused %+v", reply, err, want)
		}
		homeKey, officeKey := key(t, w, "home"), key(t, w, "office")
		if r := readRecord(t, worker); r.Controller.Key != homeKey.String() || len(r.Superseded) != 0 {
			t.Fatalf("seedbox's record is %+v, want home %s and nothing superseded", r, homeKey)
		}
		if rows := officeRows(t, w, officeAPIKey); len(rows) != 1 {
			t.Fatalf("office lists %+v, want only itself, since seedbox never sent Pair", rows)
		}
		reply, err = worker.JoinOp(t.Context(), mint(t, w, "home", w.Manifest().APIKey))
		if err != nil || reply.Refused == nil || reply.Refused.Kind != pairing.AlreadyJoined {
			t.Fatalf("the join op with home's code = %+v, %v; want already joined", reply, err)
		}

		worker.Stop()
		worker.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_JOIN_CODE="+code.String()) })
		synctest.Wait()
		if r := readRecord(t, worker); r.Controller.Key != officeKey.String() || len(r.Superseded) != 1 || r.Superseded[0].Key != homeKey.String() {
			t.Fatalf("seedbox's record is %+v, want office %s with home %s superseded", r, officeKey, homeKey)
		}
		rows := officeRows(t, w, officeAPIKey)
		i := slices.IndexFunc(rows, func(n nodeapi.Node) bool { return n.Name == "seedbox" })
		if i < 0 || rows[i].Link.State != "up" || rows[i].Facts == nil || rows[i].Facts.AppliedVersion != rows[i].IntentVersion {
			t.Fatalf("office lists %+v, want seedbox up with its intent applied", rows)
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
