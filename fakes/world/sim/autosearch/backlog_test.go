package autosearch_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

// TestBacklogSearchQueuesAGroupOnceItIsDueAndSkipsWhileOff adds a movie
// no Release names without searching it. The first Backlog search gives
// its group a due time and queues nothing, the scheduled run at the row's
// next search time queues it and the run finds nothing, and with Backlog
// search off in Settings the unit skips with the settings' sentence.
func TestBacklogSearchQueuesAGroupOnceItIsDueAndSkipsWhileOff(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launch(t, switches(true, false))
		synctest.Wait()
		work := w.add(lanterns, "")
		expect(t, "the first run", w.ran(backlogTask).lines(), []string{"run-now|succeeded", "Due searches|home|succeeded||Queued 0 searches."})
		due := w.wantedRow(work).NextAt
		if due == nil {
			t.Fatal("after Backlog search Wanted gives the movie no next search")
		}

		time.Sleep(time.Until(*due))
		synctest.Wait()
		simtest.Eventually(t, "the queued run finds nothing", func() bool {
			r := w.wantedRow(work)
			return r.Last != nil && *r.Last == "nothing_found" && r.State == "idle"
		})

		w.settings(`{"search.backlogOn":false}`)
		expect(t, "the run while off", w.ran(backlogTask).lines(), []string{"run-now|skipped", "Due searches|home|skipped|Backlog search is off in Settings.|"})
	})
}
