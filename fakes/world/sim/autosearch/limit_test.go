package autosearch_test

import (
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

// searchAll queues work from Wanted's Search all, in the bulk lane, whose
// lines claim only the slots RSS sync leaves.
func (w world) searchAll(work string) {
	w.t.Helper()
	if r := w.admin(http.MethodPost, "/works/search", `{"workIds":["`+work+`"]}`); r.Status != http.StatusAccepted {
		w.t.Fatalf("searchWorks = %d %s, want 202", r.Status, r.Body)
	}
	synctest.Wait()
}

func (w world) expectStarved(when, work string) {
	w.t.Helper()
	limits := w.wanted().Status.Limits
	if len(limits) != 1 || limits[0].IndexerName != "Fixture Site" || !limits[0].Starved || limits[0].Until != nil {
		w.t.Fatalf("%s Wanted's limits are %+v, want Fixture Site starved", when, limits)
	}
	g := w.searching(work).Groups
	if len(g) != 1 || g[0].State != "searching" || g[0].Behind == nil || !g[0].Behind.Starved || deref(g[0].Behind.IndexerName) != "Fixture Site" {
		w.t.Fatalf("%s the movie's groups are %+v, want one searching behind Fixture Site starved", when, g)
	}
}

// TestATrackerRSSSyncUsesUpStarvesBacklogSearchAndKeepsItsLimit gives
// Fixture Site one search an hour with RSS sync every 15 minutes, so RSS
// sync keeps the whole limit. A movie queued from Search all reads
// Starved in Wanted's limits and in its group's behind, and still does
// after three hours, in which RSS sync reads the feed once an hour,
// skips at the limit between, and no hour holds more than one query.
func TestATrackerRSSSyncUsesUpStarvesBacklogSearchAndKeepsItsLimit(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launch(t, switches(false, true), limit(1))
		synctest.Wait()
		work := w.add(lanterns, "")
		w.searchAll(work)
		w.expectStarved("once queued", work)

		time.Sleep(3 * time.Hour)
		synctest.Wait()
		w.expectStarved("three hours on", work)
		qs := w.queries(w.indexer())
		if most := mostInAnHour(qs); most != 1 || len(qs) < 3 {
			t.Fatalf("Fixture Site's queries are %+v, %d in one hour, want one read an hour and none past its limit of 1", qs, most)
		}
		if slices.ContainsFunc(qs, func(q query) bool { return q.Use == "search" }) {
			t.Fatalf("Fixture Site's queries are %+v, want RSS sync's alone", qs)
		}
		var skips int
		for _, r := range w.runs(rssTask) {
			if r.Result == "skipped" && *r.Units[0].Cause == "Skipped because Fixture Site is at its search limit until "+qs[len(qs)-1].ReservedAt.Add(time.Hour).Format("15:04")+"." {
				skips++
			}
		}
		if skips == 0 {
			t.Fatalf("RSS sync's runs are %+v, want skips at the limit until an hour after its last read", w.runs(rssTask))
		}
	})
}

// TestATrackerWithTwoSearchesAnHourSearchesUnderRSSSync gives Fixture
// Site two searches an hour with RSS sync on. RSS sync's reserve leaves
// one, so the movie queued from Search all is searched within the hour,
// and no hour holds more than two queries.
func TestATrackerWithTwoSearchesAnHourSearchesUnderRSSSync(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launch(t, switches(false, true), limit(2))
		synctest.Wait()
		work := w.add(lanterns, "")
		w.searchAll(work)
		queued := time.Now()
		var row wantedRow
		simtest.Eventually(t, "the movie's run finds nothing", func() bool {
			row = w.wantedRow(work)
			return row.Last != nil && *row.Last == "nothing_found"
		})
		if took := time.Since(queued); took > time.Hour {
			t.Fatalf("the movie's run ended %s after Search all, want within the hour", took)
		}
		time.Sleep(3 * time.Hour)
		synctest.Wait()
		qs := w.queries(w.indexer())
		if most := mostInAnHour(qs); most > 2 {
			t.Fatalf("Fixture Site's queries are %+v, %d in one hour, want at most its limit of 2", qs, most)
		}
	})
}
