package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// GrabCells is files.Inflight over the Grabs: each cell takes its latest
// Grab's state, downloading once past fetching and failed once failed.
type GrabCells struct{}

func (GrabCells) GrabCells(ctx context.Context, r storage.Reader[storage.Library], ws []works.WorkID) (map[files.CellKey]files.GrabState, error) {
	rows, err := db.New(r).ListTargetsOfWorks(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := map[files.CellKey]files.GrabState{}
	for _, row := range rows {
		key := files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}
		switch row.Stage {
		case Failed:
			out[key] = files.GrabFailed
		case Fetching, Imported:
			delete(out, key)
		case Downloading, Probing, Transferring, Importing:
			out[key] = files.GrabDownloading
		}
	}
	return out, nil
}

// Unfinished lists every Grab not yet imported or failed.
func (GrabCells) Unfinished(ctx context.Context, r storage.Reader[storage.Library]) ([]storage.UUID, error) {
	rows, err := db.New(r).ListOpenGrabs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]storage.UUID, len(rows))
	for i, row := range rows {
		out[i] = row.ID.UUID
	}
	return out, nil
}

// Settle records u on each unfinished Grab of work all of whose cells are
// in gone. Settling one twice changes nothing.
func (GrabCells) Settle(ctx context.Context, tx *storage.Tx[storage.Library], work works.WorkID, gone map[files.CellKey]bool, u files.Unwanted) (int, error) {
	q := db.New(tx)
	rows, err := q.ListTargetsOfWork(ctx, work)
	if err != nil {
		return 0, err
	}
	var order []GrabID
	left := map[GrabID]bool{}
	for _, row := range rows {
		if row.Stage == Imported || row.Stage == Failed {
			continue
		}
		if _, seen := left[row.ID]; !seen {
			order = append(order, row.ID)
			left[row.ID] = false
		}
		if !gone[files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}] {
			left[row.ID] = true
		}
	}
	n := 0
	for _, id := range order {
		if left[id] {
			continue
		}
		if err := q.SettleGrab(ctx, db.SettleGrabParams{ID: id, Unwanted: &u}); err != nil {
			return 0, err
		}
		n++
	}
	return n, nil
}
