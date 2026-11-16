package grabs_test

import (
	"flag"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/notice/noticetest"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var update = flag.Bool("update", false, "rewrite the rendered messages under testdata/notices")

func TestAGrabRecordsGrabbedWithItsScoreAtEachTargetAndAJoinRecordsNone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hd := grabstest.HD[0]
		other := hd
		other.Name = "Other"
		w := grabstest.New(t, grabstest.Options{
			Sites: []string{"Quick Site"}, Season: 10, Goals: []targets.Spec{hd, other},
			Grabs: grabs.DefaultTimings(), Indexers: indexers.DefaultTimings(), Formats: formats.DefaultTimings(),
		})
		pack := heard(w, grabstest.Pack(1, 9), "1080p.WEB-DL", 'p')
		monitor(t, w, false)
		if _, err := w.Grabs.DecideRun(t.Context(), w.Work, w.Items, []indexers.Release{pack}); err != nil {
			t.Fatal(err)
		}
		got := noticetest.Of(w.Notices, grabs.GrabbedRow)
		if len(got) != 1 {
			t.Fatalf("the run recorded %d Grabbed, want 1", len(got))
		}
		f := got[0]
		if f.Work.ID != w.Work || f.Work.Label != "The Attic" || f.Indexer != "Quick Site" || f.Source != grabs.FromSearch || f.FoundBy != grabs.FoundBySearch {
			t.Errorf("Grabbed tells %+v, want The Attic, found by Automatic search on Quick Site", f)
		}
		if want := []grabs.ScoreAt{{Target: "HD"}, {Target: "Other"}}; !slices.Equal(f.Scores, want) {
			t.Errorf("Grabbed scores %+v, want one per Target the pack fills: %+v", f.Scores, want)
		}
		if q, _ := f.Release.Quality.Get(); q != rel.WEBDL1080p {
			t.Errorf("Grabbed tells the quality %v, want WEBDL-1080p", q)
		}
		monitor(t, w, true)
		if _, err := w.Grabs.DecideRun(t.Context(), w.Work, w.Items[4:5], []indexers.Release{pack}); err != nil {
			t.Fatal(err)
		}
		if n := len(noticetest.Of(w.Notices, grabs.GrabbedRow)); n != 1 {
			t.Fatalf("after a run that joins the pack %d Grabbed are recorded, want still 1", n)
		}
	})
}

// failedWith fails one Grab of the World's first Item per code, in order,
// and answers the codes Grab failed was recorded for.
func failedWith(t *testing.T, codes ...grabs.Code) []grabs.Code {
	t.Helper()
	w := failureWorld(t)
	for i, c := range codes {
		failWith(t, w, grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", byte('a'+i))), c)
	}
	var out []grabs.Code
	for _, f := range noticetest.Of(w.Notices, grabs.GrabFailedRow) {
		if f.Work.ID != w.Work || f.Stage != grabs.Fetching || f.Release != "The.Attic.2020.E01.1080p.WEB-DL-GRP" {
			t.Errorf("Grab failed tells %+v, want The Attic's Grab at fetching", f)
		}
		out = append(out, f.Code)
	}
	return out
}

func TestOnlyAFailureActivityListsAsNeedingTheAdminRecordsGrabFailed(t *testing.T) {
	for _, c := range []struct {
		name  string
		codes []grabs.Code
		want  []grabs.Code
	}{
		{"a cancel", []grabs.Code{grabs.Canceled}, nil},
		{"a declined Import", []grabs.Code{grabs.Declined}, nil},
		{"a failure SlipStream searches again for", []grabs.Code{grabs.ReleaseFault}, nil},
		{"a failure that waits for the Admin", []grabs.Code{grabs.NameTaken}, []grabs.Code{grabs.NameTaken}},
		{"the failure that stops searching again", []grabs.Code{grabs.ReleaseFault, grabs.ReleaseFault, grabs.ReleaseFault}, []grabs.Code{grabs.ReleaseFault}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				if got := failedWith(t, c.codes...); !slices.Equal(got, c.want) {
					t.Fatalf("Grab failed was recorded for %v, want %v", got, c.want)
				}
			})
		})
	}
}

func TestAManualGrabNamesWhoPickedItAndItsBlocklistFailureRecordsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		id := made(t, a.grabbed(t, v, rowPick(t, v))[0])
		got := noticetest.Of(a.notices, grabs.GrabbedRow)
		if len(got) != 1 || got[0].FoundBy != grabs.FoundByManual || !got[0].By.APIKey {
			t.Fatalf("the Manual Grab recorded Grabbed %+v, want one picked by the API key", got)
		}
		if words := grabs.FoundByWords(got[0].FoundBy, got[0].Indexer, got[0].By); words != "The API key picked it in Manual search from Better Site." {
			t.Errorf("its found-by sentence is %q", words)
		}
		if err := a.g.FailWith(t.Context(), id, grabs.Blocklisted); err != nil {
			t.Fatal(err)
		}
		if got := noticetest.Of(a.notices, grabs.GrabFailedRow); len(got) != 0 {
			t.Fatalf("a Manual Grab the Blocklist held recorded Grab failed %+v", got)
		}
	})
}

func TestGrabsMessagesReadAsWritten(t *testing.T) {
	heat := notice.Work{
		ID: works.WorkID{UUID: must(storage.ParseUUID("019a8f5e-0000-7000-8000-000000000001"))}, Label: "Heat (1995)", Type: "book",
		Nouns:  notice.Nouns{Work: works.Noun{One: "book", Many: "books"}, Item: works.Noun{One: "book", Many: "books"}, Single: true},
		Poster: artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/heat.jpg"},
	}
	facts := rel.Facts{Quality: rel.Known(rel.WEBDL1080p), Group: rel.Known("FLUX"), Size: rel.Known(int64(6_200_000_000))}
	for _, f := range grabs.FoundBys {
		noticetest.Golden(t, grabs.GrabbedRow, grabs.GrabbedFacts{
			Work: heat, Release: facts, Indexer: "TorrentLeech", Source: grabs.FromSearch, FoundBy: f, By: notice.Actor{Name: "Sam"},
			Scores: []grabs.ScoreAt{{Target: "1080p", Score: 1250}, {Target: "720p", Score: 900}},
		}, "testdata/notices", "grabbed-"+string(f), *update)
	}
	noticetest.Golden(t, grabs.GrabbedRow, grabs.GrabbedFacts{
		Work: heat, Release: facts, Indexer: "TorrentLeech", Source: grabs.FromManual, FoundBy: grabs.FoundByManual, By: notice.Actor{APIKey: true},
		Scores: []grabs.ScoreAt{{Target: "1080p", Score: 1250}},
	}, "testdata/notices", "grabbed-api-key", *update)
	for _, c := range []grabs.Code{grabs.NameTaken, grabs.ImportFailed} {
		noticetest.Golden(t, grabs.GrabFailedRow, grabs.GrabFailedFacts{
			Work: heat, Code: c, Stage: grabs.Importing, Release: "Heat.1995.1080p.WEB-DL-FLUX", Source: grabs.FromSearch,
		}, "testdata/notices", "grab-failed-"+string(c), *update)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
