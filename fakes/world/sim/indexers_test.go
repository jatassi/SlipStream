package sim_test

import (
	"net/http"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/indexers"
)

// TestAnIndexerFailingForTheLongBackoffWarnsAndTheHealthCheckRunsEverySixHours
// has the fixture site answer every Torznab request 500 from just after
// start. The first failure's Test starts the long backoff, and the warning
// appears 6 hours later with no request in between. The health check runs
// every 6 hours plus at most its jitter, and each run succeeds because
// the Test ran.
func TestAnIndexerFailingForTheLongBackoffWarnsAndTheHealthCheckRunsEverySixHours(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), inUTC)
		a := w.API()
		synctest.Wait()
		listed := simtest.Decode[struct{ Indexers []struct{ ID string } }](t, "listIndexers", a.Admin("home", http.MethodGet, "/api/v1/indexers", ""))
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
		if items := simtest.HealthItems(t, a, "indexers"); len(items) != 0 {
			t.Fatalf("a minute before the long backoff the Indexers category holds %+v, want nothing", items)
		}
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		it, ok := simtest.HealthItems(t, a, "indexers")["failing."+id]
		if want := "Every request to Fixture Site has failed since " + failed.UTC().Format("15:04") + "."; !ok || it.Level != "warn" || len(it.Text) < len(want) || it.Text[:len(want)] != want {
			t.Fatalf("past the long backoff the failing item is %+v, want a warning starting %q", it, want)
		}

		time.Sleep(time.Until(failed.Add(13 * time.Hour)))
		synctest.Wait()
		d := simtest.GetTask(t, a, indexers.HealthCheckTaskID)
		expect(t, "the health check's runs over 13 hours", d.RunLines(), []string{"schedule|succeeded|1 Indexer", "schedule|succeeded|1 Indexer"})
		if gap := d.Runs[0].StartedAt.Sub(d.Runs[1].StartedAt); gap < 6*time.Hour || gap > 6*time.Hour+5*time.Minute {
			t.Errorf("the health check ran %s after its last run, want 6 hours plus at most its 5-minute jitter", gap)
		}
		if _, ok := simtest.HealthItems(t, a, "indexers")["failing."+id]; !ok {
			t.Error("the health check's failed Tests cleared the long-backoff warning")
		}
	})
}

type limitUse struct {
	Used   int        `json:"used"`
	FreeAt *time.Time `json:"freeAt"`
}

// TestASearchLimitFreesAnHourAfterTheSearch gives Fixture Site one search
// an hour. A search uses it until the hour since it passes, and the next
// second the limit is free again.
func TestASearchLimitFreesAnHourAfterTheSearch(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), inUTC)
		a := w.API()
		synctest.Wait()
		id := simtest.Decode[struct{ Indexers []struct{ ID string } }](t, "listIndexers", a.Admin("home", http.MethodGet, "/api/v1/indexers", "")).Indexers[0].ID
		if r := a.Admin("home", http.MethodPatch, "/api/v1/indexers/"+id, `{"searches":{"count":1,"per":"hour"}}`); r.Status != http.StatusOK {
			t.Fatalf("updateIndexer = %d %s, want 200", r.Status, r.Body)
		}
		searches := func() limitUse {
			return simtest.Decode[struct{ Searches limitUse }](t, "getIndexer", a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id, "")).Searches
		}
		simtest.Search(t, w, simtest.AddOrchard(t, w))
		searched := time.Now()
		synctest.Wait()
		used := searches()
		if used.Used != 1 || used.FreeAt == nil || used.FreeAt.Sub(searched) < time.Hour-time.Second || used.FreeAt.Sub(searched) > time.Hour+time.Second {
			t.Fatalf("after the search Fixture Site's searches are %+v, want its one used until an hour after %s", used, searched)
		}
		time.Sleep(time.Until(used.FreeAt.Add(time.Second)))
		synctest.Wait()
		if free := searches(); free.Used != 0 || free.FreeAt != nil {
			t.Errorf("an hour after the search Fixture Site's searches are %+v, want none used", free)
		}
	})
}

type queryPage struct {
	Queries []struct {
		ID         string    `json:"id"`
		ReservedAt time.Time `json:"reservedAt"`
	} `json:"queries"`
	NextOffset *int `json:"nextOffset"`
}

// TestHousekeepingDeletesIndexerQueriesPastTheirKeep keeps Indexer
// queries 12 hours in place of 30 days. It tests Fixture Site, then lets
// 18 hours pass, in which the health check tests it every 6 hours. After
// Housekeeping, the first Test's row is gone and every row left is from
// the last 12 hours.
func TestHousekeepingDeletesIndexerQueriesPastTheirKeep(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := simtest.Home(t)
		s.Timings.Indexers.KeepQueries = 12 * time.Hour
		w := sim.Launch(t, s, sim.WithWalk(), inUTC)
		a := w.API()
		synctest.Wait()
		id := simtest.Decode[struct{ Indexers []struct{ ID string } }](t, "listIndexers", a.Admin("home", http.MethodGet, "/api/v1/indexers", "")).Indexers[0].ID
		if r := a.Admin("home", http.MethodPost, "/api/v1/indexers/"+id+"/test", ""); r.Status != http.StatusOK {
			t.Fatalf("testIndexer = %d %s, want 200", r.Status, r.Body)
		}
		first := simtest.Decode[queryPage](t, "listIndexerQueries", a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id+"/queries", "")).Queries[0].ID
		time.Sleep(18 * time.Hour)
		synctest.Wait()
		simtest.RunNow(t, a, "housekeeping")
		synctest.Wait()
		cutoff := time.Now().Add(-s.Timings.Indexers.KeepQueries)
		rows := 0
		for offset := 0; ; {
			page := simtest.Decode[queryPage](t, "listIndexerQueries", a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id+"/queries?offset="+strconv.Itoa(offset), ""))
			for _, q := range page.Queries {
				rows++
				if q.ID == first || q.ReservedAt.Before(cutoff) {
					t.Fatalf("after Housekeeping a query from %s is kept, want none older than 12 hours", q.ReservedAt)
				}
			}
			if page.NextOffset == nil {
				break
			}
			offset = *page.NextOffset
		}
		if rows == 0 {
			t.Error("Housekeeping deleted every query, want the last 12 hours' health checks kept")
		}
	})
}
