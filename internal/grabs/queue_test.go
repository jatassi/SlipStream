package grabs_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/works"
)

func queueWorld(t *testing.T, o grabstest.Options) *grabstest.World {
	t.Helper()
	o.Grabs, o.Indexers, o.Formats, o.JobCap = grabs.DefaultTimings(), indexers.DefaultTimings(), formats.DefaultTimings(), searching.DefaultTimings().JobCap
	return grabstest.New(t, o)
}

func press(t *testing.T, w *grabstest.World, scope grabs.Scope) []grabs.Queued {
	t.Helper()
	q, err := w.Grabs.SearchWork(t.Context(), identity.Admin{}, w.Work, scope)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func enqueue(t *testing.T, w *grabstest.World, trigger searches.Trigger) {
	t.Helper()
	if err := w.Grabs.Enqueue(t.Context(), w.Work, trigger); err != nil {
		t.Fatal(err)
	}
}

func levels(n int) []grabstest.Level {
	out := make([]grabstest.Level, n)
	for i := range out {
		out[i] = grabstest.Level{Label: []string{"Winter", "Spring", "Summer"}[i], Items: []string{"Scarf"}}
	}
	return out
}

// lineStates is each open run's line states, by its group.
func lineStates(q grabstest.Queue) map[indexers.GroupKey][]string {
	out := map[indexers.GroupKey][]string{}
	for _, r := range q.Runs {
		for _, l := range r.Lines {
			out[r.Group] = append(out[r.Group], l.Indexer+" "+l.State)
		}
	}
	return out
}

// runOf is the latest run of group the Searches list names.
func runOf(t *testing.T, w *grabstest.World, group indexers.GroupKey) grabs.Search {
	t.Helper()
	runs, err := w.Grabs.Searches(t.Context(), identity.Admin{}, w.Work)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range runs {
		if r.Group == group {
			return r
		}
	}
	t.Fatalf("the Searches list has no run of %s in %+v", group, runs)
	return grabs.Search{}
}

func lineOutcomes(r grabs.Search) []string {
	var out []string
	for _, l := range r.Indexers {
		word := "sent"
		if l.Outcome != nil {
			word = string(l.Outcome.Kind)
		}
		out = append(out, l.Name+" "+word)
	}
	return out
}

func TestASecondPressOnAQueuedOrSearchingGroupMerges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Levels: []grabstest.Level{{Label: "Winter", Items: []string{"Scarf", "Mitten"}}}})
		w.Answer("Quick Site", time.Minute, searching.Answer{})
		group := indexers.LevelGroup(w.Level)
		want := []grabs.Queued{{Key: group, Label: "Winter"}}
		if got := press(t, w, grabs.Scope{}); !reflect.DeepEqual(got, want) {
			t.Fatalf("the first press answers %+v, want %+v", got, want)
		}
		want[0].Merged = true
		if got := press(t, w, grabs.Scope{}); !reflect.DeepEqual(got, want) {
			t.Fatalf("a press on the queued group answers %+v, want %+v", got, want)
		}
		w.Run()
		if got := press(t, w, grabs.Scope{}); !reflect.DeepEqual(got, want) {
			t.Fatalf("a press on the searching group answers %+v, want %+v", got, want)
		}
		q := w.Queue(t)
		if len(q.Runs) != 1 || !slices.Equal(q.Runs[0].Items, w.Items) || q.Due[0].Lane != 0 || !q.Due[0].Searching {
			t.Fatalf("the queue holds %+v, want one run of both Items and nothing queued behind it", q)
		}
	})
}

func TestAnAddBeforeOrAfterAPressLeavesThePressInCharge(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []searches.Trigger
	}{
		{"add, then press", []searches.Trigger{searches.Add, searches.Search}},
		{"press, then add", []searches.Trigger{searches.Search, searches.Add}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
				for _, s := range tc.steps {
					if s == searches.Search {
						press(t, w, grabs.Scope{})
					} else {
						enqueue(t, w, s)
					}
				}
				want := []grabstest.Due{{
					Work: w.Work, Group: indexers.LevelGroup(w.Level), Lane: pace.Now, Trigger: searches.Search,
					Items: []works.ItemID{w.Item}, Gone: true,
				}}
				if got := w.Queue(t).Due; !reflect.DeepEqual(got, want) {
					t.Fatalf("the queue holds %+v, want the press's trigger and gone cells %+v", got, want)
				}
			})
		})
	}
}

