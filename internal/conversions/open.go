package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
)

// Open is which of fs an unfinished Conversion reads as its source, for
// S48 (Moves).
func Open(ctx context.Context, r storage.Reader[storage.Library], fs []files.ID) (map[files.ID]bool, error) {
	ids, err := db.New(r).ListReading(ctx, fs)
	if err != nil {
		return nil, err
	}
	out := make(map[files.ID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
