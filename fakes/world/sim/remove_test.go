package sim_test

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestARemovedWorkerJoinsAgainKeepingItsOldTorrent removes seedbox while
// it seeds an imported Grab: it needs a write-off, and once written off
// seedbox records the removal, restarts Unclaimed saying who removed it,
// and keeps the file. Joined again with a new Join Code it is a new Node
// that keeps the torrent it still holds, and adoption moves the written-off
// Torrent's record to it.
func TestARemovedWorkerJoinsAgainKeepingItsOldTorrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), simtest.OnSeedbox(func(e *node.Env) { e.Engine = box.wrap(e.Engine) }))
		simtest.Search(t, w, simtest.AddOrchard(t, w))
		simtest.ExpectImported(t, simtest.Await(t, w))
		a := w.API()
		old := seedbox(t, a).ID
		lib := w.Node("home").Half().Library
		torrent := onlyTorrent(t, lib)

		if r := a.Admin("home", http.MethodDelete, "/api/v1/nodes/"+old, ""); r.Status != http.StatusConflict || !strings.Contains(r.Body, `"code":"node_holds_work"`) {
			t.Fatalf("removing seedbox while it seeds = %d %s, want 409 node_holds_work", r.Status, r.Body)
		}
		if r := a.Admin("home", http.MethodDelete, "/api/v1/nodes/"+old+"?writeOff=true", ""); r.Status != http.StatusNoContent {
			t.Fatalf("removing seedbox with its work written off = %d %s, want 204", r.Status, r.Body)
		}
		added := box.adds()
		w.Node("seedbox").AwaitRestart()
		a = w.API()

		status := simtest.Decode[struct {
			Claim   string `json:"claim"`
			Removed *struct {
				Controller string `json:"controller"`
			} `json:"removed"`
		}](t, "getStatus", a.Get("seedbox", "/api/v1/status"))
		if status.Claim != "unclaimed" || status.Removed == nil || status.Removed.Controller != "home" {
			t.Fatalf("seedbox's status after the removal reads %+v, want unclaimed and removed by home", status)
		}
		if data, err := os.ReadFile(filepath.Join(w.Node("seedbox").DataDir().Path(), pairing.RecordFile)); err != nil || !strings.Contains(string(data), `"removed":`) {
			t.Fatalf("seedbox's pairing record reads %s, %v; want the removal in it", data, err)
		}
		expectDownloadKept(t, w)

		joined := a.Join("home", "seedbox", w.Node("seedbox").Ready().SetupCode)
		if joined == old {
			t.Fatalf("seedbox joined again as %s, its old id; want a new Node", joined)
		}
		var names []string
		for _, r := range a.Nodes("home") {
			names = append(names, r.Name)
		}
		slices.Sort(names)
		if !slices.Equal(names, []string{"home", "seedbox"}) {
			t.Fatalf("home lists %v, want home and the new seedbox only", names)
		}
		simtest.Eventually(t, "the new seedbox is up", func() bool { return seedbox(t, a).Link.State == "up" })
		simtest.Eventually(t, "adoption grants the new seedbox Downloading", func() bool { return stateOf(seedbox(t, a), "downloading") == "holding" })
		a.SetRoles("home", joined, "searching", "downloading")
		simtest.Eventually(t, "the new seedbox runs Downloading", func() bool {
			f := seedbox(t, a).Facts
			return f != nil && slices.Contains(f.Running, "downloading")
		})
		if n := box.adds() - added; n != 1 {
			t.Errorf("seedbox's engines added %d torrents once it was removed, want the one it kept", n)
		}
		expectMovedTo(t, lib, torrent, joined)
	})
}

// onlyTorrent is the one Torrent lib records.
func onlyTorrent(t *testing.T, lib *storage.DB[storage.Library]) downloadclients.ID {
	t.Helper()
	var id downloadclients.ID
	if err := lib.QueryRowContext(t.Context(), "SELECT id FROM downloadclients_torrent").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// expectMovedTo fails t unless lib records Torrent id live on the Node
// joined.
func expectMovedTo(t *testing.T, lib *storage.DB[storage.Library], id downloadclients.ID, joined string) {
	t.Helper()
	var on role.NodeID
	if err := lib.QueryRowContext(t.Context(), "SELECT node_id FROM downloadclients_torrent WHERE id = ? AND parted_at IS NULL", id).Scan(&on); err != nil || on.String() != joined {
		t.Errorf("home records Torrent %s live on %s, %v; want it moved to the new seedbox %s", id, on, err, joined)
	}
}
