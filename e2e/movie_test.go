package e2e

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const movieType = `{"id":"movie","name":"Movies","icon":"film","enabled":true,"work":{"one":"Movie","many":"Movies"},"levels":[],"item":{"one":"Movie","many":"Movies"},` +
	`"fields":[{"key":"english_title","label":"English title","tier":"work","kind":"text"},{"key":"original_title","label":"Original title","tier":"work","kind":"text"},{"key":"original_language","label":"Original language","tier":"work","kind":"text"},` +
	`{"key":"year","label":"Year","tier":"work","kind":"number"},{"key":"release_date","label":"Release date","tier":"work","kind":"date"},{"key":"runtime","label":"Runtime","tier":"work","kind":"duration"},` +
	`{"key":"status","label":"Status","tier":"work","kind":"text"},{"key":"certification","label":"Certification","tier":"work","kind":"text"},{"key":"certification_country","label":"Certification country","tier":"work","kind":"text"},` +
	`{"key":"genres","label":"Genres","tier":"work","kind":"texts"},{"key":"tagline","label":"Tagline","tier":"work","kind":"text"},{"key":"overview","label":"Overview","tier":"work","kind":"text"},` +
	`{"key":"studio","label":"Studio","tier":"work","kind":"text"},{"key":"collection","label":"Collection","tier":"work","kind":"text"},{"key":"credits","label":"Cast and crew","tier":"work","kind":"people"},` +
	`{"key":"alternative_titles","label":"Alternative titles","tier":"work","kind":"texts"},{"key":"imdb_id","label":"IMDb id","tier":"work","kind":"text"},{"key":"trailer_site","label":"Trailer site","tier":"work","kind":"text"},` +
	`{"key":"trailer_key","label":"Trailer key","tier":"work","kind":"text"},{"key":"awards","label":"Awards","tier":"work","kind":"text"}],` +
	`"artwork":[{"key":"poster","label":"Poster","tier":"work","shape":"poster"},{"key":"backdrop","label":"Backdrop","tier":"work","shape":"backdrop"},` +
	`{"key":"title_treatment","label":"Title treatment","tier":"work","shape":"logo"},{"key":"studio_logo","label":"Studio logo","tier":"work","shape":"logo"}]}`

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
