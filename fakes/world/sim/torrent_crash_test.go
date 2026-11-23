package sim_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
)

// handoffSteps are the steps of a torrent's Move from seedbox's built-in
// client to Transmission on den, in the order it takes them.
var handoffSteps = []string{"take_lane", "place_torrent", moves.RecordPlace, "arrived", "stop", "ask_start", "switch_torrent", "parted"}

// placedAtDen is where den's Transmission keeps The Quiet Orchard's
// payload.
func placedAtDen(w *sim.World) string {
	return filepath.Join(w.ClientFolder("den", "transmission"), "movie", orchard+".mkv")
}

// TestATorrentMoveCrashedBeforeEachStepEndsMovedOnce stops the Controller
// before each step of a torrent's Move across Nodes and starts it again:
// the Move ends done with the payload whole in den's Transmission,
// fake-tracker never sees two peers started at once, and seedbox's
// Torrent is removed as moved.
func TestATorrentMoveCrashedBeforeEachStepEndsMovedOnce(t *testing.T) {
	t.Parallel()
	crashBeforeEachHandoffStep(t, movestest.Fault{Name: "restart"})
}

// TestATorrentMoveCrashedWithSeedboxOfflineEndsMovedOnce is
// TestATorrentMoveCrashedBeforeEachStepEndsMovedOnce with seedbox offline
// until a minute after the restart.
func TestATorrentMoveCrashedWithSeedboxOfflineEndsMovedOnce(t *testing.T) {
	t.Parallel()
	crashBeforeEachHandoffStep(t, movestest.Fault{Name: "restart, seedbox offline", Offline: "seedbox"})
}

// TestATorrentMoveCrashedWithDenOfflineEndsMovedOnce is
// TestATorrentMoveCrashedBeforeEachStepEndsMovedOnce with den offline
// until a minute after the restart.
func TestATorrentMoveCrashedWithDenOfflineEndsMovedOnce(t *testing.T) {
	t.Parallel()
	crashBeforeEachHandoffStep(t, movestest.Fault{Name: "restart, den offline", Offline: "den"})
}

func crashBeforeEachHandoffStep(t *testing.T, fault movestest.Fault) {
	t.Helper()
	for _, step := range handoffSteps {
		t.Run(step, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { crashedMoveEndsMovedOnce(t, step, fault) })
		})
	}
}

func crashedMoveEndsMovedOnce(t *testing.T, step string, fault movestest.Fault) {
	crash := movestest.CrashAt(step)
	w := launchMoves(t, crash.Option())
	s := seedOrchard(t, w, kinds[0])
	batch := moveTorrent(t, w, s, client{"den", "transmission"}.side(t, w))
	crash.Reached(t)
	movestest.World{World: w}.Restart(t, fault)
	if unit := ended(t, w, batch); unit.Phase != moves.PhaseDone {
		t.Fatalf("the unit ended %s %+v (%s), want done", unit.Phase, unit.Failure, unit.Words)
	}
	if got := wantedOf(t, w, s); !slices.Equal(got, []string{"den:started"}) {
		t.Errorf("the hash's Torrents are %v, want den's alone, started", got)
	}
	if got := goalRunOf(t, w, s).removed; got != string(downloadclients.Moved) {
		t.Errorf("seedbox's Torrent was removed %q, want moved", got)
	}
	if got, err := os.ReadFile(placedAtDen(w)); err != nil || !bytes.Equal(got, sim.Payload(orchard)) {
		t.Errorf("den's Transmission holds %d bytes of the payload (%v), want it whole", len(got), err)
	}
	if wrong := oneLive(w.Announces(s.hash)); wrong != "" {
		t.Error(wrong)
	}
}

// TestACancelAfterAnUnrecordedPlaceLeavesNothingAtTheDestination stops
// the Controller once the destination placed the payload in its client's
// folder and before any arriving Torrent recorded it, across Nodes into
// den's Transmission and on one Node into seedbox's qBittorrent, and
// cancels the Move with the destination offline: once it is back, the
// payload is gone from that folder and seedbox still seeds.
func TestACancelAfterAnUnrecordedPlaceLeavesNothingAtTheDestination(t *testing.T) {
	t.Parallel()
	for _, to := range []client{{"den", "transmission"}, {"seedbox", "qbittorrent"}} {
		t.Run(to.String(), func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				crash := movestest.CrashAt(moves.RecordPlace)
				w := launchMoves(t, crash.Option())
				s := seedOrchard(t, w, kinds[0])
				batch := moveTorrent(t, w, s, to.side(t, w))
				crash.Reached(t)
				placed := filepath.Join(w.ClientFolder(to.node, to.kind), "movie", orchard+".mkv")
				if _, err := os.Stat(placed); err != nil {
					t.Fatalf("%s placed nothing before the crash: %v", to, err)
				}
				home, dest := w.Node("home"), w.Node(to.node)
				link := home.Ready().Listeners.Link
				dest.Stop()
				home.Stop()
				home.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_LINK_LISTEN="+link.String()) })
				_, units, err := home.Half().Moves.Get(t.Context(), batch, -1, 10)
				if err != nil || len(units) != 1 {
					t.Fatalf("batch %s has %d units (%v), want 1", batch, len(units), err)
				}
				if err := home.Half().Moves.Cancel(t.Context(), units[0].Unit); err != nil {
					t.Fatal(err)
				}
				time.Sleep(time.Minute)
				dest.Start()
				within(t, time.Hour, "the undo discards the placed payload", func() bool {
					_, err := os.Stat(placed)
					return os.IsNotExist(err)
				})
				within(t, time.Hour, "seedbox seeds alone", func() bool {
					return slices.Equal(wantedOf(t, w, s), []string{"seedbox:started"}) && seeding(t, home.Half()) == 1
				})
			})
		})
	}
}
