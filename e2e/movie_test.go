package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const movieType = `{"id":"movie","name":"Movies","enabled":true,"work":{"one":"Movie","many":"Movies"},"levels":[],"item":{"one":"Movie","many":"Movies"},` +
	`"fields":[{"key":"year","label":"Year","tier":"work","kind":"number"},{"key":"released","label":"Released","tier":"work","kind":"date"},` +
	`{"key":"runtime","label":"Runtime","tier":"work","kind":"duration"}]}`

func TestAMovieIsAddedMonitoredFromTheCatalog(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	a := api(t, run)

	worldtest.Expect(t, "searchCatalog for quiet orchard", a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query=quiet+orchard", ""), worldtest.Reply{
		Status: 200,
		Body:   `{"candidates":[{"key":"9000001","label":"The Quiet Orchard","fields":[{"key":"year","kind":"number","number":2021}],"workId":null}]}` + "\n",
	})

	first := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created {
		t.Fatalf("addWork created %t, work %s: %v", first.Created, first.Work, err)
	}
	var ids []string
	checkOutline(t, "the added movie", tr, &ids, "#1 movie The Quiet Orchard (The Quiet Orchard) 9000001 [year=2021 released=2021-04-16 runtime=6240000ms]")
	if len(tr.Items) != 1 || tr.Items[0].Key != "movie" || tr.Items[0].Label != "The Quiet Orchard" || !tr.Items[0].Monitored || tr.Items[0].LevelID != nil {
		t.Fatalf("items = %+v, want one monitored Item movie under the Work", tr.Items)
	}
}
