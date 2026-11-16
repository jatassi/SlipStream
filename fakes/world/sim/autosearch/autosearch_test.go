// Package autosearch_test is the simulation tier of Automatic search,
// Backlog search and RSS sync: what each task's runs say, what the queue
// reads behind a tight search limit, and what a Controller restarted on
// the same database does mid-search.
package autosearch_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/tasks"
)

const (
	backlogTask tasks.TaskID = "backlog-search"
	rssTask     tasks.TaskID = "rss-sync"
)

// lanterns is a movie of home's fake-tmdb that no Release names, so it
// stays wanted.
const lanterns = "9000101"

var inUTC = sim.Edit(func(_ string, e *node.Env) { e.Zone = time.UTC })

type world struct {
	*sim.World
	t *testing.T
}

func launch(t *testing.T, edits ...func(*scenario.Scenario)) world {
	t.Helper()
	s := simtest.Home(t)
	for _, e := range edits {
		e(&s)
	}
	w := sim.Launch(t, s, sim.WithWalk(), inUTC)
	return world{World: w, t: t}
}

func switches(backlog, rss bool) func(*scenario.Scenario) {
	return func(s *scenario.Scenario) {
		s.Library.AutomaticSearch = &scenario.LibraryAutomaticSearch{BacklogSearch: &backlog, RSSSync: &rss}
	}
}

func limit(count int) func(*scenario.Scenario) {
	return func(s *scenario.Scenario) {
		s.Library.Indexers[0].Limit = &scenario.LibraryLimit{Count: count, Per: "hour"}
	}
}

func (w world) admin(method, path, body string) worldtest.Reply {
	return w.API().Admin("home", method, "/api/v1"+path, body)
}

func (w world) indexer() string {
	w.t.Helper()
	return simtest.Decode[struct{ Indexers []struct{ ID string } }](w.t, "listIndexers", w.admin(http.MethodGet, "/indexers", "")).Indexers[0].ID
}

// add adds the movie of tmdb monitored, searching it as search says, none
// when empty, and returns its Work.
func (w world) add(tmdb, search string) string {
	w.t.Helper()
	body := fmt.Sprintf(`{"contentType":"movie","key":%q,"preset":"all"`, tmdb)
	if search != "" {
		body += fmt.Sprintf(`,"search":%q`, search)
	}
	return simtest.Decode[struct {
		Work struct{ ID string } `json:"work"`
	}](w.t, "addWork", w.admin(http.MethodPost, "/works", body+"}")).Work.ID
}

func (w world) settings(values string) {
	w.t.Helper()
	if r := w.admin(http.MethodPatch, "/search-settings", `{"values":`+values+`}`); r.Status != http.StatusOK {
		w.t.Fatalf("updateSearchSettings %s = %d %s", values, r.Status, r.Body)
	}
}

type unitRun struct {
	Label   string  `json:"label"`
	Node    string  `json:"node"`
	Outcome string  `json:"outcome"`
	Cause   *string `json:"cause"`
	Note    *string `json:"note"`
}

func (u unitRun) line() string {
	return strings.Join([]string{u.Label, u.Node, u.Outcome, deref(u.Cause), deref(u.Note)}, "|")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type taskRun struct {
	ID      int64     `json:"id"`
	Trigger string    `json:"trigger"`
	Result  string    `json:"result"`
	Units   []unitRun `json:"units"`
	Started time.Time `json:"startedAt"`
}

func (r taskRun) lines() []string {
	out := []string{r.Trigger + "|" + r.Result}
	for _, u := range r.Units {
		out = append(out, u.line())
	}
	return out
}

func (w world) runs(id tasks.TaskID) []taskRun {
	w.t.Helper()
	return simtest.Decode[struct {
		Runs []taskRun `json:"runs"`
	}](w.t, "getTask", w.admin(http.MethodGet, "/tasks/"+string(id), "")).Runs
}

func (w world) ran(id tasks.TaskID) taskRun {
	w.t.Helper()
	started := simtest.RunNow(w.t, w.API(), id)
	var run taskRun
	simtest.Eventually(w.t, fmt.Sprintf("%s's run %d ends", id, started), func() bool {
		i := slices.IndexFunc(w.runs(id), func(r taskRun) bool { return r.ID == started })
		if i < 0 {
			return false
		}
		run = w.runs(id)[i]
		return run.Result != ""
	})
	return run
}

func expect(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s:\n got %q\nwant %q", what, got, want)
	}
}

