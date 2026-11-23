package sim_test

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
)

// TestATorrentWaitsForItsGrabsImportBeforeItSwitches grabs The Quiet
// Orchard and holds its Import before the probe, then moves its Torrent
// from seedbox to den: the unit waits at importing while the Import is
// open, and once the Import finishes the Grab is imported and the
// Torrent moves.
func TestATorrentWaitsForItsGrabsImportBeforeItSwitches(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		hold, reached, free := holdImport()
		w := launchMoves(t, hold)
		simtest.Search(t, w, simtest.AddOrchard(t, w))
		<-reached
		defer free()
		var s seeded
		var hash []byte
		if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT id, info_hash FROM downloadclients_torrent
			WHERE removed_at IS NULL`).Scan(&s.id, &hash); err != nil {
			t.Fatal(err)
		}
		s.hash = bencode.InfoHash(hash)
		batch := moveTorrent(t, w, s, moves.Side{Node: nodeID(t, w, "den")})
		waitsForTheImport(t, w, batch)
		free()
		simtest.ExpectImported(t, simtest.Await(t, w))
		if unit := ended(t, w, batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v (%s), want done", unit.Phase, unit.Failure, unit.Words)
		}
		if got := wantedOf(t, w, s); len(got) != 1 || got[0] != "den:started" {
			t.Errorf("the hash's Torrents are %v, want den's alone, started", got)
		}
		if got := goalRunOf(t, w, s).removed; got != string(downloadclients.Moved) {
			t.Errorf("the source was removed %q, want moved", got)
		}
	})
}

// holdImport holds home's first Import before its probe until free runs,
// closing reached once it holds.
func holdImport() (hold sim.Option, reached <-chan struct{}, free func()) {
	at, release := make(chan struct{}), make(chan struct{})
	var held, released atomic.Bool
	hold = sim.Edit(func(name string, e *node.Env) {
		if name != "home" {
			return
		}
		e.Test.BeforeImport = func(_ context.Context, s imports.Step) {
			if s.At == imports.AtProbe && held.CompareAndSwap(false, true) {
				close(at)
				<-release
			}
		}
	})
	return hold, at, func() {
		if released.CompareAndSwap(false, true) {
			close(release)
		}
	}
}

// waitsForTheImport is batch's one unit once it waits at its switch for
// the Import, failing if it gets past its switch first.
func waitsForTheImport(t *testing.T, w *sim.World, batch moves.BatchID) moves.View {
	t.Helper()
	var unit moves.View
	within(t, 30*time.Minute, "the unit waits for the Import", func() bool {
		_, units, err := w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
		if err != nil || len(units) != 1 {
			t.Fatalf("batch %s has %d units (%v), want 1", batch, len(units), err)
		}
		unit = units[0]
		if unit.Phase != moves.PhaseQueued && unit.Phase != moves.PhaseTransfer && unit.Phase != moves.PhasePlace && unit.Phase != moves.PhaseSwitch {
			t.Fatalf("the unit is %s %+v with the Import open, want it waiting before its switch", unit.Phase, unit.Failure)
		}
		return unit.Phase == moves.PhaseSwitch && unit.Waiting == moves.WaitImporting
	})
	return unit
}
