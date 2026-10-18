package sim_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/pairing"
)

// unreachable is where home advertises its link, so a Join Code names an
// address no Node can dial, as behind carrier-grade NAT.
var unreachable = func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_LINK_ADVERTISE=127.0.0.250:8485") }

// nodePort is the port slipstream join --node-port pins seedbox's link to,
// so the address home dials holds across restarts.
const nodePort = 8485

// redialMax is timings.link.redial_max: home dials a Node that wasn't
// running again within it.
const redialMax = time.Minute

func cgnatHome(t *testing.T) *sim.World {
	t.Helper()
	s, err := scenario.Load("cgnat-home")
	if err != nil {
		t.Fatal(err)
	}
	w := sim.Launch(t, s)
	h := w.Node("home")
	h.Stop()
	h.Start(unreachable)
	return w
}

// TestAControllerDialsANodeBehindCGNAT runs slipstream join on the stopped
// seedbox with a code that names no address it can reach. The Join prints
// seedbox's Node address and waits; dialForCode on home pairs it over
// home's session. home records the address and dials seedbox from then
// on, across a restart of both Nodes.
func TestAControllerDialsANodeBehindCGNAT(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := cgnatHome(t)
		a := w.API()
		box := w.Node("seedbox")
		box.Stop()
		minted := a.MintJoinCode("home")
		code, err := joincode.Parse(minted.Code)
		if err != nil {
			t.Fatal(err)
		}

		pending := make(chan pairing.Pending, 1)
		joined := make(chan error, 1)
		go func() {
			_, err := box.Join(t.Context(), code, nodePort, func(p pairing.Pending) { pending <- p })
			joined <- err
		}()
		var p pairing.Pending
		select {
		case p = <-pending:
		case err := <-joined:
			t.Fatalf("the Join = %v before it awaited home's dial", err)
		}
		if want := []string{"Can't reach the Controller at 127.0.0.250:8485."}; p.State != pairing.AwaitingDial || !slices.Equal(p.Lines()[:1], want) {
			t.Fatalf("the Join awaits as %v saying %q, want awaiting-dial saying %q", p.State, p.Lines(), want)
		}
		worldtest.Expect(t, "dialForCode", a.DialForCode("home", minted.ID, p.Address.String()), worldtest.Reply{Status: http.StatusAccepted, Body: ""})
		if err := <-joined; err != nil {
			t.Fatalf("the Join = %v, want joined over home's dial", err)
		}

		box.Start()
		time.Sleep(redialMax)
		dialed := []string{box.Ready().Listeners.Link.String()}
		if got := p.Address.Addresses(); len(got) != 1 || got[0].String() != dialed[0] {
			t.Fatalf("the Node address names %v, want seedbox's link at %s", got, dialed[0])
		}
		checkDialed(t, a, dialed)
		h := w.Node("home")
		h.Stop()
		box.Stop()
		h.Start(unreachable)
		box.Start()
		time.Sleep(redialMax)
		checkDialed(t, w.API(), dialed)
	})
}

// TestASetupJoinAwaitsTheDial joins the running seedbox from its first-run
// page. claimWorker answers 202 with the Node address, a Controller claim
// meanwhile answers join_pending, and once home dials, getWorkerClaim
// reads the Join as joined and the topology draws the link from home.
func TestASetupJoinAwaitsTheDial(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := cgnatHome(t)
		a := w.API()
		minted := a.MintJoinCode("home")
		setupCode := w.Node("seedbox").Ready().SetupCode
		body := `{"code":"` + minted.Code + `"}`
		reply := a.Setup("seedbox", setupCode, http.MethodPost, "/api/v1/setup/worker", body)
		var claim struct {
			State       string `json:"state"`
			NodeAddress string `json:"nodeAddress"`
		}
		answered(t, reply, http.StatusAccepted, &claim)
		if claim.State != "awaiting-dial" || claim.NodeAddress == "" {
			t.Fatalf("claimWorker = %s, want awaiting-dial with a Node address", reply.Body)
		}
		if r := a.Setup("seedbox", setupCode, http.MethodPost, "/api/v1/setup/controller", `{"email":"olu@example.test","displayName":"Olu Bello","pin":"1357"}`); r.Status != http.StatusConflict || !strings.Contains(r.Body, `"code":"join_pending"`) {
			t.Fatalf("claimController while the Join awaits = %d %s, want 409 join_pending", r.Status, r.Body)
		}

		worldtest.Expect(t, "dialForCode", a.DialForCode("home", minted.ID, claim.NodeAddress), worldtest.Reply{Status: http.StatusAccepted, Body: ""})
		synctest.Wait()
		answered(t, a.Setup("seedbox", setupCode, http.MethodGet, "/api/v1/setup/worker", ""), http.StatusOK, &claim)
		if claim.State != "joined" {
			t.Fatalf("getWorkerClaim reads the Join as %s, want joined", claim.State)
		}
		box := seedbox(t, a)
		if box.Link.State != "up" || box.Dial.By != "controller" {
			t.Fatalf("home lists seedbox %s dialed by %q, want up and dialed by the controller", box.Link.State, box.Dial.By)
		}
		topo := decode[topologyView](t, "getTopology", a.Admin("home", http.MethodGet, "/api/v1/nodes/topology", ""))
		if l := topo.Links; len(l) != 1 || l[0].B != box.ID || l[0].Dialer != topo.Nodes[0].ID || l[0].State != "up" {
			t.Fatalf("the topology's links are %+v, want seedbox's link up and dialed by home", l)
		}
	})
}

func answered(t *testing.T, r worldtest.Reply, status int, v any) {
	t.Helper()
	if r.Status != status {
		t.Fatalf("got %d %s, want %d", r.Status, r.Body, status)
	}
	if err := json.Unmarshal([]byte(r.Body), v); err != nil {
		t.Fatalf("%s: %v", r.Body, err)
	}
}

// checkDialed checks home lists seedbox up on the session home dialed at
// addresses.
func checkDialed(t *testing.T, a *worldtest.API, addresses []string) {
	t.Helper()
	box := seedbox(t, a)
	if box.Link.State != "up" || box.Dial.By != "controller" || !slices.Equal(box.Dial.ControllerDials, addresses) {
		t.Fatalf("home lists seedbox %s dialed by %q at %v, want up, dialed by the controller at %v", box.Link.State, box.Dial.By, box.Dial.ControllerDials, addresses)
	}
}
