package grabs_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
)

// grabTitled is the view of the Grab of the Release titled title.
func grabTitled(t *testing.T, w *grabstest.World, title string) grabs.View {
	t.Helper()
	views, err := w.Grabs.List(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.Release.Title == title {
			return v
		}
	}
	t.Fatalf("no Grab of %s", title)
	return grabs.View{}
}

func TestAGrabKeepsWhatFoundItsReleaseOnceANewerRunReplacesItsRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		first := heard(w, "E01", "1080p.WEB-DL", 'a')
		run, err := w.Grabs.DecideRun(t.Context(), w.Work, w.Items, []indexers.Release{first})
		if err != nil {
			t.Fatal(err)
		}
		v := grabTitled(t, w, first.Title)
		if v.Search != run || v.FoundBy != grabs.FoundBySearch {
			t.Fatalf("the Grab reads run %v found by %q, want run %v found by search", v.Search, v.FoundBy, run)
		}
		failWith(t, w, v.ID, grabs.ReleaseFault)
		second := heard(w, "E01", "1080p.WEB-DL", 'b')
		w.Answer("Quick Site", time.Second, searching.Answer{Releases: []searching.Release{second.Release}})
		w.Run()
		time.Sleep(time.Minute)
		synctest.Wait()
		made := grabTitled(t, w, second.Title)
		runs, err := w.Grabs.Searches(t.Context(), identity.Admin{}, w.Work)
		if err != nil {
			t.Fatal(err)
		}
		if len(runs) == 0 || runs[0].Trigger != searches.FailedGrab || made.Search != runs[0].ID || made.FoundBy != grabs.FoundByFailedGrab {
			t.Fatalf("the next Grab reads run %v found by %q, want the first of %+v, a run after a failed Grab, found by failed_grab", made.Search, made.FoundBy, runs)
		}
		w.Answer("Quick Site", time.Second, searching.Answer{})
		press(t, w, grabs.Scope{})
		time.Sleep(time.Minute)
		synctest.Wait()
		if runs, err = w.Grabs.Searches(t.Context(), identity.Admin{}, w.Work); err != nil {
			t.Fatal(err)
		}
		for _, r := range runs {
			if r.ID == made.Search {
				t.Fatalf("the Searches list is %+v, want run %v replaced by a newer run", runs, made.Search)
			}
		}
		again := grabTitled(t, w, second.Title)
		if !again.Search.IsZero() || again.FoundBy != grabs.FoundByFailedGrab {
			t.Fatalf("the Grab reads run %v found by %q, want no run found by failed_grab", again.Search, again.FoundBy)
		}
	})
}

func TestAnRSSGrabNamesTheIndexerWhoseFeedFoundIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		web := uploaded("E01", "1080p.WEB-DL", 'a', time.Hour)
		feeds(w, web)
		syncFeed(t, w)
		runs, err := w.Grabs.Searches(t.Context(), identity.Admin{}, w.Work)
		if err != nil {
			t.Fatal(err)
		}
		v := grabTitled(t, w, web.Title)
		if len(runs) != 1 || v.Search != runs[0].ID || v.FoundBy != grabs.FoundByRSS || v.RSSIndexer != "Quick Site" {
			t.Fatalf("the RSS Grab reads run %v found by %q from %q, want run %+v found by rss from Quick Site", v.Search, v.FoundBy, v.RSSIndexer, runs)
		}
	})
}
