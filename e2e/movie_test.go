package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/identity"
)

const movieType = `{"id":"movie","name":"Movies","icon":"film","enabled":true,"work":{"one":"Movie","many":"Movies"},"levels":[],"item":{"one":"Movie","many":"Movies"},"fields":[` +
	`{"key":"english_title","label":"English title","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"original_title","label":"Original title","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"original_language","label":"Original language","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"year","label":"Year","tier":"work","kind":"number","in":["facts"],"list":{"column":true,"sort":true,"filter":true},"with":null,"art":null},` +
	`{"key":"release_date","label":"Release date","tier":"work","kind":"date","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"runtime","label":"Runtime","tier":"work","kind":"duration","in":["facts"],"list":{"column":true,"sort":true,"filter":false},"with":null,"art":null},` +
	`{"key":"status","label":"Status","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"certification","label":"Certification","tier":"work","kind":"text","in":["facts","about"],"list":{"column":true,"sort":true,"filter":true},"with":"certification_country","art":null},` +
	`{"key":"certification_country","label":"Certification country","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"genres","label":"Genres","tier":"work","kind":"texts","in":["facts","about"],"list":{"column":false,"sort":false,"filter":true},"with":null,"art":null},` +
	`{"key":"tagline","label":"Tagline","tier":"work","kind":"text","in":["tagline"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"overview","label":"Overview","tier":"work","kind":"text","in":["overview"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"studio","label":"Studio","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":"studio_logo"},` +
	`{"key":"collection","label":"Collection","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"credits","label":"Cast and crew","tier":"work","kind":"people","in":["credits"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"alternative_titles","label":"Alternative titles","tier":"work","kind":"texts","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"imdb_id","label":"IMDb id","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"trailer_site","label":"Trailer site","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"trailer_key","label":"Trailer key","tier":"work","kind":"text","in":["details"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null},` +
	`{"key":"awards","label":"Awards","tier":"work","kind":"text","in":["about"],"list":{"column":false,"sort":false,"filter":false},"with":null,"art":null}],` +
	`"artwork":[{"key":"poster","label":"Poster","tier":"work","shape":"poster"},{"key":"backdrop","label":"Backdrop","tier":"work","shape":"backdrop"},{"key":"title_treatment","label":"Title treatment","tier":"work","shape":"logo"},{"key":"studio_logo","label":"Studio logo","tier":"work","shape":"logo"}],"source":"TMDB","items":"one","art":{"poster":"poster","backdrop":"backdrop","title":"title_treatment"},"ratings":["imdb","tmdb","rottentomatoes","metacritic"]}`

// artworkURL is an artwork URL, whose token each install signs with its
// own key.
var artworkURL = regexp.MustCompile(`"/api/v1/artwork/[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+"`)

func artworkMasked(r worldtest.Reply) worldtest.Reply {
	r.Body = artworkURL.ReplaceAllString(r.Body, `"ART"`)
	return r
}

func TestAMovieIsAddedMonitoredFromTheCatalog(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	a := api(t, run)

	worldtest.Expect(t, "searchCatalog for quiet orchard", artworkMasked(a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query=quiet+orchard", "")), worldtest.Reply{
		Status: 200,
		Body:   `{"candidates":[{"key":"9000001","label":"The Quiet Orchard","original":"","overview":"","fields":[{"key":"year","kind":"number","number":2021}],"posterUrl":"ART","workId":null}]}` + "\n",
	})

	first := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created {
		t.Fatalf("addWork created %t, work %s: %v", first.Created, first.Work, err)
	}
	var ids []string
	checkOutline(t, "the added movie", tr, &ids, "#1 movie The Quiet Orchard (The Quiet Orchard) 9000001 [english_title=The Quiet Orchard original_title=The Quiet Orchard original_language=en year=2021 release_date=2021-04-16 runtime=6240000ms status=Released imdb_id=tt9000001]")
	if len(tr.Items) != 1 || tr.Items[0].Key != "movie" || tr.Items[0].Label != "The Quiet Orchard" || !tr.Items[0].Monitored || tr.Items[0].LevelID != nil {
		t.Fatalf("items = %+v, want one monitored Item movie under the Work", tr.Items)
	}
}

