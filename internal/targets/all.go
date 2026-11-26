package targets

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
)

// All reads every Target in r.
func All(ctx context.Context, r storage.Reader[file]) ([]Target, error) {
	return listTargets(ctx, db.New(r))
}
