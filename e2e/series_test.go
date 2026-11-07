package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const (
	lanternKeepers = "1001"
	harborNights   = "1002"
	signalNine     = "1003"
)

const seriesType = `{"id":"series","name":"TV","icon":"tv","enabled":true,"work":{"one":"Series","many":"Series"},"levels":[{"one":"Season","many":"Seasons"}],"aside":{"one":"Special","many":"Specials"},"item":{"one":"Episode","many":"Episodes"},"fields":[` +
	`{"key":"years","label":"Years","tier":"work","kind":"text","in":["facts"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"year","label":"First aired year","tier":"work","kind":"number","in":["details"],"list":{"column":true,"sort":true,"filter":true},"with":null,"art":null},` +
	`{"key":"network","label":"Network","tier":"work","kind":"text","in":["facts","about"],"list":{"column":true,"sort":true,"filter":true},"with":null,"art":"network_logo"},` +
	`{"key":"certification","label":"Certification","tier":"work","kind":"text","in":["facts","about"],"list":{"column":true,"sort":true,"filter":true},"with":"certification_country","art":null},` +
	`{"key":"certification_country","label":"Certification country","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"genres","label":"Genres","tier":"work","kind":"texts","in":["facts","about"],"list":{"column":false,"sort":false,"filter":true},"with":null,"art":null},` +
	`{"key":"status","label":"Production status","tier":"work","kind":"text","in":["facts","about"],"list":{"column":true,"sort":false,"filter":true},"with":null,"art":null},` +
	`{"key":"seasons","label":"Seasons","tier":"work","kind":"number","in":["about"],"list":{"column":true,"sort":true,"filter":false},"with":null,"art":null},` +
	`{"key":"episodes","label":"Episodes","tier":"work","kind":"number","in":["about"],"list":{"column":true,"sort":true,"filter":false},"with":null,"art":null},` +
	`{"key":"overview","label":"Overview","tier":"work","kind":"text","in":["overview"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"creators","label":"Created by","tier":"work","kind":"people","in":["credits"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"credits","label":"Cast","tier":"work","kind":"people","in":["credits"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"original_title","label":"Original title","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"original_language","label":"Original language","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"country","label":"Country","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"first_aired","label":"First aired","tier":"work","kind":"date","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"last_aired","label":"Last aired","tier":"work","kind":"date","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"next_aired","label":"Next airing","tier":"work","kind":"date","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"awards","label":"Awards","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"english_title","label":"English title","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"episode_order","label":"Episode order","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"tvdb_id","label":"TVDB id","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"tmdb_id","label":"TMDB id","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"imdb_id","label":"IMDb id","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"trailer_site","label":"Trailer site","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"trailer_key","label":"Trailer key","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"season","label":"Season","tier":"level","depth":1,"kind":"number","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"overview","label":"Overview","tier":"level","depth":1,"kind":"text","in":["overview"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"code","label":"Episode","tier":"item","kind":"text","in":["code"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"air_date","label":"Air date","tier":"item","kind":"date","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"runtime","label":"Runtime","tier":"item","kind":"duration","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"overview","label":"Overview","tier":"item","kind":"text","in":["overview"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"season","label":"Season","tier":"item","kind":"number","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"episode","label":"Episode number","tier":"item","kind":"number","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"absolute","label":"Absolute number","tier":"item","kind":"number","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"imdb_id","label":"IMDb id","tier":"item","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null}],` +
	`"artwork":[{"key":"poster","label":"Poster","tier":"work","shape":"poster"},{"key":"backdrop","label":"Backdrop","tier":"work","shape":"backdrop"},{"key":"title_treatment","label":"Title treatment","tier":"work","shape":"logo"},{"key":"network_logo","label":"Network logo","tier":"work","shape":"logo"},{"key":"season_poster","label":"Season poster","tier":"level","depth":1,"shape":"poster"},{"key":"still","label":"Still","tier":"item","shape":"backdrop"}],"source":"TVDB","items":"many","art":{"poster":"poster","backdrop":"backdrop","title":"title_treatment"},"ratings":["imdb","tmdb","rottentomatoes","metacritic"],"availability":"Next airing","releases":"air"}`

