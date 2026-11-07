package e2e

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// tests counts the Tests in Indexer id's query rows.
func (w *walked) tests(t *testing.T, id string) int {
	t.Helper()
	var page struct {
		Queries []queryView `json:"queries"`
	}
	decode(t, "listIndexerQueries", w.a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id+"/queries", ""), &page)
	n := 0
	for _, q := range page.Queries {
		if q.Use == "test" {
			n++
		}
	}
	return n
}

func (w *walked) refuse(t *testing.T, code int) {
	t.Helper()
	if _, err := w.run.Cue(worldtest.Context(t), site.Refuse.Name, json.RawMessage(`{"code":`+strconv.Itoa(code)+`}`)); err != nil {
		t.Fatal(err)
	}
}

// TestAStoppedIndexerRaisesOneItemTheHealthCheckSkipsAndATestClears has
// the fixture site refuse Fixture Site's account: its Test stops it, which
// raises its Fail and, as the only Indexer, the movies' Fail. Once the
// site serves again the health check tests Fixture Mirror and leaves
// Fixture Site alone, and an Admin's Test clears its item.
func TestAStoppedIndexerRaisesOneItemTheHealthCheckSkipsAndATestClears(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	var listed struct {
		Indexers []struct {
			ID string `json:"id"`
		} `json:"indexers"`
	}
	decode(t, "listIndexers", w.a.Admin("home", http.MethodGet, "/api/v1/indexers", ""), &listed)
	fixture := listed.Indexers[0].ID
	if items := healthItems(t, w.a, "indexers"); len(items) != 0 {
		t.Fatalf("the Indexers category holds %+v before any stop, want nothing", items)
	}

	w.refuse(t, 101)
	worldtest.Expect(t, "testIndexer while the site refuses the account", w.a.Admin("home", http.MethodPost, "/api/v1/indexers/"+fixture+"/test", ""),
		problemBecause(http.StatusUnprocessableEntity, "indexer_test_failed", "The Indexer's Test failed",
			"Fixture Site refused the API key with 'Account suspended'. Check the account, then test again."))
	stopped := "stopped." + fixture
	worldtest.Until(t, "the stop and the movies raise their items", func() (bool, error) {
		return len(healthItems(t, w.a, "indexers")) == 2, nil
	})
	items := healthItems(t, w.a, "indexers")
	if it := items[stopped]; it.Level != "fail" || it.Subject != "Fixture Site" ||
		it.Text != "Fixture Site refused the API key with 'Account suspended'. Check the account, then test the Indexer." ||
		it.Action == nil || it.Action.Kind != "link" || it.Action.Href != "/settings/indexers/"+fixture {
		t.Errorf("the stop's item is %+v, want a Fail naming the fix and linking to the Indexer", it)
	}
	if it := items["unserved.movie"]; it.Level != "fail" || it.Text != "No Indexer can search for movies. Add an Indexer or turn one on." ||
		it.Action == nil || it.Action.Href != "/settings/indexers" {
		t.Errorf("the movies' item is %+v, want a Fail linking to Settings > Indexers", it)
	}

	w.refuse(t, 0)
	mirror := w.addMirror(t, "")
	worldtest.Until(t, "Fixture Mirror serves the movies", func() (bool, error) {
		_, open := healthItems(t, w.a, "indexers")["unserved.movie"]
		return !open, nil
	})
	before, mirrored := w.tests(t, fixture), w.tests(t, mirror)
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/tasks/indexer-health-check/runs", ""); r.Status != http.StatusAccepted {
		t.Fatalf("runTask = %d %s, want 202", r.Status, r.Body)
	}
	var detail struct {
		Runs []struct {
			Result string `json:"result"`
			Units  []struct {
				Label   string `json:"label"`
				Outcome string `json:"outcome"`
			} `json:"units"`
		} `json:"runs"`
	}
	worldtest.Until(t, "the health check finishes", func() (bool, error) {
		decode(t, "getTask", w.a.Admin("home", http.MethodGet, "/api/v1/tasks/indexer-health-check", ""), &detail)
		return len(detail.Runs) > 0 && detail.Runs[0].Result != "", nil
	})
	if r := detail.Runs[0]; r.Result != "succeeded" || len(r.Units) != 1 || r.Units[0].Label != "Fixture Mirror" || r.Units[0].Outcome != "succeeded" {
		t.Fatalf("the health check ran %+v, want Fixture Mirror alone, succeeded", r)
	}
	if got := w.tests(t, fixture); got != before {
		t.Errorf("the health check tested the stopped Fixture Site: %d Tests, want %d", got, before)
	}
	if got := w.tests(t, mirror); got != mirrored+1 {
		t.Errorf("Fixture Mirror has %d Tests after the health check, want %d", got, mirrored+1)
	}

	if r := w.a.Admin("home", http.MethodPost, "/api/v1/indexers/"+fixture+"/test", ""); r.Status != http.StatusOK {
		t.Fatalf("testIndexer once the site serves = %d %s, want 200", r.Status, r.Body)
	}
	worldtest.Until(t, "a passing Test clears the stop", func() (bool, error) { return len(healthItems(t, w.a, "indexers")) == 0, nil })
}
