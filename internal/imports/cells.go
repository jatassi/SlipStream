package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Cells is files.Inflight: grabs' cells, and every holding path an
// unfinished Import's file or Grab claims. Stateless, so the composition
// root builds it before grabs.New and imports.Open.
type Cells struct{ Grabs grabs.GrabCells }

func (c Cells) GrabCells(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[files.CellKey]files.GrabCell, error) {
	return c.Grabs.GrabCells(ctx, r, ws)
}

// Holding is every path in a root's holding folder an unfinished Import's
// file claims, its Dest from the decision on while it isn't final or
// failed for the Admin to retry, and each open Grab's.
func (c Cells) Holding(ctx context.Context, r storage.Reader[file]) (map[roots.ID]map[string]bool, error) {
	out, err := c.Grabs.Holding(ctx, r)
	if err != nil {
		return nil, err
	}
	rows, err := db.New(r).ListHolding(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if out[row.RootID] == nil {
			out[row.RootID] = map[string]bool{}
		}
		out[row.RootID][*row.Dest] = true
	}
	return out, nil
}

func (c Cells) ReleaseTitles(ctx context.Context, r storage.Reader[file]) (map[storage.UUID]string, error) {
	return c.Grabs.ReleaseTitles(ctx, r)
}
