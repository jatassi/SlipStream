package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Existing is which of ids are Grabs still kept, for History's links.
func Existing(ctx context.Context, r storage.Reader[file], ids []GrabID) (map[GrabID]bool, error) {
	out := map[GrabID]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := db.New(r).ListExisting(ctx, ids)
	for _, id := range rows {
		out[id] = true
	}
	return out, err
}
