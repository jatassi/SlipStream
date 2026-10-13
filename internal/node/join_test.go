package node

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/system"
)

// machine is one Node's env on a shared in-memory network, at addr.
func machine(t *testing.T, net *fault.Network, addr, name string) (Env, *fault.Host) {
	t.Helper()
	env, _ := testEnv(t)
	env.Environ = []string{
		"SLIPSTREAM_TORRENT_LISTEN=" + addr + ":0",
		"SLIPSTREAM_HTTP_LISTEN=" + addr + ":0",
		"SLIPSTREAM_LINK_LISTEN=" + addr + ":0",
		"SLIPSTREAM_NAME=" + name,
	}
	host := net.Host(netip.MustParseAddr(addr))
	return inMemoryOn(env, host), host
}

type nodeRow struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Self          bool     `json:"self"`
	Roles         []string `json:"roles"`
	IntentVersion string   `json:"intentVersion"`
	Link          struct {
		State string `json:"state"`
		Build *struct {
			Version string `json:"version"`
		} `json:"build"`
	} `json:"link"`
	Facts *struct {
		Assigned       []string `json:"assigned"`
		AppliedVersion string   `json:"appliedVersion"`
	} `json:"facts"`
}

func decode[T any](t *testing.T, r reply, status int) T {
	t.Helper()
	var v T
	if r.status != status {
		t.Fatalf("got %d %s, want %d", r.status, r.body, status)
	}
	if err := json.Unmarshal([]byte(r.body), &v); err != nil {
		t.Fatalf("%s: %v", r.body, err)
	}
	return v
}

func (c client) admin(apiKey, method, path, body string) reply {
	c.t.Helper()
	k := call{method: method, path: path, headers: map[string]string{"X-Api-Key": apiKey}}
	if body != "" {
		k.body = strings.NewReader(body)
	}
	return c.do(k)
}

// claimedHome claims home and mints a Join Code with its API key.
func claimedHome(t *testing.T, home *Node, hc client) (apiKey, code string) {
	t.Helper()
	claimed := decode[struct {
		APIKey string `json:"apiKey"`
	}](t, hc.claim(home.Ready().SetupCode, "1234"), http.StatusCreated)
	minted := decode[struct {
		Code string `json:"code"`
	}](t, hc.admin(claimed.APIKey, http.MethodPost, "/api/v1/nodes/join-codes", ""), http.StatusCreated)
	parsed, err := joincode.Parse(minted.Code)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := parsed.Addresses(), []netip.AddrPort{home.Ready().Listeners.Link}; !slices.Equal(got, want) {
		t.Fatalf("the Join Code names %v, want the link listener %v", got, want)
	}
	return claimed.APIKey, minted.Code
}

type joinedWorker struct {
	NodeID     string `json:"nodeId"`
	Controller struct {
		Name string `json:"name"`
		Key  string `json:"key"`
		Link string `json:"link"`
	} `json:"controller"`
}

// joinBox joins box through its Setup gate and checks it is a Worker
// holding only local.db.
func joinBox(t *testing.T, box *Node, bc client, boxDir DataDir, code, apiKey string) joinedWorker {
	t.Helper()
	join := func() reply {
		return bc.do(call{method: http.MethodPost, path: "/api/v1/setup/worker", body: strings.NewReader(`{"code":"` + code + `"}`), headers: map[string]string{"X-Setup-Code": box.Ready().SetupCode}})
	}
	joined := decode[joinedWorker](t, join(), http.StatusCreated)
	expect(t, join(), 409, problem(409, "already_claimed", "This Node is already set up"))
	expect(t, bc.do(call{method: http.MethodGet, path: "/api/v1/status"}), 200, `{"claim":"worker"}`+"\n")
	expect(t, bc.admin(apiKey, http.MethodGet, "/api/v1/system", ""), 409, problem(409, "worker_node", "This Node is a Worker. Manage it from its Controller."))
	if !loggedClaimed(t, boxDir, "worker") {
		t.Fatal("the Worker logged no claimed record with claim worker")
	}
	holds := dataDirHolds(t, boxDir)
	if !slices.Contains(holds, "node.key") || !slices.Contains(holds, "pairing.json") || !slices.Contains(holds, "local.db") || slices.Contains(holds, "library.db") {
		t.Fatalf("the Worker's data dir holds %v, want node.key, pairing.json and local.db and no library.db", holds)
	}
	return joined
}

// listed is listNodes on hc as home's row and box's.
func listed(t *testing.T, hc client, apiKey string) (self, worker nodeRow) {
	t.Helper()
	list := decode[struct {
		Nodes []nodeRow `json:"nodes"`
	}](t, hc.admin(apiKey, http.MethodGet, "/api/v1/nodes", ""), http.StatusOK).Nodes
	if len(list) != 2 {
		t.Fatalf("listNodes has %d rows, want home and box", len(list))
	}
	if list[0].Self {
		return list[0], list[1]
	}
	return list[1], list[0]
}

// TestAWorkerJoinsThroughTheSetupGate runs a Controller and an Unclaimed
// Node on one in-memory network: the Unclaimed one joins with a Join Code
// through claimWorker, listNodes shows it up, and setNodeRoles answers once
// it has the new Roles.
func TestAWorkerJoinsThroughTheSetupGate(t *testing.T) {
	net := fault.NewNetwork()
	homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
	boxEnv, boxHost := machine(t, net, "127.0.0.3", "box")
	synctest.Test(t, func(t *testing.T) {
		home, hc := startOn(t, homeEnv, homeHost)
		apiKey, code := claimedHome(t, home, hc)
		box, bc := startOn(t, boxEnv, boxHost)
		joined := joinBox(t, box, bc, boxEnv.DataDir, code, apiKey)
		if joined.Controller.Name != "home" || joined.Controller.Key != home.Ready().NodeKey || joined.Controller.Link != home.Ready().Listeners.Link.String() {
			t.Fatalf("claimWorker answered controller %+v, want home, %s, %s", joined.Controller, home.Ready().NodeKey, home.Ready().Listeners.Link)
		}

		synctest.Wait()
		self, worker := listed(t, hc, apiKey)
		if self.Name != "home" || self.Link.State != "local" || worker.Name != "box" || worker.ID != joined.NodeID || worker.Link.State != "up" {
			t.Fatalf("listNodes rows: %s %s, %s %s %s; want home local and box up as %s", self.Name, self.Link.State, worker.Name, worker.ID, worker.Link.State, joined.NodeID)
		}
		if worker.Link.Build == nil || self.Link.Build == nil || worker.Link.Build.Version != self.Link.Build.Version {
			t.Fatalf("box's build is not home's: %+v, %+v", worker.Link.Build, self.Link.Build)
		}

		set := decode[nodeRow](t, hc.admin(apiKey, http.MethodPut, "/api/v1/nodes/"+worker.ID+"/roles", `{"roles":["searching"]}`), http.StatusOK)
		if set.Link.State != "up" || set.Facts == nil || !slices.Equal(set.Facts.Assigned, []string{"searching"}) || set.Facts.AppliedVersion != set.IntentVersion {
			t.Fatalf("setNodeRoles answered %+v with facts %+v, want up with searching applied at %s", set.Link, set.Facts, set.IntentVersion)
		}
		if box.Ready().Claim != system.Unclaimed {
			t.Fatalf("box was ready as %s, want unclaimed", box.Ready().Claim)
		}
	})
}
