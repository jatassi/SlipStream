package e2e

import (
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func (w *walked) navAlert(t *testing.T) string {
	t.Helper()
	var c struct {
		Alerts struct {
			Nodes *string `json:"nodes"`
		} `json:"alerts"`
	}
	decode(t, "getCounts", w.a.Admin("home", http.MethodGet, "/api/v1/counts", ""), &c)
	if c.Alerts.Nodes == nil {
		return "none"
	}
	return *c.Alerts.Nodes
}

// indexerWaits is why the one Indexer's searches wait, or "none".
func (w *walked) indexerWaits(t *testing.T) string {
	t.Helper()
	var list struct {
		Indexers []struct {
			Waits *struct {
				Kind string `json:"kind"`
			} `json:"waits"`
		} `json:"indexers"`
	}
	decode(t, "listIndexers", w.a.Admin("home", http.MethodGet, "/api/v1/indexers", ""), &list)
	if len(list.Indexers) != 1 {
		t.Fatalf("listIndexers lists %d Indexers, want 1", len(list.Indexers))
	}
	if list.Indexers[0].Waits == nil {
		return "none"
	}
	return list.Indexers[0].Waits.Kind
}

// expectSearch searches once and checks the run finishes outcome with the
// one Indexer's line.
func (w *walked) expectSearch(t *testing.T, when, outcome, line string) searchView {
	t.Helper()
	s := w.searched(t, w.search(t))
	if lines := s.lines(); *s.Outcome != outcome || len(lines) != 1 || lines[0] != line {
		t.Fatalf("a search %s finishes %s with lines %q, want %s with %q", when, *s.Outcome, lines, outcome, line)
	}
	return s
}

// TestAnOfflineSeedboxsSearchesGoHomePastTheGrace stops seedbox, the
// Indexer's home and the Node that downloads, with the offline grace at its
// 1 min floor. Within the grace the Indexer waits for seedbox, so a search
// finds nothing. Past it the nav alert fails and home covers
// Searching, so a search grabs through home and its Grab waits for a Node
// that downloads. Once seedbox is back and settled, seedbox fetches and
// downloads the Grab and the alert clears.
func TestAnOfflineSeedboxsSearchesGoHomePastTheGrace(t *testing.T) {
	t.Parallel()
	w := begin(t, "home")
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/nodes/settings", `{"offlineGrace":60}`); r.Status != http.StatusOK {
		t.Fatalf("updateNodeSettings = %d %s, want 200", r.Status, r.Body)
	}
	box := w.a.Node("seedbox").NodeID
	if got := w.navAlert(t); got != "none" {
		t.Fatalf("with seedbox in step the nav alert is %s, want none", got)
	}

	act(t, w.run, "seedbox", world.StopNode)
	worldtest.Until(t, "the Indexer waits for seedbox", func() (bool, error) { return w.indexerWaits(t) != "none", nil })
	w.expectSearch(t, "within the grace", "nothing_acceptable", "Fixture Site waiting: Waiting for seedbox")

	worldtest.Until(t, "seedbox lapses", func() (bool, error) { return w.navAlert(t) == "fail", nil })
	if got := w.indexerWaits(t); got != "none" {
		t.Fatalf("past the grace the Indexer waits %s, want none", got)
	}
	grabbed := w.expectSearch(t, "past the grace, through home", "grabbed", "Fixture Site found: Found 2 Releases")

	act(t, w.run, "seedbox", world.StartNode)
	w.ended(t, *grabbed.GrabID)
	if w.grab.Stage != "imported" || w.grab.NodeID != box || w.grab.FetchNodeID == nil || *w.grab.FetchNodeID != box {
		t.Fatalf("the Grab ended %s, fetched by %v and downloaded by %s; want imported, both by seedbox %s", w.grab.Stage, w.grab.FetchNodeID, w.grab.NodeID, box)
	}
	worldtest.Until(t, "the nav alert clears once seedbox settled", func() (bool, error) { return w.navAlert(t) == "none", nil })
}
