package e2e

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

// TestANativeIndexerOfEachFamilySearchesAndGrabsThroughItsPort searches
// each of ported-sites' five movies with every native Indexer, the one
// of a new family each time at priority 10 above the others' 25. Each
// search finds the movie's release on all five, and its Grab comes from
// the Indexer at 10, fetched through its port, and imports.
func TestANativeIndexerOfEachFamilySearchesAndGrabsThroughItsPort(t *testing.T) {
	t.Parallel()
	w := begin(t, "ported-sites")
	ids := w.indexers(t)
	previous := ""
	for i, name := range []string{"PassThePopcorn", "Torrent Bytes", "AvistaZ", "IPTorrents", "NorBits"} {
		if previous != "" {
			w.prioritize(t, ids[previous].ID, 25)
			w.addWork(t, strconv.Itoa(9000001+i))
		}
		w.prioritize(t, ids[name].ID, 10)
		w.grabsFrom(t, name, ids[name].ID, len(ids))
		previous = name
	}
}

func (w *walked) prioritize(t *testing.T, id string, priority int) {
	t.Helper()
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/indexers/"+id, `{"priority":`+strconv.Itoa(priority)+`}`); r.Status != http.StatusOK {
		t.Fatalf("updateIndexer %s to priority %d = %d %s, want 200", id, priority, r.Status, r.Body)
	}
}

// addWork adds the movie key, monitored, as w's Work.
func (w *walked) addWork(t *testing.T, key string) {
	t.Helper()
	made := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(made.Work, &tr); err != nil || !made.Created {
		t.Fatalf("addWork %s created %t: %v; want a created Work", key, made.Created, err)
	}
	w.work = tr.ID
}

// grabsFrom searches w's Work, which each of the indexers Indexers finds
// once, and checks its Grab came from Indexer id, named name, and
// imported.
func (w *walked) grabsFrom(t *testing.T, name, id string, indexers int) {
	t.Helper()
	found := w.searched(t, w.search(t))
	for _, l := range found.Indexers {
		if l.Outcome == nil || *l.Outcome != "found" || l.Releases != 1 {
			t.Errorf("%s's search ended %v with %d Releases: %s, want its one release found", l.Name, l.Outcome, l.Releases, l.Sentence)
		}
	}
	if len(found.Indexers) != indexers || found.GrabID == nil {
		t.Fatalf("the search = %+v, want a line for each of the %d Indexers and a Grab", found, indexers)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("%s's Grab ended %s with %+v after %v, want imported", name, w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	if from := w.grabbedFrom(t, w.grab.ID); from != id {
		t.Errorf("the Grab came from Indexer %s, want %s's %s", from, name, id)
	}
}
