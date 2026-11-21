package sim_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

const metainfoProcedure = "/slipstream.downloading.v1.DownloadingService/Metainfo"

// gate holds every Metainfo call a Node serves until it opens.
type gate struct {
	mu     sync.Mutex
	asked  int
	opened chan struct{}
}

func (g *gate) wrap(e *node.Env) {
	outer := e.Test.Lockstep
	e.Test.Lockstep = func(h http.Handler) http.Handler {
		return outer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == metainfoProcedure {
				g.mu.Lock()
				g.asked++
				g.mu.Unlock()
				<-g.opened
			}
			h.ServeHTTP(w, r)
		}))
	}
}

func (g *gate) calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.asked
}

// TestANewPairingKeepsEverythingUntilAdopted has seedbox seed 40 torrents
// for home, then pairs it with office. While office adopts them, seedbox
// reads keeping for Downloading, never draining, under one Reconciling
// link in Health. Once adopted, office records all 40 with their folders,
// seedbox holds Downloading, granted as it was adopted, and its engine
// still holds every torrent and removed none.
func TestANewPairingKeepsEverythingUntilAdopted(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		const held = 40
		box := &engines{}
		swarm := enginetest.NewWorld()
		swarm.Tracker(idleTracker, enginetest.TrackerRules{Interval: 30 * time.Minute, MinInterval: 30 * time.Minute})
		g := &gate{opened: make(chan struct{})}
		s := simtest.Home(t)
		s.Nodes = append(s.Nodes, scenario.Node{Name: "office"})
		w := sim.Launch(t, s, simtest.OnSeedbox(func(e *node.Env) {
			e.Engine = box.wrap(swarm.Open)
			g.wrap(e)
		}))
		home := w.Node("home").Half()
		attachSeeded(t, home, swarm, nodeID(t, w, "seedbox"), held)
		simtest.Eventually(t, "seedbox seeds every torrent for home", func() bool { return seeding(t, home) == held })

		key := office(t, w)
		worker := w.Node("seedbox")
		if reply, err := worker.JoinOp(t.Context(), mint(t, w, "office", key)); err != nil || reply.Refused != nil {
			t.Fatalf("the join op with office's code = %+v, %v; want joined", reply, err)
		}
		worker.AwaitRestart()
		synctest.Wait()

		seen := map[string]bool{}
		downloading := func() string {
			st := officeDownloading(t, w, key)
			seen[st] = true
			return st
		}
		simtest.Eventually(t, "office asks seedbox for a torrent's metainfo", func() bool {
			downloading()
			return g.calls() > 0
		})
		if st := downloading(); st != "keeping" {
			t.Errorf("while office adopts, seedbox reads %q for Downloading, want keeping", st)
		}
		if links := reconcilingLinks(t, w, key); links != 1 {
			t.Errorf("while office adopts, its Health links to Reconciling %d times, want once", links)
		}

		close(g.opened)
		simtest.Eventually(t, "seedbox holds Downloading for office", func() bool { return downloading() == "holding" })
		if seen["draining"] {
			t.Errorf("seedbox read draining for Downloading at some point: %v", seen)
		}
		adopted, granted := adoptedOn(t, w.Node("office").Half())
		if adopted != held || granted != 1 {
			t.Errorf("office records %d adopted torrents with their folders and %d Nodes granted as adopted, want %d and 1", adopted, granted, held)
		}
		if box.torrents() != held || len(box.removals()) != 0 {
			t.Errorf("seedbox's engine holds %d torrents having removed %d, want %d and none", box.torrents(), len(box.removals()), held)
		}
	})
}

// officeDownloading is seedbox's Downloading state on office's Nodes
// page, unpaired before office lists it.
func officeDownloading(t *testing.T, w *sim.World, key string) string {
	t.Helper()
	rows := officeRows(t, w, key)
	i := slices.IndexFunc(rows, func(n nodeapi.Node) bool { return n.Name == "seedbox" })
	if i < 0 {
		return "unpaired"
	}
	j := slices.IndexFunc(rows[i].Roles, func(r nodeapi.RoleView) bool { return r.Role == "downloading" })
	if j < 0 {
		return "none"
	}
	return rows[i].Roles[j].State
}

