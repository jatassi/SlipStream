package e2e

import (
	"net/http"
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
			ID        string `json:"id"`
			CreatedAt string `json:"createdAt"`
		} `json:"indexers"`
	}
	reply := api(t, run).Admin("home", http.MethodGet, "/api/v1/indexers", "")
	decode(t, "listIndexers", reply, &listed)
	ix := listed.Indexers[0]
	worldtest.Expect(t, "listIndexers", reply, worldtest.Reply{Status: 200, Body: `{"indexers":[{"id":"` + ix.ID + `","name":"Fixture Site","protocol":"torznab",` +
		`"url":"` + site + `/torznab","homeNodeId":"` + m.Nodes[1].NodeID + `","apiKeySet":true,` +
		`"searchModes":[{"mode":"movie","params":["q","tmdbid","imdbid"]},{"mode":"search","params":["q"]}],"createdAt":"` + ix.CreatedAt + `"}]}` + "\n"})
}
