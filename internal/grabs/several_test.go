package grabs_test

import (
	"context"
	"maps"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/media"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// goneFile gives work's first Item a Bluray file that a scan found gone,
// so its cell is gone.
func goneFile(t *testing.T, w *grabstest.World, work works.WorkID) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		tree, err := w.Config.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		root, err := roots.OfWork(ctx, tx, work)
		if err != nil {
			return err
		}
		f, err := files.Add(ctx, tx, files.File{
			WorkID: work, Items: []works.ItemID{tree.Items[0].ID}, Cut: rel.Standard, RootID: root.ID, Path: work.String() + "/Scarf.mkv",
			Size: 1 << 30, Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: rel.Bluray1080p,
		}, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE files_file SET gone_at = ? WHERE id = ?", storage.Now(), f.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func searchSeveral(t *testing.T, w *grabstest.World, s grabs.Several) grabs.SeveralQueued {
	t.Helper()
	q, err := w.Grabs.SearchSeveral(t.Context(), identity.Admin{}, s)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestSearchAllLeavesGoneOnlyWorksOutAndCountsThem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		gone, missing := w.Work, w.AddWork()
		goneFile(t, w, gone)
		got := searchSeveral(t, w, grabs.Several{Wanted: &grabs.WantedFilter{}})
		outcomes := map[works.WorkID]grabs.WorkOutcome{}
		for _, o := range got.Works {
			outcomes[o.Work] = o
		}
		want := map[works.WorkID]grabs.WorkOutcome{missing: {Work: missing}, gone: {Work: gone, Skipped: grabs.SkipGoneOnly}}
		if got.Queued != 1 || got.Merged != 0 || got.LeftGone != 1 || !maps.Equal(outcomes, want) {
			t.Fatalf("Search all answers %+v, want the missing Work queued and the gone one left out and counted", got)
		}
		if q := w.Queue(t); len(q.Due) != 1 || q.Due[0].Work != missing || q.Due[0].Lane != pace.Bulk || q.Due[0].Trigger != searches.SearchAll || q.Due[0].Gone {
			t.Fatalf("after Search all the queue holds %+v, want the missing Work alone in bulk", q.Due)
		}
		got = searchSeveral(t, w, grabs.Several{Wanted: &grabs.WantedFilter{Reasons: []files.Reason{files.ReasonGone}}})
		if want := []grabs.WorkOutcome{{Work: gone}}; got.Queued != 1 || got.LeftGone != 0 || len(got.Works) != 1 || got.Works[0] != want[0] {
			t.Fatalf("Search all filtered to File gone answers %+v, want the gone Work alone queued", got)
		}
		for _, d := range w.Queue(t).Due {
			if d.Work == gone && (d.Lane != pace.Bulk || !d.Gone) {
				t.Fatalf("the gone Work is queued %+v, want bulk with its gone cells", d)
			}
		}
	})
}

func TestTheLibrarysBulkSearchSkipsAWorkThatWantsNothingAndMergesOneQueued(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		other, gone := w.AddWork(), w.AddWork()
		goneFile(t, w, gone)
		press(t, w, grabs.Scope{})
		nothing := works.WorkID{UUID: storage.NewUUID()}
		got := searchSeveral(t, w, grabs.Several{Works: []works.WorkID{w.Work, other, gone, nothing}})
		want := []grabs.WorkOutcome{{Work: w.Work, Merged: true}, {Work: other}, {Work: gone, Skipped: grabs.SkipGoneOnly}, {Work: nothing, Skipped: grabs.SkipNothingWanted}}
		if got.Queued != 1 || got.Merged != 1 || got.LeftGone != 1 || len(got.Works) != len(want) {
			t.Fatalf("the bulk Search answers %+v, want %+v", got, want)
		}
		for i := range want {
			if got.Works[i] != want[i] {
				t.Errorf("Work %d's outcome is %+v, want %+v", i, got.Works[i], want[i])
			}
		}
	})
}

func TestCancelQueuedClearsTheBulkLaneAloneAndKeepsDueTimes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Levels: []grabstest.Level{
			{Label: "Winter", Items: []string{"Scarf"}}, {Label: "Spring", Items: []string{"Kite"}},
		}})
		bulk := w.AddWork()
		press(t, w, grabs.Scope{})
		searchSeveral(t, w, grabs.Several{Works: []works.WorkID{bulk}})
		dueBefore := map[indexers.GroupKey]time.Time{}
		for _, d := range w.Queue(t).Due {
			dueBefore[d.Group] = w.DueOf(t, d.Work, d.Group)
		}
		n, err := w.Grabs.CancelQueued(t.Context(), identity.Admin{})
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Fatalf("Cancel queued searches canceled %d, want the bulk Work's 2 groups", n)
		}
		for _, d := range w.Queue(t).Due {
			wantLane := pace.Lane(0)
			if d.Work == w.Work {
				wantLane = pace.Now
			}
			if d.Lane != wantLane || wantLane == 0 && d.Items != nil {
				t.Errorf("after the cancel %s's group %s is queued %+v, want lane %d", d.Work, d.Group, d, wantLane)
			}
			if got := w.DueOf(t, d.Work, d.Group); !got.Equal(dueBefore[d.Group]) {
				t.Errorf("after the cancel %s's group %s is due at %v, want %v as before", d.Work, d.Group, got, dueBefore[d.Group])
			}
		}
	})
}