type seriesView struct {
	ID             string     `json:"id"`
	Label          string     `json:"label"`
	GoneUpstreamAt *string    `json:"goneUpstreamAt"`
	MonitorNew     bool       `json:"monitorNew"`
	MonitorAsides  bool       `json:"monitorAsides"`
	Fields         []field    `json:"fields"`
	Levels         []treeNode `json:"levels"`
	Items          []treeNode `json:"items"`
}

// seasons is v's levels in position order, each with its episodes' codes:
// + for monitored, and the level or episode marked when withdrawn.
func (v seriesView) seasons() string {
	var out []string
	for _, l := range v.Levels {
		head := l.Label
		if l.WithdrawnAt != nil {
			head += " withdrawn"
		}
		line := []string{head + ":"}
		for _, it := range v.Items {
			if it.LevelID == nil || *it.LevelID != l.ID {
				continue
			}
			episode := it.code()
			if it.Monitored {
				episode += "+"
			}
			if it.WithdrawnAt != nil {
				episode += " withdrawn"
			}
			line = append(line, episode)
		}
		out = append(out, strings.Join(line, " "))
	}
	return strings.Join(out, "\n")
}

// item is v's episode with TVDB id key.
func (v seriesView) item(t *testing.T, key string) treeNode {
	t.Helper()
	for _, it := range v.Items {
		if it.Key == key {
			return it
		}
	}
	t.Fatalf("%s has no episode %s", v.Label, key)
	return treeNode{}
}

func (v seriesView) level(id string) string {
	for _, l := range v.Levels {
		if l.ID == id {
			return l.Label
		}
	}
	return ""
}

func (v seriesView) text(key string) string {
	for _, f := range v.Fields {
		if f.Key == key {
			return f.Text
		}
	}
	return ""
}

func (n treeNode) code() string {
	for _, f := range n.Fields {
		if f.Key == "code" {
			return f.Text
		}
	}
	return ""
}

// seriesAlone is the series Scenario on its Controller alone, holding
// every Role.
func seriesAlone(t *testing.T) *world.Run {
	t.Helper()
	s, err := scenario.Load("series")
	if err != nil {
		t.Fatal(err)
	}
	s.Name, s.Nodes = "series-alone", []scenario.Node{{Name: "home", Setup: scenario.SetupController}}
	return worldtest.LaunchScenario(t, s)
}

func addSeries(t *testing.T, a *worldtest.API, key, preset string, asides bool) seriesView {
	t.Helper()
	body := fmt.Sprintf(`{"contentType":"series","key":%q,"preset":%q,"includeAsides":%t}`, key, preset, asides)
	got := decode(t, "addWork "+key+" with "+preset, a.Admin("home", http.MethodPost, "/api/v1/works", body), &struct {
		Created bool       `json:"created"`
		Work    seriesView `json:"work"`
	}{})
	if !got.Created {
		t.Fatalf("addWork %s created nothing", key)
	}
	return got.Work
}

func getSeries(t *testing.T, a *worldtest.API, id string) seriesView {
	t.Helper()
	return *decode(t, "getWork", a.Admin("home", http.MethodGet, "/api/v1/works/"+id, ""), &seriesView{})
}

func refreshSeries(t *testing.T, a *worldtest.API, v seriesView, outcome string) seriesView {
	t.Helper()
	worldtest.Expect(t, "refreshWorks "+v.Label, a.Admin("home", http.MethodPost, "/api/v1/works/refresh", `{"workIds":["`+v.ID+`"]}`),
		worldtest.Reply{Status: 200, Body: `{"outcomes":[{"workId":"` + v.ID + `",` + outcome + `}]}` + "\n"})
	return getSeries(t, a, v.ID)
}

func checkSeasons(t *testing.T, what string, v seriesView, want ...string) {
	t.Helper()
	if got := v.seasons(); got != strings.Join(want, "\n") {
		t.Fatalf("%s:\n%s\nwant\n%s", what, got, strings.Join(want, "\n"))
	}
}

type presetCount struct {
	Preset      string `json:"preset"`
	Monitors    int    `json:"monitors"`
	MonitorsNew bool   `json:"monitorsNew"`
}

