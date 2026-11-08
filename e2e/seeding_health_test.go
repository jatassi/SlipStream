package e2e

import (
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type windowView struct {
	Name       string  `json:"name"`
	State      string  `json:"state"`
	AtRisk     string  `json:"atRisk"`
	QuietSince *string `json:"quietSince"`
}

// window reads tracker's one window.
func (w *walked) window(t *testing.T, tracker string) windowView {
	t.Helper()
	var list struct {
		Windows []windowView `json:"windows"`
	}
	decode(t, "listTrackerWindows", w.a.Admin("home", http.MethodGet, "/api/v1/trackers/"+tracker+"/windows", ""), &list)
	if len(list.Windows) != 1 {
		t.Fatalf("the Tracker holds windows %+v, want one", list.Windows)
	}
	return list.Windows[0]
}

// torrentAction asks for action on Torrent id, then waits for torrentIn.
func (w *walked) torrentAction(t *testing.T, action, id, state string) {
	t.Helper()
	if r := w.a.Admin("home", http.MethodPost, "/api/v1/torrents/actions", `{"action":"`+action+`","ids":["`+id+`"]}`); r.Status != http.StatusOK {
		t.Fatalf("torrentActions %s = %d %s, want 200", action, r.Status, r.Body)
	}
	w.torrentIn(t, id, state)
}

// torrentIn waits for Torrent id's Node to report it in state, with the
// bytes it downloaded, which open its window.
func (w *walked) torrentIn(t *testing.T, id, state string) {
	t.Helper()
	worldtest.Until(t, "the Torrent is "+state, func() (bool, error) {
		var row struct {
			State      string `json:"state"`
			Pending    string `json:"pending"`
			Downloaded int64  `json:"downloaded"`
		}
		decode(t, "getTorrent", w.a.Admin("home", http.MethodGet, "/api/v1/torrents/"+id, ""), &row)
		return row.State == state && row.Pending == "" && row.Downloaded > 0, nil
	})
}

// TestATorrentThatOwesAndIsPausedIsAtRiskAndAHealthWarningOnceTheDelayPasses
// grabs The Quiet Orchard from heron, whose 3-minute rule it still owes,
// and pauses it. The next Seeding goals run puts its window at risk, and
// with the warn delay at 1 minute Health warns of Heron once it passes.
// Resuming it and running Seeding goals again clears both.
func TestATorrentThatOwesAndIsPausedIsAtRiskAndAHealthWarningOnceTheDelayPasses(t *testing.T) {
	t.Parallel()
	w := begin(t, "seeding")
	heron := w.ruled(t)["Heron"].ID
	torrent := *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+w.grabWork(t), ""), &trackedGrab{}).TorrentID
	w.torrentIn(t, torrent, "seeding")
	w.runTask(t, "seeding-goals")
	if win := w.window(t, heron); win.State != "owing" || win.AtRisk != "no" || win.QuietSince != nil {
		t.Fatalf("seeding Orchard's window is %+v, want owing and not at risk", win)
	}

	w.torrentAction(t, "pause", torrent, "paused")
	w.runTask(t, "seeding-goals")
	if win := w.window(t, heron); win.State != "owing" || win.AtRisk != "yes" || win.QuietSince == nil {
		t.Fatalf("paused Orchard's window is %+v, want owing and at risk since the run", win)
	}
	key := heron + ".at-risk"
	if it, ok := healthItems(t, w.a, "seeding")[key]; ok {
		t.Fatalf("before the warn delay passes Health holds %+v, want no Seeding item", it)
	}
	var item healthItem
	worldtest.Until(t, "Health warns of Heron", func() (bool, error) {
		var ok bool
		item, ok = healthItems(t, w.a, "seeding")[key]
		return ok, nil
	})
	want := "1 torrent on Heron stopped seeding before Heron's rule was met. Resume or retry it before Heron counts a hit and run."
	if item.Level != "warn" || item.Text != want || item.Action == nil || item.Action.Href != "/settings/trackers/"+heron+"?tab=problems" {
		t.Fatalf("Health warns %+v, want a warning %q linking to Heron's Problems", item, want)
	}

	w.torrentAction(t, "resume", torrent, "seeding")
	w.runTask(t, "seeding-goals")
	if win := w.window(t, heron); win.AtRisk != "no" || win.QuietSince != nil {
		t.Fatalf("resumed Orchard's window is %+v, want not at risk", win)
	}
	worldtest.Until(t, "Health clears Heron's warning", func() (bool, error) {
		_, ok := healthItems(t, w.a, "seeding")[key]
		return !ok, nil
	})
}
