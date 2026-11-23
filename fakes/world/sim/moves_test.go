package sim_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/moves"
)

// TestAMoveSwitchesAndCleansUp moves The Quiet Orchard and its subtitle
// from den's movies-e to movies-f, another volume on den, and to
// seedbox's movies-d through a Transfer: both files arrive whole, the
// Work's rows name the new root, and movies-e keeps neither.
func TestAMoveSwitchesAndCleansUp(t *testing.T) {
	t.Parallel()
	for _, to := range []string{"movies-f", "movies-d"} {
		t.Run(to, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				w := movestest.Launch(t)
				batch := w.Start(t, to)
				if unit := w.Ended(t, batch); unit.Phase != moves.PhaseDone {
					t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
				}
				w.Invariants(t, batch)
				w.Moved(t, to)
			})
		})
	}
}

// TestACanceledMoveLeavesNothingAtTheDestination cancels a unit before
// its switch and sees its undo remove what it placed.
func TestACanceledMoveLeavesNothingAtTheDestination(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := movestest.Launch(t)
		batch := w.Start(t, "movies-f")
		if _, _, err := w.Moves().CancelBatch(t.Context(), batch); err != nil {
			t.Fatal(err)
		}
		simtest.Eventually(t, "the canceled unit is undone", func() bool {
			u := w.Unit(t, batch)
			return u.Phase == moves.PhaseCanceled && u.Undone
		})
		for rel := range w.Was {
			if _, err := os.Stat(filepath.Join(w.Roots["movies-f"].Path, filepath.FromSlash(rel))); !os.IsNotExist(err) {
				t.Errorf("a canceled Move left %s in movies-f: %v", rel, err)
			}
		}
		w.Invariants(t, batch)
	})
}

// TestAMoveCrashedBeforeEachStepEndsMovedOnce stops the Controller just
// before each step of a Move on one Node, from movies-e to movies-f, and
// starts it again, alone or with den offline until a minute after: the
// invariants hold at the restart, and the Move ends done with every file
// whole at the destination once.
func TestAMoveCrashedBeforeEachStepEndsMovedOnce(t *testing.T) {
	t.Parallel()
	crashBeforeEachStep(t, "movies-f")
}

// TestAMoveAcrossNodesCrashedBeforeEachStepEndsMovedOnce is
// TestAMoveCrashedBeforeEachStepEndsMovedOnce across Nodes, to seedbox's
// movies-d, with den or seedbox offline.
func TestAMoveAcrossNodesCrashedBeforeEachStepEndsMovedOnce(t *testing.T) {
	t.Parallel()
	crashBeforeEachStep(t, "movies-d")
}

func crashBeforeEachStep(t *testing.T, to string) {
	t.Helper()
	for _, step := range movestest.Steps {
		for _, fault := range movestest.Faults(t, to) {
			t.Run(step+"/"+fault.Name, func(t *testing.T) {
				t.Parallel()
				synctest.Test(t, func(t *testing.T) {
					crash := movestest.CrashAt(step)
					w := movestest.Launch(t, crash.Option())
					batch := w.Start(t, to)
					crash.Reached(t)
					w.Restart(t, fault)
					w.Invariants(t, batch)
					if unit := w.Ended(t, batch); unit.Phase != moves.PhaseDone {
						t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
					}
					w.Invariants(t, batch)
					w.Moved(t, to)
				})
			})
		}
	}
}
