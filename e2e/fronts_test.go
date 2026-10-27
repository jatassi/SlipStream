package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type searchView struct {
	ID         string     `json:"id"`
	Indexers   []lineView `json:"indexers"`
	Outcome    *string    `json:"outcome"`
	Candidates int        `json:"candidates"`
	GrabID     *string    `json:"grabId"`
}

// lineView is one Indexer's line in a search.
type lineView struct {
	IndexerID string  `json:"indexerId"`
	Name      string  `json:"name"`
	Outcome   *string `json:"outcome"`
	Releases  int     `json:"releases"`
	Sentence  string  `json:"sentence"`
}

// lines is each of s's Indexer lines as its name, outcome and sentence.
func (s searchView) lines() []string {
	var out []string
	for _, l := range s.Indexers {
		outcome := "asking"
		if l.Outcome != nil {
			outcome = *l.Outcome
		}
		out = append(out, l.Name+" "+outcome+": "+l.Sentence)
	}
	return out
}

// searched reads w's Work's searches until the run id has finished.
func (w *walked) searched(t *testing.T, id string) searchView {
	t.Helper()
	var found searchView
	worldtest.Until(t, "search "+id+" finishes", func() (bool, error) {
		var list struct {
			Searches []searchView `json:"searches"`
		}
		decode(t, "getWorkSearches", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/searches", ""), &list)
		i := slices.IndexFunc(list.Searches, func(s searchView) bool { return s.ID == id })
		if i < 0 {
			return false, nil
		}
		found = list.Searches[i]
		return found.Outcome != nil, nil
	})
	return found
}

func (w *walked) setFault(t *testing.T, r fault.Rule, spelled string) {
	t.Helper()
	f, err := fault.Parse(spelled)
	if err != nil {
		t.Fatal(err)
	}
	r.Fault = f
	if _, err := w.run.SetFault(worldtest.Context(t), r); err != nil {
		t.Fatal(err)
	}
}

func (w *walked) payload(t *testing.T) world.Release {
	t.Helper()
	i := slices.IndexFunc(w.m.Releases, func(r world.Release) bool { return r.Name == release })
	if i < 0 {
		t.Fatalf("the manifest lists no release %s", release)
	}
	return w.m.Releases[i]
}

func TestHeldSwarmCompletesWhenCleared(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	w.setFault(t, fault.Rule{ID: "swarm-hold", Target: "seeders", Origin: "seedbox"}, "throttle=16KiB/s")
	s := w.searched(t, w.search(t))
	if s.Outcome == nil || *s.Outcome != "grabbed" || s.GrabID == nil {
		t.Fatalf("the search ended %+v, want grabbed", s)
	}
	ctx := worldtest.Context(t)
	started, err := events.Wait(ctx, w.m.RunDir, events.Query{Kind: tracker.Announces.Kind, Node: "seedbox", Match: map[string]string{"event": "started"}})
	if err != nil {
		t.Fatal(err)
	}
	size := w.payload(t).Size
	var reports []int64
	worldtest.Until(t, "the held Grab reports its bytes rising three times", func() (bool, error) {
		g := decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+*s.GrabID, ""), &grabView{})
		if g.Stage != "fetching" && g.Stage != "downloading" {
			return false, fmt.Errorf("the held Grab is %s after %v, want downloading", g.Stage, g.stages())
		}
		if g.Progress == nil {
			return false, nil
		}
		if len(reports) == 0 || g.Progress.Done > reports[len(reports)-1] {
			reports = append(reports, g.Progress.Done)
		}
		return len(reports) == 4, nil
	})
	if last := reports[len(reports)-1]; last >= size {
		t.Fatalf("the held Grab reported %v of a %d byte payload, want fewer than the payload", reports, size)
	}
	seeders := string(tracker.SeedersTarget(release))
	closed, err := events.Select(w.m.RunDir, events.Query{Kind: "front.close", Node: "seedbox", Match: map[string]string{"target": seeders}, After: started.Seq})
	if err != nil {
		t.Fatal(err)
	}
	if len(closed) > 0 {
		t.Fatalf("the seedbox's seeder front closed %s while the swarm was held, want it open", closed[0].Data)
	}

	if _, err := w.run.ClearFault(ctx, "swarm-hold"); err != nil {
		t.Fatal(err)
	}
	if _, err := events.Wait(ctx, w.m.RunDir, events.Query{Kind: tracker.Announces.Kind, Node: "seedbox", Match: map[string]string{"event": "completed"}}); err != nil {
		t.Fatal(err)
	}
	w.ended(t, *s.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	checkPlaced(t, w.m, 1)
}