const saltRoad = "9000005"

const saltRoadFields = `[{"key":"english_title","kind":"text","text":"The Salt Road"},{"key":"original_title","kind":"text","text":"La Route du Sel"},{"key":"original_language","kind":"text","text":"fr"},` +
	`{"key":"year","kind":"number","number":2022},{"key":"release_date","kind":"date","date":"2022-09-14"},{"key":"runtime","kind":"duration","ms":6720000},{"key":"status","kind":"text","text":"Released"},` +
	`{"key":"certification","kind":"text","text":"PG-13"},{"key":"certification_country","kind":"text","text":"US"},{"key":"genres","kind":"texts","texts":["Drama","Adventure"]},` +
	`{"key":"tagline","kind":"text","text":"Every grain counts."},{"key":"overview","kind":"text","text":"Two sisters walk the salt road."},{"key":"studio","kind":"text","text":"Fennel Films"},{"key":"collection","kind":"text","text":"Salt Trilogy"},` +
	`{"key":"credits","kind":"people","people":[{"name":"Ines Varga","role":"Director","department":"Directing","photoUrl":"ART"},{"name":"Ola Brandt","role":"Screenplay","department":"Writing","photoUrl":"ART"},` +
	`{"name":"Mara Quell","role":"Ysolde","department":"Acting","photoUrl":"ART"},{"name":"Tove Lind","role":"Anouk","department":"Acting","photoUrl":"ART"}]},` +
	`{"key":"alternative_titles","kind":"texts","texts":["Salt Road"]},{"key":"imdb_id","kind":"text","text":"tt9000005"},{"key":"trailer_site","kind":"text","text":"YouTube"},{"key":"trailer_key","kind":"text","text":"sa1tR0adTr"}`

const awardsField = `,{"key":"awards","kind":"text","text":"1 win \u0026 3 nominations"}]`

const tmdbRating = `{"source":"tmdb","label":"TMDB","value":7.2,"scale":10,"votes":410,"fetchedAt":"AT"}`

const omdbRatings = `{"source":"imdb","label":"IMDb","value":7.1,"scale":10,"votes":12345,"fetchedAt":"AT"},{"source":"metacritic","label":"Metacritic","value":74,"scale":100,"votes":null,"fetchedAt":"AT"},` +
	`{"source":"rottentomatoes","label":"Rotten Tomatoes critics","value":88,"scale":100,"votes":null,"fetchedAt":"AT"}`

var fetchedAt = regexp.MustCompile(`"fetchedAt":"[^"]+"`)

type movieView struct {
	ID             string          `json:"id"`
	GoneUpstreamAt *string         `json:"goneUpstreamAt"`
	Fields         json.RawMessage `json:"fields"`
	Ratings        json.RawMessage `json:"ratings"`
	Artwork        []artRef        `json:"artwork"`
	Links          json.RawMessage `json:"links"`
}

type artRef struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

func (m movieView) fields() string {
	return artworkURL.ReplaceAllString(string(m.Fields), `"ART"`)
}

func (m movieView) ratings() string {
	return fetchedAt.ReplaceAllString(string(m.Ratings), `"fetchedAt":"AT"`)
}

func (m movieView) art(t *testing.T, kind string) string {
	t.Helper()
	i := slices.IndexFunc(m.Artwork, func(a artRef) bool { return a.Kind == kind })
	if i < 0 {
		t.Fatalf("the Work's artwork %+v has no %s", m.Artwork, kind)
	}
	return m.Artwork[i].URL
}

// moviesAlone is the movies Scenario on its Controller alone, holding
// every Role.
func moviesAlone(t *testing.T) *world.Run {
	t.Helper()
	s, err := scenario.Load("movies")
	if err != nil {
		t.Fatal(err)
	}
	s.Name, s.Nodes = "movies-alone", []scenario.Node{{Name: "home", Setup: scenario.SetupController}}
	return worldtest.LaunchScenario(t, s)
}

