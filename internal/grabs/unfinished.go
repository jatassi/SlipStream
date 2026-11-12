package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// UnfinishedOf counts in r the Grabs of each of ws not yet imported or
// failed. A Work with none has no entry.
func UnfinishedOf(ctx context.Context, r storage.Reader[storage.Library], ws []works.WorkID) (map[works.WorkID]int, error) {
	rows, err := db.New(r).CountUnfinishedOfWorks(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]int, len(rows))
	for _, row := range rows {
		out[row.WorkID] = int(row.Grabs)
	}
	return out, nil
}
