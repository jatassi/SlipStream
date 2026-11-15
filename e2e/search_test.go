package e2e

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// addMirror creates Fixture Mirror, a second Torznab Indexer on the
// fixture site at priority 10, so it outranks Fixture Site's 25 and lists
// the same Releases. fields are more JSON fields of its draft.
func (w *walked) addMirror(t *testing.T, fields string) string {
	t.Helper()
	body := `{"kind":"torznab","name":"Fixture Mirror","address":"https://` + site.Host(site.Torznab) + site.TorznabPath + `","apiKey":"` + site.APIKey(w.s) + `","on":true,"automaticSearch":true,"priority":10` + fields + `}`
	reply := w.a.Admin("home", http.MethodPost, "/api/v1/indexers", body)
	if reply.Status != http.StatusCreated {
		t.Fatalf("createIndexer = %d %s, want 201", reply.Status, reply.Body)
	}
	reply.Status = http.StatusOK
	return decode(t, "createIndexer", reply, &struct {
		ID string `json:"id"`
	}{}).ID
}

// grabbedFrom reads Grab id's Indexer.
func (w *walked) grabbedFrom(t *testing.T, id string) string {
	t.Helper()
	g := decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+id, ""), &grabView{})
	if g.IndexerID == nil {
		t.Fatalf("Grab %s names no Indexer", id)
	}
	return *g.IndexerID
}

type queryView struct {
	Use      string  `json:"use"`
	Outcome  *string `json:"outcome"`
	Releases int     `json:"releases"`
}

func expectLines(t *testing.T, when string, s searchView, want ...string) {
	t.Helper()
	if got := s.lines(); !slices.Equal(got, want) {
		t.Fatalf("%s the search's lines are %q, want %q", when, got, want)
	}
}

// TestASecondSearchMergesIntoTheRunningOneWithinEachIndexersSearchLimit
// gives both Indexers a search limit of one an hour and searches twice
// while the first run waits on a slow feed. The second merges into it, so
// the one run asks each Indexer once, counts the Release each lists once
// and grabs it from Fixture Mirror, the better priority.
func TestASecondSearchMergesIntoTheRunningOneWithinEachIndexersSearchLimit(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	mirror := w.addMirror(t, `,"searches":{"count":1,"per":"hour"}`)
	var listed struct {
		Indexers []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"indexers"`
	}
	decode(t, "listIndexers", w.a.Admin("home", http.MethodGet, "/api/v1/indexers", ""), &listed)
	for _, ix := range listed.Indexers {
		if r := w.a.Admin("home", http.MethodPatch, "/api/v1/indexers/"+ix.ID, `{"searches":{"count":1,"per":"hour"}}`); r.Status != http.StatusOK {
			t.Fatalf("updateIndexer %s = %d %s, want 200", ix.Name, r.Status, r.Body)
		}
	}
	w.setFault(t, fault.Rule{ID: "slow-feed", Target: "fixture-site", Origin: "seedbox", Route: "torznab.api", Times: 2}, "slow-body=1KiB/s")

	first := w.search(t)
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/works/"+w.work+"/search", ""); r.Status != http.StatusAccepted || !strings.Contains(r.Body, `"state":"merged"`) {
		t.Fatalf("the second searchWork = %d %s, want 202 merged into the running search", r.Status, r.Body)
	}

	grabbed := w.searched(t, first)
	expectLines(t, "first", grabbed, "Fixture Mirror found: Found 2 Releases", "Fixture Site found: Found 2 Releases")
	if *grabbed.Outcome != "grabbed" || grabbed.Candidates != 2 || len(grabbed.GrabIDs) != 1 {
		t.Fatalf("the first search ended %+v, want 2 candidates and a Grab", grabbed)
	}
	if got := w.grabbedFrom(t, grabbed.GrabIDs[0]); got != mirror {
		t.Fatalf("the Grab is from Indexer %s, want Fixture Mirror %s", got, mirror)
	}
	var list struct {
		Searches []searchView `json:"searches"`
	}
	if decode(t, "listSearches", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/searches", ""), &list); len(list.Searches) != 1 {
		t.Fatalf("the Work's searches are %+v, want the one run both presses share", list.Searches)
	}
}

// TestAnIndexerTheSiteLimitsLeavesTheOtherToAnswer answers the first of
// the two Indexers' searches 429.
func TestAnIndexerTheSiteLimitsLeavesTheOtherToAnswer(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	w.addMirror(t, "")
	w.setFault(t, fault.Rule{ID: "indexer-429", Target: "fixture-site", Origin: "seedbox", Route: "torznab.api", Times: 1}, "status=429,retry-after=30s")

	s := w.searched(t, w.search(t))
	lines := s.lines()
	limited := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, " limited: Limit reached until ") })
	found := slices.IndexFunc(lines, func(l string) bool { return strings.HasSuffix(l, " found: Found 2 Releases") })
	if *s.Outcome != "grabbed" || s.Candidates != 2 || len(lines) != 2 || limited < 0 || found < 0 {
		t.Fatalf("the search ended %+v with lines %q, want one Indexer limited and the other's 2 Releases grabbed", s, lines)
	}
	if got := w.grabbedFrom(t, s.GrabIDs[0]); got != s.Indexers[found].IndexerID {
		t.Fatalf("the Grab is from Indexer %s, want %s, the one that answered", got, s.Indexers[found].IndexerID)
	}
}

// TestAnIndexerPastTheSearchWaitIsLateAndItsAnswerStillLands sets the
// search wait to 10s through the API and delays the first of the two
// Indexers' feeds 15s: the search grabs from the other, and the late one's
// query still completes.
func TestAnIndexerPastTheSearchWaitIsLateAndItsAnswerStillLands(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	w.addMirror(t, "")
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/indexer-settings", `{"values":{"indexers.searchWait":10}}`); r.Status != http.StatusOK {
		t.Fatalf("updateIndexerSettings = %d %s, want 200", r.Status, r.Body)
	}
	w.setFault(t, fault.Rule{ID: "slow-feed", Target: "fixture-site", Origin: "seedbox", Route: "torznab.api", Times: 1}, "delay=15s")

	run := w.searched(t, w.search(t))
	lines := run.lines()
	late := slices.IndexFunc(lines, func(l string) bool { return strings.HasSuffix(l, " late: Didn't finish in time") })
	found := slices.IndexFunc(lines, func(l string) bool { return strings.HasSuffix(l, " found: Found 2 Releases") })
	if *run.Outcome != "grabbed" || len(lines) != 2 || late < 0 || found < 0 {
		t.Fatalf("the search ended %+v with lines %q, want one Indexer late and the other's Release grabbed", run, lines)
	}
	id := run.Indexers[late].IndexerID
	worldtest.Until(t, "the late Indexer's search completes", func() (bool, error) {
		var page struct {
			Queries []queryView `json:"queries"`
		}
		decode(t, "listIndexerQueries", w.a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id+"/queries", ""), &page)
		i := slices.IndexFunc(page.Queries, func(q queryView) bool { return q.Use == "search" })
		return i >= 0 && page.Queries[i].Outcome != nil && *page.Queries[i].Outcome == "ok" && page.Queries[i].Releases == 2, nil
	})
}