func TestAddASeriesWithEachPreset(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)

	var preview struct {
		Presets struct {
			Asides  int           `json:"asides"`
			Choices []presetCount `json:"choices"`
		} `json:"presets"`
	}
	decode(t, "previewCatalogRecord", a.Admin("home", http.MethodGet, "/api/v1/content-types/series/catalog/"+lanternKeepers, ""), &preview)
	if preview.Presets.Asides != 2 {
		t.Fatalf("the preview counts %d specials, want 2", preview.Presets.Asides)
	}
	counts := map[string]presetCount{}
	for _, c := range preview.Presets.Choices {
		counts[c.Preset] = c
	}

	for _, c := range []struct {
		preset  string
		asides  bool
		seasons []string
	}{
		{"all", false, []string{"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01+ S02E02+ S02E03+", "Season 3: S03E01+ S03E02+ S03E03+", "Specials: S00E01 S00E02"}},
		{"all", true, []string{"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01+ S02E02+ S02E03+", "Season 3: S03E01+ S03E02+ S03E03+", "Specials: S00E01+ S00E02+"}},
		{"future", false, []string{"Season 1: S01E01 S01E02 S01E03", "Season 2: S02E01 S02E02 S02E03", "Season 3: S03E01 S03E02 S03E03", "Specials: S00E01 S00E02"}},
		{"missing", false, []string{"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01+ S02E02+ S02E03+", "Season 3: S03E01+ S03E02+ S03E03+", "Specials: S00E01 S00E02"}},
		{"existing", false, []string{"Season 1: S01E01 S01E02 S01E03", "Season 2: S02E01 S02E02 S02E03", "Season 3: S03E01 S03E02 S03E03", "Specials: S00E01 S00E02"}},
		{"first", true, []string{"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01 S02E02 S02E03", "Season 3: S03E01 S03E02 S03E03", "Specials: S00E01+ S00E02+"}},
		{"latest", false, []string{"Season 1: S01E01 S01E02 S01E03", "Season 2: S02E01 S02E02 S02E03", "Season 3: S03E01+ S03E02+ S03E03+", "Specials: S00E01 S00E02"}},
		{"none", false, []string{"Season 1: S01E01 S01E02 S01E03", "Season 2: S02E01 S02E02 S02E03", "Season 3: S03E01 S03E02 S03E03", "Specials: S00E01 S00E02"}},
	} {
		what := fmt.Sprintf("The Lantern Keepers added with %s, specials %t", c.preset, c.asides)
		added := addSeries(t, a, lanternKeepers, c.preset, c.asides)
		checkSeasons(t, what, added, c.seasons...)
		stored := getSeries(t, a, added.ID)
		monitored := strings.Count(stored.seasons(), "+")
		want := counts[c.preset]
		if c.asides {
			want.Monitors += preview.Presets.Asides
		}
		if monitored != want.Monitors || stored.MonitorNew != want.MonitorsNew || stored.MonitorAsides != c.asides {
			t.Fatalf("%s monitors %d, new %t, specials %t; the preview counted %+v", what, monitored, stored.MonitorNew, stored.MonitorAsides, want)
		}
		worldtest.Expect(t, "deleteWorks "+what, a.Admin("home", http.MethodPost, "/api/v1/works/delete", `{"workIds":["`+added.ID+`"],"files":"leave"}`),
			worldtest.Reply{Status: 200, Body: `{"plan":{"works":1,"files":0,"kept":0,"bytes":0,"downloading":0},"outcomes":[{"workId":"` + added.ID + `","outcome":"done"}]}` + "\n"})
	}

	signal := addSeries(t, a, signalNine, "future", false)
	checkSeasons(t, "Signal Nine added with future", signal, "Season 1: S01E01 S01E02+ S01E03+")
	var library struct {
		Rows []struct {
			ID          string         `json:"id"`
			Status      string         `json:"status"`
			Items       int            `json:"items"`
			AvailableOn string         `json:"availableOn"`
			Values      map[string]any `json:"values"`
		} `json:"rows"`
	}
	decode(t, "listLibrary for TV", a.Admin("home", http.MethodGet, "/api/v1/library?contentType=series", ""), &library)
	if len(library.Rows) != 1 {
		t.Fatalf("TV's Library lists %+v, want Signal Nine alone", library.Rows)
	}
	row := library.Rows[0]
	if row.ID != signal.ID || row.Items != 2 || row.AvailableOn != "2027-01-15" ||
		row.Status != "unreleased" || fmt.Sprint(row.Values) != "map[episodes:3 genres:[Animation Crime] network:Signal Nine seasons:1 status:Continuing year:2026]" {
		t.Fatalf("Signal Nine's Library row = %+v, want 2 unreleased Items, next airing 2027-01-15 and its listed fields", row)
	}
}

