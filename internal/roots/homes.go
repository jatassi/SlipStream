package roots

import (
	"context"

	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// HomesOf reads the root each of ws lives in, inside a caller's read. A
// rootless Work is left out.
func HomesOf(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]ID, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.RootsWork, error) { return q.ListEveryHome(ctx) }, func(ids []works.WorkID) ([]db.RootsWork, error) {
		return q.ListHomesOf(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]ID, len(rows))
	for _, row := range rows {
		out[row.WorkID] = row.RootID
	}
	return out, nil
}
