package pacetest_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace/pacetest"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
)

var t0 = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

func groups(n int, lane pace.Lane, trigger searches.Trigger, at time.Duration) []pacetest.Arrival {
	var out []pacetest.Arrival
	for i := range n {
		out = append(out, pacetest.Arrival{At: at, Group: fmt.Sprintf("%s %02d", trigger, i), Entry: pace.Entry{Lane: lane, Trigger: trigger, Holds: true}})
	}
	return out
}

func sim(idx pacetest.Indexer, arrivals []pacetest.Arrival, manuals ...time.Duration) pacetest.Sim {
	return pacetest.Sim{
		Start: t0, Indexers: []pacetest.Indexer{idx}, Arrivals: arrivals, Manuals: manuals,
		Wait: 2 * time.Minute, Interval: 8 * time.Hour, Longest: 7 * 24 * time.Hour, For: 3 * time.Hour,
	}
}

func TestALimitOfTwoAnHourWithTenQueuedGroups(t *testing.T) {
	tight := pacetest.Indexer{Name: "Tight", AtOnce: 2, Limit: indexers.Limit{Count: 2, Per: indexers.Hour}, Answer: time.Minute, RSSEvery: 15 * time.Minute, RSSPages: 1}
	arrivals := append(groups(10, pace.Backlog, searches.Backlog, 0), groups(1, pace.Now, searches.Search, 2*time.Minute)...)
	r := sim(tight, arrivals, time.Minute).Run()

	if got := r.Queries["Tight"]; !reflect.DeepEqual(got, []int{2, 2, 2, 1}) {
		t.Errorf("queries per hour = %v, want the limit of 2 each hour and never past it", got)
	}
	if got := r.Manual["Tight"]; !reflect.DeepEqual(got, map[string]int{"searched": 1}) {
		t.Errorf("Manual search = %v, want its one query ahead of the 10 queued groups", got)
	}
	if got := r.Lines[pace.Now]; !reflect.DeepEqual(got, map[string]int{string(indexers.LimitReached): 1}) || !r.Skips[0].Equal(t0.Add(time.Hour)) {
		t.Errorf("the now press = %v skipped until %v, want limit_reached until %s", got, r.Skips, t0.Add(time.Hour))
	}
	if got := r.Lines[pace.Backlog]; len(got) != 0 || r.Drained != 0 || r.Starved != 0 {
		t.Errorf("Backlog search lines = %v, drained at %s, %d minutes Starved; want every line waiting, not Starved", got, r.Drained, r.Starved)
	}
	if r.PeakOpen > tight.AtOnce {
		t.Errorf("open runs peaked at %d, want at most Searches at once (%d)", r.PeakOpen, tight.AtOnce)
	}
	if got := r.RSS["Tight"]; got.Due != 13 || got.Made != 6 || got.Slots+r.Manual["Tight"]["searched"] != 7 {
		t.Errorf("RSS sync = %+v, want every query beside the Manual search to be RSS sync's", got)
	}
}

func TestTorrentLeechDrainsTenGroupsTwoAtATimeBesideRSSSync(t *testing.T) {
	leech := pacetest.Indexer{Name: "TorrentLeech", AtOnce: 2, Limit: indexers.Limit{Count: 50, Per: indexers.Hour}, Answer: time.Minute, RSSEvery: 15 * time.Minute, RSSPages: 1}
	r := sim(leech, groups(10, pace.Backlog, searches.Backlog, 0)).Run()

	if r.Lines[pace.Backlog]["searched"] != 10 || r.Drained != 5*time.Minute {
		t.Errorf("Backlog search lines = %v, drained at %s; want all 10 searched by 5m, two at a time", r.Lines[pace.Backlog], r.Drained)
	}
	if got := r.RSS["TorrentLeech"]; got.Due != got.Made {
		t.Errorf("RSS sync = %+v, want every fetch made", got)
	}
	if r.PeakOpen > leech.AtOnce {
		t.Errorf("open runs peaked at %d, want at most %d", r.PeakOpen, leech.AtOnce)
	}
}