func TestA429FromTheIndexerLimitsOneSearch(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	w.setFault(t, fault.Rule{ID: "indexer-429", Target: "fixture-site", Origin: "seedbox", Route: "torznab.api", Times: 1}, "status=429,retry-after=2s")

	first := w.searched(t, w.search(t))
	lines := first.lines()
	if *first.Outcome != "nothing_acceptable" || first.Candidates != 0 || first.GrabID != nil ||
		len(lines) != 1 || !strings.HasPrefix(lines[0], "Fixture Site limited: Limit reached until ") {
		t.Fatalf("the first search ended %+v with lines %q, want nothing_acceptable with Fixture Site limited", first, lines)
	}
	fired, err := events.Select(w.m.RunDir, events.Query{Kind: events.FaultFired})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range fired {
		got = append(got, r.Node+" "+string(r.Data))
	}
	if want := []string{`seedbox {"rule":"indexer-429","target":"fixture-site","fault":"status=429,retry-after=2s","match":1}`}; !slices.Equal(got, want) {
		t.Fatalf("fault.fired records = %q, want %q", got, want)
	}

	worldtest.Until(t, "the Indexer waits out Retry-After", func() (bool, error) { return w.indexerWaits(t) == "none", nil })
	second := w.searched(t, w.search(t))
	if *second.Outcome != "grabbed" || second.GrabID == nil {
		t.Fatalf("the second search ended %+v with lines %q, want grabbed", second, second.lines())
	}
}

func TestALinkCutMidTransferResumes(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	link := string(world.LinkTarget("home"))
	completed := &events.Query{Kind: tracker.Announces.Kind, Match: map[string]string{"event": "completed"}}
	w.setFault(t, fault.Rule{ID: "link-cut", Target: fault.Target(link), Times: 1, After: completed}, "cut=256KiB")
	s := w.searched(t, w.search(t))
	if s.Outcome == nil || *s.Outcome != "grabbed" || s.GrabID == nil {
		t.Fatalf("the search ended %+v, want grabbed", s)
	}
	w.ended(t, *s.GrabID)
	if want := []string{"fetching", "downloading", "probing", "transferring", "importing", "imported"}; !slices.Equal(w.grab.stages(), want) {
		t.Fatalf("the Grab's history is %v with %+v, want %v", w.grab.stages(), w.grab.Failure, want)
	}
	fired, err := events.Select(w.m.RunDir, events.Query{Kind: events.FaultFired})
	if err != nil {
		t.Fatal(err)
	}
	if len(fired) != 1 || string(fired[0].Data) != `{"rule":"link-cut","target":"link/home","fault":"cut=256KiB","match":1}` {
		t.Fatalf("fault.fired records = %+v, want link-cut once", fired)
	}
	onLink := map[string]string{"target": link}
	cut, err := events.Select(w.m.RunDir, events.Query{Kind: "front.close", Match: map[string]string{"target": link, "cause": "cut"}, After: fired[0].Seq})
	if err != nil || len(cut) != 1 {
		t.Fatalf("link/home closed by the cut %+v, %v; want once", cut, err)
	}
	var carried struct{ In int64 }
	if err := json.Unmarshal(cut[0].Data, &carried); err != nil || carried.In < 256<<10 {
		t.Fatalf("the cut connection carried %s, want the Transfer's 256 KiB from seedbox", cut[0].Data)
	}
	if _, err := events.Wait(worldtest.Context(t), w.m.RunDir, events.Query{Kind: "front.open", Match: onLink, After: cut[0].Seq}); err != nil {
		t.Fatalf("no front.open on link/home after the cut: %v", err)
	}
	checkPlaced(t, w.m, 1)
}
