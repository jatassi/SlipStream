package grabs_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/works"
)

// byQuery is a Searching Node whose Indexers answer each search job at
// once by its query's text: a text lists names its Releases, any other
// none. It keeps the text of each search job, in the order each first
// reached it.
type byQuery struct {
	lists map[string][]searching.Release

	mu    sync.Mutex
	seen  map[string]bool
	texts []string
}

func (b *byQuery) Run(_ context.Context, j searching.Job) (searching.Poll, error) {
	a := searching.Answer{Started: time.Now()}
	if c, ok := j.Call.(searching.SearchCall); ok {
		b.mu.Lock()
		if b.seen == nil {
			b.seen = map[string]bool{}
		}
		if !b.seen[j.ID] {
			b.seen[j.ID] = true
			b.texts = append(b.texts, c.Query.Text)
		}
		b.mu.Unlock()
		a.Releases = b.lists[c.Query.Text]
	}
	return searching.Poll{Started: a.Started, Answer: &a}, nil
}

func (*byQuery) ReachSolver(context.Context, searching.Solver) (int, error) {
	return 0, errors.New("the scripted Node has no Challenge solver")
}

// asked is the texts searched since the last call, the group's first, as
// a run's first phase goes before its second, then the rest in text
// order, as a run's lines answer in any order.
func (b *byQuery) asked() []string {
	synctest.Wait()
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.texts
	b.texts = nil
	if len(out) > 1 {
		slices.Sort(out[1:])
	}
	return out
}

// webdl is a WEB-DL of The Attic covering the episodes mark names.
func webdl(mark string) searching.Release {
	return searching.Release{
		Title: "The.Attic.2020." + mark + ".1080p.WEB-DL-GRP", Size: 4 << 30, Seeders: 10,
		Link: searching.LinkOf("https://site.test/dl/" + mark),
	}
}

// settings applies p to the Automatic search settings.
func settings(t *testing.T, w *grabstest.World, p map[string]string) {
	t.Helper()
	patch := setting.Patch{}
	for k, v := range p {
		patch[k] = json.RawMessage(v)
	}
	if err := w.Grabs.Settings().Apply(t.Context(), patch); err != nil {
		t.Fatal(err)
	}
}

// steady keeps the World's Items out of the full-pace window, so a run's
// gap follows only what it heard.
func steady(t *testing.T, w *grabstest.World) {
	t.Helper()
	settings(t, w, map[string]string{grabs.FullPace.Key: "0"})
}

func TestAPressSearchesAloneOnlyTheItemsItsGroupsReleasesLeave(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		site := &byQuery{lists: map[string][]searching.Release{
			"The Attic":     {webdl(grabstest.Pack(1, 2))},
			"The Attic E03": {webdl(grabstest.Pack(3, 3))},
		}}
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3, Remote: site})
		press(t, w, grabs.Scope{})
		w.Run()
		if got := site.asked(); !slices.Equal(got, []string{"The Attic", "The Attic E03"}) {
			t.Fatalf("the press searched %q, want the season, then E03 alone, which its pack leaves", got)
		}
		if r := runOf(t, w, indexers.LevelGroup(w.Level)); r.Outcome != grabs.Grabbed || r.Took != 2 {
			t.Fatalf("the run is %s with %d Grabs, want grabbed with the pack and E03's single", r.Outcome, r.Took)
		}
	})
}

func TestBacklogSearchSearchesItemsAloneOnlyOnTheirFirstSearchOrAtTheLongestGap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		site := &byQuery{}
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3, Remote: site})
		steady(t, w)
		settings(t, w, map[string]string{grabs.LongestGap.Key: "86400"})
		w.Run()
		season, alone := []string{"The Attic"}, []string{"The Attic", "The Attic E02", "The Attic E03", "The Attic Scarf"}
		for i, want := range [][]string{alone, season, alone} {
			enqueue(t, w, searches.Backlog)
			if got := site.asked(); !slices.Equal(got, want) {
				t.Fatalf("Backlog search %d searched %q, want %q: each Item alone on its first search and once the gap doubled to the longest gap of a day", i+1, got, want)
			}
		}
	})
}

func TestTheGapResetsAfterAGrabDoublesAfterNothingAndStaysWhenEveryIndexerFailed(t *testing.T) {
	grabbable := searching.Answer{Releases: []searching.Release{{
		Title: "The.Attic.2020.1080p.BluRay.x264", Size: 8 << 30, Seeders: 10, Link: searching.LinkOf("https://site.test/attic.torrent"),
	}}}
	failed := searching.Answer{Outcome: &searching.Failing{Err: errors.New("the Indexer responded with status 500")}}
	for _, tc := range []struct {
		name string
		then searching.Answer
		gap  time.Duration
	}{
		{"took", grabbable, 2 * time.Hour},
		{"every Indexer failed", failed, 4 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, BacklogEvery: 2 * time.Hour})
				steady(t, w)
				group := indexers.LevelGroup(w.Level)
				w.Run()
				enqueue(t, w, searches.Backlog)
				if gap := gapOf(t, w, group); gap != 4*time.Hour {
					t.Fatalf("after a run that found nothing the next search is %v later, want double Backlog search's 2 hours", gap)
				}
				w.Answer("Quick Site", 0, tc.then)
				enqueue(t, w, searches.Backlog)
				if gap := gapOf(t, w, group); gap != tc.gap {
					t.Fatalf("after a run that %s the next search is %v later, want %v", tc.name, gap, tc.gap)
				}
			})
		})
	}
}

// gapOf is how long after group's latest run finished it is due again.
func gapOf(t *testing.T, w *grabstest.World, group indexers.GroupKey) time.Duration {
	t.Helper()
	synctest.Wait()
	return w.DueAt(t, group).Sub(runOf(t, w, group).Finished)
}

func TestAPressOnOneCellKeepsEveryOtherItemsLastResult(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3})
		w.Run()
		scarf, e02, e03 := w.Items[0], w.Items[1], w.Items[2]
		for _, items := range [][]works.ItemID{nil, {e02}} {
			scope := grabs.Scope{}
			if items != nil {
				scope = grabs.Within(works.Within{Items: items})
			}
			press(t, w, scope)
			synctest.Wait()
		}
		if got := searchedBy(t, w); !slices.EqualFunc(got, [][]works.ItemID{{e02}, {scarf, e02, e03}}, slices.Equal) {
			t.Fatalf("after a press on E02 the runs searched %v, want E02's and the season's, which holds Scarf's and E03's last result", got)
		}
		press(t, w, grabs.Within(works.Within{Items: []works.ItemID{scarf, e03}}))
		synctest.Wait()
		if got := searchedBy(t, w); !slices.EqualFunc(got, [][]works.ItemID{{scarf, e03}, {e02}}, slices.Equal) {
			t.Fatalf("after a press on Scarf and E03 the runs searched %v, want only each Item's latest", got)
		}
	})
}

// searchedBy is the Items each of the Work's runs searched, newest first.
func searchedBy(t *testing.T, w *grabstest.World) [][]works.ItemID {
	t.Helper()
	runs, err := w.Grabs.Searches(t.Context(), identity.Admin{}, w.Work)
	if err != nil {
		t.Fatal(err)
	}
	out := make([][]works.ItemID, len(runs))
	for i, r := range runs {
		out[i] = r.Items
		slices.SortFunc(out[i], func(a, b works.ItemID) int { return slices.Index(w.Items, a) - slices.Index(w.Items, b) })
	}
	return out
}
