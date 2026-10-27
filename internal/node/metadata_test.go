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
	boxID             string
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
		} else {
			c.boxID = n.ID
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

// TestAProviderTestLeavesFromTheWorker tests TMDB, whose real host no Node
// in this network reaches: box makes the call, and status names box.
func TestAProviderTestLeavesFromTheWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		status := decode[struct {
			State   string `json:"state"`
			Details struct {
				Nodes []struct {
					NodeID  string `json:"nodeId"`
					Reached bool   `json:"reached"`
				} `json:"nodes"`
			} `json:"details"`
		}](t, c.hc.admin(c.apiKey, http.MethodPost, "/api/v1/metadata/providers/tmdb/test", ""), http.StatusOK)
		if n := status.Details.Nodes; status.State != "unreachable" || len(n) != 1 || n[0].NodeID != c.boxID || n[0].Reached {
			t.Fatalf("testMetadataProvider = %+v, want TMDB unreachable from box %s alone", status, c.boxID)
		}
	})
}

type refreshRun struct {
	Result  string `json:"result"`
	Summary string `json:"summary"`
	Units   []struct {
		Label   string `json:"label"`
		Node    string `json:"node"`
		Outcome string `json:"outcome"`
		Cause   string `json:"cause"`
	} `json:"units"`
}

// refreshRuns is Metadata refresh's runs, newest first.
func (c *metadataCluster) refreshRuns(t *testing.T) []refreshRun {
	t.Helper()
	return decode[struct {
		Runs []refreshRun `json:"runs"`
	}](t, c.hc.admin(c.apiKey, http.MethodGet, "/api/v1/tasks/metadata-refresh", ""), http.StatusOK).Runs
}

func (c *metadataCluster) refreshAll(t *testing.T) reply {
	t.Helper()
	return c.hc.admin(c.apiKey, http.MethodPost, "/api/v1/content-types/fixture/refresh", "")
}

