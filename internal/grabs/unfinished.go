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

// CancelTransfersIn cancels in tx the Transfer of each Grab of ws, for a
// delete whose Write takes those Grabs with their Works. Call
// TouchTransfers after the commit.
func (gr *Grabs) CancelTransfersIn(ctx context.Context, tx *storage.Tx[storage.Library], ws []works.WorkID) error {
	ids, err := db.New(tx).ListGrabsOfWorks(ctx, ws)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := gr.cancelTransfer(ctx, tx, id); err != nil {
			return err
		}
	}
	return nil
}

// TouchTransfers tells the Transfers loop a delete's Write cancelled
// Transfers.
func (gr *Grabs) TouchTransfers() { gr.cfg.Transfers.Touch() }
