package e2e

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/catalog"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/site/definitions"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// runTask runs task id now and waits for that run to succeed.
func (w *walked) runTask(t *testing.T, id string) {
	t.Helper()
	var detail struct {
		Task struct {
			LastRun *struct {
				ID     int    `json:"id"`
				Result string `json:"result"`
			} `json:"lastRun"`
		} `json:"task"`
	}
	path := "/api/v1/tasks/" + id
	before := 0
	if last := decode(t, "getTask", w.a.Admin("home", http.MethodGet, path, ""), &detail).Task.LastRun; last != nil {
		before = last.ID
	}
	if r := w.a.Admin("home", http.MethodPost, path+"/runs", ""); r.Status != http.StatusAccepted {
		t.Fatalf("runTask %s = %d %s, want 202", id, r.Status, r.Body)
	}
	worldtest.Until(t, id+" runs", func() (bool, error) {
		decode(t, "getTask", w.a.Admin("home", http.MethodGet, path, ""), &detail)
		return detail.Task.LastRun != nil && detail.Task.LastRun.ID > before, nil
	})
	if r := detail.Task.LastRun.Result; r != "succeeded" {
		t.Fatalf("%s %s, want succeeded", id, r)
	}
}

func (w *walked) cueCatalog(t *testing.T, name string) {
	t.Helper()
	if _, err := w.run.Cue(worldtest.Context(t), name, json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	w.runTask(t, "indexer-catalog-refresh")
}

func (w *walked) pinned(t *testing.T, id string) string {
	t.Helper()
	var v struct {
		Definition struct {
			Version string `json:"version"`
		} `json:"definition"`
	}
	return decode(t, "getIndexer", w.a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id, ""), &v).Definition.Version
}

// TestLarkspurMovesToACatalogUpdateOnlyWhenItLoads refreshes the catalog
// after an update of Larkspur that loads, which Larkspur moves to, and
// after one that doesn't, which leaves it on the version it runs and
// raises a Warn.
func TestLarkspurMovesToACatalogUpdateOnlyWhenItLoads(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	larkspur := w.indexers(t)["Larkspur"].ID
	first := w.pinned(t, larkspur)

	w.cueCatalog(t, catalog.Update.Name)
	moved := w.pinned(t, larkspur)
	if moved == first {
		t.Fatalf("after an update that loads Larkspur still runs the version from %s", first)
	}
	if items := healthItems(t, w.a, "indexers"); len(items) != 0 {
		t.Errorf("Health holds %+v after an update that loads, want nothing", items)
	}

	w.cueCatalog(t, catalog.BadUpdate.Name)
	if got := w.pinned(t, larkspur); got != moved {
		t.Errorf("after an update that doesn't load Larkspur runs the version from %s, want %s", got, moved)
	}
	it := healthItems(t, w.a, "indexers")["definition."+larkspur]
	if it.Level != "warn" || !strings.HasPrefix(it.Text, "The catalog's new version of Larkspur doesn't load. The Indexer keeps running the version from ") ||
		it.Action == nil || it.Action.Href != "/settings/indexers/"+larkspur {
		t.Errorf("the bad update's item is %+v, want a Warn linking to Larkspur", it)
	}
}