func TestRefreshAddsASeasonUnderMonitorNew(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)
	following := addSeries(t, a, lanternKeepers, "all", false)
	trying := addSeries(t, a, harborNights, "first", false)
	cue(t, run, "tvdb-add-season", `{"id":1001}`)
	cue(t, run, "tvdb-add-season", `{"id":1002}`)

	t.Run("on", func(t *testing.T) {
		checkSeasons(t, "The Lantern Keepers, monitoring new episodes, once refreshed", refreshSeries(t, a, following, `"outcome":"done"`),
			"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01+ S02E02+ S02E03+", "Season 3: S03E01+ S03E02+ S03E03+", "Season 4: S04E01+ S04E02+ S04E03+", "Specials: S00E01 S00E02")
	})
	t.Run("off", func(t *testing.T) {
		checkSeasons(t, "Harbor Nights, on its first season, once refreshed", refreshSeries(t, a, trying, `"outcome":"done"`),
			"Season 1: S01E01+ S01E02+", "Season 2: S02E01 S02E02", "Season 3: S03E01 S03E02")
	})
}

func TestSpecialMovesToSeason3AsTheSameEpisode(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)
	added := addSeries(t, a, lanternKeepers, "all", true)
	before := added.item(t, "100102")

	cue(t, run, "tvdb-move-episode", `{"id":100102,"season":3,"number":4}`)
	refreshed := refreshSeries(t, a, added, `"outcome":"done"`)
	checkSeasons(t, "The Lantern Keepers once A Winter Light moves to season 3", refreshed,
		"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01+ S02E02+ S02E03+", "Season 3: S03E01+ S03E02+ S03E03+ S03E04+", "Specials: S00E01+")
	after := refreshed.item(t, "100102")
	if after.ID != before.ID || after.Label != "A Winter Light" || after.LevelID == nil || refreshed.level(*after.LevelID) != "Season 3" {
		t.Fatalf("A Winter Light once moved = %+v in %s, want the same Item %s in Season 3", after, refreshed.level(*after.LevelID), before.ID)
	}
}

func TestRemovedEpisodeIsWithdrawn(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)
	added := addSeries(t, a, lanternKeepers, "all", false)
	before := added.item(t, "100122")

	cue(t, run, "tvdb-remove-episode", `{"id":100122}`)
	refreshed := refreshSeries(t, a, added, `"outcome":"done"`)
	checkSeasons(t, "The Lantern Keepers once Fog Signal is removed", refreshed,
		"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01+ S02E03+ S02E02+ withdrawn", "Season 3: S03E01+ S03E02+ S03E03+", "Specials: S00E01 S00E02")
	if after := refreshed.item(t, "100122"); after.ID != before.ID {
		t.Fatalf("Fog Signal once removed is %s, want the same Item %s", after.ID, before.ID)
	}
	var library struct {
		Rows []struct {
			Items int `json:"items"`
		} `json:"rows"`
	}
	decode(t, "listLibrary", a.Admin("home", http.MethodGet, "/api/v1/library?contentType=series", ""), &library)
	if len(library.Rows) != 1 || library.Rows[0].Items != 8 {
		t.Fatalf("the Library rows = %+v, want The Lantern Keepers counting 8 Items without Fog Signal", library.Rows)
	}
}

func TestOrderChangeKeepsEpisodes(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)
	added := addSeries(t, a, lanternKeepers, "all", true)
	keep := added.item(t, "100113")
	options := func(preview bool) worldtest.Reply {
		return a.Admin("home", http.MethodPut, "/api/v1/works/"+added.ID+"/options", fmt.Sprintf(`{"type":{"order":"dvd"},"preview":%t}`, preview))
	}

	var change struct {
		Withdraws int `json:"withdraws"`
	}
	decode(t, "a preview of DVD order", options(true), &change)
	if change.Withdraws != 4 {
		t.Fatalf("DVD order withdraws %d episodes, want the 4 it doesn't list", change.Withdraws)
	}
	if got := getSeries(t, a, added.ID); got.seasons() != added.seasons() {
		t.Fatalf("the preview changed the series:\n%s", got.seasons())
	}
	decode(t, "setWorkOptions to DVD order", options(false), &change)
	dvd := getSeries(t, a, added.ID)
	checkSeasons(t, "The Lantern Keepers in DVD order", dvd,
		"Season 1: S01E01+ S01E02+ S01E03+", "Season 2: S02E01+ S02E02+ S02E03+", "Specials: S00E01+",
		"Season 1 withdrawn:", "Season 2 withdrawn:", "Season 3 withdrawn: S03E01+ withdrawn S03E02+ withdrawn S03E03+ withdrawn", "Specials withdrawn: S00E02+ withdrawn")
	if moved := dvd.item(t, "100113"); moved.ID != keep.ID || moved.code() != "S01E02" || dvd.text("episode_order") != "DVD" {
		t.Fatalf("Salt and Oil in DVD order = %+v, %s order; want the same Item %s as S01E02", moved, dvd.text("episode_order"), keep.ID)
	}
}

