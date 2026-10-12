package clip

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/clip/internal/db"
)

func Get(ctx context.Context, r storage.Reader[storage.Library], id ID) (db.Clip, error) {
	return db.New(r).GetClip(ctx, id)
}

func Forget(ctx context.Context, tx *storage.Tx[storage.Library], id ID) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM clip WHERE id = ?", id)
	return err
}
