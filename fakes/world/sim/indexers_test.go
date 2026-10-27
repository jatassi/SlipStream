package sim_test

import (
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/indexers"
)

// TestAnIndexerFailingForTheLongBackoffWarnsAndTheHealthCheckRunsEverySixHours
// has the fixture site answer every Torznab request 500 from just after
// start. The first failure's Test starts the long backoff, and the warning
// appears 6 hours later with no request in between. The health check runs
// every 6 hours plus at most its jitter, and each run succeeds because
// the Test ran.
func TestAnIndexerFailingForTheLongBackoffWarnsAndTheHealthCheckRunsEverySixHours(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithWalk(), inUTC)
		a := w.API()
		synctest.Wait()
		listed := decode[struct{ Indexers []struct{ ID string } }](t, "listIndexers", a.Admin("home", http.MethodGet, "/api/v1/indexers", ""))
		id := listed.Indexers[0].ID
		if _, err := w.Faults().Set(fault.Rule{ID: "down", Target: "fixture-site", Route: "torznab.api", Fault: fault.Status{Code: http.StatusInternalServerError}}); err != nil {
			t.Fatal(err)
		}
		if r := a.Admin("home", http.MethodPost, "/api/v1/indexers/"+id+"/test", ""); r.Status != http.StatusUnprocessableEntity {
			t.Fatalf("testIndexer with the site down = %d %s, want 422", r.Status, r.Body)
		}
		failed := time.Now()

		time.Sleep(time.Until(failed.Add(6*time.Hour - time.Minute)))
		synctest.Wait()
		if items := healthItems(t, a, "indexers"); len(items) != 0 {
			t.Fatalf("a minute before the long backoff the Indexers category holds %+v, want nothing", items)
		}
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		it, ok := healthItems(t, a, "indexers")["failing."+id]
		if want := "Every request to Fixture Site has failed since " + failed.UTC().Format("15:04") + "."; !ok || it.Level != "warn" || len(it.Text) < len(want) || it.Text[:len(want)] != want {
			t.Fatalf("past the long backoff the failing item is %+v, want a warning starting %q", it, want)
		}

		time.Sleep(time.Until(failed.Add(13 * time.Hour)))
		synctest.Wait()
		d := getTask(t, a, indexers.HealthCheckTaskID)
		expect(t, "the health check's runs over 13 hours", d.runs(), []string{"schedule|succeeded|1 Indexer", "schedule|succeeded|1 Indexer"})
		if gap := d.Runs[0].StartedAt.Sub(d.Runs[1].StartedAt); gap < 6*time.Hour || gap > 6*time.Hour+5*time.Minute {
			t.Errorf("the health check ran %s after its last run, want 6 hours plus at most its 5-minute jitter", gap)
		}
		if _, ok := healthItems(t, a, "indexers")["failing."+id]; !ok {
			t.Error("the health check's failed Tests cleared the long-backoff warning")
		}
	})
}
