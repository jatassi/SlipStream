package sim_test

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestARootMoveDeletesTheRootItEmpties moves every Work of movies-e to
// movies-f with "Delete this root when it's empty": once the unit ends,
// movies-e is gone, The Quiet Orchard lives in movies-f, and the batch
// finishes.
func TestARootMoveDeletesTheRootItEmpties(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := movestest.Launch(t)
		m := w.Moves()
		from, to := w.Root(t, "movies-e"), w.Root(t, "movies-f")
		plan, err := m.Preview(t.Context(), moves.Ask{
			Origin: moves.OriginRoot, Roots: []moves.RootTo{{Root: from, To: to}},
			Options: moves.Options{DeleteEmptiedRoots: true},
		})
		if err != nil || plan.Movable() != 1 {
			t.Fatalf("the preview is %+v (%v), want The Quiet Orchard moving", plan.Units, err)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
		if err != nil {
			t.Fatal(err)
		}
		simtest.Eventually(t, "the batch finishes", func() bool {
			b, _, err := m.Get(t.Context(), started.Batch, -1, 10)
			return err == nil && b.FinishedAt != nil
		})
		if unit := w.Unit(t, started.Batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		for _, r := range simtest.Decode[struct {
			Roots []simtest.RootRow `json:"roots"`
		}](t, "listRoots", w.API().Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots {
			if r.ID == from.String() {
				t.Errorf("movies-e is still a root after the Move emptied it: %+v", r)
			}
		}
		var root storage.UUID
		err = w.Library(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			return r.QueryRowContext(ctx, `SELECT root_id FROM roots_work WHERE work_id = ?`, w.Work).Scan(&root)
		})
		if err != nil || root != to.UUID {
			t.Errorf("The Quiet Orchard lives in %s (%v), want movies-f", root, err)
		}
		w.Moved(t, "movies-f")
	})
}
