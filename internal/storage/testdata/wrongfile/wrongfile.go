package wrongfile

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
)

func putOutbox(context.Context, *storage.Tx[storage.Local]) error     { return nil }
func readOutbox(context.Context, storage.Reader[storage.Local]) error { return nil }

func Wrong(ctx context.Context, lib *storage.DB[storage.Library]) error {
	return lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := putOutbox(ctx, tx); err != nil {
			return err
		}
		return readOutbox(ctx, tx)
	})
}
