package sim_test

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage"
)

func onTorrents(t *testing.T, w *sim.World, query string, args ...any) {
	t.Helper()
	err := w.Node("home").Half().Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func torrentsOf(t *testing.T, w *sim.World, hash string) int64 {
	t.Helper()
	var n int64
	if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT count(*) FROM downloadclients_torrent WHERE hex(info_hash) = upper(?)", hash).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestAttachFollowsTheTorrentsState fails seedbox's first Grab at probing,
// so its Torrent stays complete and seeding, sets that Torrent's state,
// and searches again: the second Grab attaches to the Torrent, and the
// Torrent's state alone decides what it does.
func TestAttachFollowsTheTorrentsState(t *testing.T) {
	for _, c := range []struct {
		name string
		set  string
		// then checks the second Grab and the Torrents of its hash.
		then func(t *testing.T, w *sim.World, first grabView)
	}{
		{name: "seeding imports at once", then: func(t *testing.T, w *sim.World, first grabView) {
			next := awaitSecond(t, w)
			expectImported(t, next)
			if next.NodeID != first.NodeID || torrentsOf(t, w, first.InfoHash) != 1 {
				t.Fatalf("the second Grab downloaded on %s with %d Torrents of its hash, want seedbox %s's one", next.NodeID, torrentsOf(t, w, first.InfoHash), first.NodeID)
			}
		}},
		{
			name: "failed fails the Grab",
			set:  `UPDATE downloadclients_torrent SET report = '{"at":"2026-10-19T21:00:00Z","engine":"failed","error":"disk on fire","done":0}', reported_at = 1`,
			then: func(t *testing.T, w *sim.World, _ grabView) {
				next := awaitSecond(t, w)
				expectFailed(t, next, "downloading", "download_failed", "disk on fire")
			},
		},
		{
			name: "Removing waits for the confirmation, then adds a new Torrent",
			set:  `UPDATE downloadclients_torrent SET removed_at = 1, removed_reason = 'admin'`,
			then: func(t *testing.T, w *sim.World, first grabView) {
				time.Sleep(time.Minute)
				synctest.Wait()
				if gs := grabsListed(t, w); len(gs) != 2 || gs[0].Stage != "fetching" {
					t.Fatalf("while seedbox removes the Torrent the Grabs are %+v, want the second waiting at fetching", gs)
				}
				onTorrents(t, w, `UPDATE downloadclients_torrent SET parted_at = 2, freed = 0`)
				next := awaitSecond(t, w)
				expectImported(t, next)
				if n := torrentsOf(t, w, first.InfoHash); n != 2 || next.NodeID != first.NodeID {
					t.Fatalf("the second Grab downloaded on %s with %d Torrents of its hash, want a new one on seedbox %s beside the parted one", next.NodeID, n, first.NodeID)
				}
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Media = &failsOnce{Prober: e.Media} }))
				work := addOrchard(t, w)
				search(t, w, work)
				first := await(t, w)
				if first.Stage != "failed" || first.Failure == nil || first.Failure.Stage != "probing" {
					t.Fatalf("the first Grab ended %s with %+v, want failed at probing", first.Stage, first.Failure)
				}
				if c.set != "" {
					onTorrents(t, w, c.set)
				}
				search(t, w, work)
				c.then(t, w, first)
			})
		})
	}
}

// awaitSecond waits in fake time until the world's second Grab is done,
// and returns it.
func awaitSecond(t *testing.T, w *sim.World) grabView {
	t.Helper()
	var next grabView
	eventually(t, "the second Grab finishes", func() bool {
		gs := grabsListed(t, w)
		if len(gs) == 2 {
			next = gs[0]
		}
		return next.Stage == "imported" || next.Stage == "failed"
	})
	return next
}

// TestARemoveBetweenTheReadAndTheCommitFailsTheGrab removes the Torrent
// after the reconciler read it complete and before it commits the move to
// probing: the commit sees the removal and fails the Grab with removed.
func TestARemoveBetweenTheReadAndTheCommitFailsTheGrab(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var lib atomic.Pointer[storage.DB[storage.Library]]
		w := sim.Launch(t, home(t), sim.WithWalk(), sim.Edit(func(name string, e *node.Env) {
			if name != "home" {
				return
			}
			e.Test.BeforeStep = func(ctx context.Context, s grabs.Stage) {
				if s != grabs.Downloading {
					return
				}
				err := lib.Load().Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
					_, err := tx.ExecContext(ctx, `UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = 'admin' WHERE removed_at IS NULL`, storage.Now())
					return err
				})
				if err != nil {
					t.Error(err)
				}
			}
		}))
		lib.Store(w.Node("home").Half().Library)
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectFailed(t, g, "downloading", "removed", "The Admin removed this Grab's torrent")
		if got := g.stages(); !slices.Equal(got, []string{"fetching", "downloading", "failed"}) {
			t.Fatalf("the Grab's history is %v, want [fetching downloading failed]", got)
		}
	})
}
