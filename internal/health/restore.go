package health

import (
	"context"

	"github.com/jatassi/slipstream/internal/health/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// DeleteItems deletes every stored Health item in tx. A restored library's
// items were the old Controller's, and each check raises its own again.
func DeleteItems(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).DeleteAllItems(ctx)
}
