package e2e

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func act(t *testing.T, run *world.Run, name string, actions ...world.NodeAction) world.Node {
	t.Helper()
	var entry world.Node
	for _, a := range actions {
		var err error
		if entry, err = run.Node(worldtest.Context(t), name, a); err != nil {
			t.Fatalf("%s %s: %v", a, name, err)
		}
	}
	return entry
}

func row(t *testing.T, rows []worldtest.Row, id string) worldtest.Row {
	t.Helper()
	i := slices.IndexFunc(rows, func(r worldtest.Row) bool { return r.ID == id })
	if i < 0 {
		t.Fatalf("listNodes has no row %s: %+v", id, rows)
	}
	return rows[i]
}

func TestARestartedNodeKeepsItsPort(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	box := api(t, run).Node("seedbox")
	restarted := act(t, run, "seedbox", world.KillNode, world.StartNode)
	if restarted.State != world.NodeRunning || restarted.PID == box.PID || restarted.Listeners["link"] != box.Listeners["link"] || restarted.Listeners["http"] != box.Listeners["http"] {
		t.Fatalf("seedbox restarted %s as PID %d on link %s and http %s; want running as a new PID on link %s and http %s",
			restarted.State, restarted.PID, restarted.Listeners["link"], restarted.Listeners["http"], box.Listeners["link"], box.Listeners["http"])
	}
	a := api(t, run)
	worldtest.Expect(t, "seedbox GET /api/v1/status", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"worker"}` + "\n"})
	worldtest.Until(t, "home lists seedbox's link up", func() (bool, error) {
		return row(t, a.Nodes("home"), box.NodeID).Link.State == "up", nil
	})
	doctor(t, run)
}

func TestWipedWorkerPairsAgain(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	box := api(t, run).Node("seedbox")
	if wiped := act(t, run, "seedbox", world.StopNode, world.WipeNode, world.StartNode); wiped.State != world.NodeRunning {
		t.Fatalf("seedbox is %s after its wipe and start, want running", wiped.State)
	}
	a := api(t, run)
	worldtest.Expect(t, "seedbox GET /api/v1/status after the wipe", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"unclaimed"}` + "\n"})

	id := a.Join("home", "seedbox", worldtest.SetupCode(t, run, "seedbox"))
	worldtest.Expect(t, "seedbox GET /api/v1/status after it joined", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"worker"}` + "\n"})
	rows := a.Nodes("home")
	again, was := row(t, rows, id), row(t, rows, box.NodeID)
	if len(rows) != 3 || id == box.NodeID || again.Name != "seedbox-2" || again.Link.State != "up" || len(again.Roles) != 0 || was.Link.State != "down" {
		t.Fatalf("listNodes on home after seedbox joined again as %s = %+v; want its wiped row down and a new row seedbox-2, up and holding no Roles", id, rows)
	}
}
