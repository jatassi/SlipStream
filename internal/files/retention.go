package files

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Retention is the Import records step: every import record older than
// Keep history for. A live file keeps how it was placed, and a forced fill
// its Import, on their own rows.
func (f *Files) Retention() housekeeping.Step {
	return housekeeping.Aged("import-records", "Import records", setting.Noun{One: "import record", Many: "import records"}, "",
		func(ctx context.Context, before time.Time, _ struct{}, limit int) (housekeeping.Swept[struct{}], error) {
			var n int64
			err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
				var err error
				n, err = db.New(tx).DeleteImportedBefore(ctx, db.DeleteImportedBeforeParams{Before: storage.FromTime(before), Limit: int64(limit)})
				return err
			})
			return housekeeping.Swept[struct{}]{Seen: int(n), Deleted: int(n)}, err
		})
}
