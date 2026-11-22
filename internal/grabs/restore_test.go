package grabs_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
)

// TestOpenRunsAreCancelled catches a queue search mid-run, as a Backup
// would, and ends it as a restore does: the run is gone and the Work's
// queued search stays queued.
func TestOpenRunsAreCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newStreaming(t, quick)
		g := s.open()
		stop := running(t, g)
		if _, err := g.SearchWork(t.Context(), identity.Admin{}, s.work, grabs.Scope{}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		synctest.Wait()
		open := `SELECT count(*) FROM grabs_search WHERE finished_at IS NULL AND manual_id IS NULL`
		stop()
		due := s.count(t, `SELECT count(*) FROM grabs_due`)
		if n := s.count(t, open); n != 1 {
			t.Fatalf("%d queue runs are open mid-search, want 1", n)
		}
		if err := s.lib.Write(t.Context(), grabs.CancelOpenRuns); err != nil {
			t.Fatal(err)
		}
		if n := s.count(t, open); n != 0 {
			t.Fatalf("%d queue runs are open after the restore, want none", n)
		}
		if n := s.count(t, `SELECT count(*) FROM grabs_due`); n != due {
			t.Fatalf("%d searches are queued after the restore, want the %d before it", n, due)
		}
	})
}
