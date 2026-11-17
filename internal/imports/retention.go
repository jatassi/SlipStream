package imports

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Retention is the Finished Imports step: Imports that finished before
// Keep history for and whose Grab is gone or never was. An Import whose
// Grab the Finished Grabs step keeps stays with it. Its files, Items and
// fills go with it.
func (im *Imports) Retention() housekeeping.Step {
	return housekeeping.Aged("finished-imports", "Finished Imports", setting.Noun{One: "Import", Many: "Imports"}, "",
		func(ctx context.Context, before time.Time, _ struct{}, limit int) (housekeeping.Swept[struct{}], error) {
			var n int64
			err := im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
				var err error
				n, err = db.New(tx).DeletePrunable(ctx, db.DeletePrunableParams{Before: storage.FromTime(before), Limit: int64(limit)})
				return err
			})
			return housekeeping.Swept[struct{}]{Seen: int(n), Deleted: int(n)}, err
		})
}
