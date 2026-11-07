package e2e

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/challengesolver"
	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// solves are the hosts fake-challenge-solver cleared, as "<node> <host>".
func solves(t *testing.T, m world.Manifest) []string {
	t.Helper()
	recs, err := events.Select(m.RunDir, events.Query{Kind: challengesolver.Solved.Kind})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range recs {
		var s challengesolver.Solve
		if err := json.Unmarshal(r.Data, &s); err != nil {
			t.Fatal(err)
		}
		out = append(out, r.Node+" "+s.Host)
	}
	return out
}

// TestCindersChallengeClearsFromANodeWithASolverAndStopsOnOneWithout adds
// Cinder at setup from seedbox, whose Challenge solver clears its
// challenge, so the Admin pasted only Cinder's session cookie. Once den
// holds Searching, the same cookie from den, which has no solver, stops at
// the challenge. Setting den's solver refuses while den can't reach it,
// and saves once it can.
func TestCindersChallengeClearsFromANodeWithASolverAndStopsOnOneWithout(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("indexers")
	if err != nil {
		t.Fatal(err)
	}
	run := worldtest.Launch(t, "indexers")
	m := run.Manifest()
	a := api(t, run)
	ids := map[string]string{}
	for _, n := range m.Nodes {
		ids[n.Name] = n.NodeID
	}
	a.SetRoles("home", ids["den"], "searching")
	front := func(node string) string {
		i := slices.IndexFunc(m.Fronts, func(f world.Front) bool { return f.Target == challengesolver.Fake.Name && f.Node == node })
		if i < 0 {
			t.Fatalf("the manifest lists no %s front for %s", challengesolver.Fake.Name, node)
		}
		return m.Fronts[i].URL
	}
	cinderHost := site.Host(site.Cinder)

	var list struct {
		Indexers []homedIndexer `json:"indexers"`
	}
	decode(t, "listIndexers", a.Admin("home", http.MethodGet, "/api/v1/indexers", ""), &list)
	i := slices.IndexFunc(list.Indexers, func(ix homedIndexer) bool { return ix.Name == "Cinder" })
	if i < 0 || list.Indexers[i].HomeNodeID != ids["seedbox"] || list.Indexers[i].Session == nil || list.Indexers[i].Session.NodeID != ids["seedbox"] {
		t.Fatalf("listIndexers = %+v, want Cinder homed on seedbox holding a session there", list.Indexers)
	}
	cinder := list.Indexers[i].ID
	if got := solves(t, m); len(got) == 0 || slices.ContainsFunc(got, func(s string) bool { return s != "seedbox "+cinderHost }) {
		t.Fatalf("fake-challenge-solver cleared %q, want Cinder's host for seedbox only", got)
	}
	passes(t, "testIndexer Cinder", a.Admin("home", http.MethodPost, "/api/v1/indexers/"+cinder+"/test", "{}"))

	worldtest.Expect(t, "getChallengeSolver seedbox", a.Admin("home", http.MethodGet, "/api/v1/nodes/"+ids["seedbox"]+"/challenge-solver", ""), worldtest.Reply{
		Status: http.StatusOK, Body: `{"address":"` + front("seedbox") + `","timeout":60}` + "\n",
	})
	worldtest.Expect(t, "getChallengeSolver den", a.Admin("home", http.MethodGet, "/api/v1/nodes/"+ids["den"]+"/challenge-solver", ""), worldtest.Reply{
		Status: http.StatusOK, Body: `{"address":null,"timeout":60}` + "\n",
	})

	draft, err := json.Marshal(map[string]any{
		"kind": "catalog", "name": "Cinder from den", "definition": site.Cinder, "homeNodeId": ids["den"],
		"cookie": map[string]string{"cookie": site.SessionCookie(s), "userAgent": site.BrowserUserAgent},
	})
	if err != nil {
		t.Fatal(err)
	}
	reply := a.Admin("home", http.MethodPost, "/api/v1/indexers/test", string(draft))
	if want := "Cinder from den is behind a browser challenge, and den has no Challenge solver. Set one on den, or paste a browser cookie."; reply.Status != http.StatusUnprocessableEntity ||
		!strings.Contains(reply.Body, `"code":"indexer_test_failed"`) || !strings.Contains(reply.Body, want) {
		t.Fatalf("testIndexerDraft from den = %d %s, want a failed Test saying %q", reply.Status, reply.Body, want)
	}

	setDen := `{"address":"` + front("den") + `","timeout":30}`
	refuse, err := fault.Parse("refuse")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.SetFault(worldtest.Context(t), fault.Rule{ID: "den-solver-down", Target: fault.Target(challengesolver.Fake.Name), Origin: "den", Fault: refuse}); err != nil {
		t.Fatal(err)
	}
	worldtest.Expect(t, "setChallengeSolver den while it can't reach the solver", a.Admin("home", http.MethodPut, "/api/v1/nodes/"+ids["den"]+"/challenge-solver", setDen), worldtest.Reply{
		Status: http.StatusUnprocessableEntity,
		Body: `{"status":422,"code":"challenge_solver_unreachable","title":"The Node can't reach the Challenge solver","detail":"den can't reach a Challenge solver at ` +
			front("den") + `. Check the address and that the solver is running beside den."}` + "\n",
	})
	worldtest.Expect(t, "getChallengeSolver den after the refusal", a.Admin("home", http.MethodGet, "/api/v1/nodes/"+ids["den"]+"/challenge-solver", ""), worldtest.Reply{
		Status: http.StatusOK, Body: `{"address":null,"timeout":60}` + "\n",
	})
	if _, err := run.ClearFault(worldtest.Context(t), "den-solver-down"); err != nil {
		t.Fatal(err)
	}
	worldtest.Expect(t, "setChallengeSolver den", a.Admin("home", http.MethodPut, "/api/v1/nodes/"+ids["den"]+"/challenge-solver", setDen), worldtest.Reply{
		Status: http.StatusOK, Body: `{"address":"` + front("den") + `","timeout":30}` + "\n",
	})
	passes(t, "testIndexerDraft from den with its solver", a.Admin("home", http.MethodPost, "/api/v1/indexers/test", string(draft)))
	if got := solves(t, m); !slices.Contains(got, "den "+cinderHost) {
		t.Fatalf("fake-challenge-solver cleared %q, want Cinder's host for den too", got)
	}
}

type homedIndexer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	HomeNodeID string `json:"homeNodeId"`
	Session    *struct {
		NodeID string `json:"nodeId"`
	} `json:"session"`
}

func passes(t *testing.T, what string, r worldtest.Reply) {
	t.Helper()
	if r.Status != http.StatusOK {
		t.Fatalf("%s = %d %s, want 200", what, r.Status, r.Body)
	}
}
