package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// The series-scale Scenario's two generated series: 12 seasons of 100
// episodes, and 40 seasons of 250.
const (
	longSeries    = "60000001"
	longerSeries  = "60000002"
	addBudget     = 5 * time.Second
	refreshBudget = 30 * time.Second
)

func seriesScale(t *testing.T) *world.Run {
	t.Helper()
	s, err := scenario.Load("series-scale")
	if err != nil {
		t.Fatal(err)
	}
	return worldtest.LaunchScenario(t, s)
}

// size is v's live seasons and episodes and its monitored episodes.
func (v seriesView) size() string {
	seasons, episodes, monitored := 0, 0, 0
	for _, l := range v.Levels {
		if l.WithdrawnAt == nil {
			seasons++
		}
	}
	for _, it := range v.Items {
		if it.WithdrawnAt == nil {
			episodes++
		}
		if it.Monitored {
			monitored++
		}
	}
	return fmt.Sprintf("%d seasons, %d episodes, %d monitored", seasons, episodes, monitored)
}

func timedAdd(t *testing.T, a *worldtest.API, key string) (seriesView, time.Duration) {
	t.Helper()
	start := time.Now()
	v := addSeries(t, a, key, "all", false)
	return v, time.Since(start)
}

func TestAddA1200EpisodeSeriesInSeconds(t *testing.T) {
	t.Parallel()
	run := seriesScale(t)
	a := api(t, run)

	added, took := timedAdd(t, a, longSeries)
	t.Logf("adding 1,200 episodes took %s against a %s budget", took.Round(time.Millisecond), addBudget)
	if want := "12 seasons, 1200 episodes, 1200 monitored"; added.size() != want || took > addBudget {
		t.Fatalf("the add answered %s in %s, want %s within %s", added.size(), took, want, addBudget)
	}
	if got := getSeries(t, a, added.ID).size(); got != added.size() {
		t.Fatalf("getWork reads %s, want what the add answered", got)
	}
}

func TestRefreshA10000EpisodeSeries(t *testing.T) {
	t.Parallel()
	run := seriesScale(t)
	a := api(t, run)
	added, took := timedAdd(t, a, longerSeries)
	t.Logf("adding 10,000 episodes took %s", took.Round(time.Millisecond))
	if want := "40 seasons, 10000 episodes, 10000 monitored"; added.size() != want {
		t.Fatalf("the add answered %s, want %s", added.size(), want)
	}

	cue(t, run, "tvdb-add-season", `{"id":`+longerSeries+`}`)
	start := time.Now()
	refreshed := refreshSeries(t, a, added, `"outcome":"done"`)
	took = time.Since(start)
	t.Logf("refreshing 10,250 episodes took %s against a %s budget", took.Round(time.Millisecond), refreshBudget)
	if want := "41 seasons, 10250 episodes, 10250 monitored"; refreshed.size() != want || took > refreshBudget {
		t.Fatalf("the refresh left %s in %s, want %s within %s", refreshed.size(), took, want, refreshBudget)
	}
	for _, it := range added.Items {
		if got := refreshed.item(t, it.Key); got.ID != it.ID {
			t.Fatalf("episode %s is Item %s once refreshed, want %s", it.Key, got.ID, it.ID)
		}
	}
}

func TestFailedMiddlePageChangesNothing(t *testing.T) {
	t.Parallel()
	run := seriesScale(t)
	a := api(t, run)
	added := addSeries(t, a, longSeries, "all", false)

	cue(t, run, "tvdb-add-season", `{"id":`+longSeries+`}`)
	setFault(t, run, fault.Rule{ID: "middle-page", Target: "tvdb", Route: "episodes", Skip: 1}, "status=500")
	worldtest.Expect(t, "refreshWorks with the second page failing", a.Admin("home", http.MethodPost, "/api/v1/works/refresh", `{"workIds":["`+added.ID+`"]}`),
		worldtest.Reply{Status: 200, Body: `{"outcomes":[{"workId":"` + added.ID + `","outcome":"failed","problem":{"code":"metadata_unavailable","title":"The metadata source didn't respond. Try again later.","detail":"TVDB: responded with 500 Internal Server Error"}}]}` + "\n"})
	if fired := selected(t, run, events.Query{Kind: events.FaultFired, Match: map[string]string{"rule": "middle-page"}}); len(fired) == 0 {
		t.Fatal("no episodes page failed")
	}
	if got := getSeries(t, a, added.ID); got.seasons() != added.seasons() {
		t.Fatalf("the failed refresh changed the series to %s, want %s", got.size(), added.size())
	}

	if _, err := run.ClearFault(worldtest.Context(t), "middle-page"); err != nil {
		t.Fatal(err)
	}
	if got := refreshSeries(t, a, added, `"outcome":"done"`).size(); got != "13 seasons, 1300 episodes, 1300 monitored" {
		t.Fatalf("the refresh once every page answers left %s, want the new season in", got)
	}
}
