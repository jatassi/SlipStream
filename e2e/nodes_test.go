package e2e

import (
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	worldtest.Expect(t, "seedbox GET /api/v1/status", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"worker","passkeys":{"available":false,"reason":"ip_address"}}` + "\n"})
	worldtest.Until(t, "home lists seedbox's link up", func() (bool, error) {
		return row(t, a.Nodes("home"), box.NodeID).Link.State == "up", nil
	})
	doctor(t, run)
}

func TestANodeWhoseWebPortIsHeldFallsBack(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	a := api(t, run)
	box := a.Node("seedbox")
	configured := box.Listeners["http"]
	act(t, run, "seedbox", world.StopNode)
	held, err := new(net.ListenConfig).Listen(worldtest.Context(t), "tcp", configured)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Close() }()

	bound := act(t, run, "seedbox", world.StartNode).Listeners["http"]
	if port, was := netip.MustParseAddrPort(bound).Port(), netip.MustParseAddrPort(configured).Port(); port <= was || port >= was+10 {
		t.Fatalf("seedbox bound http %s with %s held, want one of the next 9 ports", bound, configured)
	}
	a = api(t, run)
	linkUp(t, a, box.NodeID, "")
	r := a.Admin("home", http.MethodGet, "/api/v1/nodes/"+box.NodeID+"/system", "")
	var report struct {
		HTTP struct {
			Configured string `json:"configured"`
		} `json:"http"`
		Listeners struct {
			HTTP string `json:"http"`
		} `json:"listeners"`
	}
	if err := json.Unmarshal([]byte(r.Body), &report); r.Status != http.StatusOK || err != nil || report.HTTP.Configured != configured || report.Listeners.HTTP != bound {
		t.Fatalf("home getNodeSystem seedbox = %d %s; want http.configured %s and listeners.http %s", r.Status, r.Body, configured, bound)
	}

	logged, err := os.ReadFile(filepath.Join(box.DataDir, "logs", "slipstream.log"))
	if err != nil {
		t.Fatal(err)
	}
	warn := `"level":"WARN","msg":"web port taken, serving on a later one","subsystem":"http","configured":"` + configured + `","bound":"` + bound + `"}`
	if !strings.Contains(string(logged), warn) {
		t.Fatalf("seedbox's slipstream.log holds no %s:\n%s", warn, logged)
	}
}

func TestAWipedWorkerPairsAgainAndTakesBackItsName(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	box := api(t, run).Node("seedbox")
	if wiped := act(t, run, "seedbox", world.StopNode, world.WipeNode, world.StartNode); wiped.State != world.NodeRunning {
		t.Fatalf("seedbox is %s after its wipe and start, want running", wiped.State)
	}
	a := api(t, run)
	worldtest.Expect(t, "seedbox GET /api/v1/status after the wipe", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"unclaimed","passkeys":{"available":false,"reason":"ip_address"}}` + "\n"})

	id := a.Join("home", "seedbox", worldtest.SetupCode(t, run, "seedbox"))
	worldtest.Expect(t, "seedbox GET /api/v1/status after it joined", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"worker","passkeys":{"available":false,"reason":"ip_address"}}` + "\n"})
	worldtest.Until(t, "home lists seedbox-2's link up", func() (bool, error) {
		return row(t, a.Nodes("home"), id).Link.State == "up", nil
	})
	rows := a.Nodes("home")
	again, was := row(t, rows, id), row(t, rows, box.NodeID)
	if len(rows) != 3 || id == box.NodeID || again.Name != "seedbox-2" || again.Link.State != "up" || len(again.Assigned()) != 0 || was.Link.State != "down" {
		t.Fatalf("listNodes on home after seedbox joined again as %s = %+v; want its wiped row down and a new row seedbox-2, up and holding no Roles", id, rows)
	}

	worldtest.Expect(t, "home DELETE seedbox's wiped row", a.Admin("home", http.MethodDelete, "/api/v1/nodes/"+box.NodeID, ""), worldtest.Reply{Status: 204})
	if renamed := a.Admin("home", http.MethodPatch, "/api/v1/nodes/"+id, `{"name":"seedbox"}`); renamed.Status != 200 {
		t.Fatalf("home PATCH seedbox-2 to seedbox = %d %s, want 200", renamed.Status, renamed.Body)
	}
	rows = a.Nodes("home")
	if again := row(t, rows, id); len(rows) != 2 || again.Name != "seedbox" || again.Link.State != "up" {
		t.Fatalf("listNodes on home after the wiped row left and seedbox-2 took its name = %+v; want home and seedbox %s, up", rows, id)
	}
}
