package sim_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim/simtest"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/works"
)

// goalPayload is 6000 bytes, so an upload of 10 bytes a second reaches a
// ratio of 1.0 in 10 minutes.
var goalPayload = bytes.Repeat([]byte("goal"), 1500)

// seeded is a Torrent the Controller handed seedbox, and its info hash.
type seeded struct {
	id   downloadclients.ID
	hash bencode.InfoHash
}

// addTracker adds a Tracker named name to home's library, as an Indexer
// reaching it would.
func addTracker(t *testing.T, w *sim.World, name string, privacy indexers.Privacy) indexers.TrackerID {
	t.Helper()
	id := indexers.TrackerID{UUID: storage.NewUUID()}
	onTorrents(t, w, `INSERT INTO indexers_tracker (id, name, privacy, created_at) VALUES (?, ?, ?, 0)`, id, name, string(privacy))
	return id
}

// seedFor has home hand seedbox a private torrent of name announcing to
// idleTracker, its Release reached through tracker, whose payload swarm
// holds.
func seedFor(t *testing.T, w *sim.World, swarm *enginetest.World, tracker indexers.TrackerID, name string) seeded {
	t.Helper()
	box := nodeID(t, w, "seedbox")
	raw, hash, err := releases.Metainfo(name+".mkv", nil, goalPayload, idleTracker, false)
	if err != nil {
		t.Fatal(err)
	}
	swarm.Payload(hash, goalPayload)
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	controller := w.Node("home").Half()
	var id downloadclients.ID
	err = controller.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		want := downloadclients.Want{
			Target: downloadclients.Target{Node: box}, Metainfo: m, Raw: raw, ContentType: works.TypeID("movie"), Tracker: tracker,
			WrittenOff: func(role.NodeID) bool { return false },
		}
		var err error
		id, _, err = downloadclients.Attach(ctx, tx, want, storage.Now())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	controller.Clients.Touch(box)
	return seeded{id: id, hash: hash}
}

func put(t *testing.T, a *worldtest.API, path, body string) {
	t.Helper()
	if r := a.Admin("home", http.MethodPut, path, body); r.Status != http.StatusOK {
		t.Fatalf("PUT %s = %d %s, want 200", path, r.Status, r.Body)
	}
}

// goalRun is a Torrent as the goals left it: why it was removed, its goal
// run's action and kept, each "" for none, and when the run acted and its
// window was met, zero for none.
type goalRun struct {
	removed, action, kept string
	actedAt, metAt        int64
}