// reconcilingLinks counts the Nodes items in office's Health that link to
// Reconciling.
func reconcilingLinks(t *testing.T, w *sim.World, key string) int {
	t.Helper()
	n := 0
	for _, it := range officeHealth(t, w, key) {
		if it.Action != nil && it.Action.Href == "/reconciling" {
			n++
		}
	}
	return n
}

// adoptedOn counts the live Torrents c adopted with their folders, and the
// Nodes it granted Downloading in the Write that adopted them.
func adoptedOn(t *testing.T, c node.Half) (adopted, granted int) {
	t.Helper()
	if err := c.Library.QueryRowContext(t.Context(), "SELECT count(*) FROM downloadclients_torrent WHERE adopted = 1 AND save_folder IS NOT NULL AND parted_at IS NULL").Scan(&adopted); err != nil {
		t.Fatal(err)
	}
	if err := c.Library.QueryRowContext(t.Context(), "SELECT count(*) FROM reconciling_node WHERE adopted_at IS NOT NULL AND granted_at = adopted_at").Scan(&granted); err != nil {
		t.Fatal(err)
	}
	return adopted, granted
}

// officeHealth is every Nodes item in office's Health report.
func officeHealth(t *testing.T, w *sim.World, key string) map[string]simtest.HealthItem {
	t.Helper()
	return simtest.Decode[simtest.HealthReport](t, "getSystemHealth", w.API().Keyed("office", key, http.MethodGet, "/api/v1/system/health", "")).Items(t, "nodes")
}

// TestEveryStubGateNowReadsState begins Reconciling on home for a restore
// onto a new machine, so it runs on, with Orchard seeding on seedbox under
// a goal of a ratio of 0.5 then pause. A search is refused as
// reconciling, and once Orchard's goal is met the goals preview says it
// waits on Reconciling and the goal never acts.
func TestEveryStubGateNowReadsState(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: 5 * time.Minute, MinInterval: 5 * time.Minute}, &engines{})
		a := w.API()
		nyaa := addTracker(t, w, "Nyaa", indexers.Public)
		put(t, a, fmt.Sprintf("/api/v1/trackers/%s/goal", nyaa), `{"combine":"first","ratio":0.5,"seedSeconds":null,"action":"pause"}`)
		orchard := seedFor(t, w, swarm, nyaa, "Orchard")
		home := w.Node("home").Half()
		simtest.Eventually(t, "Orchard seeds", func() bool { return seeding(t, home) == 1 })
		err := home.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "restore-1", Cause: reconciling.Restore, Backup: "nightly", TakenAt: time.Now(), NewMachine: true})
		})
		if err != nil {
			t.Fatal(err)
		}

		if r := a.Admin("home", http.MethodPost, "/api/v1/works/"+storage.NewUUID().String()+"/search", ""); r.Status != http.StatusConflict || !strings.Contains(r.Body, `"reconciling"`) {
			t.Errorf("searchWork while Reconciling = %d %s, want 409 reconciling", r.Status, r.Body)
		}
		swarm.Transfer(orchard.hash, 0, 10)
		type preview struct {
			Waits []struct {
				Torrent struct{ ID string }
				Why     string
			}
		}
		within(t, time.Hour, "Orchard's met goal waits on Reconciling", func() bool {
			p := simtest.Decode[preview](t, "previewSeedingGoals", a.Admin("home", http.MethodGet, "/api/v1/torrents/goals/preview", ""))
			return slices.ContainsFunc(p.Waits, func(w struct {
				Torrent struct{ ID string }
				Why     string
			}) bool {
				return w.Torrent.ID == orchard.id.String() && w.Why == "reconciling"
			})
		})
		time.Sleep(time.Hour)
		synctest.Wait()
		if got := goalRunOf(t, w, orchard); got.action != "" || got.removed != "" {
			t.Errorf("while Reconciling Orchard's goal left it %+v, want it untouched", got)
		}
	})
}
