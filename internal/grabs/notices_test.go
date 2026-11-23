package grabs_test

import (
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

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
		got := entrytest.Of(w.History, grabs.GrabbedRow)
		if len(got) != 1 {
			t.Fatalf("the run recorded %d Grabbed, want 1", len(got))
		}
		f := got[0]
		if f.Work.ID != w.Work || f.Work.Label != "The Attic" || f.Indexer != "Quick Site" || f.Source != grabs.FromSearch || f.FoundBy != grabs.FoundBySearch {
			t.Errorf("Grabbed tells %+v, want The Attic, found by a search on Quick Site", f)
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
		if n := len(entrytest.Of(w.History, grabs.GrabbedRow)); n != 1 {
			t.Fatalf("after a run that joins the pack %d Grabbed are recorded, want still 1", n)
		}
	})
}

// failedWith fails one Grab of the World's first Item per code, in order,
// and answers the codes Grab failed was recorded for and those it sends.
func failedWith(t *testing.T, codes ...grabs.Code) (recorded, sent []grabs.Code) {
	t.Helper()
	w := failureWorld(t)
	for i, c := range codes {
		failWith(t, w, grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", byte('a'+i))), c)
	}
	abouts := entrytest.Abouts(w.History, grabs.GrabFailedRow)
	for i, f := range entrytest.Of(w.History, grabs.GrabFailedRow) {
		if f.Work.ID != w.Work || f.Stage != grabs.Fetching || f.Release != "The.Attic.2020.E01.1080p.WEB-DL-GRP" || f.Grab.IsZero() {
			t.Errorf("Grab failed tells %+v, want The Attic's Grab at fetching", f)
		}
		if abouts[i].Quiet == f.NeedsYou || abouts[i].Grab != f.Grab.UUID {
			t.Errorf("Grab failed %+v is about %+v, want it quiet unless it needs the Admin, and about its Grab", f, abouts[i])
		}
		recorded = append(recorded, f.Code)
		if f.NeedsYou {
			sent = append(sent, f.Code)
		}
	}
	return recorded, sent
}

func TestEveryFailureIsRecordedAndOnlyOneActivityListsAsNeedingTheAdminIsSent(t *testing.T) {
	fault := grabs.ReleaseFault
	for _, c := range []struct {
		name  string
		codes []grabs.Code
		sent  []grabs.Code
	}{
		{"a cancel", []grabs.Code{grabs.Canceled}, nil},
		{"a declined Import", []grabs.Code{grabs.Declined}, nil},
		{"a failure SlipStream searches again for", []grabs.Code{fault}, nil},
		{"a failure that waits for the Admin", []grabs.Code{grabs.NameTaken}, []grabs.Code{grabs.NameTaken}},
		{"the failure that stops searching again", []grabs.Code{fault, fault, fault}, []grabs.Code{fault}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				recorded, sent := failedWith(t, c.codes...)
				if !slices.Equal(recorded, c.codes) || !slices.Equal(sent, c.sent) {
					t.Fatalf("Grab failed was recorded for %v and sent for %v, want %v and %v", recorded, sent, c.codes, c.sent)
				}
			})
		})
	}
}

func TestAFetchingDropThatADecisionReplacesIsRecordedQuiet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		bad := newRelease(t)
		raw, err := bad.link.Torrent(bad.info)
		if err != nil {
			t.Fatal(err)
		}
		tw := newTwinWorld(t, twin{sites: []twinSite{
			{name: "Better Site", priority: 10, torrent: raw, releases: []searching.Release{listing(bluRay, 90, "")}},
			{name: "Other Site", priority: 25, releases: []searching.Release{listing(webDL, 5, "")}},
		}})
		grabstest.Blocked(t, tw.lib, blockEntry(bad.link.InfoHash, bluRay))
		if got, want := tw.searchedAutomatically(t), "Other Site "+webDL; got != want {
			t.Fatalf("the run's Grab is %q, want %q", got, want)
		}
		failed := entrytest.Of(tw.notices, grabs.GrabFailedRow)
		abouts := entrytest.Abouts(tw.notices, grabs.GrabFailedRow)
		if len(failed) == 0 || failed[0].Code != grabs.Blocklisted || failed[0].Stage != grabs.Fetching || failed[0].NeedsYou || !abouts[0].Quiet {
			t.Fatalf("the replaced Grab recorded %+v about %+v, want a quiet Grab failed at fetching first", failed, abouts)
		}
		if n := len(entrytest.Of(tw.notices, grabs.GrabbedRow)); n != 2 {
			t.Fatalf("the run recorded %d Grabbed, want the dropped Grab and the one in its place", n)
		}
	})
}

