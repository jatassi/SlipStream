package subtitles

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
)

// Writing is each of fs with a Speech Map building or a Sidecar write
// open beside it, read in r.
func Writing(ctx context.Context, r storage.Reader[storage.Library], fs []files.ID) (map[files.ID]bool, error) {
	ids, err := db.New(r).ListWriting(ctx, fs)
	if err != nil {
		return nil, err
	}
	out := make(map[files.ID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
