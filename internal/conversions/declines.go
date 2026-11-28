package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
)

// declineIf remembers the pair when the Conversion ends for why, so the
// drain never queues it again until its source changes or the Admin asks
// for it by name.
func declineIf(why CancelWhy) effect {
	return func(ctx context.Context, m *move) error {
		if m.row.CancelWhy != why || m.row.SourceFile == nil || m.row.Target == nil {
			return nil
		}
		return m.q.Decline(ctx, db.DeclineParams{SourceFileID: *m.row.SourceFile, TargetID: *m.row.Target})
	}
}
