package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
)

// Open is which of fs an unfinished Conversion reads or made, for S48
// (Moves).
func Open(context.Context, storage.Reader[storage.Library], []files.ID) (map[files.ID]bool, error) {
	return nil, errNotBuilt
}
