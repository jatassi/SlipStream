package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// searchingSeries is the series Scenario on its Controller alone, holding
// every Role, with The Lantern Keepers added monitoring every episode and
// its specials.
func searchingSeries(t *testing.T) (*walked, seriesView) {
	t.Helper()
	s, err := scenario.Load("series")
	if err != nil {
		t.Fatal(err)
	}
	s.Name, s.Nodes = "series-alone", []scenario.Node{{Name: "home", Setup: scenario.SetupController}}
	run := worldtest.LaunchScenario(t, s)
	w := &walked{s: s, run: run, m: run.Manifest(), a: api(t, run)}
	v := addSeries(t, w.a, lanternKeepers, "all", true)
	w.work = v.ID
	return w, v
}

// searchWithin searches w's Work within body's levels and Items and
// returns the runs, which must be n.
func (w *walked) searchWithin(t *testing.T, body string, n int) []string {
	t.Helper()
	reply := w.a.Admin("home", http.MethodPost, "/api/v1/works/"+w.work+"/search", body)
	var started struct {
		SearchIDs []string `json:"searchIds"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &started); err != nil || reply.Status != http.StatusAccepted || len(started.SearchIDs) != n {
		t.Fatalf("searchWork %s = %d %s, want 202 with %d searches", body, reply.Status, reply.Body, n)
	}
	return started.SearchIDs
}

func itemsBody(ids ...string) string {
	return `{"levelIds":[],"itemIds":["` + strings.Join(ids, `","`) + `"]}`
}

// episodeRun is a finished search run of one episode and what it refused.
type episodeRun struct {
	ItemID  string  `json:"itemId"`
	Outcome *string `json:"outcome"`
	GrabID  *string `json:"grabId"`
	Refused []struct {
		Title   string `json:"title"`
		Reasons []struct {
			Code  string `json:"code"`
			Value string `json:"value"`
		} `json:"reasons"`
	} `json:"refused"`
	Indexers []lineView `json:"indexers"`
}

// refusals is each refused title with its first reason and that reason's
// value.
func (r episodeRun) refusals() []string {
	var out []string
	for _, f := range r.Refused {
		out = append(out, f.Title+" "+f.Reasons[0].Code+" "+f.Reasons[0].Value)
	}
	return out
}

// finished reads w's Work's searches until every run of ids finished.
func (w *walked) finished(t *testing.T, ids []string) map[string]episodeRun {
	t.Helper()
	out := map[string]episodeRun{}
	worldtest.Until(t, "the searches finish", func() (bool, error) {
		var list struct {
			Searches []struct {
				ID string `json:"id"`
				episodeRun
			} `json:"searches"`
		}
		decode(t, "listSearches", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/searches", ""), &list)
		for _, s := range list.Searches {
			if slices.Contains(ids, s.ID) && s.Outcome != nil {
				out[s.ID] = s.episodeRun
			}
		}
		return len(out) == len(ids), nil
	})
	return out
}

// tvSearches are the tvsearch and free-text searches fixture-site served,
// each as its site, mode and parameters, sorted. An Indexer's test is a
// free-text search with no query, left out.
func tvSearches(t *testing.T, w *walked) []string {
	t.Helper()
	var out []string
	for _, rec := range selected(t, w.run, events.Query{Kind: site.Searched.Kind}) {
		var s site.Search
		if err := json.Unmarshal(rec.Data, &s); err != nil {
			t.Fatal(err)
		}
		if s.T == "tvsearch" || s.T == "search" && s.Q != "" {
			out = append(out, fmt.Sprintf("%s t=%s q=%q tvdbid=%s season=%q ep=%s", s.Site, s.T, s.Q, s.TVDBID, s.Season, s.Ep))
		}
	}
	slices.Sort(out)
	return out
}

// filed is each Item of work holding a library file in a cell.
func filed(t *testing.T, w *walked) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, c := range decode(t, "getWorkCells", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), &cellList{}).Cells {
		if c.File != nil {
			out[c.ItemID] = true
		}
	}
	return out
}

// imported waits for each run's Grab to be imported.
func (w *walked) imported(t *testing.T, runs map[string]episodeRun) {
	t.Helper()
	for id, r := range runs {
		if r.GrabID == nil {
			t.Fatalf("search %s of %s grabbed nothing: %+v", id, r.ItemID, r)
		}
		w.ended(t, *r.GrabID)
		if w.grab.Stage != "imported" {
			t.Fatalf("the Grab of %s ended %s with %+v, want imported", r.ItemID, w.grab.Stage, w.grab.Failure)
		}
	}
}

func TestEpisodeSearchSendsSeasonAndEp(t *testing.T) {
	t.Parallel()
	w, v := searchingSeries(t)
	s01e03, special := v.item(t, "100113").ID, v.item(t, "100101").ID
	runs := w.finished(t, w.searchWithin(t, itemsBody(s01e03, special), 2))

	if got, want := tvSearches(t, w), []string{
		`torznab t=tvsearch q="" tvdbid=1001 season="0" ep=1`,
		`torznab t=tvsearch q="" tvdbid=1001 season="1" ep=3`,
	}; !slices.Equal(got, want) {
		t.Fatalf("fixture-site served %q, want one tvsearch per episode by TVDB id, season and ep, season 0 for the special", got)
	}
	for _, r := range runs {
		want := []string{
			"The.Lantern.Keepers.S01E02.720p.WEB-DL.H264-FKW wrong_item S01E02",
			"The.Lantern.Keepers.S01.1080p.WEB-DL.H264-FKW covers_more S01",
		}
		if r.ItemID == special {
			want = nil
		}
		if r.Outcome == nil || *r.Outcome != "grabbed" || !slices.Equal(r.refusals(), want) {
			t.Fatalf("the search of %s ended %v refusing %q, want grabbed refusing %q", r.ItemID, r.Outcome, r.refusals(), want)
		}
	}
	w.imported(t, runs)
	if got := filed(t, w); len(got) != 2 || !got[s01e03] || !got[special] {
		t.Fatalf("the episodes with files are %v, want S01E03 %s and S00E01 %s", got, s01e03, special)
	}
}

func TestLevelSearchSearchesEachWantedEpisode(t *testing.T) {
	t.Parallel()
	w, v := searchingSeries(t)
	var season1 string
	for _, l := range v.Levels {
		if l.Label == "Season 1" {
			season1 = l.ID
		}
	}
	if r := w.a.Admin("home", http.MethodPut, "/api/v1/works/"+v.ID+"/monitored", `{"levelIds":[],"itemIds":["`+v.item(t, "100112").ID+`"],"on":false}`); r.Status != http.StatusNoContent {
		t.Fatalf("setWorkMonitored S01E02 off = %d %s, want 204", r.Status, r.Body)
	}
	var listed struct {
		Indexers []struct {
			ID string `json:"id"`
		} `json:"indexers"`
	}
	decode(t, "listIndexers", w.a.Admin("home", http.MethodGet, "/api/v1/indexers", ""), &listed)
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/indexers/"+listed.Indexers[0].ID, `{"searches":{"count":2,"per":"hour"}}`); r.Status != http.StatusOK {
		t.Fatalf("updateIndexer = %d %s, want 200", r.Status, r.Body)
	}

	runs := w.finished(t, w.searchWithin(t, `{"levelIds":["`+season1+`"],"itemIds":[]}`, 2))
	searched := map[string]bool{}
	for _, r := range runs {
		searched[r.ItemID] = true
	}
	if !searched[v.item(t, "100111").ID] || !searched[v.item(t, "100113").ID] {
		t.Fatalf("Season 1's Search searched %v, want S01E01 and S01E03, the wanted episodes, and not S01E02", searched)
	}
	if got, want := tvSearches(t, w), []string{
		`torznab t=tvsearch q="" tvdbid=1001 season="1" ep=1`,
		`torznab t=tvsearch q="" tvdbid=1001 season="1" ep=3`,
	}; !slices.Equal(got, want) {
		t.Fatalf("fixture-site served %q, want one tvsearch per wanted episode", got)
	}

	limited := w.finished(t, w.searchWithin(t, "", 8))
	var lines []string
	for _, r := range limited {
		lines = append(lines, r.Indexers[0].Name+" "+*r.Indexers[0].Outcome)
	}
	if n := len(tvSearches(t, w)); n != 2 || slices.ContainsFunc(lines, func(l string) bool { return l != "Fixture Site limit_reached" }) {
		t.Fatalf("the series' Search of its 8 other wanted episodes ran lines %q with %d tvsearches served, want each held at the Indexer's limit of 2 searches an hour", lines, n)
	}
}

func TestAnotherSeriesTVDBIDIsTheWrongWork(t *testing.T) {
	t.Parallel()
	w, v := searchingSeries(t)
	body := `{"kind":"torznab","name":"Fixture Words","address":"https://` + site.Host(site.Prowlarr) + site.TorznabPath + `","apiKey":"` + site.APIKey(w.s) + `","on":true,"automaticSearch":true,"priority":10}`
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/indexers", body); r.Status != http.StatusCreated {
		t.Fatalf("createIndexer = %d %s, want 201", r.Status, r.Body)
	}
	runs := w.finished(t, w.searchWithin(t, itemsBody(v.item(t, "100113").ID), 1))

	if got, want := tvSearches(t, w), []string{
		`prowlarr t=search q="The Lantern Keepers S01E03" tvdbid= season="" ep=`,
		`torznab t=tvsearch q="" tvdbid=1001 season="1" ep=3`,
	}; !slices.Equal(got, want) {
		t.Fatalf("fixture-site served %q, want the free-text search naming the episode on the site without TV ids", got)
	}
	for _, r := range runs {
		if r.Outcome == nil || *r.Outcome != "grabbed" || !slices.Contains(r.refusals(), "The.Lantern.Keepers.S01E03.Proper.720p.WEB-DL.H264-NTB wrong_work ") {
			t.Fatalf("the search ended %v refusing %q, want grabbed with Harbor Nights' release named S01E03 refused as wrong_work", r.Outcome, r.refusals())
		}
	}
}

// TestEpisodeFilesStayWithTheirEpisodes grabs A Winter Light, a special,
// and Salt and Oil, S01E03, then moves the special into season 3 and
// changes the order to DVD, which renumbers Salt and Oil S01E02 and
// doesn't list the special: each keeps its file, and deleting the series
// counts both.
func TestEpisodeFilesStayWithTheirEpisodes(t *testing.T) {
	t.Parallel()
	w, v := searchingSeries(t)
	special, salt := v.item(t, "100102").ID, v.item(t, "100113").ID
	w.imported(t, w.finished(t, w.searchWithin(t, itemsBody(special, salt), 2)))
	both := func(when string) {
		t.Helper()
		if got := filed(t, w); len(got) != 2 || !got[special] || !got[salt] {
			t.Fatalf("%s the episodes with files are %v, want A Winter Light %s and Salt and Oil %s", when, got, special, salt)
		}
	}
	both("once grabbed")

	cue(t, w.run, "tvdb-move-episode", `{"id":100102,"season":3,"number":4}`)
	moved := refreshSeries(t, w.a, v, `"outcome":"done"`)
	if at := moved.item(t, "100102"); at.ID != special || at.code() != "S03E04" {
		t.Fatalf("A Winter Light once moved is %+v, want the same Item as S03E04", at)
	}
	both("once A Winter Light moved to season 3")

	if r := w.a.Admin("home", http.MethodPut, "/api/v1/works/"+v.ID+"/options", `{"type":{"order":"dvd"},"preview":false}`); r.Status != http.StatusOK {
		t.Fatalf("setWorkOptions to DVD order = %d %s, want 200", r.Status, r.Body)
	}
	dvd := getSeries(t, w.a, v.ID)
	if at := dvd.item(t, "100113"); at.ID != salt || at.code() != "S01E02" {
		t.Fatalf("Salt and Oil in DVD order is %+v, want the same Item as S01E02", at)
	}
	if at := dvd.item(t, "100102"); at.WithdrawnAt == nil {
		t.Fatalf("A Winter Light in DVD order is %+v, want it withdrawn, since DVD order doesn't list it", at)
	}
	both("in DVD order")

	worldtest.Expect(t, "deleteWorks preview", w.a.Admin("home", http.MethodPost, "/api/v1/works/delete", `{"workIds":["`+v.ID+`"],"files":"delete","preview":true}`),
		worldtest.Reply{Status: 200, Body: `{"plan":{"works":1,"files":2,"kept":0,"bytes":` + strconv.FormatInt(filedBytes(t, w), 10) + `,"downloading":0},"outcomes":[{"workId":"` + v.ID + `","outcome":"done"}]}` + "\n"})
}

// filedBytes is the size of w's Work's library files.
func filedBytes(t *testing.T, w *walked) int64 {
	t.Helper()
	sizes := map[string]int64{}
	for _, c := range decode(t, "getWorkCells", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), &cellList{}).Cells {
		if c.File != nil {
			sizes[c.File.ID] = c.File.Size
		}
	}
	var total int64
	for _, s := range sizes {
		total += s
	}
	return total
}
