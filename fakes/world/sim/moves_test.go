package sim_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

const orchard = "The Quiet Orchard (2021) {tmdb-9000001}/The Quiet Orchard (2021).mkv"

// movingWorld launches the moves Scenario with The Quiet Orchard in den's
// movies-e, and answers its Work once the root's first scan recorded it,
// with den's roots by folder.
func movingWorld(t *testing.T) (w *sim.World, work works.WorkID, byFolder map[string]simtest.RootRow) {
	t.Helper()
	s, err := scenario.Load("moves")
	if err != nil {
		t.Fatal(err)
	}
	s.Library.Folders = append(s.Library.Folders, scenario.LibraryFolder{Folder: "movies-e", Files: []scenario.MediaFile{{
		Path: orchard, Video: scenario.MediaVideo{Width: 1920, Height: 1080, Seconds: 6, HDR: "sdr"},
	}}})
	w = sim.Launch(t, s, sim.WithWalk())
	byFolder = map[string]simtest.RootRow{}
	for _, r := range simtest.Decode[struct {
		Roots []simtest.RootRow `json:"roots"`
	}](t, "listRoots", w.API().Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots {
		byFolder[filepath.Base(r.Path)] = r
	}
	simtest.Eventually(t, "movies-e's first scan records The Quiet Orchard", func() bool {
		err := w.Node("home").Half().Library.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			return r.QueryRowContext(ctx, `SELECT work_id FROM files_file LIMIT 1`).Scan(&work)
		})
		return err == nil
	})
	return w, work, byFolder
}

func rootID(t *testing.T, r simtest.RootRow) roots.ID {
	t.Helper()
	id, err := storage.ParseUUID(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	return roots.ID{UUID: id}
}

// ended is the one unit of batch once it is done or failed.
func ended(t *testing.T, m *moves.Moves, batch moves.BatchID) moves.View {
	t.Helper()
	var unit moves.View
	simtest.Eventually(t, "the unit finishes", func() bool {
		_, units, err := m.Get(t.Context(), batch, -1, 10)
		if err != nil || len(units) != 1 {
			return false
		}
		unit = units[0]
		return unit.Phase == moves.PhaseDone || unit.Phase == moves.PhaseFailed
	})
	return unit
}

// TestASameNodeMoveCopiesSwitchesAndCleansUp moves a Work from den's
// movies-e to movies-f, another volume: the copy lands, the Work's row
// names movies-f, and the source file is gone.
func TestASameNodeMoveCopiesSwitchesAndCleansUp(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, work, byFolder := movingWorld(t)
		m := w.Node("home").Half().Moves
		to := rootID(t, byFolder["movies-f"])
		plan, err := m.Preview(t.Context(), moves.Ask{Origin: moves.OriginWork, Works: []moves.WorkTo{{Work: work, To: to}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Units) != 1 || plan.Units[0].Verdict.Color == moves.Red {
			t.Fatalf("the preview is %+v, want one unit that moves", plan.Units)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, plan.Movable())
		if err != nil {
			t.Fatal(err)
		}
		again, err := m.Start(t.Context(), identity.Admin{}, plan.Token, plan.Movable())
		if err != nil || again.Batch != started.Batch {
			t.Fatalf("a replayed Start answered %+v, %v, want batch %s again", again, err, started.Batch)
		}
		if unit := ended(t, m, started.Batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		var at roots.ID
		err = w.Node("home").Half().Library.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			return r.QueryRowContext(ctx, `SELECT root_id FROM files_file WHERE work_id = ?`, work).Scan(&at)
		})
		if err != nil || at != to {
			t.Fatalf("the Work's file is in root %s (%v), want movies-f %s", at, err, to)
		}
		if _, err := os.Stat(filepath.Join(byFolder["movies-f"].Path, filepath.FromSlash(orchard))); err != nil {
			t.Errorf("the copy isn't in movies-f: %v", err)
		}
		if _, err := os.Stat(filepath.Join(byFolder["movies-e"].Path, filepath.FromSlash(orchard))); !os.IsNotExist(err) {
			t.Errorf("the source file is still in movies-e: %v", err)
		}
	})
}

// TestACanceledMoveLeavesNothingAtTheDestination cancels a unit before
// its switch and sees its undo remove what it placed.
func TestACanceledMoveLeavesNothingAtTheDestination(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, work, byFolder := movingWorld(t)
		m := w.Node("home").Half().Moves
		to := rootID(t, byFolder["movies-f"])
		plan, err := m.Preview(t.Context(), moves.Ask{Origin: moves.OriginWork, Works: []moves.WorkTo{{Work: work, To: to}}})
		if err != nil {
			t.Fatal(err)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, plan.Movable())
		if err != nil {
			t.Fatal(err)
		}
		if err := m.CancelBatch(t.Context(), started.Batch); err != nil {
			t.Fatal(err)
		}
		simtest.Eventually(t, "the canceled unit is undone", func() bool {
			_, units, err := m.Get(t.Context(), started.Batch, -1, 10)
			return err == nil && len(units) == 1 && units[0].Phase == moves.PhaseCanceled && units[0].Undone
		})
		if _, err := os.Stat(filepath.Join(byFolder["movies-f"].Path, filepath.FromSlash(orchard))); !os.IsNotExist(err) {
			t.Errorf("a canceled Move left its copy in movies-f: %v", err)
		}
		if _, err := os.Stat(filepath.Join(byFolder["movies-e"].Path, filepath.FromSlash(orchard))); err != nil {
			t.Errorf("a canceled Move lost its source file: %v", err)
		}
	})
}
