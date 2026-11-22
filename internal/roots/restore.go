package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// DeleteVolumes deletes every volume sample in tx. A restored library's
// samples were taken on the old Controller's machine.
func DeleteVolumes(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).DeleteAllVolumes(ctx)
}
