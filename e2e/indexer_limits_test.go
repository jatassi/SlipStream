package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type limitUse struct {
	Count  int     `json:"count"`
	Per    string  `json:"per"`
	Used   int     `json:"used"`
	FreeAt *string `json:"freeAt"`
}

// standing is what the limit and backoff tests read of an Indexer.
type standing struct {
	Searches limitUse `json:"searches"`
	Grabs    limitUse `json:"grabs"`
	Status   struct {
		State string  `json:"state"`
		Rung  int     `json:"rung"`
		Until *string `json:"until"`
	} `json:"status"`
}

func (w *walked) standing(t *testing.T, id string) standing {
	t.Helper()
	return *decode(t, "getIndexer", w.a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id, ""), &standing{})
}

func (w *walked) patchIndexer(t *testing.T, id, body string) {
	t.Helper()
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/indexers/"+id, body); r.Status != http.StatusOK {
		t.Fatalf("updateIndexer %s = %d %s, want 200", body, r.Status, r.Body)
	}
}

// TestSearchAndGrabLimitsHoldAcrossAControllerRestart gives Quarry the
// best priority, two searches an hour and one grab a day. The first search
// grabs from Quarry. After home restarts, Quarry's page still counts both,
// the next search asks Quarry but grabs from another Indexer, and the one
// after skips Quarry at its search limit.
func TestSearchAndGrabLimitsHoldAcrossAControllerRestart(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	quarry := w.indexers(t)["Quarry"].ID
	w.patchIndexer(t, quarry, `{"priority":1,"searches":{"count":2,"per":"hour"},"grabs":{"count":1,"per":"day"}}`)

	first := w.searched(t, w.search(t))
	if *first.Outcome != "grabbed" || w.grabbedFrom(t, *first.GrabID) != quarry {
		t.Fatalf("the first search ended %+v, want a Grab from Quarry", first)
	}

	box := w.a.Node("seedbox").NodeID
	linked := linkUp(t, w.a, box, "")
	act(t, w.run, "home", world.RestartNode)
	w.a = api(t, w.run)
	linkUp(t, w.a, box, linked)
	if s := w.standing(t, quarry); s.Searches.Used != 1 || s.Grabs.Used != 1 || s.Searches.FreeAt != nil || s.Grabs.FreeAt == nil {
		t.Fatalf("after the restart Quarry has used %+v and %+v, want one of two searches and its one grab", s.Searches, s.Grabs)
	}

	lines := w.searchAnother(t, "9000002")
	expectLine(t, "within the search limit", lines, "Quarry", "found: Found 1 Release")
	second := w.searched(t, w.lastSearch(t))
	if second.GrabID == nil || w.grabbedFrom(t, *second.GrabID) == quarry {
		t.Errorf("the second search ended %+v with lines %q, want a Grab from another Indexer than Quarry, at its grab limit", second, second.lines())
	}
	lines = w.searchAnother(t, "9000003")
	if !strings.HasPrefix(lines["Quarry"], "limit_reached: Limit reached until ") {
		t.Errorf("past the search limit Quarry's line is %q, want limit_reached", lines["Quarry"])
	}
}

// lastSearch is the id of the newest search of w's Work.
func (w *walked) lastSearch(t *testing.T) string {
	t.Helper()
	var list struct {
		Searches []searchView `json:"searches"`
	}
	decode(t, "getWorkSearches", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/searches", ""), &list)
	if len(list.Searches) == 0 {
		t.Fatal("the Work has no search")
	}
	return list.Searches[0].ID
}

// TestAFailingIndexerClimbsTheLadderAndStepsDownOnASuccess puts Quarry in
// maintenance, where every page answers 503: each search past its backoff
// fails and climbs a rung of the 1s, 2s, 4s ladder. Out of maintenance the
// next search finds the Release and steps down one rung.
func TestAFailingIndexerClimbsTheLadderAndStepsDownOnASuccess(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	quarry := w.indexers(t)["Quarry"].ID
	maintain := func(on bool) {
		args, err := json.Marshal(site.SiteArgs{Site: site.Quarry, On: on})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.run.Cue(worldtest.Context(t), site.Maintenance.Name, args); err != nil {
			t.Fatal(err)
		}
	}
	climbed := func(key string, rung int) {
		t.Helper()
		worldtest.Until(t, "Quarry's backoff passes", func() (bool, error) { return w.backedOffPast(t, quarry), nil })
		line := w.searchAnother(t, key)["Quarry"]
		if !strings.HasPrefix(line, "failing: Isn't responding. SlipStream retries at ") {
			t.Fatalf("Quarry's line in maintenance is %q, want failing", line)
		}
		if s := w.standing(t, quarry).Status; s.State != "failing" || s.Rung != rung {
			t.Fatalf("Quarry's status is %+v, want failing on rung %d", s, rung)
		}
	}

	maintain(true)
	climbed("9000002", 1)
	climbed("9000003", 2)
	maintain(false)
	worldtest.Until(t, "Quarry's backoff passes", func() (bool, error) { return w.backedOffPast(t, quarry), nil })
	expectLine(t, "out of maintenance", w.searchAnother(t, "9000004"), "Quarry", "found: Found 1 Release")
	if s := w.standing(t, quarry).Status; s.State != "ok" || s.Rung != 1 {
		t.Errorf("after a success Quarry's status is %+v, want ok one rung down, on 1", s)
	}
}

// backedOffPast reports whether Indexer id's backoff has passed.
func (w *walked) backedOffPast(t *testing.T, id string) bool {
	t.Helper()
	s := w.standing(t, id).Status
	if s.Until == nil {
		return true
	}
	until, err := time.Parse(time.RFC3339, *s.Until)
	return err == nil && time.Now().After(until)
}
