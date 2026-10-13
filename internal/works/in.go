package works

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
)

// TreeIn is Get inside the caller's own Read, for an owner that reads the
// tree with its own rows in one snapshot.
func (w *Works) TreeIn(ctx context.Context, r storage.Reader[file], id WorkID) (Tree, error) {
	return w.read(ctx, r, id)
}
