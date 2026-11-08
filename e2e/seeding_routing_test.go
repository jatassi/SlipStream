package e2e

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// privacyOf is the privacy of the Tracker named name.
func (w *walked) privacyOf(t *testing.T, name string) (id, privacy string) {
	t.Helper()
	var list struct {
		Trackers []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Privacy string `json:"privacy"`
		} `json:"trackers"`
	}
	decode(t, "listTrackers", w.a.Admin("home", http.MethodGet, "/api/v1/trackers", ""), &list)
	for _, tr := range list.Trackers {
		if tr.Name == name {
			return tr.ID, tr.Privacy
		}
	}
	t.Fatalf("listTrackers has no %s", name)
	return "", ""
}

type routedTracker struct {
	Name   string `json:"name"`
	Stored bool   `json:"stored"`
}

// routedHere is the Trackers each Node's built-in client lists, by Node
// name.
func (w *walked) routedHere(t *testing.T) map[string][]routedTracker {
	t.Helper()
	var list struct {
		Cards []struct {
			NodeName string          `json:"nodeName"`
			Trackers []routedTracker `json:"trackers"`
		} `json:"cards"`
	}
	decode(t, "listDownloadClients", w.a.Admin("home", http.MethodGet, "/api/v1/download-clients", ""), &list)
	out := map[string][]routedTracker{}
	for _, c := range list.Cards {
		out[c.NodeName] = c.Trackers
	}
	return out
}

// grabOn adds movie key, grabs it and checks its Grab imported after it
// downloaded on node.
func (w *walked) grabOn(t *testing.T, key, name, node string) {
	t.Helper()
	w.addMovie(t, key)
	w.grabWork(t)
	if w.grab.NodeID != node {
		t.Fatalf("%s's Grab downloaded on %s, want %s", key, w.grab.NodeID, name)
	}
}

// awaitClientAway reads the Controller's log stream s until Grab id's
// step waits for its client's Node.
func awaitClientAway(t *testing.T, s *bufio.Scanner, id string) {
	t.Helper()
	for s.Scan() {
		var r struct {
			Subsystem string `json:"subsystem"`
			Message   string `json:"msg"`
			Attrs     struct {
				Grab string `json:"grab"`
				Err  string `json:"err"`
			} `json:"attrs"`
		}
		if json.Unmarshal([]byte(strings.TrimPrefix(s.Text(), "data: ")), &r) == nil && r.Subsystem == "grabs" &&
			r.Message == "a Grab's step waits" && r.Attrs.Grab == id && r.Attrs.Err == "the Node of the Grab's client is out of step" {
			return
		}
	}
	t.Fatalf("the log stream ended before Grab %s waited for its client's Node: %v", id, s.Err())
}

// TestAPrivateTrackersGrabsDownloadOnTheirHomeOrTheRouteAndWaitForAnOfflineHome
// grabs from heron, whose Indexer's home is seedbox, while the Movies root
// is on den. Heron's first Grab goes by the root to den, since its Tracker
// reads unknown until that fetch shows a private torrent. Its next Grab
// downloads on seedbox and a public release on den. With a route from
// seedbox to den's built-in client, the next Grab downloads on den and
// den's card lists Heron. Once the route is deleted and seedbox lapses, a
// Grab from heron waits at fetching for seedbox until the Admin sends it
// to den, where it downloads and imports.
func TestAPrivateTrackersGrabsDownloadOnTheirHomeOrTheRouteAndWaitForAnOfflineHome(t *testing.T) {
	t.Parallel()
	w := begin(t, "seeding-routing")
	box, den := w.a.Node("seedbox").NodeID, w.a.Node("den").NodeID

	w.grabWork(t)
	if w.grab.NodeID != den {
		t.Fatalf("Heron's first Grab downloaded on %s, want den %s by the Movies root", w.grab.NodeID, den)
	}
	heron, privacy := w.privacyOf(t, "Heron")
	if privacy != "private" {
		t.Fatalf("after its first Grab Heron reads %s, want private", privacy)
	}
	w.grabOn(t, "9000002", "seedbox", box)
	w.grabOn(t, "9000003", "den", den)
	if got := w.routedHere(t); len(got["seedbox"]) != 1 || got["seedbox"][0].Name != "Heron" || got["seedbox"][0].Stored || len(got["den"]) != 0 {
		t.Fatalf("with no route the cards list %+v, want Heron on seedbox by default", got)
	}

	route := "/api/v1/trackers/" + heron + "/routes/" + box
	if r := w.a.Admin("home", http.MethodPut, route, `{"builtInOn":"`+den+`"}`); r.Status != http.StatusOK {
		t.Fatalf("setTrackerRoute = %d %s, want 200", r.Status, r.Body)
	}
	if got := w.routedHere(t); len(got["den"]) != 1 || got["den"][0].Name != "Heron" || !got["den"][0].Stored || len(got["seedbox"]) != 0 {
		t.Fatalf("with the route the cards list %+v, want Heron on den by the Admin's pick", got)
	}
	w.grabOn(t, "9000004", "den", den)
	if r := w.a.Admin("home", http.MethodDelete, route, ""); r.Status != http.StatusOK {
		t.Fatalf("clearTrackerRoute = %d %s, want 200", r.Status, r.Body)
	}

	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/nodes/settings", `{"offlineGrace":60}`); r.Status != http.StatusOK {
		t.Fatalf("updateNodeSettings = %d %s, want 200", r.Status, r.Body)
	}
	act(t, w.run, "seedbox", world.StopNode)
	worldtest.Until(t, "seedbox lapses", func() (bool, error) { return w.navAlert(t) == "fail", nil })
	w.addMovie(t, "9000005")
	logs := followLogs(t, w.run, w.a.Node("home").NodeID)
	s := w.searched(t, w.search(t))
	if s.Outcome == nil || *s.Outcome != "grabbed" {
		t.Fatalf("the search past the grace ended %+v, want a Grab", s)
	}
	awaitClientAway(t, logs, *s.GrabID)
	grab := "/api/v1/grabs/" + *s.GrabID
	w.grab = *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, grab, ""), &grabView{})
	if w.grab.Stage != "fetching" || w.grab.FetchNodeID != nil {
		t.Fatalf("with seedbox lapsed Heron's Grab is %s, fetched by %v; want fetching and waiting for seedbox", w.grab.Stage, w.grab.FetchNodeID)
	}

	if r := w.a.Admin("home", http.MethodPost, grab+"/send-to", `{"builtInOn":"`+den+`"}`); r.Status != http.StatusNoContent {
		t.Fatalf("sendGrabTo = %d %s, want 204", r.Status, r.Body)
	}
	w.ended(t, *s.GrabID)
	if w.grab.Stage != "imported" || w.grab.NodeID != den {
		t.Fatalf("once sent Heron's Grab ended %s on %s after %v, want imported on den %s", w.grab.Stage, w.grab.NodeID, w.grab.stages(), den)
	}
}
