package moves

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Retention deletes the batches finished before "Keep history for" whose
// every unit ended and was undone; their units and files go with them. A
// batch with a unit not done is never deleted.
func Retention(lib *storage.DB[file]) housekeeping.Step {
	return housekeeping.Aged("finished-moves", "Finished Moves", tasks.Pauses, setting.Noun{One: "Move", Many: "Moves"}, "that are still running",
		func(ctx context.Context, before time.Time, _ struct{}, limit int) (housekeeping.Swept[struct{}], error) {
			var out housekeeping.Swept[struct{}]
			err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
				n, err := db.New(tx).DeleteFinishedBatches(ctx, db.DeleteFinishedBatchesParams{Before: storage.FromTime(before), N: int64(limit)})
				out = housekeeping.Swept[struct{}]{Seen: int(n), Deleted: int(n)}
				return err
			})
			return out, err
		})
}
