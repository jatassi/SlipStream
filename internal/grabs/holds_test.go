package grabs_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// postedAgo is a Release of The Attic naming mark at quality, published
// ago before now.
func postedAgo(mark, quality string, ago time.Duration) searching.Release {
	title := "The.Attic.2020." + mark + "." + quality + "-GRP"
	return searching.Release{Title: title, Size: 4 << 30, Seeders: 10, Published: time.Now().Add(-ago), Link: searching.LinkOf("https://site.test/dl/" + title)}
}

// backlog has Quick Site answer every search with rs, runs Backlog search
// of the World's Work and answers the run of group once it decided.
func backlog(t *testing.T, w *grabstest.World, group indexers.GroupKey, rs ...searching.Release) grabs.Search {
	t.Helper()
	w.Answer("Quick Site", time.Second, searching.Answer{Releases: rs})
	enqueue(t, w, searches.Backlog)
	time.Sleep(time.Minute)
	return runOf(t, w, group)
}

// outcomesOf is each Release run listed, as "<title> <outcome>".
func outcomesOf(t *testing.T, w *grabstest.World, run grabs.SearchID) []string {
	t.Helper()
	ls, err := w.Grabs.Releases(t.Context(), identity.Admin{}, run)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, l := range ls {
		out = append(out, l.Title+" "+string(l.Outcome))
	}
	return out
}

func TestBacklogSearchWaitsOutTheHoldUnlessTheReleaseIsFinal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		w.Hold(0, 2*time.Hour)
		w.Run()
		group := indexers.LevelGroup(w.Level)
		web := postedAgo("E01", "1080p.WEB-DL", 30*time.Minute)
		run := backlog(t, w, group, web)
		if got, _ := grabbed(t, w); run.Outcome != grabs.Waiting || len(got) != 0 {
			t.Fatalf("the run is %s with Grabs %v, want it waiting with none", run.Outcome, got)
		}
		if got, want := outcomesOf(t, w, run.ID), []string{web.Title + " waiting"}; !slices.Equal(got, want) {
			t.Errorf("the run lists %v, want %v", got, want)
		}
		if got := w.Waits(t); len(got) != 1 || !got[0].Until.Equal(web.Published.Add(2*time.Hour)) {
			t.Fatalf("the waits are %+v, want the cell waiting until 2 hours after the WEB-DL was published", got)
		}
		bluray := postedAgo("E01", "1080p.BluRay", 10*time.Minute)
		run = backlog(t, w, group, bluray)
		if got, _ := grabbed(t, w); run.Outcome != grabs.Grabbed || len(got[bluray.Title]) != 1 {
			t.Fatalf("the run is %s with Grabs %v, want the BluRay, final at the cutoff, grabbed at once", run.Outcome, got)
		}
		if got := w.Waits(t); len(got) != 0 {
			t.Errorf("the waits are %+v, want the Grab to end the cell's wait", got)
		}
	})
}

// TestAReleaseMeantForTwoTargetsGoesOnceEitherStopsWaiting holds a
// WEB-DL wanted at two Targets, one holding 1 hour and the other 3: it
// waits until the earlier, then a WEB-DL past the first Hold is grabbed
// for both.
func TestAReleaseMeantForTwoTargetsGoesOnceEitherStopsWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hd := grabstest.HD[0]
		other := hd
		other.Name = "Other"
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Goals: []targets.Spec{hd, other}})
		w.Hold(0, time.Hour)
		w.Hold(1, 3*time.Hour)
		w.Run()
		group := indexers.LevelGroup(w.Level)
		fresh := postedAgo("E01", "1080p.WEB-DL", 30*time.Minute)
		backlog(t, w, group, fresh)
		waits := w.Waits(t)
		if len(waits) != 2 || !waits[0].Until.Equal(fresh.Published.Add(time.Hour)) || !waits[1].Until.Equal(waits[0].Until) {
			t.Fatalf("the waits are %+v, want both cells waiting until the shorter Hold ends", waits)
		}
		older := postedAgo("E01", "1080p.WEB-DL.x264", 2*time.Hour)
		if run := backlog(t, w, group, older); run.Outcome != grabs.Grabbed {
			t.Fatalf("the run is %s, want it to grab the WEB-DL published past the 1-hour Hold", run.Outcome)
		}
		views, err := w.Grabs.List(t.Context(), identity.Admin{})
		if err != nil {
			t.Fatal(err)
		}
		if len(views) != 1 || len(views[0].Targets) != 2 {
			t.Fatalf("the Grabs are %+v, want one holding both Targets' cells", views)
		}
	})
}

