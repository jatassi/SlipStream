package downloadclients

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// DeleteVolumes deletes every download folder's volume sample in tx. A
// restored library's samples were taken on the old Controller's machine.
func DeleteVolumes(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).DeleteAllVolumes(ctx)
}