func TestAPressPromotesTheOpenBacklogRunThatCoversIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Tight Site"}, Levels: levels(2)})
		w.Limit("Tight Site", 1, indexers.Hour)
		enqueue(t, w, searches.Backlog)
		w.Run()
		first, second := indexers.LevelGroup(w.Levels[0]), indexers.LevelGroup(w.Levels[1])
		if got := lineStates(w.Queue(t)); !slices.Equal(got[second], []string{"Tight Site planned"}) {
			t.Fatalf("the Backlog runs' lines are %v, want Spring's waiting for the limit", got)
		}
		if got := press(t, w, grabs.Within(works.Within{Levels: []works.LevelID{w.Levels[1]}})); len(got) != 1 || !got[0].Merged {
			t.Fatalf("a press on Spring answers %+v, want it merged into Spring's run", got)
		}
		synctest.Wait()
		r := runOf(t, w, second)
		if r.Trigger != searches.Search || r.Lane != "now" || r.Finished.IsZero() || !slices.Equal(lineOutcomes(r), []string{"Tight Site limit_reached"}) {
			t.Fatalf("Spring's run is %s %s finished %v with lines %v, want a finished Search in the now lane skipped limit_reached",
				r.Trigger, r.Lane, r.Finished, lineOutcomes(r))
		}
		if r := runOf(t, w, first); r.Trigger != searches.Backlog {
			t.Fatalf("Winter's run is named %s, want backlog", r.Trigger)
		}
	})
}

func TestAPressThatAddsItemsQueuesAgainBehindTheRunningSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Levels: []grabstest.Level{{Label: "Winter", Items: []string{"Scarf", "Mitten"}}}})
		w.Answer("Quick Site", 10*time.Second, searching.Answer{})
		scarf, mitten := w.Items[0], w.Items[1]
		press(t, w, grabs.Within(works.Within{Items: []works.ItemID{scarf}}))
		w.Run()
		if got := press(t, w, grabs.Scope{}); len(got) != 1 || got[0].Merged {
			t.Fatalf("a press adding Mitten answers %+v, want it queued again", got)
		}
		q := w.Queue(t)
		again := grabstest.Due{Work: w.Work, Group: indexers.LevelGroup(w.Level), Lane: pace.Now, Trigger: searches.Search, Items: []works.ItemID{mitten}, Gone: true, Searching: true}
		if !reflect.DeepEqual(q.Due, []grabstest.Due{again}) || len(q.Runs) != 1 || !slices.Equal(q.Runs[0].Items, []works.ItemID{scarf}) {
			t.Fatalf("the queue holds %+v, want Mitten queued behind the run of Scarf", q)
		}
		time.Sleep(10 * time.Second)
		if q := w.Queue(t); len(q.Runs) != 1 || !slices.Equal(q.Runs[0].Items, []works.ItemID{mitten}) || q.Due[0].Lane != 0 {
			t.Fatalf("once Scarf's run ends the queue holds %+v, want Mitten's run started", q)
		}
	})
}

func TestTheSearchLimitHoldsBacklogSearchAndSkipsAPress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Tight Site"}, Levels: levels(2), RSSEvery: 15 * time.Minute})
		w.Limit("Tight Site", 2, indexers.Hour)
		enqueue(t, w, searches.Backlog)
		w.Run()
		winter, spring := indexers.LevelGroup(w.Levels[0]), indexers.LevelGroup(w.Levels[1])
		if got := lineOutcomes(runOf(t, w, winter)); !slices.Equal(got, []string{"Tight Site no_releases"}) {
			t.Fatalf("Winter's Backlog line is %v, want it searched", got)
		}
		if got := lineStates(w.Queue(t)); !slices.Equal(got[spring], []string{"Tight Site planned"}) {
			t.Fatalf("Spring's Backlog line is %v, want it waiting with the last slot kept for RSS sync", got)
		}
		if _, _, err := w.Grabs.StartManualSearch(t.Context(), identity.Admin{}, w.Work, grabs.ManualScope{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if n := len(w.Node.Jobs()); n != 2 {
			t.Fatalf("Tight Site ran %d jobs, want Winter's and the Manual search's", n)
		}
		press(t, w, grabs.Within(works.Within{Levels: []works.LevelID{w.Levels[0]}}))
		synctest.Wait()
		if got := lineOutcomes(runOf(t, w, winter)); !slices.Equal(got, []string{"Tight Site limit_reached"}) {
			t.Fatalf("the press on Winter has lines %v, want Tight Site skipped limit_reached", got)
		}
		if got := lineStates(w.Queue(t)); !slices.Equal(got[spring], []string{"Tight Site planned"}) {
			t.Fatalf("Spring's Backlog line is %v after the press, want it still waiting", got)
		}
		time.Sleep(time.Hour + time.Minute)
		if got := lineOutcomes(runOf(t, w, spring)); !slices.Equal(got, []string{"Tight Site no_releases"}) || len(w.Node.Jobs()) != 3 {
			t.Fatalf("once the window frees a slot Spring's line is %v after %d jobs, want it searched third", got, len(w.Node.Jobs()))
		}
	})
}

