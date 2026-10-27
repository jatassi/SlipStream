package e2e

import (
	"maps"
	"net/http"
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestHomeSeedsTheFixtureSiteIndexerOnTheSeedbox(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	m := run.Manifest()
	var site string
	for _, f := range m.Fronts {
		if f.Target == "fixture-site" && f.Node == "seedbox" {
			site = f.URL
		}
	}
	var listed struct {
		Indexers []struct {
			ID     string `json:"id"`
			Status struct {
				SuccessAt string `json:"successAt"`
			} `json:"status"`
			CreatedAt string `json:"createdAt"`
		} `json:"indexers"`
	}
	reply := api(t, run).Admin("home", http.MethodGet, "/api/v1/indexers", "")
	decode(t, "listIndexers", reply, &listed)
	ix := listed.Indexers[0]
	address, err := url.Parse(site)
	if err != nil {
		t.Fatal(err)
	}
	worldtest.Expect(t, "listIndexers", reply, worldtest.Reply{Status: 200, Body: `{"indexers":[{"id":"` + ix.ID + `","name":"Fixture Site","kind":"torznab",` +
		`"homeNodeId":"` + m.Nodes[1].NodeID + `","on":true,"automaticSearch":true,"rss":true,"allowFailover":false,"priority":25,"categories":null,` +
		`"searches":{"count":0,"per":"hour","used":0,"freeAt":null},"grabs":{"count":0,"per":"hour","used":0,"freeAt":null},` +
		`"status":{"state":"ok","cause":null,"text":null,"until":null,"since":null,"nodeId":null,"solverTried":false,"rung":0,"lastFailure":null,"successAt":"` + ix.Status.SuccessAt + `"},` +
		`"sentence":"Fixture Site is working.","fix":null,"signsIn":false,"trackerIds":[],"waits":null,"session":null,"feedHost":"` + address.Host + `",` +
		`"searchModes":[{"mode":"movie","params":["q","tmdbid","imdbid"]},{"mode":"search","params":["q"]}],"revision":1,"createdAt":"` + ix.CreatedAt + `"}]}` + "\n"})
}

func TestHomeDownloadsTheIndexerCatalogFromFakeCatalogAtStart(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "home")
	var front string
	for _, f := range run.Manifest().Fronts {
		if f.Target == "indexer-catalog" && f.Node == "home" {
			front = f.URL
		}
	}
	type entry struct {
		ID     string  `json:"id"`
		Origin string  `json:"origin"`
		Reason *string `json:"reason"`
	}
	var c struct {
		RefreshedAt *string `json:"refreshedAt"`
		Address     string  `json:"address"`
		IsDefault   bool    `json:"addressIsDefault"`
		Notice      *string `json:"notice"`
		Entries     []entry `json:"entries"`
	}
	a := api(t, run)
	worldtest.Until(t, "the Indexer catalog refresh downloads the catalog", func() (bool, error) {
		decode(t, "getIndexerCatalog", a.Admin("home", http.MethodGet, "/api/v1/indexer-catalog", ""), &c)
		return c.RefreshedAt != nil, nil
	})
	if c.Address != front+"/master" || !c.IsDefault || c.Notice != nil {
		t.Errorf("the catalog is at %s (default %t) with notice %v, want fake-catalog's default %s/master", c.Address, c.IsDefault, c.Notice, front)
	}
	reasons := map[string]string{}
	for _, e := range c.Entries {
		reasons[e.ID] = "runs"
		if e.Reason != nil {
			reasons[e.ID] = *e.Reason
		}
	}
	want := map[string]string{"broken": "it has no links", "cinder": "runs", "larkspur": "runs", "quarry": "runs"}
	if !maps.Equal(reasons, want) {
		t.Errorf("the catalog lists %v, want %v", reasons, want)
	}
}
