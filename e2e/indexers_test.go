package e2e

import (
	"maps"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

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
		if e.Origin == "native" {
			continue
		}
		reasons[e.ID] = "runs"
		if e.Reason != nil {
			reasons[e.ID] = *e.Reason
		}
	}
	want := map[string]string{"broken": "it has no links", "cinder": "runs", "ferry": "runs", "larkspur": "runs", "marsh": "runs", "quarry": "runs"}
	if !maps.Equal(reasons, want) {
		t.Errorf("the catalog lists %v, want %v", reasons, want)
	}
}