func TestAnIndexerAlwaysLateSkipsItsStuckLinesLateWhileTheOthersDecide(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site", "Stuck Site"}, Levels: levels(3)})
		w.AtOnce("Stuck Site", 1)
		w.Answer("Stuck Site", time.Hour, searching.Answer{})
		enqueue(t, w, searches.Backlog)
		w.Run()
		time.Sleep(3 * indexers.SearchWait.Default)
		q := w.Queue(t)
		if len(q.Runs) != 0 {
			t.Fatalf("the queue holds runs %+v, want every group decided", q.Runs)
		}
		for _, l := range w.Levels {
			if got := lineOutcomes(runOf(t, w, indexers.LevelGroup(l))); !slices.Equal(got, []string{"Quick Site no_releases", "Stuck Site late"}) {
				t.Fatalf("a run's lines are %v, want Quick Site's answer and Stuck Site late", got)
			}
		}
		stuck := 0
		for _, j := range w.Node.Jobs() {
			if j.Indexer == w.Sites["Stuck Site"].String() {
				stuck++
			}
		}
		if stuck != 1 {
			t.Fatalf("Stuck Site was sent %d jobs, want one: its later lines skip late while it is stuck", stuck)
		}
	})
}

func TestARestartMidDispatchAsksNoLineTwice(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		w.Answer("Quick Site", 30*time.Second, searching.Answer{})
		press(t, w, grabs.Scope{})
		w.Run()
		time.Sleep(10 * time.Second)
		w.Restart()
		time.Sleep(30 * time.Second)
		r := runOf(t, w, indexers.LevelGroup(w.Level))
		if r.Finished.IsZero() || !slices.Equal(lineOutcomes(r), []string{"Quick Site no_releases"}) || len(w.Node.Jobs()) != 1 {
			t.Fatalf("after the restart the run is finished %v with lines %v after %d jobs, want the re-ask joining the one job",
				r.Finished, lineOutcomes(r), len(w.Node.Jobs()))
		}
	})
}

func TestNothingDispatchesWhileReconciling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		w.Reconciling(true)
		if _, err := w.Grabs.SearchWork(t.Context(), identity.Admin{}, w.Work, grabs.Scope{}); !errors.Is(err, grabs.ErrReconciling) {
			t.Fatalf("SearchWork while Reconciling = %v, want %v", err, grabs.ErrReconciling)
		}
		enqueue(t, w, searches.Backlog)
		w.Run()
		if q := w.Queue(t); len(q.Runs) != 0 || len(q.Due) != 1 || q.Due[0].Lane != pace.Backlog || len(w.Node.Jobs()) != 0 {
			t.Fatalf("while Reconciling the queue holds %+v after %d jobs, want the Backlog row queued and nothing sent", q, len(w.Node.Jobs()))
		}
		w.Reconciling(false)
		press(t, w, grabs.Scope{})
		synctest.Wait()
		if r := runOf(t, w, indexers.LevelGroup(w.Level)); r.Trigger != searches.Search || r.Finished.IsZero() {
			t.Fatalf("once Reconciling ends the run is %+v, want the press searched", r)
		}
	})
}

func TestARunDecidesOnlyOnceEveryLineAnswered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site", "Slow Site"}})
		w.Answer("Quick Site", 5*time.Second, searching.Answer{Releases: []searching.Release{{
			Title: "The.Attic.2020.1080p.BluRay.x264", Size: 8 << 30, Seeders: 10, Link: searching.LinkOf("https://site.test/attic.torrent"),
		}}})
		w.Answer("Slow Site", time.Minute, searching.Answer{})
		press(t, w, grabs.Scope{})
		w.Run()
		time.Sleep(10 * time.Second)
		group := indexers.LevelGroup(w.Level)
		if r := runOf(t, w, group); !r.Finished.IsZero() || !slices.Equal(lineOutcomes(r), []string{"Quick Site found", "Slow Site sent"}) {
			t.Fatalf("while Slow Site searches the run is finished %v with lines %v, want it open", r.Finished, lineOutcomes(r))
		}
		time.Sleep(time.Minute)
		if r := runOf(t, w, group); r.Outcome != grabs.Grabbed || r.Candidates != 1 {
			t.Fatalf("once every line answered the run is %s of %d candidates, want grabbed of the one Release", r.Outcome, r.Candidates)
		}
	})
}

func TestARefusedRunKeepsItsDueTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		enqueue(t, w, searches.Backlog)
		group := indexers.LevelGroup(w.Level)
		due := w.DueAt(t, group)
		if _, err := w.Indexers.Update(t.Context(), identity.Admin{}, w.Sites["Quick Site"], indexers.Edit{AutoSearch: new(false)}, indexers.Typed{}); err != nil {
			t.Fatal(err)
		}
		w.Run()
		if r := runOf(t, w, group); r.Outcome != grabs.NoIndexer || !r.Finished.Equal(r.Started) {
			t.Fatalf("the run is %+v, want refused no_indexer at its start", r)
		}
		if q := w.Queue(t); len(q.Runs) != 0 || q.Due[0].Lane != 0 || !w.DueAt(t, group).Equal(due) {
			t.Fatalf("the queue holds %+v due %v, want the row out of its lane and still due %v", q, w.DueAt(t, group), due)
		}
	})
}
