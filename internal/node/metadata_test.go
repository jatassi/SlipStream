package node

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/works"
)

// metadataCluster is home, the Controller, and box, a Worker holding the
// Metadata Role alone, both registering one Fixture.
type metadataCluster struct {
	home, box *Node
	hc        client
	apiKey    string
	fx        *contenttypetest.Fixture
	// homeOpen and boxOpen are each Node's provider half in fx.Opens.
	homeOpen, boxOpen int
}

// startMetadataCluster runs inside the caller's bubble.
func startMetadataCluster(t *testing.T) *metadataCluster {
	t.Helper()
	c := &metadataCluster{fx: contenttypetest.New()}
	c.fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{{Key: "b1", Label: "Winter"}}}})
	net := fault.NewNetwork()
	homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
	boxEnv, boxHost := machine(t, net, "127.0.0.3", "box")
	homeEnv.Types, boxEnv.Types = contenttype.NewSet(c.fx.Def()), contenttype.NewSet(c.fx.Def())
	var code string
	var bc client
	c.home, c.hc = startOn(t, homeEnv, homeHost)
	c.apiKey, code = claimedHome(t, c.home, c.hc)
	c.homeOpen = len(c.fx.Opens()) - 1
	c.box, bc = startOn(t, boxEnv, boxHost)
	joinBox(t, c.box, bc, boxEnv.DataDir, code, c.apiKey)
	synctest.Wait()
	c.boxOpen = len(c.fx.Opens()) - 1

	board := decode[struct {
		Basis string `json:"basis"`
		Nodes []struct {
			ID         string   `json:"id"`
			Controller bool     `json:"controller"`
			Assigned   []string `json:"assigned"`
		} `json:"nodes"`
	}](t, c.hc.admin(c.apiKey, http.MethodGet, "/api/v1/nodes/roles", ""), http.StatusOK)
	type assignment struct {
		NodeID string   `json:"nodeId"`
		Roles  []string `json:"roles"`
	}
	var set []assignment
	for _, n := range board.Nodes {
		roles := []string{"metadata"}
		if n.Controller {
			roles = slices.DeleteFunc(n.Assigned, func(r string) bool { return r == "metadata" })
		}
		set = append(set, assignment{NodeID: n.ID, Roles: roles})
	}
	body, err := json.Marshal(map[string]any{"basis": board.Basis, "set": set})
	if err != nil {
		t.Fatal(err)
	}
	decode[struct{}](t, c.hc.admin(c.apiKey, http.MethodPut, "/api/v1/nodes/roles", string(body)), http.StatusOK)
	synctest.Wait()
	return c
}

func (c *metadataCluster) search(t *testing.T) reply {
	t.Helper()
	return c.hc.admin(c.apiKey, http.MethodGet, "/api/v1/content-types/fixture/catalog?query=attic", "")
}

// lastServedBy is the provider half that answered the latest call.
func (c *metadataCluster) lastServedBy(t *testing.T) int {
	t.Helper()
	calls := c.fx.Calls()
	if len(calls) == 0 {
		t.Fatal("no provider half answered a call")
	}
	return calls[len(calls)-1].Open
}

func TestAWorkerOpensOnlyProviderHalves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		if opens, libraries := len(c.fx.Opens()), len(c.fx.Libraries()); opens != 2 || libraries != 1 || c.boxOpen == c.homeOpen {
			t.Fatalf("home and box opened %d provider halves and %d library halves, want one provider half each and the Controller's library half", opens, libraries)
		}
		if o := c.fx.Opens()[c.boxOpen]; o.Deps.Lines == nil || o.Deps.TMDB == nil || o.Deps.OMDb == nil {
			t.Fatalf("box opened its provider half with %+v, want its own call layer and clients", o.Deps)
		}
	})
}

func TestTheCallsLeaveFromTheWorkerHoldingMetadata(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		if r := c.search(t); r.status != http.StatusOK || !strings.Contains(r.body, `"label":"The Attic"`) {
			t.Fatalf("searchCatalog = %d %s, want The Attic", r.status, r.body)
		}
		if got := c.lastServedBy(t); got != c.boxOpen {
			t.Fatalf("the search was answered by provider half %d, want box's %d", got, c.boxOpen)
		}
		if r := c.hc.admin(c.apiKey, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`); r.status != http.StatusOK {
			t.Fatalf("addWork = %d %s", r.status, r.body)
		}
		if got := c.lastServedBy(t); got != c.boxOpen {
			t.Fatalf("the add's fetch was answered by provider half %d, want box's %d", got, c.boxOpen)
		}
	})
}

func TestInsideTheGraceMetadataIsUnavailableWithWhy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		c.box.Stop()
		waitDone(t, c.box)
		time.Sleep(2 * time.Minute)
		calls := len(c.fx.Calls())
		r := c.search(t)
		var p struct {
			Code   string `json:"code"`
			Detail string `json:"detail"`
		}
		if err := json.Unmarshal([]byte(r.body), &p); err != nil || r.status != http.StatusBadGateway || p.Code != "metadata_unavailable" ||
			!strings.HasPrefix(p.Detail, "Metadata: box is offline since ") || !strings.HasSuffix(p.Detail, ", and no other Node reads metadata") {
			t.Fatalf("searchCatalog = %d %s, want metadata_unavailable saying box is offline", r.status, r.body)
		}
		if len(c.fx.Calls()) != calls {
			t.Fatal("a provider half answered while no Node may read metadata")
		}
	})
}

func TestPastTheGraceTheControllerCovers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		c.box.Stop()
		waitDone(t, c.box)
		time.Sleep(16 * time.Minute)
		if r := c.search(t); r.status != http.StatusOK || !strings.Contains(r.body, `"label":"The Attic"`) {
			t.Fatalf("searchCatalog = %d %s, want The Attic", r.status, r.body)
		}
		if got := c.lastServedBy(t); got != c.homeOpen {
			t.Fatalf("the search was answered by provider half %d, want home's %d", got, c.homeOpen)
		}
	})
}
