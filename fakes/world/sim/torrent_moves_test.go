package sim_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// launchMoves starts the moves Scenario, its external clients added,
// with seedbox and den running their built-in engines in swarm.
func launchMoves(t *testing.T) (w *sim.World, swarm *enginetest.World) {
	t.Helper()
	s, err := scenario.Load("moves")
	if err != nil {
		t.Fatal(err)
	}
	swarm = enginetest.NewWorld()
	swarm.Tracker(idleTracker, enginetest.TrackerRules{Interval: 5 * time.Minute, MinInterval: 5 * time.Minute})
	return sim.Launch(t, s, sim.WithWalk(), sim.Edit(func(name string, e *node.Env) {
		if name == "seedbox" || name == "den" {
			e.Engine = swarm.Open
		}
	})), swarm
}

// moveTorrent previews and starts the Move of s to to, failing unless it
// is movable.
func moveTorrent(t *testing.T, w *sim.World, s seeded, to moves.Side) moves.BatchID {
	t.Helper()
	m := w.Node("home").Half().Moves
	plan, err := m.Preview(t.Context(), moves.Ask{Origin: moves.OriginTorrent, Torrents: []moves.TorrentTo{{Torrent: s.id, To: to}}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Movable() != 1 {
		t.Fatalf("the preview is %+v, want the torrent movable", plan.Units)
	}
	started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
	if err != nil {
		t.Fatal(err)
	}
	return started.Batch
}

// ended is batch's one unit once it is done or failed.
func ended(t *testing.T, w *sim.World, batch moves.BatchID) moves.View {
	t.Helper()
	var unit moves.View
	within(t, 2*time.Hour, "the torrent's Move ends", func() bool {
		_, units, err := w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
		if err != nil || len(units) != 1 {
			return false
		}
		unit = units[0]
		return unit.Phase == moves.PhaseDone || unit.Phase == moves.PhaseFailed
	})
	return unit
}

// clientOn is the id of on's external Download Client of kind.
func clientOn(t *testing.T, w *sim.World, on, kind string) *downloadclients.ClientID {
	t.Helper()
	var id downloadclients.ClientID
	simtest.Eventually(t, on+"'s "+kind+" client is paired", func() bool {
		return w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT c.id FROM downloadclients_client c
			JOIN nodes_node n ON n.id = c.node_id WHERE n.name = ? AND c.kind = ?`, on, kind).Scan(&id) == nil
	})
	return &id
}

// TestATorrentMovesBetweenBuiltInClientsWithoutTwoPeersStarted moves
// Orchard, seeding in seedbox's built-in client, to den's: the payload
// crosses once, the guard waits out seedbox's last announce, seedbox
// stops before den starts, and the hash ends live on den alone with
// seedbox's copy removed as moved.
func TestATorrentMovesBetweenBuiltInClientsWithoutTwoPeersStarted(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, swarm := launchMoves(t)
		orchard := seedFor(t, w, swarm, addTracker(t, w, "ratiopool", indexers.Private), "Orchard")
		simtest.Eventually(t, "Orchard seeds on seedbox", func() bool { return seeding(t, w.Node("home").Half()) == 1 })

		batch := moveTorrent(t, w, orchard, moves.Side{Node: nodeID(t, w, "den")})
		if unit := ended(t, w, batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v (%s), want done", unit.Phase, unit.Failure, unit.Words)
		}
		if got := wantedOf(t, w, orchard); !slices.Equal(got, []string{"den:started"}) {
			t.Errorf("the hash's Torrents are %v, want den's alone, started", got)
		}
		if got := goalRunOf(t, w, orchard).removed; got != string(downloadclients.Moved) {
			t.Errorf("seedbox's Orchard was removed %q, want moved", got)
		}
		stopped, started := announced(swarm, "stopped"), announced(swarm, "started")
		if len(stopped) == 0 || len(started) < 2 || started[len(started)-1].At.Before(stopped[len(stopped)-1].At) {
			t.Errorf("the tracker heard started %v and stopped %v, want seedbox's stop before den's start", started, stopped)
		}
	})
}

// wantedOf is each Torrent of s's hash as Node:wanted, removed ones left
// out.
func wantedOf(t *testing.T, w *sim.World, s seeded) []string {
	t.Helper()
	rows, err := w.Node("home").Half().Library.QueryContext(t.Context(), `SELECT n.name || ':' || t.wanted
		FROM downloadclients_torrent t JOIN nodes_node n ON n.id = t.node_id
		WHERE t.info_hash = ? AND t.removed_at IS NULL ORDER BY t.arriving_unit IS NOT NULL`, s.hash[:])
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// TestARefusedStartSeedsFromTheSourceAgainKeepingTheCopy moves Orchard
// from seedbox's built-in client to Transmission on den and to
// qBittorrent on seedbox, whose trackers never answer their start: each
// Move fails switch_refused with seedbox seeding again and the
// destination's copy kept paused, and a cancel then removes the copy.
func TestARefusedStartSeedsFromTheSourceAgainKeepingTheCopy(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ node, kind string }{{"den", "transmission"}, {"seedbox", "qbittorrent"}} {
		t.Run(c.kind, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				w, swarm := launchMoves(t)
				orchard := seedFor(t, w, swarm, addTracker(t, w, "ratiopool", indexers.Private), "Orchard")
				simtest.Eventually(t, "Orchard seeds on seedbox", func() bool { return seeding(t, w.Node("home").Half()) == 1 })

				batch := moveTorrent(t, w, orchard, moves.Side{Node: nodeID(t, w, c.node), Client: clientOn(t, w, c.node, c.kind)})
				unit := ended(t, w, batch)
				if unit.Phase != moves.PhaseFailed || unit.Failure.Code != moves.SwitchRefused {
					t.Fatalf("the unit ended %s %+v, want failed switch_refused", unit.Phase, unit.Failure)
				}
				if got, want := wantedOf(t, w, orchard), []string{"seedbox:started", c.node + ":paused"}; !slices.Equal(got, want) {
					t.Errorf("the hash's Torrents are %v, want %v", got, want)
				}
				if err := w.Node("home").Half().Moves.Cancel(t.Context(), unit.Unit); err != nil {
					t.Fatal(err)
				}
				within(t, time.Hour, "the cancel removes the kept copy", func() bool {
					return slices.Equal(wantedOf(t, w, orchard), []string{"seedbox:started"})
				})
			})
		})
	}
}