func (c *metadataCluster) add(t *testing.T, key string) {
	t.Helper()
	if r := c.hc.admin(c.apiKey, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"`+key+`"}`); r.status != http.StatusOK {
		t.Fatalf("addWork %s = %d %s", key, r.status, r.body)
	}
}

func TestARefreshAllRunsOnTheWorkerHoldingMetadata(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		c.add(t, "w1")
		time.Sleep(time.Minute)
		started := decode[struct {
			RunID int64 `json:"runId"`
		}](t, c.refreshAll(t), http.StatusAccepted)
		synctest.Wait()
		runs := c.refreshRuns(t)
		if started.RunID == 0 || len(runs) != 1 || runs[0].Summary != "1 Work" || len(runs[0].Units) != 1 || runs[0].Units[0].Node != "box" || runs[0].Units[0].Outcome != "succeeded" {
			t.Fatalf("Metadata refresh ran %+v after refreshContentType, want one Work refreshed on box", runs)
		}
		if got := c.lastServedBy(t); got != c.boxOpen {
			t.Fatalf("the refresh was answered by provider half %d, want box's %d", got, c.boxOpen)
		}
		if r := c.hc.admin(c.apiKey, http.MethodPost, "/api/v1/content-types/nothing/refresh", ""); r.status != http.StatusNotFound || !strings.Contains(r.body, `"code":"content_type_not_found"`) {
			t.Fatalf("refreshContentType of an unknown Content Type = %d %s, want 404 content_type_not_found", r.status, r.body)
		}
	})
}

func TestTheRefreshAllRunNamesWhenItRanAndWhetherItIsOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		c.add(t, "w1")
		time.Sleep(time.Minute)
		type state struct {
			LastRunAt *time.Time `json:"lastRunAt"`
			Running   bool       `json:"running"`
		}
		read := func() state {
			t.Helper()
			return decode[state](t, c.hc.admin(c.apiKey, http.MethodGet, "/api/v1/content-types/fixture/refresh", ""), http.StatusOK)
		}
		if s := read(); s.LastRunAt != nil || s.Running {
			t.Fatalf("getContentTypeRefresh before any Refresh all = %+v, want none run and none open", s)
		}
		letGo := c.fx.Hold(1)
		asked := time.Now()
		decode[struct{}](t, c.refreshAll(t), http.StatusAccepted)
		synctest.Wait()
		if s := read(); s.LastRunAt == nil || !s.LastRunAt.Equal(asked) || !s.Running {
			t.Fatalf("getContentTypeRefresh during the run = %+v, want run at %v and running", s, asked)
		}
		letGo()
		synctest.Wait()
		if s := read(); s.LastRunAt == nil || !s.LastRunAt.Equal(asked) || s.Running {
			t.Fatalf("getContentTypeRefresh after the run = %+v, want run at %v and not running", s, asked)
		}
		if r := c.hc.admin(c.apiKey, http.MethodGet, "/api/v1/content-types/nothing/refresh", ""); r.status != http.StatusNotFound || !strings.Contains(r.body, `"code":"content_type_not_found"`) {
			t.Fatalf("getContentTypeRefresh of an unknown Content Type = %d %s, want 404 content_type_not_found", r.status, r.body)
		}
	})
}

func TestAUnitWhoseHolderIsAwayIsSkippedNotFailed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		c.add(t, "w1")
		c.box.Stop()
		waitDone(t, c.box)
		time.Sleep(2 * time.Minute)
		calls := len(c.fx.Calls())
		decode[struct{}](t, c.refreshAll(t), http.StatusAccepted)
		synctest.Wait()
		runs := c.refreshRuns(t)
		if len(runs) != 1 || runs[0].Result != "skipped" || len(runs[0].Units) != 1 ||
			runs[0].Units[0].Outcome != "skipped" || runs[0].Units[0].Cause != "home no longer holds Metadata" {
			t.Fatalf("Metadata refresh ran %+v while box was away, want the Work skipped because home no longer holds Metadata", runs)
		}
		if len(c.fx.Calls()) != calls {
			t.Fatal("a provider half answered while box, the only Node holding Metadata, was away")
		}
	})
}

func TestRefreshAllDuringARunKeepsTheRequestAndAnswersTaskRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := startMetadataCluster(t)
		c.fx.Put("w2", contenttypetest.Record{Tree: works.Fetched{Label: "The Cellar", Children: []works.Entry{{Key: "b1", Label: "Spring"}}}})
		c.add(t, "w1")
		c.add(t, "w2")
		time.Sleep(time.Minute)
		c.fx.Fail("w2", works.ErrUnavailable)
		decode[struct{}](t, c.refreshAll(t), http.StatusAccepted)
		synctest.Wait()
		time.Sleep(time.Minute)

		c.fx.Fail("w2", nil)
		letGo := c.fx.Hold(1)
		decode[struct{}](t, c.hc.admin(c.apiKey, http.MethodPost, "/api/v1/tasks/metadata-refresh/runs", ""), http.StatusAccepted)
		synctest.Wait()
		if r := c.refreshAll(t); r.status != http.StatusConflict || !strings.Contains(r.body, `"code":"task_running"`) {
			t.Fatalf("refreshContentType during a run = %d %s, want 409 task_running", r.status, r.body)
		}
		letGo()
		synctest.Wait()
		if runs := c.refreshRuns(t); len(runs) != 2 || runs[0].Summary != "1 Work" || runs[0].Units[0].Label != "The Cellar" {
			t.Fatalf("the run open during the request ran %+v, want only The Cellar, which failed before", runs)
		}

		decode[struct{}](t, c.hc.admin(c.apiKey, http.MethodPost, "/api/v1/tasks/metadata-refresh/runs", ""), http.StatusAccepted)
		synctest.Wait()
		if runs := c.refreshRuns(t); len(runs) != 3 || runs[0].Summary != "1 Work" || runs[0].Units[0].Label != "The Attic" {
			t.Fatalf("the next run ran %+v, want The Attic, refreshed before the kept request", runs)
		}
	})
}