func addSaltRoad(t *testing.T, a *worldtest.API) movieView {
	t.Helper()
	got := decode(t, "addWork The Salt Road", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+saltRoad+`","monitor":true}`), &struct {
		Created bool      `json:"created"`
		Work    movieView `json:"work"`
	}{})
	if !got.Created {
		t.Fatal("addWork The Salt Road created nothing")
	}
	return got.Work
}

func setFault(t *testing.T, run *world.Run, r fault.Rule, spelled string) events.Record {
	t.Helper()
	f, err := fault.Parse(spelled)
	if err != nil {
		t.Fatal(err)
	}
	r.Fault = f
	rec, err := run.SetFault(worldtest.Context(t), r)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func cue(t *testing.T, run *world.Run, name, args string) {
	t.Helper()
	if _, err := run.Cue(worldtest.Context(t), name, json.RawMessage(args)); err != nil {
		t.Fatal(err)
	}
}

func selected(t *testing.T, run *world.Run, q events.Query) []events.Record {
	t.Helper()
	recs, err := events.Select(run.Manifest().RunDir, q)
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func opened(node, target string, after uint64) events.Query {
	return events.Query{Kind: "front.open", Node: node, Match: map[string]string{"target": target}, After: after}
}

func TestAMovieIsAddedWithItsWholeRecordFromThePreviewsOneFetch(t *testing.T) {
	t.Parallel()
	run := moviesAlone(t)
	a := api(t, run)

	worldtest.Expect(t, "searchCatalog for salt road", artworkMasked(a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query=salt+road", "")), worldtest.Reply{
		Status: 200,
		Body:   `{"candidates":[{"key":"9000005","label":"The Salt Road","original":"La Route du Sel","overview":"Two sisters walk the salt road.","fields":[{"key":"year","kind":"number","number":2022}],"posterUrl":"ART","workId":null}]}` + "\n",
	})

	preview := decode(t, "previewCatalogRecord", a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog/"+saltRoad, ""), &struct {
		InLibrary bool      `json:"inLibrary"`
		Work      movieView `json:"work"`
	}{})
	if preview.InLibrary || preview.Work.fields() != saltRoadFields+"]" || preview.Work.ratings() != "["+tmdbRating+"]" || len(preview.Work.Artwork) != 4 {
		t.Fatalf("the preview is in the library %t with fields\n%s\nratings %s and %d artwork\nwant out of it with\n%s]\n[%s] and 4", preview.InLibrary, preview.Work.fields(), preview.Work.ratings(), len(preview.Work.Artwork), saltRoadFields, tmdbRating)
	}
	if omdb := selected(t, run, opened("home", "omdb", 0)); len(omdb) != 0 {
		t.Fatalf("the preview reached OMDb: %+v", omdb)
	}

	set := setFault(t, run, fault.Rule{ID: "second-fetch", Target: "tmdb", Route: "movie"}, "status=500")
	added := addSaltRoad(t, a)
	if want := saltRoadFields + awardsField; added.fields() != want {
		t.Fatalf("the added movie's fields are\n%s\nwant\n%s", added.fields(), want)
	}
	if want := "[" + omdbRatings + "," + tmdbRating + "]"; added.ratings() != want {
		t.Fatalf("the added movie's ratings are\n%s\nwant\n%s", added.ratings(), want)
	}
	if fired := selected(t, run, events.Query{Kind: events.FaultFired, Match: map[string]string{"rule": "second-fetch"}}); len(fired) != 0 {
		t.Fatalf("the add fetched the movie from TMDB again: %+v", fired)
	}
	if omdb := selected(t, run, opened("home", "omdb", set.Seq)); len(omdb) != 1 {
		t.Fatalf("the add opened %d connections to OMDb, want 1", len(omdb))
	}
	stored := decode(t, "getWork", a.Admin("home", http.MethodGet, "/api/v1/works/"+added.ID, ""), &movieView{})
	if stored.fields() != added.fields() || stored.ratings() != added.ratings() {
		t.Fatalf("getWork reads fields\n%s\nratings %s\nwant what the add answered", stored.fields(), stored.ratings())
	}
	if want := saltRoadLinks; string(stored.Links) != want {
		t.Fatalf("getWork links to\n%s\nwant\n%s", stored.Links, want)
	}
}

const saltRoadLinks = `[{"key":"imdb","label":"IMDb","url":"https://www.imdb.invalid/title/tt9000005","kind":"page"},` +
	`{"key":"tmdb","label":"TMDB","url":"https://www.tmdb.invalid/movie/9000005","kind":"page"},` +
	`{"key":"trakt","label":"Trakt","url":"https://trakt.invalid/search/tmdb/9000005?id_type=movie","kind":"page"},` +
	`{"key":"trailer","label":"Trailer","url":"https://www.youtube.invalid/watch?v=sa1tR0adTr","kind":"trailer"}]`

type served struct {
	status      int
	cache, kind string
	body        []byte
}

// fetchArtwork GETs an artwork URL at size in session, as a browser
// loading an image does.
func fetchArtwork(t *testing.T, a *worldtest.API, session, url, size string) served {
	t.Helper()
	req, err := http.NewRequestWithContext(worldtest.Context(t), http.MethodGet, a.Node("home").URL+url+"?size="+size, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", identity.CookieName+"="+session)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return served{res.StatusCode, res.Header.Get("Cache-Control"), res.Header.Get("Content-Type"), body}
}

func (s served) width(t *testing.T, what string) int {
	t.Helper()
	c, _, err := image.DecodeConfig(bytes.NewReader(s.body))
	if err != nil {
		t.Fatalf("%s is %d %s, %d bytes that don't decode: %v", what, s.status, s.kind, len(s.body), err)
	}
	return c.Width
}

const immutable = "private, max-age=31536000, immutable"

func TestArtworkIsServedAtEachSizeAndNeverFromAHalfImage(t *testing.T) {
	t.Parallel()
	run := moviesAlone(t)
	a := api(t, run)
	session := a.SignIn("home")
	work := addSaltRoad(t, a)
	poster := work.art(t, "poster")

	width := 0
	for _, size := range []string{"small", "medium", "large", "original"} {
		got := fetchArtwork(t, a, session, poster, size)
		if got.status != http.StatusOK || got.kind != "image/jpeg" || got.cache != immutable {
			t.Fatalf("the poster at %s = %d %s %q, want 200 image/jpeg %q", size, got.status, got.kind, got.cache, immutable)
		}
		w := got.width(t, "the poster at "+size)
		if w <= width {
			t.Fatalf("the poster at %s is %d wide, want wider than the size below it at %d", size, w, width)
		}
		width = w
	}

	forged := poster[:len(poster)-4] + "AAAA"
	if got := fetchArtwork(t, a, session, forged, "small"); got.status != http.StatusNotFound || got.cache != "no-store" ||
		string(got.body) != `{"status":404,"code":"artwork_unavailable","title":"SlipStream has no such image to show","detail":"SlipStream didn't issue that artwork token"}`+"\n" {
		t.Fatalf("a forged token = %d %q %s, want 404 no-store artwork_unavailable", got.status, got.cache, got.body)
	}

	small := fetchArtwork(t, a, session, poster, "small")
	treatment := work.art(t, "title_treatment")
	setFault(t, run, fault.Rule{ID: "half-image", Target: "image-cdn", Route: "image"}, "truncate")
	if got := fetchArtwork(t, a, session, treatment, "small"); got.status != http.StatusNotFound || got.cache != "no-store" ||
		!strings.HasPrefix(string(got.body), `{"status":404,"code":"artwork_unavailable","title":"SlipStream has no such image to show","detail":"the image couldn't be fetched: `) ||
		!strings.Contains(string(got.body), `. SlipStream tries again later."`) {
		t.Fatalf("the title treatment cut off mid-answer = %d %q %s, want 404 no-store artwork_unavailable", got.status, got.cache, got.body)
	}
	if got := fetchArtwork(t, a, session, poster, "small"); got.status != http.StatusOK || !bytes.Equal(got.body, small.body) {
		t.Fatalf("the stored poster while image-cdn cuts answers off = %d, %d bytes; want 200 with the %d bytes it served before", got.status, len(got.body), len(small.body))
	}
	if _, err := run.ClearFault(worldtest.Context(t), "half-image"); err != nil {
		t.Fatal(err)
	}
	var whole served
	worldtest.Until(t, "the title treatment is fetched again past timings.metadata.artwork_retry", func() (bool, error) {
		whole = fetchArtwork(t, a, session, treatment, "small")
		return whole.status == http.StatusOK, nil
	})
	if whole.cache != immutable || whole.kind != "image/png" || whole.width(t, "the title treatment once whole") == 0 {
		t.Fatalf("the title treatment once whole = %q %s, want %q image/png", whole.cache, whole.kind, immutable)
	}
}

type taskRun struct {
	ID      int     `json:"id"`
	Result  *string `json:"result"`
	Summary string  `json:"summary"`
	Units   []struct {
		Label   string `json:"label"`
		Outcome string `json:"outcome"`
	} `json:"units"`
}

func refreshRun(t *testing.T, a *worldtest.API, id int) taskRun {
	t.Helper()
	var found taskRun
	worldtest.Until(t, fmt.Sprintf("Metadata refresh run %d ends", id), func() (bool, error) {
		var task struct {
			Runs []taskRun `json:"runs"`
		}
		decode(t, "getTask metadata-refresh", a.Admin("home", http.MethodGet, "/api/v1/tasks/metadata-refresh", ""), &task)
		i := slices.IndexFunc(task.Runs, func(r taskRun) bool { return r.ID == id })
		if i < 0 {
			return false, nil
		}
		found = task.Runs[i]
		return found.Result != nil, nil
	})
	return found
}

func TestAMovieRefreshesAloneAndWithEveryMovieAndKeepsItsRecordOnceGone(t *testing.T) {
	t.Parallel()
	run := moviesAlone(t)
	a := api(t, run)
	work := addSaltRoad(t, a)
	refresh := func(why, outcome string) {
		t.Helper()
		worldtest.Expect(t, why, a.Admin("home", http.MethodPost, "/api/v1/works/refresh", `{"workIds":["`+work.ID+`"]}`),
			worldtest.Reply{Status: 200, Body: `{"outcomes":[{"workId":"` + work.ID + `",` + outcome + `}]}` + "\n"})
	}

	cue(t, run, "tmdb-change", `{"id":9000005}`)
	revised := strings.Replace(saltRoadFields, "Two sisters walk the salt road.", "Two sisters walk the salt road. Revision 1.", 1) + awardsField
	recounted := strings.Replace(work.ratings(), `"votes":410`, `"votes":411`, 1)
	refresh("refreshWorks", `"outcome":"done"`)
	if got := decode(t, "getWork after refreshWorks", a.Admin("home", http.MethodGet, "/api/v1/works/"+work.ID, ""), &movieView{}); got.fields() != revised || got.ratings() != recounted {
		t.Fatalf("the refreshed movie reads fields\n%s\nratings %s\nwant\n%s\n%s", got.fields(), got.ratings(), revised, recounted)
	}

	var queued struct {
		RunID int `json:"runId"`
	}
	all := a.Admin("home", http.MethodPost, "/api/v1/content-types/movie/refresh", "")
	if err := json.Unmarshal([]byte(all.Body), &queued); err != nil || all.Status != http.StatusAccepted {
		t.Fatalf("refreshContentType = %d %s, want 202 with a runId", all.Status, all.Body)
	}
	if r := refreshRun(t, a, queued.RunID); *r.Result != "succeeded" || r.Summary != "1 Work" || len(r.Units) != 1 || r.Units[0].Label != "The Salt Road" || r.Units[0].Outcome != "succeeded" {
		t.Fatalf("refresh all's run = %+v, want succeeded with The Salt Road refreshed", r)
	}

	cue(t, run, "tmdb-delete", `{"id":9000005}`)
	refresh("refreshWorks once TMDB deleted it", `"outcome":"skipped","reason":"work_gone_upstream"`)
	gone := decode(t, "getWork once gone", a.Admin("home", http.MethodGet, "/api/v1/works/"+work.ID, ""), &movieView{})
	if gone.GoneUpstreamAt == nil || gone.fields() != revised || gone.ratings() != recounted {
		t.Fatalf("the gone movie reads goneUpstreamAt %v with fields\n%s\nratings %s\nwant it marked and its record kept", gone.GoneUpstreamAt, gone.fields(), gone.ratings())
	}
}

type providerView struct {
	ID      string  `json:"id"`
	Summary string  `json:"summary"`
	Key     string  `json:"key"`
	State   string  `json:"state"`
	Until   *string `json:"until"`
}

func provider(t *testing.T, a *worldtest.API, id string) providerView {
	t.Helper()
	var list struct {
		Providers []providerView `json:"providers"`
	}
	decode(t, "listMetadataProviders", a.Admin("home", http.MethodGet, "/api/v1/metadata/providers", ""), &list)
	i := slices.IndexFunc(list.Providers, func(p providerView) bool { return p.ID == id })
	if i < 0 {
		t.Fatalf("listMetadataProviders = %+v, want %s among them", list.Providers, id)
	}
	return list.Providers[i]
}

var rateLimited = regexp.MustCompile(`^\{"status":502,"code":"metadata_unavailable","title":"The metadata source didn't respond. Try again later.","detail":"TMDB: is rate limited until (\S+)Z"\}\n$`)

func TestAProviderFaultEndsInItsStatusAndProblem(t *testing.T) {
	t.Parallel()
	run := moviesAlone(t)
	a := api(t, run)
	search := func(query string) worldtest.Reply {
		return a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query="+query, "")
	}

	setFault(t, run, fault.Rule{ID: "down", Target: "tmdb"}, "refuse")
	if got := search("orchard"); got.Status != http.StatusBadGateway || !strings.Contains(got.Body, `"detail":"TMDB: didn't respond: `) {
		t.Fatalf("searchCatalog with TMDB refusing connections = %d %s, want 502 metadata_unavailable", got.Status, got.Body)
	}
	if p := provider(t, a, "tmdb"); p.State != "unreachable" {
		t.Fatalf("TMDB refusing connections = %+v, want unreachable", p)
	}
	if _, err := run.ClearFault(worldtest.Context(t), "down"); err != nil {
		t.Fatal(err)
	}
	worldtest.Until(t, "TMDB tests itself working again past timings.metadata.retest_min", func() (bool, error) {
		return provider(t, a, "tmdb").State == "working", nil
	})

	cue(t, run, "omdb-cap", `{}`)
	addSaltRoad(t, a)
	midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour).Format(time.RFC3339)
	if p := provider(t, a, "omdb"); p.State != "out_of_allowance" || p.Until == nil || *p.Until != midnight || p.Summary != "OMDb is out of today's allowance. It resets automatically." {
		t.Fatalf("OMDb once capped = %+v, want out_of_allowance until %s", p, midnight)
	}

	cue(t, run, "tmdb-revoke-key", `{}`)
	worldtest.Expect(t, "searchCatalog with the key revoked", search("orchard"), problemBecause(502, "metadata_refused", "The metadata source refused SlipStream's key", "TMDB refused SlipStream's key"))
	if p := provider(t, a, "tmdb"); p.State != "key_refused" || p.Key != "none" || p.Summary != "This build has no TMDB key. Add your own." {
		t.Fatalf("TMDB with its key revoked = %+v, want key_refused with no key in this build", p)
	}
	saved := a.Admin("home", http.MethodPut, "/api/v1/metadata/providers/tmdb/key", `{"key":"ada-own-key"}`)
	var p providerView
	if err := json.Unmarshal([]byte(saved.Body), &p); err != nil || saved.Status != http.StatusOK || p.State != "working" || p.Key != "yours" {
		t.Fatalf("setMetadataKey = %d %s, want 200 working on yours", saved.Status, saved.Body)
	}
	if got := search("orchard"); got.Status != http.StatusOK {
		t.Fatalf("searchCatalog on the Admin's key = %d %s, want 200", got.Status, got.Body)
	}

	setFault(t, run, fault.Rule{ID: "limited", Target: "tmdb", Route: "search"}, "status=429,retry-after=30s")
	got := search("quiet")
	m := rateLimited.FindStringSubmatch(got.Body)
	if got.Status != http.StatusBadGateway || m == nil {
		t.Fatalf("searchCatalog answered 429 = %d %s, want 502 metadata_unavailable rate limited until a time", got.Status, got.Body)
	}
	if p := provider(t, a, "tmdb"); p.State != "rate_limited" || p.Until == nil || !strings.HasPrefix(*p.Until, m[1]) {
		t.Fatalf("TMDB once limited = %+v, want rate_limited until %sZ", p, m[1])
	}
	if again := search("harbor"); again.Status != http.StatusBadGateway {
		t.Fatalf("searchCatalog inside the limit = %d %s, want 502", again.Status, again.Body)
	}
	if fired := selected(t, run, events.Query{Kind: events.FaultFired, Match: map[string]string{"rule": "limited"}}); len(fired) != 1 {
		t.Fatalf("TMDB saw %d searches inside the limit, want only the first", len(fired))
	}
}

func TestMetadataOnAWorkerCallsFromItsFrontAndLeavesNoKeyOnItsDisk(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "movies")
	a := api(t, run)
	m := run.Manifest()
	home, seedbox := m.Nodes[0], m.Nodes[1]
	a.SetRoles("home", seedbox.NodeID, "searching", "downloading", "metadata")
	a.SetRoles("home", home.NodeID, "storage", "converting", "subtitles", "media-server-link", "notifying", "serving")
	cut := setFault(t, run, fault.Rule{ID: "home-cut-off", Target: "tmdb", Origin: "home"}, "refuse")

	const key = "seedbox-never-keeps-this"
	saved := a.Admin("home", http.MethodPut, "/api/v1/metadata/providers/tmdb/key", `{"key":"`+key+`"}`)
	if saved.Status != http.StatusOK || !strings.Contains(saved.Body, `"nodeId":"`+seedbox.NodeID+`","reached":true`) {
		t.Fatalf("setMetadataKey = %d %s, want 200 tested from seedbox", saved.Status, saved.Body)
	}
	addSaltRoad(t, a)
	if from := selected(t, run, opened("seedbox", "tmdb", cut.Seq)); len(from) == 0 {
		t.Fatal("seedbox opened no connection to TMDB, want the test and the add to leave from it")
	}
	if from := selected(t, run, opened("home", "tmdb", cut.Seq)); len(from) != 0 {
		t.Fatalf("home opened %d connections to TMDB once seedbox held Metadata, want none", len(from))
	}

	setFault(t, run, fault.Rule{ID: "seedbox-cut-off", Target: "tmdb", Origin: "seedbox"}, "status=404")
	if got := a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query=orchard", ""); got.Status != http.StatusBadGateway || !strings.Contains(got.Body, `"code":"metadata_unavailable"`) {
		t.Fatalf("searchCatalog with seedbox's TMDB front failing = %d %s, want 502 metadata_unavailable", got.Status, got.Body)
	}

	act(t, run, "seedbox", world.StopNode)
	if err := filepath.WalkDir(seedbox.DataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		if d.Name() == "library.db" {
			t.Errorf("seedbox keeps a library at %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read seedbox's %s: %w", path, err)
		}
		if bytes.Contains(data, []byte(key)) {
			t.Errorf("seedbox keeps the TMDB key in %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
