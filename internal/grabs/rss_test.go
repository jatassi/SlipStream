package grabs_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/media"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

func rssWorld(t *testing.T) *grabstest.World {
	t.Helper()
	return queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
}

// uploaded is a Release of The Attic naming mark at quality, posted ago
// before now, with info hash hash and its own GUID.
func uploaded(mark, quality string, hash byte, ago time.Duration) searching.Release {
	h := bencode.InfoHash(slices.Repeat([]byte{hash}, 20))
	title := "The.Attic.2020." + mark + "." + quality + "-GRP"
	return searching.Release{
		Title: title, Size: 4 << 30, Seeders: 10, InfoHash: &h, GUID: title + "#" + string(hash), Published: time.Now().Add(-ago),
		Link: searching.LinkOf("https://site.test/dl/" + title),
	}
}

// feeds has Quick Site's feed list rs, newest first, from now on.
func feeds(w *grabstest.World, rs ...searching.Release) {
	w.Node.AnswerCall(w.Sites["Quick Site"], indexerstest.Recent(0), 0, searching.Answer{Releases: rs})
}

// syncFeed runs Quick Site's RSS sync unit once and answers its note.
func syncFeed(t *testing.T, w *grabstest.World) string {
	t.Helper()
	note, err := w.Grabs.SyncFeed(t.Context(), w.Sites["Quick Site"])
	if err != nil {
		t.Fatal(err)
	}
	return note
}

// skipped is the owned skip Quick Site's RSS sync unit ends with.
func skipped(t *testing.T, w *grabstest.World) string {
	t.Helper()
	_, err := w.Grabs.SyncFeed(t.Context(), w.Sites["Quick Site"])
	skip, ok := errors.AsType[*tasks.SkipError](err)
	if !ok || !skip.Owned {
		t.Fatalf("RSS sync ends with %v, want an owned skip", err)
	}
	return skip.Cause
}

func patchSettings(t *testing.T, w *grabstest.World, key string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Grabs.Settings().Apply(t.Context(), setting.Patch{key: raw}); err != nil {
		t.Fatal(err)
	}
}

// grabsOf is every Grab's Release title and source.
func grabsOf(t *testing.T, w *grabstest.World) []string {
	t.Helper()
	views, err := w.Grabs.List(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, v := range views {
		out = append(out, v.Release.Title+" by "+string(v.Source))
	}
	return out
}

// heldFile gives the World's first Item a Bluray-1080p file, which meets
// the HD Target, gone when gone.
func heldFile(t *testing.T, w *grabstest.World, gone bool) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		root, err := roots.OfWork(ctx, tx, w.Work)
		if err != nil {
			return err
		}
		f, err := files.Add(ctx, tx, files.File{
			WorkID: w.Work, Items: []works.ItemID{w.Item}, Cut: rel.Standard, RootID: root.ID, Path: w.Work.String() + "/Scarf.mkv",
			Size: 1 << 30, Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: rel.Bluray1080p,
		}, w.Goals)
		if err != nil || !gone {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE files_file SET gone_at = ? WHERE id = ?", storage.Now(), f.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRSSSyncGrabsANewUploadOnceAndMovesPastIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		web := uploaded("E01", "1080p.WEB-DL", 'a', time.Hour)
		other := searching.Release{Title: "Another.Show.2019.1080p.WEB-DL-GRP", Size: 4 << 30, GUID: "other", Published: time.Now().Add(-time.Hour)}
		feeds(w, web, other)
		if got, want := syncFeed(t, w), "Read 2 new Releases. 1 matched a Work, and 1 was grabbed."; got != want {
			t.Fatalf("the note is %q, want %q", got, want)
		}
		if got, want := grabsOf(t, w), []string{web.Title + " by rss"}; !slices.Equal(got, want) {
			t.Fatalf("the Grabs are %v, want %v", got, want)
		}
		runs, err := w.Grabs.Searches(t.Context(), identity.Admin{}, w.Work)
		if err != nil {
			t.Fatal(err)
		}
		if len(runs) != 1 || runs[0].Trigger != searches.RSS || runs[0].Outcome != grabs.Grabbed || !slices.Equal(runs[0].Items, []works.ItemID{w.Item}) {
			t.Fatalf("the Searches list is %+v, want one RSS run that grabbed for the first Item", runs)
		}
		if got, want := syncFeed(t, w), "Read no new Releases."; got != want {
			t.Fatalf("the next note is %q, want %q: the feed's position is past the upload", got, want)
		}
		if got := grabsOf(t, w); len(got) != 1 {
			t.Fatalf("the Grabs are %v, want the one Grab alone", got)
		}
	})
}

func TestRSSSyncSkipsWhileReconcilingAndAfterARestartDecidesTheSameUploadsOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		web := uploaded("E01", "1080p.WEB-DL", 'a', time.Hour)
		feeds(w, web)
		w.Reconciling(true)
		want := "Skipped because SlipStream is Reconciling after a restore. RSS sync starts again once Reconciling finishes."
		if got := skipped(t, w); got != want {
			t.Fatalf("the skip is %q, want %q", got, want)
		}
		if got := grabsOf(t, w); len(got) != 0 {
			t.Fatalf("the Grabs are %v, want none while Reconciling", got)
		}
		w.Restart()
		w.Reconciling(false)
		if got, want := syncFeed(t, w), "Read 1 new Release. 1 matched a Work, and 1 was grabbed."; got != want {
			t.Fatalf("after the restart the note is %q, want %q: the skipped fetch wrote no position", got, want)
		}
		if got := syncFeed(t, w); got != "Read no new Releases." {
			t.Fatalf("the next note is %q, want the upload decided once", got)
		}
		if got := grabsOf(t, w); len(got) != 1 {
			t.Fatalf("the Grabs are %v, want one", got)
		}
	})
}