// TestACustomDefinitionRunsAnIndexerOnlyWhenItLoadsUnderItsOwnID pastes
// definitions: one that doesn't load and one with a catalog id are
// refused, and Quarry's under a new id runs an Indexer that passes its
// Test on home.
func TestACustomDefinitionRunsAnIndexerOnlyWhenItLoadsUnderItsOwnID(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	paste := func(yaml string) worldtest.Reply {
		b, err := json.Marshal(map[string]string{"yaml": yaml})
		if err != nil {
			t.Fatal(err)
		}
		return w.a.Admin("home", http.MethodPost, "/api/v1/indexer-catalog/custom", string(b))
	}
	read := func(file string) string {
		b, err := fs.ReadFile(definitions.FS, file)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	worldtest.Expect(t, "createCustomDefinition of Larkspur's third version", paste(read("larkspur-v3.yml")),
		problemBecause(http.StatusUnprocessableEntity, "definition_unloadable", "SlipStream can't run this definition yet", `It logs in with "twofactor", which SlipStream doesn't run.`))
	worldtest.Expect(t, "createCustomDefinition of Quarry", paste(read("quarry.yml")), worldtest.Reply{
		Status: http.StatusConflict, Body: `{"status":409,"code":"definition_id_taken","title":"The Indexer catalog already has a definition with this id"}` + "\n",
	})
	if r := paste(strings.Replace(read("quarry.yml"), "id: quarry", "id: quarry-mine", 1)); r.Status != http.StatusCreated {
		t.Fatalf("createCustomDefinition of quarry-mine = %d %s, want 201", r.Status, r.Body)
	}

	settings, _ := site.Credentials(w.s, site.Quarry, "home")
	draft, err := json.Marshal(map[string]any{
		"kind": "catalog", "name": "Quarry Mine", "definition": "quarry-mine", "homeNodeId": w.a.Node("home").NodeID,
		"on": true, "automaticSearch": true, "values": settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/indexers", string(draft)); r.Status != http.StatusCreated {
		t.Fatalf("createIndexer from quarry-mine = %d %s, want 201", r.Status, r.Body)
	}
}

// TestDeletingAnIndexerKeepsItsGrabsUnderItsTracker deletes the Indexer
// the first search grabbed from, whose page listed the Grab: the Grab keeps
// its Tracker, the Tracker stays, and the search still lists the line it
// answered.
func TestDeletingAnIndexerKeepsItsGrabsUnderItsTracker(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	s := w.searched(t, w.search(t))
	if s.GrabID == nil {
		t.Fatalf("the search ended %+v, want a Grab", s)
	}
	from := w.grabbedFrom(t, *s.GrabID)
	g := decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+*s.GrabID, ""), &trackedGrab{})
	if g.TrackerID == nil {
		t.Fatalf("Grab %s names no Tracker", *s.GrabID)
	}

	var page struct {
		Grabs []struct {
			ID string `json:"id"`
		} `json:"grabs"`
	}
	decode(t, "listIndexerGrabs", w.a.Admin("home", http.MethodGet, "/api/v1/indexers/"+from+"/grabs", ""), &page)
	if len(page.Grabs) != 1 || page.Grabs[0].ID != *s.GrabID {
		t.Errorf("the Indexer's page lists Grabs %+v, want the search's Grab %s", page.Grabs, *s.GrabID)
	}

	if r := w.a.Admin("home", http.MethodDelete, "/api/v1/indexers/"+from, ""); r.Status != http.StatusNoContent {
		t.Fatalf("deleteIndexer = %d %s, want 204", r.Status, r.Body)
	}
	after := decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+*s.GrabID, ""), &struct {
		IndexerID *string `json:"indexerId"`
		TrackerID *string `json:"trackerId"`
	}{})
	if after.IndexerID != nil || after.TrackerID == nil || *after.TrackerID != *g.TrackerID {
		t.Errorf("after the delete the Grab names Indexer %v and Tracker %v, want no Indexer and Tracker %s", after.IndexerID, after.TrackerID, *g.TrackerID)
	}
	var tracker trackerView
	decode(t, "getTracker", w.a.Admin("home", http.MethodGet, "/api/v1/trackers/"+*g.TrackerID, ""), &tracker)
	if tracker.ID != *g.TrackerID || slices.Contains(tracker.IndexerIDs, from) {
		t.Errorf("the Tracker is %+v after its Indexer's delete, want it kept without the Indexer", tracker)
	}
	if again := w.searched(t, s.ID); len(again.Indexers) != len(s.Indexers) {
		t.Errorf("after the delete the search lists %q, want its %d lines", again.lines(), len(s.Indexers))
	}
}
