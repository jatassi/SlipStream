//go:build scale

package e2e

import (
	"testing"
	"time"
)

// The series-scale Scenario's 40 seasons of 250 episodes. refreshBudget
// times a wall clock, which a shared CI runner can't keep, so this test
// runs only on demand: go test -tags scale -run TestRefreshA10000EpisodeSeries ./e2e/
const (
	longerSeries  = "60000002"
	refreshBudget = 30 * time.Second
)

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
	refreshed := refreshSeries(t, a, added)
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
