package sim_test

import (
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestPreviewQueryCount previews 1,000 movies, half on seedbox and half on
// den, to den's movies-f, and holds the preview to its query budget: a
// few queries a page of Works and a few for the whole preview, never a
// query a Work.
func TestPreviewQueryCount(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		const n, each, whole = 1000, 20, 20
		w := movestest.Seed(t, n)
		to := w.Root(t, "movies-f")
		ask := moves.Ask{Origin: moves.OriginWork}
		for _, id := range w.Seeded {
			ask.Works = append(ask.Works, moves.WorkTo{Work: id, To: to})
		}
		ctx, queries := storage.Counted(t.Context())
		plan, err := w.Moves().Preview(ctx, ask)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Movable() != n {
			t.Errorf("the preview moves %d of %d movies: %s", plan.Movable(), n, plan.Recommendation)
		}
		pages := (n + moves.DefaultTimings().PlanBatch - 1) / moves.DefaultTimings().PlanBatch
		if q, budget := queries(), int64(whole+each*pages); q > budget {
			t.Errorf("the preview of %d movies in %d pages ran %d queries, want at most %d", n, pages, q, budget)
		}
	})
}
