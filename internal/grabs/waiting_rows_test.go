package grabs_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestARunsGrabLeavingFetchingKeepsTheCandidateItsOtherCellWaitsOn ends
// the kept Releases of a run that grabbed E01 and waits at E02, as a Grab
// leaving fetching does.
func TestARunsGrabLeavingFetchingKeepsTheCandidateItsOtherCellWaitsOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 2})
		w.Hold(0, 2*time.Hour)
		w.Run()
		bluray := postedAgo("E01", "1080p.BluRay", 10*time.Minute)
		web := postedAgo("E02", "1080p.WEB-DL", 10*time.Minute)
		backlog(t, w, indexers.LevelGroup(w.Level), bluray, web)
		time.Sleep(10 * time.Minute)
		views, err := w.Grabs.List(t.Context(), identity.Admin{})
		if err != nil {
			t.Fatal(err)
		}
		if len(views) != 1 || views[0].Release.Title != bluray.Title {
			t.Fatalf("the Grabs are %+v, want the final BluRay grabbed at once", views)
		}
		if err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return db.New(tx).EndReleasesOfGrab(ctx, views[0].ID)
		}); err != nil {
			t.Fatalf("ending the BluRay run's kept Releases: %v", err)
		}
		w.AdvanceTo(web.Published.Add(2 * time.Hour))
		if got, _ := grabbed(t, w); len(got[web.Title]) != 1 {
			t.Fatalf("the Grabs are %v, want the WEB-DL grabbed once its wait ended", got)
		}
	})
}
