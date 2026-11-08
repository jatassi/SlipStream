package trackers

import (
	"context"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
)

// RepointTracker is trackers' indexers.Repoint, run in a merge's Write
// before from is deleted. Each of from's rules moves to into unless into
// has one from the same source, so an Admin's rule beats one from
// Releases and into's wins between two of one source. into keeps its own
// goal, else takes from's, and its own route from each home, else from's.
// Windows only change Tracker. Whatever stays on from is deleted with it.
func RepointTracker(ctx context.Context, tx *storage.Tx[file], from, into indexers.TrackerID) error {
	q := db.New(tx)
	if err := q.RepointRules(ctx, db.RepointRulesParams{IntoID: into, FromID: from}); err != nil {
		return err
	}
	if err := q.RepointGoal(ctx, db.RepointGoalParams{IntoID: into, FromID: from}); err != nil {
		return err
	}
	if err := q.RepointRoutes(ctx, db.RepointRoutesParams{IntoID: into, FromID: from}); err != nil {
		return err
	}
	return q.RepointWindows(ctx, db.RepointWindowsParams{IntoID: into, FromID: from})
}
