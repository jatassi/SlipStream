package e2e

import (
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type healthItem struct {
	Key     string `json:"key"`
	Level   string `json:"level"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	Since   string `json:"since"`
	Action  *struct {
		Kind string `json:"kind"`
		Href string `json:"href"`
	} `json:"action"`
}

type healthReport struct {
	Level      *string `json:"level"`
	Categories []struct {
		ID    string       `json:"id"`
		Items []healthItem `json:"items"`
	} `json:"categories"`
}

func (r healthReport) items(t *testing.T, id string) map[string]healthItem {
	t.Helper()
	for _, c := range r.Categories {
		if c.ID == id {
			out := map[string]healthItem{}
			for _, it := range c.Items {
				out[it.Key] = it
			}
			return out
		}
	}
	t.Fatalf("the Health report has no category %s", id)
	return nil
}

func healthItems(t *testing.T, a *worldtest.API, id string) map[string]healthItem {
	t.Helper()
	return decode(t, "getSystemHealth", a.Admin("home", http.MethodGet, "/api/v1/system/health", ""), &healthReport{}).items(t, id)
}

type healthChecked struct {
	Report    healthReport `json:"report"`
	Unreached []struct {
		Key   string `json:"key"`
		Words string `json:"words"`
	} `json:"unreached"`
}

func healthAlerts(t *testing.T, a *worldtest.API) (health, nodes *string) {
	t.Helper()
	var c struct {
		Alerts struct {
			Health *string `json:"health"`
			Nodes  *string `json:"nodes"`
		} `json:"alerts"`
	}
	decode(t, "getCounts", a.Admin("home", http.MethodGet, "/api/v1/counts", ""), &c)
	return c.Alerts.Health, c.Alerts.Nodes
}

// TestANodeOfflinePastTheGraceIsANodesErrorUntilItIsBack stops seedbox
// with the offline grace at its 1 min floor. Past the grace it is a
// Nodes error linking to its page, home covering its Searching is a
// warning, the counts' Health alert fails and health.changed names nodes.
// Check now on the offline seedbox answers it unreached and leaves its
// item as it was. Once seedbox is back and settled, the items clear.
func TestANodeOfflinePastTheGraceIsANodesErrorUntilItIsBack(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	a := api(t, run)
	stream := worldtest.NewMessages(t, run.Manifest(), world.AdminIdentity)
	if r := a.Admin("home", http.MethodPatch, "/api/v1/nodes/settings", `{"offlineGrace":60}`); r.Status != http.StatusOK {
		t.Fatalf("updateNodeSettings = %d %s, want 200", r.Status, r.Body)
	}
	box := a.Node("seedbox").NodeID
	worldtest.Until(t, "the Nodes category is clear", func() (bool, error) { return len(healthItems(t, a, "nodes")) == 0, nil })
	checked := decode(t, "checkHealthCategory", a.Admin("home", http.MethodPost, "/api/v1/system/health/nodes/check", ""), &healthChecked{})
	if len(checked.Unreached) != 0 || len(checked.Report.items(t, "nodes")) != 0 {
		t.Fatalf("Check now on Nodes in step = %+v, want nothing unreached and no items", checked)
	}
	if indexers := checked.Report.items(t, "indexers"); len(indexers) != 0 {
		t.Fatalf("the Indexers category holds %+v, want nothing while Fixture Site works", indexers)
	}
	if health, _ := healthAlerts(t, a); health != nil {
		t.Fatalf("with every Node in step the Health alert is %s, want null", *health)
	}

	act(t, run, "seedbox", world.StopNode)
	worldtest.Until(t, "seedbox is a Nodes error", func() (bool, error) { return healthItems(t, a, "nodes")[box].Level == "fail", nil })
	items := healthItems(t, a, "nodes")
	lapsed := items[box]
	if lapsed.Subject != "seedbox" || lapsed.Action == nil || lapsed.Action.Kind != "link" || lapsed.Action.Href != "/nodes/"+box {
		t.Fatalf("seedbox's item is %+v, want subject seedbox and a link to /nodes/%s", lapsed, box)
	}
	if covering := items["covering.searching"]; covering.Level != "warn" || covering.Subject != "Searching on home" {
		t.Fatalf("home covering Searching is %+v, want a warning on Searching on home", covering)
	}
	for data := stream.Next("health.changed"); data != `{"categories":["nodes"]}`; {
		data = stream.Next("health.changed")
	}
	if health, nodes := healthAlerts(t, a); health == nil || *health != "fail" || nodes == nil || *nodes != "fail" {
		t.Fatalf("past the grace the counts' alerts are health %v and nodes %v, want fail and fail", health, nodes)
	}

	checked = decode(t, "checkHealthCategory", a.Admin("home", http.MethodPost, "/api/v1/system/health/nodes/check", ""), &healthChecked{})
	if len(checked.Unreached) != 1 || checked.Unreached[0].Words != "seedbox is offline." {
		t.Fatalf("Check now with seedbox offline left %+v unreached, want seedbox is offline.", checked.Unreached)
	}
	if after := checked.Report.items(t, "nodes")[box]; after.Level != "fail" || after.Since != lapsed.Since {
		t.Fatalf("after Check now seedbox's item is %+v, want it unchanged from %+v", after, lapsed)
	}
	worldtest.Expect(t, "checkHealthItem on a linked item", a.Admin("home", http.MethodPost, "/api/v1/system/health/nodes/items/"+box+"/check", ""),
		problem(http.StatusNotFound, "no_check", "There is no Check now for that"))
	worldtest.Expect(t, "checkHealthItem on no item", a.Admin("home", http.MethodPost, "/api/v1/system/health/nodes/items/nothing/check", ""),
		problem(http.StatusNotFound, "health_item_not_found", "The category has no item with that key"))
	worldtest.Expect(t, "checkHealthCategory on no category", a.Admin("home", http.MethodPost, "/api/v1/system/health/widgets/check", ""),
		problem(http.StatusNotFound, "health_category_not_found", "No Health category has that id"))

	act(t, run, "seedbox", world.StartNode)
	worldtest.Until(t, "the Nodes items clear once seedbox is back and settled", func() (bool, error) { return len(healthItems(t, a, "nodes")) == 0, nil })
	if health, _ := healthAlerts(t, a); health != nil {
		t.Fatalf("with seedbox back the Health alert is %s, want null", *health)
	}
}
