package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// GrabCells is files.GrabCells over the Grabs: each cell takes its latest
// Grab's state, downloading once past fetching and failed once failed.
type GrabCells struct{}

func (GrabCells) GrabCells(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (map[files.CellKey]files.GrabState, error) {
	rows, err := db.New(r).ListTargetsOfWork(ctx, work)
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
