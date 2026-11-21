package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// DropQueuedMoves deletes in tx every move and rename a restored Backup
// caught queued. The old Controller may have finished any of them, so the
// scan after the restore finds each file where it is.
func DropQueuedMoves(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).DeleteAllMoves(ctx)
}
