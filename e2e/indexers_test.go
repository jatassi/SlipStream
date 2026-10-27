package e2e

import (
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
		`"waits":null,"session":null,"feedHost":"` + address.Host + `",` +
		`"searchModes":[{"mode":"movie","params":["q","tmdbid","imdbid"]},{"mode":"search","params":["q"]}],"revision":1,"createdAt":"` + ix.CreatedAt + `"}]}` + "\n"})
}
