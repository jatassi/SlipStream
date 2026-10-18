package sim_test

import (
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
)

func navAlert(t *testing.T, a *worldtest.API) string {
	t.Helper()
	c := decode[struct {
		Alerts struct {
			Nodes *string `json:"nodes"`
		} `json:"alerts"`
	}](t, "getCounts", a.Admin("home", http.MethodGet, "/api/v1/counts", ""))
	if c.Alerts.Nodes == nil {
		return "none"
	}
	return *c.Alerts.Nodes
}

// indexerWaits is how the one Indexer's home Node stands while its calls
// wait for it, or "none".
func indexerWaits(t *testing.T, a *worldtest.API) string {
	t.Helper()
	list := decode[struct {
		Indexers []struct {
			Waiting *struct {
				Standing string `json:"standing"`
			} `json:"waiting"`
		} `json:"indexers"`
	}](t, "listIndexers", a.Admin("home", http.MethodGet, "/api/v1/indexers", "")).Indexers
	if len(list) != 1 {
		t.Fatalf("listIndexers lists %d Indexers, want 1", len(list))
	}
	if list[0].Waiting == nil {
		return "none"
	}
	return list[0].Waiting.Standing
}

// searched presses Search on work and waits for the run's outcome.
func searched(t *testing.T, a *worldtest.API, work string) string {
	t.Helper()
	started := decode[struct {
		SearchIDs []string `json:"searchIds"`
	}](t, "searchWork", accepted(a.Admin("home", http.MethodPost, "/api/v1/works/"+work+"/search", "")))
	if len(started.SearchIDs) != 1 {
		t.Fatalf("searchWork started %v, want one search", started.SearchIDs)
	}
	run := started.SearchIDs[0]
	var outcome string
	eventually(t, "the search finishes", func() bool {
		runs := decode[struct {
			Searches []struct {
				ID      string  `json:"id"`
				Outcome *string `json:"outcome"`
			} `json:"searches"`
		}](t, "listSearches", a.Admin("home", http.MethodGet, "/api/v1/works/"+work+"/searches", "")).Searches
		for _, r := range runs {
			if r.ID == run && r.Outcome != nil {
				outcome = *r.Outcome
			}
		}
		return outcome != ""
	})
	return outcome
}

// accepted reads a 202 as decode reads a 200.
func accepted(r worldtest.Reply) worldtest.Reply {
	if r.Status == http.StatusAccepted {
		r.Status = http.StatusOK
	}
	return r
}

func grabsOn(t *testing.T, a *worldtest.API) []grabView {
	t.Helper()
	return decode[struct {
		Grabs []grabView `json:"grabs"`
	}](t, "listGrabs", a.Admin("home", http.MethodGet, "/api/v1/grabs", "")).Grabs
}

// view is how seedbox stands in Searching, the nav alert, and how the
// Indexer waits for seedbox.
type view struct{ searching, alert, waits string }

func expectStanding(t *testing.T, a *worldtest.API, when string, want view) {
	t.Helper()
	got := view{stateOf(seedbox(t, a), "searching"), navAlert(t, a), indexerWaits(t, a)}
	if got != want {
		t.Fatalf("%s, seedbox stands %+v, want %+v", when, got, want)
	}
}

func expectSearch(t *testing.T, a *worldtest.API, work, when, want string) {
	t.Helper()
	if got := searched(t, a, work); got != want {
		t.Fatalf("a search %s finishes %s, want %s", when, got, want)
	}
}

// TestAnOfflineSeedboxsSearchesWaitThenGoHome stops seedbox, which holds
// Searching and Downloading and is the Indexer's home, with the offline
// grace at its 1 min floor. Within the grace the Indexer waits for it and a
// search finishes indexer_unavailable; past the grace home covers
// Searching, so a search grabs through home while the Grab waits for a
// Node that downloads. Once seedbox is back and settled, the Grab is
// fetched and downloaded on seedbox and the nav alert clears.
func TestAnOfflineSeedboxsSearchesWaitThenGoHome(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const blip = 15 * time.Second
		w := sim.Launch(t, home(t), sim.WithWalk(), sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Environ = append(e.Environ, "SLIPSTREAM_TIMINGS_NODES_BLIP="+blip.String())
			}
		}))
		a := w.API()
		if r := a.Admin("home", http.MethodPatch, "/api/v1/nodes/settings", `{"offlineGrace":60}`); r.Status != http.StatusOK {
			t.Fatalf("updateNodeSettings = %d %s, want 200", r.Status, r.Body)
		}
		work := addOrchard(t, w)
		synctest.Wait()
		expectStanding(t, a, "in step", view{"holding", "none", "none"})

		w.Node("seedbox").Stop()
		synctest.Wait()
		expectStanding(t, a, "just stopped", view{"away", "none", "reconnecting"})
		expectSearch(t, a, work, "within the grace", "indexer_unavailable")
		seen, err := time.Parse(time.RFC3339Nano, seedbox(t, a).Link.Since)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Until(seen.Add(blip)))
		synctest.Wait()
		expectStanding(t, a, "past the blip", view{"away", "warn", "offline"})

		time.Sleep(time.Until(seen.Add(time.Minute)))
		synctest.Wait()
		expectStanding(t, a, "past the grace", view{"lapsed", "fail", "none"})
		expectSearch(t, a, work, "past the grace, through home", "grabbed")
		time.Sleep(time.Minute)
		synctest.Wait()
		if gs := grabsOn(t, a); len(gs) != 1 || gs[0].Stage != "fetching" || gs[0].FetchNodeID != "" {
			t.Fatalf("with no Node that downloads, the Grabs are %+v; want one waiting to fetch", gs)
		}

		w.Node("seedbox").Start()
		a = w.API()
		g := await(t, w)
		expectImported(t, g)
		box := seedbox(t, a).ID
		if g.NodeID != box || g.FetchNodeID != box {
			t.Fatalf("the Grab was fetched by %s and downloaded by %s, want both seedbox %s", g.FetchNodeID, g.NodeID, box)
		}
		expectStanding(t, a, "back and settled", view{"holding", "none", "none"})
	})
}
