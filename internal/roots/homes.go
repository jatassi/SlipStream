package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// HomesOf reads the root each of ws lives in, inside a caller's read. A
// rootless Work is left out. It binds one variable per id, so a caller
// passes at most works.Batch.
func HomesOf(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]ID, error) {
	rows, err := db.New(r).ListHomesOf(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]ID, len(rows))
	for _, row := range rows {
		out[row.WorkID] = row.RootID
	}
	return out, nil
}
