package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// CancelOpenRuns ends in tx every queue search run a restored Backup
// caught open: the process that ran it is gone. Its queued search stays
// queued and runs again once Reconciling ends.
func CancelOpenRuns(ctx context.Context, tx *storage.Tx[file]) error {
	_, err := db.New(tx).DeleteOpenQueueRuns(ctx)
	return err
}
