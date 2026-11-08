package e2e

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// addShelves adds home's shelf-movies and shelf-series as a movie and a
// series root on home, as an Admin would, and answers their ids by folder.
// Each root's first library scan starts once its Node reports it ready.
func addShelves(t *testing.T, a *worldtest.API, m world.Manifest) map[string]string {
	t.Helper()
	ids := map[string]string{}
	for _, root := range []struct{ contentType, folder string }{{"movie", "shelf-movies"}, {"series", "shelf-series"}} {
		body, err := json.Marshal(map[string]string{"contentType": root.contentType, "nodeId": m.Nodes[0].NodeID, "path": filepath.Join(m.MediaDir, root.folder)})
		if err != nil {
			t.Fatal(err)
		}
		r := a.Admin("home", http.MethodPost, "/api/v1/roots", string(body))
		var made struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(r.Body), &made); r.Status != http.StatusCreated || err != nil {
			t.Fatalf("createRoot %s = %d %s, want 201", root.folder, r.Status, r.Body)
		}
		ids[root.folder] = made.ID
	}
	return ids
}
