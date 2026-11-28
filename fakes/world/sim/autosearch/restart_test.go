package autosearch_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

func (w world) delaySearches(d time.Duration) {
	w.t.Helper()
	if _, err := w.Faults().Set(fault.Rule{ID: "slow", Target: "fixture-site", Route: "torznab.api", Fault: fault.Delay(d), Times: 1}); err != nil {
		w.t.Fatal(err)
	}
}

func (w world) hold(d time.Duration) {
	w.t.Helper()
	reply := w.admin(http.MethodGet, "/targets", "")
	id := simtest.Decode[struct{ Targets []struct{ ID string } }](w.t, "listTargets", reply).Targets[0].ID
	target := simtest.Decode[struct {
		Targets []map[string]any `json:"targets"`
	}](w.t, "listTargets", reply).Targets[0]
	for _, k := range []string{"id", "profiles", "contains", "defaultKbps", "createdAt"} {
		delete(target, k)
	}
	target["holdSeconds"] = int(d / time.Second)
	body, err := json.Marshal(target)
	if err != nil {
		w.t.Fatal(err)
	}
	if r := w.admin(http.MethodPut, "/targets/"+id, string(body)); r.Status != http.StatusOK {
		w.t.Fatalf("updateTarget = %d %s, want 200", r.Status, r.Body)
	}
}

func (w world) expectOneGrab(work, trigger string, searches int) {
	w.t.Helper()
	simtest.ExpectImported(w.t, simtest.Await(w.t, w.World))
	if qs := w.queries(w.indexer()); len(qs) != searches {
		w.t.Fatalf("Fixture Site's queries are %+v, want %d", qs, searches)
	}
	runs := w.searches(work)
	if len(runs) != 1 || deref(runs[0].Trigger) != trigger || runs[0].Grabs != 1 {
		w.t.Fatalf("the movie's runs are %+v, want one %s run that grabbed once", runs, trigger)
	}
}

// TestAControllerRestartedMidDispatchSearchesOnce restarts home while
// the press's search waits on Fixture Site. The restarted Controller asks
// again under the same query, which joins the search seedbox still runs,
// and the run grabs once from it.
func TestAControllerRestartedMidDispatchSearchesOnce(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launch(t, switches(false, false))
		synctest.Wait()
		w.delaySearches(20 * time.Second)
		work := simtest.AddOrchard(t, w.World)
		simtest.Search(t, w.World, work)
		time.Sleep(10 * time.Second)
		w.restartHome()
		w.expectOneGrab(work, "search", 1)
	})
}

// TestAControllerRestartedMidWaitGrabsWhenTheHoldEnds holds home's
// Target for two hours, so the add's search waits on the 1080p Release
// it found, and restarts home within the wait. The restarted Controller
// grabs it once the wait ends.
func TestAControllerRestartedMidWaitGrabsWhenTheHoldEnds(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launch(t, switches(false, false))
		synctest.Wait()
		w.hold(2 * time.Hour)
		work := w.add("9000001", "monitored")
		var until time.Time
		simtest.Eventually(t, "the add's run waits", func() bool {
			ws := w.searching(work).Waits
			if len(ws) == 1 {
				until = ws[0].Until
			}
			return len(ws) == 1
		})
		time.Sleep(time.Hour)
		w.restartHome()
		time.Sleep(time.Until(until) - time.Minute)
		synctest.Wait()
		if gs := simtest.GrabsListed(t, w.World); len(gs) != 0 {
			t.Fatalf("a minute before the wait ends the Grabs are %+v, want none", gs)
		}
		time.Sleep(time.Minute)
		w.expectOneGrab(work, "add", 1)
	})
}

// TestAControllerRestartedMidRSSGrabsTheFeedOnce restarts home while RSS
// sync reads Fixture Site's feed. The read never moved the feed's
// position, so the next scheduled run reads the same uploads and grabs
// the movie once.
func TestAControllerRestartedMidRSSGrabsTheFeedOnce(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launch(t, switches(false, true))
		synctest.Wait()
		work := simtest.AddOrchard(t, w.World)
		w.delaySearches(20 * time.Second)
		simtest.RunNow(t, w.API(), rssTask)
		time.Sleep(10 * time.Second)
		w.restartHome()
		time.Sleep(20 * time.Minute)
		synctest.Wait()
		w.expectOneGrab(work, "rss", 2)
		var got []string
		for _, r := range w.runs(rssTask) {
			got = append(got, r.lines()...)
		}
		expect(t, "RSS sync's runs", got, []string{
			"schedule|succeeded", "Fixture Site|seedbox|succeeded||Read 2 new releases. 2 matched a Work, and 1 was grabbed.",
			"run-now|canceled", "Fixture Site||canceled||",
		})
	})
}