func TestGoneFromTVDBKeepsEverything(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)
	added := addSeries(t, a, lanternKeepers, "all", false)

	cue(t, run, "tvdb-delete", `{"id":1001}`)
	gone := refreshSeries(t, a, added, `"outcome":"skipped","reason":"work_gone_upstream"`)
	if gone.GoneUpstreamAt == nil || gone.seasons() != added.seasons() || gone.text("overview") != added.text("overview") {
		t.Fatalf("The Lantern Keepers once TVDB dropped it reads goneUpstreamAt %v with\n%s\nwant it marked and kept as\n%s", gone.GoneUpstreamAt, gone.seasons(), added.seasons())
	}
}

func TestTokenExpiryRecovers(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)
	added := addSeries(t, a, lanternKeepers, "all", false)
	setFault(t, run, fault.Rule{ID: "logins", Target: "tvdb", Route: "login"}, "slow-body=1GiB/s")

	cue(t, run, "tvdb-change", `{"id":1001}`)
	expired, err := run.Cue(worldtest.Context(t), "tvdb-expire-tokens", json.RawMessage(`{}`))
	if err != nil || !strings.Contains(string(expired.Data), `"result":{"ended":1}`) {
		t.Fatalf("tvdb-expire-tokens = %s, %v; want the add's token ended", expired.Data, err)
	}
	refreshed := refreshSeries(t, a, added, `"outcome":"done"`)
	if want := "The Lantern Keepers is a fictional series. Revision 1."; refreshed.text("overview") != want {
		t.Fatalf("the refresh past the expired token reads overview %q, want %q", refreshed.text("overview"), want)
	}
	if logins := selected(t, run, events.Query{Kind: events.FaultFired, Match: map[string]string{"rule": "logins"}}); len(logins) != 1 {
		t.Fatalf("the refresh logged in %d times, want once", len(logins))
	}
	if p := provider(t, a, "tvdb"); p.State != "working" {
		t.Fatalf("TVDB after the token expired = %+v, want working", p)
	}
}

func TestEndedSeriesThatReturnsIsRefreshed(t *testing.T) {
	t.Parallel()
	run := seriesAlone(t)
	a := api(t, run)
	added := addSeries(t, a, harborNights, "all", false)
	refreshAll := func() taskRun {
		t.Helper()
		var queued struct {
			RunID int `json:"runId"`
		}
		r := a.Admin("home", http.MethodPost, "/api/v1/tasks/metadata-refresh/runs", "")
		if err := json.Unmarshal([]byte(r.Body), &queued); err != nil || r.Status != http.StatusAccepted {
			t.Fatalf("runTask metadata-refresh = %d %s, want 202 with a runId", r.Status, r.Body)
		}
		return refreshRun(t, a, queued.RunID)
	}
	if r := refreshAll(); *r.Result != "succeeded" {
		t.Fatalf("the first Metadata refresh run = %+v, want succeeded", r)
	}

	cue(t, run, "tvdb-add-season", `{"id":1002}`)
	if r := refreshAll(); *r.Result != "succeeded" || len(r.Units) != 1 || r.Units[0].Label != "Harbor Nights" {
		t.Fatalf("the Metadata refresh run after TVDB listed Harbor Nights as updated = %+v, want Harbor Nights refreshed", r)
	}
	returned := getSeries(t, a, added.ID)
	checkSeasons(t, "Harbor Nights once it returns", returned, "Season 1: S01E01+ S01E02+", "Season 2: S02E01+ S02E02+", "Season 3: S03E01+ S03E02+")
	if returned.text("status") != "Continuing" {
		t.Fatalf("Harbor Nights once it returns reads %q, want Continuing", returned.text("status"))
	}
}