type limitUntil struct {
	IndexerName string     `json:"indexerName"`
	Until       *time.Time `json:"until"`
	Starved     bool       `json:"starved"`
}

type wantedRow struct {
	Work struct {
		ID string `json:"id"`
	} `json:"work"`
	Last   *string    `json:"lastResult"`
	NextAt *time.Time `json:"nextSearchAt"`
	State  string     `json:"state"`
}

type wantedPage struct {
	Status struct {
		Searching int          `json:"searching"`
		Queued    int          `json:"queued"`
		Limits    []limitUntil `json:"limits"`
	} `json:"status"`
	Rows []wantedRow `json:"rows"`
}

func (w world) wanted() wantedPage {
	w.t.Helper()
	return simtest.Decode[wantedPage](w.t, "listWanted", w.admin(http.MethodGet, "/wanted", ""))
}

func (w world) wantedRow(work string) wantedRow {
	w.t.Helper()
	page := w.wanted()
	i := slices.IndexFunc(page.Rows, func(r wantedRow) bool { return r.Work.ID == work })
	if i < 0 {
		w.t.Fatalf("Wanted lists no row of %s: %+v", work, page.Rows)
	}
	return page.Rows[i]
}

type group struct {
	State  string `json:"state"`
	Behind *struct {
		IndexerName *string    `json:"indexerName"`
		Until       *time.Time `json:"until"`
		Starved     bool       `json:"starved"`
	} `json:"behind"`
}

type workSearching struct {
	Groups []group `json:"groups"`
	Waits  []struct {
		Until time.Time `json:"until"`
	} `json:"waits"`
}

func (w world) searching(work string) workSearching {
	w.t.Helper()
	return simtest.Decode[workSearching](w.t, "getWorkSearching", w.admin(http.MethodGet, "/works/"+work+"/searching", ""))
}

type searchRun struct {
	Trigger *string `json:"trigger"`
	Outcome *string `json:"outcome"`
	Grabs   int     `json:"grabs"`
}

func (w world) searches(work string) []searchRun {
	w.t.Helper()
	return simtest.Decode[struct {
		Searches []searchRun `json:"searches"`
	}](w.t, "listSearches", w.admin(http.MethodGet, "/works/"+work+"/searches", "")).Searches
}

type query struct {
	Use        string    `json:"use"`
	ReservedAt time.Time `json:"reservedAt"`
}

// queries is every search and recent query of Indexer id, oldest first:
// the ones its search limit counts.
func (w world) queries(id string) []query {
	w.t.Helper()
	var out []query
	for offset := 0; ; {
		page := simtest.Decode[struct {
			Queries    []query `json:"queries"`
			NextOffset *int    `json:"nextOffset"`
		}](w.t, "listIndexerQueries", w.admin(http.MethodGet, fmt.Sprintf("/indexers/%s/queries?offset=%d", id, offset), ""))
		for _, q := range page.Queries {
			if q.Use == "search" || q.Use == "recent" {
				out = append(out, q)
			}
		}
		if page.NextOffset == nil {
			break
		}
		offset = *page.NextOffset
	}
	slices.Reverse(out)
	return out
}

func mostInAnHour(qs []query) int {
	most := 0
	for i, q := range qs {
		n := 0
		for _, later := range qs[i:] {
			if later.ReservedAt.Sub(q.ReservedAt) < time.Hour {
				n++
			}
		}
		most = max(most, n)
	}
	return most
}

// restartHome stops the Controller and starts it on the same data dir,
// listening for its Workers where it did.
func (w world) restartHome() {
	h := w.Node("home")
	link := h.Ready().Listeners.Link
	h.Stop()
	h.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_LINK_LISTEN="+link.String()) })
}