func TestAWaitingPackHoldsItsCellsSoNoSingleIsGrabbedBesideIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3})
		w.Hold(0, 2*time.Hour)
		w.Run()
		pack := postedAgo(grabstest.Pack(1, 3), "1080p.WEB-DL", 10*time.Minute)
		single := postedAgo(grabstest.Pack(3, 3), "1080p.WEB-DL", 24*time.Hour)
		run := backlog(t, w, indexers.LevelGroup(w.Level), pack, single)
		if got, _ := grabbed(t, w); run.Outcome != grabs.Waiting || len(got) != 0 {
			t.Fatalf("the run is %s with Grabs %v, want it waiting with none", run.Outcome, got)
		}
		var items []works.ItemID
		for _, wait := range w.Waits(t) {
			items = append(items, wait.Cell.Item)
		}
		if !slices.Equal(items, w.Items) {
			t.Errorf("the waiting cells are %v, want every episode the pack covers %v", items, w.Items)
		}
	})
}

func TestAPressPromotingABacklogRunWaitsForNoHold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		w.Hold(0, 2*time.Hour)
		web := postedAgo("E01", "1080p.WEB-DL", 10*time.Minute)
		w.Answer("Quick Site", time.Minute, searching.Answer{Releases: []searching.Release{web}})
		enqueue(t, w, searches.Backlog)
		w.Run()
		time.Sleep(10 * time.Second)
		if got := press(t, w, grabs.Scope{}); len(got) != 1 || !got[0].Merged {
			t.Fatalf("the press answers %+v, want it merged into the open Backlog run", got)
		}
		time.Sleep(time.Minute)
		run := runOf(t, w, indexers.LevelGroup(w.Level))
		if got, _ := grabbed(t, w); run.Trigger != searches.Search || run.Outcome != grabs.Grabbed || len(got[web.Title]) != 1 {
			t.Fatalf("the run is %s %s with Grabs %v, want the promoted press to grab the WEB-DL at once", run.Trigger, run.Outcome, got)
		}
		if got := w.Waits(t); len(got) != 0 {
			t.Errorf("the waits are %+v, want none", got)
		}
	})
}

// TestAGroupRunRefusesAReleaseOfAnotherGroupsItems presses Winter, Scarf
// and Mitten, while Quick Site lists only a Release of Spring's Boot.
func TestAGroupRunRefusesAReleaseOfAnotherGroupsItems(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Levels: []grabstest.Level{
			{Label: "Winter", Items: []string{"Scarf", "Mitten"}}, {Label: "Spring", Items: []string{"Boot"}},
		}})
		boot := postedAgo(grabstest.Pack(3, 3), "1080p.BluRay", 24*time.Hour)
		w.Answer("Quick Site", time.Second, searching.Answer{Releases: []searching.Release{boot}})
		press(t, w, grabs.Within(works.Within{Levels: []works.LevelID{w.Levels[0]}}))
		w.Run()
		time.Sleep(time.Minute)
		run := runOf(t, w, indexers.LevelGroup(w.Levels[0]))
		if got, _ := grabbed(t, w); run.Outcome != grabs.NothingAcceptable || len(got) != 0 {
			t.Fatalf("Winter's run is %s with Grabs %v, want nothing acceptable and none", run.Outcome, got)
		}
		ls, err := w.Grabs.Releases(t.Context(), identity.Admin{}, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range ls {
			if l.Outcome != grabs.ReleaseRefused || !slices.ContainsFunc(l.Reasons, func(r grabs.Worded) bool { return r.Code == targets.WrongItem }) {
				t.Errorf("Winter's run lists %s %s for %+v, want it refused wrong_item", l.Title, l.Outcome, l.Reasons)
			}
		}
	})
}