func goalRunOf(t *testing.T, w *sim.World, s seeded) goalRun {
	t.Helper()
	var g goalRun
	err := w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT coalesce(t.removed_reason, ''), coalesce(g.action, ''),
		coalesce(g.kept, ''), coalesce(g.acted_at, 0), coalesce(win.met_at, 0)
		FROM downloadclients_torrent t LEFT JOIN trackers_goal_run g ON g.torrent_id = t.id
		LEFT JOIN trackers_window win ON win.info_hash = t.info_hash WHERE t.id = ?`, s.id).
		Scan(&g.removed, &g.action, &g.kept, &g.actedAt, &g.metAt)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// within waits in fake time, a minute at a time, until ok, failing with
// what after limit.
func within(t *testing.T, limit time.Duration, what string, ok func() bool) {
	t.Helper()
	for range limit / time.Minute {
		if ok() {
			return
		}
		time.Sleep(time.Minute)
	}
	t.Fatalf("not within %s: %s", limit, what)
}

// TestSeedingGoalsActOnceTheRuleIsMet seeds Orchard on ratiopool, whose
// rule is a ratio of 1.0 and whose goal a ratio of 0.5 then remove and
// delete, and Harbor on timepool, whose rule is 30 minutes of seeding and
// whose goal 10 minutes then remove. With a margin of 10 minutes, each
// goal acts only once its window is met: Orchard once it uploaded its
// size, Harbor once the tracker credited 40 minutes. seedbox's engine
// then removes Orchard with its files and Harbor without.
func TestSeedingGoalsActOnceTheRuleIsMet(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: 5 * time.Minute, MinInterval: 5 * time.Minute}, box)
		a := w.API()
		ratiopool := addTracker(t, w, "ratiopool", indexers.Private)
		timepool := addTracker(t, w, "timepool", indexers.Private)
		if r := a.Admin("home", http.MethodPatch, "/api/v1/trackers/settings", `{"values":{"trackers.margin":600}}`); r.Status != http.StatusOK {
			t.Fatalf("updateSeedingSettings = %d %s, want 200", r.Status, r.Body)
		}
		put(t, a, fmt.Sprintf("/api/v1/trackers/%s/rule", ratiopool), `{"clears":"ratio","ratio":1,"seedSeconds":null}`)
		put(t, a, fmt.Sprintf("/api/v1/trackers/%s/goal", ratiopool), `{"combine":"first","ratio":0.5,"seedSeconds":null,"action":"remove-delete"}`)
		put(t, a, fmt.Sprintf("/api/v1/trackers/%s/rule", timepool), `{"clears":"time","ratio":null,"seedSeconds":1800}`)
		put(t, a, fmt.Sprintf("/api/v1/trackers/%s/goal", timepool), `{"combine":"first","ratio":null,"seedSeconds":600,"action":"remove"}`)

		orchard := seedFor(t, w, swarm, ratiopool, "Orchard")
		harbor := seedFor(t, w, swarm, timepool, "Harbor")
		simtest.Eventually(t, "both torrents seed", func() bool { return seeding(t, w.Node("home").Half()) == 2 })
		swarm.Transfer(orchard.hash, 0, 10)

		within(t, 3*time.Hour, "both goals act", func() bool {
			return goalRunOf(t, w, orchard).action != "" && goalRunOf(t, w, harbor).action != ""
		})
		for name, c := range map[string]struct {
			s    seeded
			want goalRun
		}{
			"Orchard": {orchard, goalRun{removed: "goal_met", action: "remove-delete"}},
			"Harbor":  {harbor, goalRun{removed: "goal_met", action: "remove"}},
		} {
			got := goalRunOf(t, w, c.s)
			if got.removed != c.want.removed || got.action != c.want.action || got.kept != "" {
				t.Errorf("%s reads %+v, want removed by its goal with %s", name, got, c.want.action)
			}
			if got.metAt == 0 || got.actedAt < got.metAt {
				t.Errorf("%s's goal acted at %d with its window met at %d, want it to act only once its window was met", name, got.actedAt, got.metAt)
			}
		}
		simtest.Eventually(t, "seedbox's engine removes both", func() bool { return len(box.removals()) == 2 })
		deletes := map[bool]int{}
		for _, r := range box.removals() {
			deletes[r.deleteFiles]++
		}
		if deletes[true] != 1 || deletes[false] != 1 {
			t.Errorf("seedbox's engine removed %+v, want Orchard with its files and Harbor without", box.removals())
		}
	})
}

// TestADrainRemovesATorrentItsGoalPausedKeepingItsCopy seeds Orchard on
// Nyaa, public with a goal of a ratio of 0.5 then pause. Once its goal
// pauses it, unticking Downloading on seedbox has the goals remove it
// keeping its copy, and seedbox's drain then ends with nothing held.
func TestADrainRemovesATorrentItsGoalPausedKeepingItsCopy(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		box := &engines{}
		w, swarm := launchSwarm(t, enginetest.TrackerRules{Interval: 5 * time.Minute, MinInterval: 5 * time.Minute}, box)
		a := w.API()
		nyaa := addTracker(t, w, "Nyaa", indexers.Public)
		put(t, a, fmt.Sprintf("/api/v1/trackers/%s/goal", nyaa), `{"combine":"first","ratio":0.5,"seedSeconds":null,"action":"pause"}`)
		orchard := seedFor(t, w, swarm, nyaa, "Orchard")
		simtest.Eventually(t, "Orchard seeds", func() bool { return seeding(t, w.Node("home").Half()) == 1 })
		swarm.Transfer(orchard.hash, 0, 10)

		within(t, time.Hour, "Orchard's goal pauses it", func() bool { return goalRunOf(t, w, orchard).action == "pause" })
		if got := goalRunOf(t, w, orchard); got.removed != "" {
			t.Fatalf("paused by its goal Orchard reads %+v, want it held", got)
		}

		id := seedbox(t, a).ID
		a.SetRoles("home", id, "searching")
		within(t, time.Hour, "the goals remove Orchard from the draining seedbox", func() bool { return goalRunOf(t, w, orchard).removed != "" })
		if got := goalRunOf(t, w, orchard); got.removed != "goal_met" || got.action != "remove" || got.kept != "draining" {
			t.Fatalf("on the draining seedbox Orchard reads %+v, want removed by its goal, kept for the drain", got)
		}
		simtest.Eventually(t, "seedbox's drain ends", func() bool {
			return box.torrents() == 0 && stateOf(seedbox(t, a), "downloading") == "none"
		})
		if got := box.removals(); len(got) != 1 || got[0].deleteFiles {
			t.Errorf("seedbox's engine removed %+v, want Orchard once with its files kept", got)
		}
	})
}
