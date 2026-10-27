package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type searchView struct {
	ID         string  `json:"id"`
	Outcome    *string `json:"outcome"`
	Candidates int     `json:"candidates"`
	GrabID     *string `json:"grabId"`
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

func TestA429FromTheIndexerFailsOneSearch(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	w.setFault(t, fault.Rule{ID: "indexer-429", Target: "fixture-site", Origin: "seedbox", Route: "torznab", Times: 1}, "status=429,retry-after=30s")

	first := w.searched(t, w.search(t))
	if first.Outcome == nil || *first.Outcome != "indexer_unavailable" || first.Candidates != 0 || first.GrabID != nil {
		t.Fatalf("the first search ended %+v, want indexer_unavailable with no candidates and no Grab", first)
	}
	fired, err := events.Select(w.m.RunDir, events.Query{Kind: events.FaultFired})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range fired {
		got = append(got, r.Node+" "+string(r.Data))
	}
	if want := []string{`seedbox {"rule":"indexer-429","target":"fixture-site","fault":"status=429,retry-after=30s","match":1}`}; !slices.Equal(got, want) {
		t.Fatalf("fault.fired records = %q, want %q", got, want)
	}

	second := w.searched(t, w.search(t))
	if second.Outcome == nil || *second.Outcome != "grabbed" || second.GrabID == nil {
		t.Fatalf("the second search ended %+v, want grabbed", second)
	}
}

// TestALinkCutMidTransferResumes cuts every link/seedbox connection at
// 256 KiB until one is cut, then lets the Transfer finish. The rule can't
// play once: the Controller's own lanes can dial before seedbox's intent
// names the Transfer, and seedbox closes those before they carry it.
func TestALinkCutMidTransferResumes(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	link := string(world.LinkTarget("seedbox"))
	completed := &events.Query{Kind: tracker.Announces.Kind, Match: map[string]string{"event": "completed"}}
	w.setFault(t, fault.Rule{ID: "link-cut", Target: fault.Target(link), After: completed}, "cut=256KiB")
	s := w.searched(t, w.search(t))
	if s.Outcome == nil || *s.Outcome != "grabbed" || s.GrabID == nil {
		t.Fatalf("the search ended %+v, want grabbed", s)
	}
	cut, err := events.Wait(worldtest.Context(t), w.m.RunDir, events.Query{Kind: "front.close", Match: map[string]string{"target": link, "cause": "cut"}})
	if err != nil {
		t.Fatalf("no link/seedbox connection was cut: %v", err)
	}
	if _, err := w.run.ClearFault(worldtest.Context(t), "link-cut"); err != nil {
		t.Fatal(err)
	}
	var carried struct{ Out int64 }
	if err := json.Unmarshal(cut.Data, &carried); err != nil || carried.Out < 256<<10 {
		t.Fatalf("the cut connection carried %s, want the Transfer's 256 KiB from seedbox", cut.Data)
	}
	w.ended(t, *s.GrabID)
	if want := []string{"fetching", "downloading", "probing", "transferring", "importing", "imported"}; !slices.Equal(w.grab.stages(), want) {
		t.Fatalf("the Grab's history is %v with %+v, want %v", w.grab.stages(), w.grab.Failure, want)
	}
	if _, err := events.Wait(worldtest.Context(t), w.m.RunDir, events.Query{Kind: "front.open", Match: map[string]string{"target": link}, After: cut.Seq}); err != nil {
		t.Fatalf("no front.open on link/seedbox after the cut: %v", err)
	}
	checkPlaced(t, w.m, 1)
}
