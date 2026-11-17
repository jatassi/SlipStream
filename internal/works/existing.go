package works

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Existing is which of ids are Works still in the library, for the bell's
// links.
func Existing(ctx context.Context, r storage.Reader[file], ids []WorkID) (map[WorkID]bool, error) {
	out := map[WorkID]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := db.New(r).ListWorksByID(ctx, ids)
	for _, row := range rows {
		out[row.ID] = true
	}
	return out, err
}