func TestAManualGrabNamesWhoPickedItAndItsBlocklistFailureIsRecordedQuiet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		id := made(t, a.grabbed(t, v, rowPick(t, v))[0])
		got := entrytest.Of(a.notices, grabs.GrabbedRow)
		if len(got) != 1 || got[0].FoundBy != grabs.FoundByManual || !got[0].By.APIKey {
			t.Fatalf("the Manual Grab recorded Grabbed %+v, want one picked by the API key", got)
		}
		if about := entrytest.Abouts(a.notices, grabs.GrabbedRow)[0]; got[0].Grab != id || about.Grab != id.UUID || about.Quiet {
			t.Errorf("Grabbed tells Grab %s and is about %+v, want it about the Grab %s", got[0].Grab, about, id)
		}
		if words := grabs.FoundByWords(got[0].FoundBy, got[0].Indexer, got[0].By); !strings.Contains(words, "API key") || !strings.Contains(words, "Better Site") {
			t.Errorf("its found-by sentence is %q, want it to name the API key and Better Site", words)
		}
		if err := a.g.FailWith(t.Context(), id, grabs.Blocklisted); err != nil {
			t.Fatal(err)
		}
		if got := entrytest.Abouts(a.notices, grabs.GrabFailedRow); len(got) != 1 || !got[0].Quiet {
			t.Fatalf("a Manual Grab the Blocklist held recorded Grab failed about %+v, want one, quiet", got)
		}
	})
}

func TestGrabsMessagesReadAsWritten(t *testing.T) {
	heat := entry.Work{
		ID: works.WorkID{UUID: must(storage.ParseUUID("019a8f5e-0000-7000-8000-000000000001"))}, Label: "Heat (1995)", Type: "book",
		Nouns:  entry.Nouns{Work: works.Noun{One: "book", Many: "books"}, Item: works.Noun{One: "book", Many: "books"}, Single: true},
		Poster: artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/heat.jpg"},
	}
	page := "/works/" + heat.ID.String()
	facts := rel.Facts{Quality: rel.Known(rel.WEBDL1080p), Group: rel.Known("FLUX"), Size: rel.Known(int64(6_200_000_000))}
	read := map[string]bool{}
	for _, f := range grabs.FoundBys {
		m := entrytest.Stored(t, grabs.GrabbedRow, grabs.GrabbedFacts{
			Work: heat, Release: facts, Indexer: "TorrentLeech", Source: grabs.FromSearch, FoundBy: f, By: entry.Actor{Name: "Sam"},
			Scores: []grabs.ScoreAt{{Target: "1080p", Score: 1250}, {Target: "720p", Score: 900}},
		}, "testdata/notices", "grabbed-"+string(f), heat.Label, "WEBDL-1080p", "FLUX", "5.8 GB", "TorrentLeech", "1080p", "1250", "720p", "900", page)
		read[strings.Join(m.Lines, "\n")] = true
	}
	if len(read) != len(grabs.FoundBys) {
		t.Errorf("%d ways of finding a release read %d ways, want each its own", len(grabs.FoundBys), len(read))
	}
	entrytest.Stored(t, grabs.GrabbedRow, grabs.GrabbedFacts{
		Work: heat, Release: facts, Indexer: "TorrentLeech", Source: grabs.FromManual, FoundBy: grabs.FoundByManual, By: entry.Actor{APIKey: true},
		Scores: []grabs.ScoreAt{{Target: "1080p", Score: 1250}},
	}, "testdata/notices", "grabbed-api-key", heat.Label, "API key", "TorrentLeech", "1250", page)
	codes := []grabs.Code{grabs.NameTaken, grabs.ImportFailed, grabs.DownloadFailed}
	read = map[string]bool{}
	for _, c := range codes {
		m := entrytest.Stored(t, grabs.GrabFailedRow, grabs.GrabFailedFacts{
			Work: heat, Code: c, Stage: grabs.Importing, Release: "Heat.1995.1080p.WEB-DL-FLUX", Source: grabs.FromSearch,
		}, "testdata/notices", "grab-failed-"+string(c), heat.Label, page)
		if m.Tone != entry.Fail {
			t.Errorf("Grab failed for %s reads at tone %v, want Fail", c, m.Tone)
		}
		read[strings.Join(m.Lines, "\n")] = true
	}
	if len(read) != len(codes) {
		t.Errorf("%d failure codes read %d ways, want each its own", len(codes), len(read))
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
