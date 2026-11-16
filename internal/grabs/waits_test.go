package grabs_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/trackers"
)

// threeTiers is a Target whose cutoff is Bluray-1080p over WEB-DL 1080p
// over HDTV-1080p, so neither a WEB-DL nor an HDTV Release is final.
var threeTiers = []targets.Spec{{
	Scores: targets.Scores{Increment: 1}, Name: "HD", Cutoff: rel.Bluray1080p,
	Tiers: [][]rel.Quality{{rel.Bluray1080p}, {rel.WEBDL1080p}, {rel.HDTV1080p}},
}}

// twoCandidates holds the World's goal 2 hours and has two Backlog search
// runs each find a Release for the cell to wait on: an HDTV, then a
// WEB-DL published later. It answers the first run and the wait's end.
func twoCandidates(t *testing.T, w *grabstest.World, web searching.Release) (grabs.Search, time.Time) {
	t.Helper()
	w.Hold(0, 2*time.Hour)
	w.Run()
	group := indexers.LevelGroup(w.Level)
	hdtv := postedAgo("E01", "1080p.HDTV", 30*time.Minute)
	first := backlog(t, w, group, hdtv)
	backlog(t, w, group, web)
	waits := w.Waits(t)
	if len(waits) != 1 || !waits[0].Until.Equal(hdtv.Published.Add(2*time.Hour)) {
		t.Fatalf("the waits are %+v, want one cell waiting until 2 hours after the HDTV was published", waits)
	}
	if got, _ := grabbed(t, w); len(got) != 0 {
		t.Fatalf("the Grabs are %v before the wait ends, want none", got)
	}
	return first, waits[0].Until
}

func TestAWaitEndingAtItsTimeGrabsTheBestCandidate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Goals: threeTiers})
		web := postedAgo("E01", "1080p.WEB-DL", 10*time.Minute)
		first, until := twoCandidates(t, w, web)
		w.AdvanceTo(until.Add(-time.Second))
		if got, _ := grabbed(t, w); len(got) != 0 {
			t.Fatalf("a second before the wait ends the Grabs are %v, want none", got)
		}
		w.AdvanceTo(until)
		if got, _ := grabbed(t, w); len(got) != 1 || len(got[web.Title]) != 1 {
			t.Fatalf("the Grabs are %v, want the WEB-DL, the better candidate, grabbed", got)
		}
		if got := w.Waits(t); len(got) != 0 {
			t.Errorf("the waits are %+v, want the wait ended", got)
		}
		last := runOf(t, w, indexers.LevelGroup(w.Level))
		if last.Outcome != grabs.Grabbed || len(last.Grabs) != 1 {
			t.Errorf("the WEB-DL's run is %s with Grabs %v, want grabbed with 1", last.Outcome, last.Grabs)
		}
		if got, want := outcomesOf(t, w, last.ID), []string{web.Title + " grabbed"}; !slices.Equal(got, want) {
			t.Errorf("the WEB-DL's run lists %v, want %v", got, want)
		}
		if got := outcomesOf(t, w, first.ID); len(got) != 1 || got[0] != "The.Attic.2020.E01.1080p.HDTV-GRP kept" {
			t.Errorf("the HDTV's run lists %v, want the HDTV kept", got)
		}
	})
}

func TestAWaitSkipsACandidateBlocklistedMeanwhile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Goals: threeTiers})
		web := postedAgo("E01", "1080p.WEB-DL", 10*time.Minute)
		hash := bencode.InfoHash{1, 2, 3}
		web.InfoHash = &hash
		_, until := twoCandidates(t, w, web)
		on := grabs.BlocklistOn{Reconciling: trackers.NotReconciling}
		if err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return on.Block(ctx, tx, grabs.BlockEntry{InfoHash: hash, Title: web.Title, Size: web.Size, Cause: grabs.CauseByHand, By: &grabs.Actor{Name: "Jackson"}})
		}); err != nil {
			t.Fatal(err)
		}
		w.AdvanceTo(until)
		if got, _ := grabbed(t, w); len(got) != 1 || len(got["The.Attic.2020.E01.1080p.HDTV-GRP"]) != 1 {
			t.Fatalf("the Grabs are %v, want the HDTV grabbed past the blocklisted WEB-DL", got)
		}
	})
}

// waitingPack has a Backlog search run find a pack of the World's three
// episodes for them to wait on 2 hours, and answers when the wait ends.
func waitingPack(t *testing.T, w *grabstest.World) (searching.Release, time.Time) {
	t.Helper()
	w.Hold(0, 2*time.Hour)
	w.Run()
	pack := postedAgo(grabstest.Pack(1, 3), "1080p.WEB-DL", 10*time.Minute)
	backlog(t, w, indexers.LevelGroup(w.Level), pack)
	if got := w.Waits(t); len(got) != 3 {
		t.Fatalf("the waits are %+v, want each of the three episodes waiting", got)
	}
	return pack, pack.Published.Add(2 * time.Hour)
}

// onlyGrab fails unless the World holds one Grab, of title, over every
// Item.
func onlyGrab(t *testing.T, w *grabstest.World, title string) {
	t.Helper()
	got, _ := grabbed(t, w)
	if len(got) != 1 || len(got[title]) != len(w.Items) {
		t.Fatalf("the Grabs are %v, want one Grab of %s over all %d episodes", got, title, len(w.Items))
	}
}

func TestGrabNowEndsEveryWaitOfTheWorkOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3})
		pack, until := waitingPack(t, w)
		n, err := w.Grabs.GrabWaiting(t.Context(), identity.Admin{}, w.Work, nil)
		if err != nil || n != 1 {
			t.Fatalf("Grab now = %d, %v, want 1 Release grabbing", n, err)
		}
		onlyGrab(t, w, pack.Title)
		if got := w.Waits(t); len(got) != 0 {
			t.Errorf("the waits are %+v, want Grab now to end all three", got)
		}
		if n, err := w.Grabs.GrabWaiting(t.Context(), identity.Admin{}, w.Work, nil); err != nil || n != 0 {
			t.Errorf("Grab now again = %d, %v, want 0, nothing waiting", n, err)
		}
		w.AdvanceTo(until.Add(time.Hour))
		onlyGrab(t, w, pack.Title)
	})
}

func TestARestartMidWaitEndsItOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3})
		pack, until := waitingPack(t, w)
		w.Restart()
		w.AdvanceTo(until)
		onlyGrab(t, w, pack.Title)
		w.Restart()
		w.AdvanceTo(until.Add(time.Hour))
		onlyGrab(t, w, pack.Title)
	})
}

func TestNoWaitEndsWhileReconciling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 3})
		pack, until := waitingPack(t, w)
		w.Reconciling(true)
		w.AdvanceTo(until.Add(time.Hour))
		if got, _ := grabbed(t, w); len(got) != 0 {
			t.Fatalf("while Reconciling the Grabs are %v, want none", got)
		}
		if got := w.Waits(t); len(got) != 3 {
			t.Fatalf("while Reconciling the waits are %+v, want all three still waiting", got)
		}
		if _, err := w.Grabs.GrabWaiting(t.Context(), identity.Admin{}, w.Work, nil); !errors.Is(err, grabs.ErrReconciling) {
			t.Fatalf("Grab now while Reconciling = %v, want ErrReconciling", err)
		}
		w.Reconciling(false)
		time.Sleep(grabs.DefaultTimings().RetryMax)
		onlyGrab(t, w, pack.Title)
	})
}
