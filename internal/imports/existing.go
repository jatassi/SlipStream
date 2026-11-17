package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Existing is which of ids are Imports still kept, for History's links.
func Existing(ctx context.Context, r storage.Reader[file], ids []ImportID) (map[ImportID]bool, error) {
	out := map[ImportID]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := db.New(r).ListExisting(ctx, ids)
	for _, id := range rows {
		out[id] = true
	}
	return out, err
}
