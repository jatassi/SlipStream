package conversions

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Retention deletes done and canceled Conversions that finished before
// Keep history for, with their attempts, and backfills older than it
// with no Conversion left. Failed Conversions are unfinished and stay.
func (c *Conversions) Retention() housekeeping.Step {
	return housekeeping.Aged("finished-conversions", "Finished Conversions", tasks.Pauses, setting.Noun{One: "Conversion", Many: "Conversions"}, "",
		func(ctx context.Context, before time.Time, _ struct{}, limit int) (housekeeping.Swept[struct{}], error) {
			var n int64
			err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				q := db.New(tx)
				var err error
				if n, err = q.DeleteFinishedBefore(ctx, db.DeleteFinishedBeforeParams{Before: storage.FromTime(before), N: int64(limit)}); err != nil {
					return err
				}
				_, err = q.DeleteEmptyBackfills(ctx, db.DeleteEmptyBackfillsParams{Before: storage.FromTime(before), N: int64(limit)})
				return err
			})
			return housekeeping.Swept[struct{}]{Seen: int(n), Deleted: int(n)}, err
		})
}
