package sim_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// beginRestore has home begin Reconciling for a restore onto a new
// machine.
func beginRestore(t *testing.T, w *sim.World) {
	t.Helper()
	err := w.Node("home").Half().Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "restore-1", Cause: reconciling.Restore, Backup: "nightly", TakenAt: time.Now(), NewMachine: true})
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestEachGateRefusesWithTheWaits begins Reconciling on home, then presses
// each refused row of the Gates table: every one answers 409 reconciling
// with a detail naming what Reconciling waits on, and pausing, resuming
// or removing a torrent refuses it as reconciling. The grabs tests prove
// the rows that need a Grab first: Cancel at downloading, a pick from a
// Manual search and Blocklist by hand.
func TestEachGateRefusesWithTheWaits(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: 5 * time.Minute, MinInterval: 5 * time.Minute}, &engines{})
		a := w.API()
		orchard := seedFor(t, w, swarm, addTracker(t, w, "Nyaa", indexers.Public), "Orchard")
		simtest.Eventually(t, "Orchard seeds", func() bool { return seeding(t, w.Node("home").Half()) == 1 })
		beginRestore(t, w)

		work, grab := storage.NewUUID().String(), storage.NewUUID().String()
		const lead = `"detail":"SlipStream is matching the restored library with the torrents and files on your Nodes. It's waiting for the old Controller's folders, `
		for _, row := range []struct{ gate, method, path, body string }{
			{"Search", http.MethodPost, "/api/v1/works/" + work + "/search", ""},
			{"Search several Works", http.MethodPost, "/api/v1/works/search", `{"workIds":["` + work + `"]}`},
			{"Manual search", http.MethodPost, "/api/v1/works/" + work + "/manual-search", `{}`},
			{"Grab from a plan", http.MethodPost, "/api/v1/works/" + work + "/waits/grab", `{}`},
			{"Retarget", http.MethodPost, "/api/v1/grabs/" + grab + "/send-to", `{"builtInOn":"` + nodeID(t, w, "seedbox").String() + `"}`},
			{"Rename", http.MethodPost, "/api/v1/renames", `{"works":[{"work":"` + work + `","digest":"0123456789abcdef"}]}`},
			{"Delete Work with files", http.MethodPost, "/api/v1/works/delete", `{"workIds":["` + work + `"],"files":"delete"}`},
		} {
			r := a.Admin("home", row.method, row.path, row.body)
			if r.Status != http.StatusConflict || !strings.Contains(r.Body, `"code":"reconciling"`) || !strings.Contains(r.Body, lead) || !strings.Contains(r.Body, "seedbox. Try again once it finishes.") {
				t.Errorf("%s while Reconciling = %d %s, want 409 reconciling naming the folders and seedbox", row.gate, r.Status, r.Body)
			}
		}
		for _, action := range []string{"pause", "resume", "remove"} {
			body := fmt.Sprintf(`{"action":%q,"ids":[%q]}`, action, orchard.id)
			if r := a.Admin("home", http.MethodPost, "/api/v1/torrents/actions", body); r.Status != http.StatusOK || !strings.Contains(r.Body, `"code":"reconciling"`) {
				t.Errorf("%s of Orchard while Reconciling = %d %s, want its torrent refused as reconciling", action, r.Status, r.Body)
			}
		}
	})
}
