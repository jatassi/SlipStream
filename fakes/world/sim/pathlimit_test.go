package sim_test

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestANodeGivenWindowsPathLimitBudgetsItsRoot launches seedbox-nas, whose
// nas writes paths of at most 259 bytes: the Controller budgets nas's
// movies root against 259, less the root folder and a separator.
func TestANodeGivenWindowsPathLimitBudgetsItsRoot(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, seedboxNAS(t), sim.WithWalk())
		movies := filepath.Join(w.Manifest().MediaDir, "movies")
		want := 259 - len(movies) - 1
		lib := w.Node("home").Half().Library
		budgets := func() map[roots.ID]int {
			var b map[roots.ID]int
			if err := lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
				var err error
				b, err = roots.BudgetsIn(ctx, rd)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			return b
		}
		simtest.Eventually(t, "nas reports its path limit", func() bool {
			b := budgets()
			return len(b) == 1 && slices.Contains(slices.Collect(maps.Values(b)), want)
		})
	})
}
