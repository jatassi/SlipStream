package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/indexers"
)

// SyncFeed runs RSS sync's unit of Indexer id once and answers its note.
func (gr *Grabs) SyncFeed(ctx context.Context, id indexers.ID) (string, error) {
	return gr.syncFeed(ctx, id)
}