func TestRSSSyncOffSkipsOwned(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		patchSettings(t, w, grabs.RSSOn.Key, false)
		if got, want := skipped(t, w), "RSS sync is off in Settings."; got != want {
			t.Fatalf("the skip is %q, want %q", got, want)
		}
	})
}

func TestAnIndexersSkipBecomesTheRSSUnitsOwnedSkip(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		w.Limit("Quick Site", 1, indexers.Hour)
		feeds(w)
		syncFeed(t, w)
		if got := skipped(t, w); got != "Skipped because Quick Site is at its search limit until 01:00." {
			t.Fatalf("the skip is %q, want the Indexer's sentence", got)
		}
	})
}

func TestTheHoldAppliesToAnRSSGrab(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		w.Hold(0, 2*time.Hour)
		web := uploaded("E01", "1080p.WEB-DL", 'a', 30*time.Minute)
		feeds(w, web)
		if got, want := syncFeed(t, w), "Read 1 new Release. 1 matched a Work, and none was grabbed."; got != want {
			t.Fatalf("the note is %q, want %q", got, want)
		}
		if got := grabsOf(t, w); len(got) != 0 {
			t.Fatalf("the Grabs are %v, want none inside the Hold", got)
		}
		if got := w.Waits(t); len(got) != 1 || !got[0].Until.Equal(web.Published.Add(2*time.Hour)) {
			t.Fatalf("the waits are %+v, want the cell waiting until 2 hours after the upload", got)
		}
	})
}

func TestRSSSyncNeverGrabsAReleaseAFailedGrabTried(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		bluray := uploaded("E01", "1080p.BluRay", 'a', time.Hour)
		failWith(t, w, grabbedOver(t, w, indexers.Release{Indexer: w.Sites["Quick Site"], Release: bluray}), grabs.ReleaseFault)
		web := uploaded("E01", "1080p.WEB-DL", 'b', time.Hour)
		feeds(w, bluray, web)
		syncFeed(t, w)
		var fetching []string
		views, err := w.Grabs.List(t.Context(), identity.Admin{})
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range views {
			if v.Stage == grabs.Fetching {
				fetching = append(fetching, v.Release.Title)
			}
		}
		if want := []string{web.Title}; !slices.Equal(fetching, want) {
			t.Fatalf("RSS sync grabbed %v, want %v: the failed BluRay is tried", fetching, want)
		}
	})
}

func TestRSSSyncTakesAProperForAMetCell(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		heldFile(t, w, false)
		proper := uploaded("E01", "1080p.BluRay.PROPER", 'a', time.Hour)
		feeds(w, proper, uploaded("E01", "1080p.BluRay", 'b', time.Hour))
		syncFeed(t, w)
		if got, want := grabsOf(t, w), []string{proper.Title + " by rss"}; !slices.Equal(got, want) {
			t.Fatalf("the Grabs are %v, want the proper alone for the Met cell", got)
		}
	})
}

func TestRSSSyncNeverGrabsForAGoneCell(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := rssWorld(t)
		heldFile(t, w, true)
		feeds(w, uploaded("E01", "1080p.BluRay", 'a', time.Hour))
		if got, want := syncFeed(t, w), "Read 1 new Release. 1 matched a Work, and none was grabbed."; got != want {
			t.Fatalf("the note is %q, want %q", got, want)
		}
		if got := grabsOf(t, w); len(got) != 0 {
			t.Fatalf("the Grabs are %v, want none: the gone cell waits for the Admin", got)
		}
	})
}

func TestRSSSyncKeepsItsSlotsOfALimitedIndexerWhileItIsOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"TorrentLeech"}, RSSEvery: 15 * time.Minute})
		w.Limit("TorrentLeech", 4, indexers.Hour)
		background := func() int {
			t.Helper()
			var rooms map[indexers.ID]indexers.Room
			err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				rooms, err = w.Indexers.Room(ctx, r)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			return rooms[w.Sites["TorrentLeech"]].Background.Free
		}
		on := background()
		patchSettings(t, w, grabs.RSSOn.Key, false)
		if off := background(); on >= off {
			t.Fatalf("Backlog search may send %d with RSS sync on and %d with it off, want fewer while RSS sync keeps its slots", on, off)
		}
	})
}

func TestBacklogSearchSearchesAnItemWhoseListedCutIsUnseen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Cuts: []rel.CutKey{rel.Standard, "extended"}, BacklogEvery: backlogEvery})
		heldFile(t, w, false)
		extended := uploaded("Extended.E01", "1080p.BluRay", 'a', time.Hour)
		w.Answer("Quick Site", time.Second, searching.Answer{Releases: []searching.Release{extended}})
		w.Run()
		backlogRun(t, w)
		later(grabs.LongestGap.Default)
		backlogRun(t, w)
		later(time.Minute)
		if got, want := grabsOf(t, w), []string{extended.Title + " by search"}; !slices.Equal(got, want) {
			t.Fatalf("the Grabs are %v, want the Extended cut a Backlog run sighted and grabbed", got)
		}
	})
}
