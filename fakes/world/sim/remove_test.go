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
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/pairing"
)

// TestARemovedWorkerJoinsAgainWithoutItsOldTorrents removes seedbox while
// it seeds an imported Grab: it needs a write-off, and once written off
// seedbox records the removal, restarts Unclaimed saying who removed it,
// and keeps the file. Joined again with a new Join Code it is a new Node,
// and Downloading ticked on it adds none of the old torrents.
func TestARemovedWorkerJoinsAgainWithoutItsOldTorrents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Engine = box.wrap(e.Engine) }))
		search(t, w, addOrchard(t, w))
		expectImported(t, await(t, w))
		a := w.API()
		old := seedbox(t, a).ID

		if r := a.Admin("home", http.MethodDelete, "/api/v1/nodes/"+old, ""); r.Status != http.StatusConflict || !strings.Contains(r.Body, `"code":"node_holds_work"`) {
			t.Fatalf("removing seedbox while it seeds = %d %s, want 409 node_holds_work", r.Status, r.Body)
		}
		if r := a.Admin("home", http.MethodDelete, "/api/v1/nodes/"+old+"?writeOff=true", ""); r.Status != http.StatusNoContent {
			t.Fatalf("removing seedbox with its work written off = %d %s, want 204", r.Status, r.Body)
		}
		added := box.adds()
		w.Node("seedbox").AwaitRestart()
		a = w.API()

		status := decode[struct {
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
		eventually(t, "the new seedbox is up", func() bool { return seedbox(t, a).Link.State == "up" })
		a.SetRoles("home", joined, "searching", "downloading")
		eventually(t, "the new seedbox runs Downloading", func() bool {
			f := seedbox(t, a).Facts
			return f != nil && slices.Contains(f.Running, "downloading")
		})
		if n := box.adds() - added; n != 0 {
			t.Fatalf("seedbox's engines added %d torrents once it was removed, want none of the removed Node's", n)
		}
	})
}
