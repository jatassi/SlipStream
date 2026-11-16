package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// CancelQueued clears every search queued in the bulk lane, Search all's
// and the Library's bulk Search, and answers how many. Each group keeps
// its due time, and a search already running finishes.
func (gr *Grabs) CancelQueued(ctx context.Context, _ identity.Admin) (int, error) {
	var (
		ws       []works.WorkID
		canceled int64
	)
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q, lane := db.New(tx), new(int64(pace.Bulk))
		var err error
		if ws, err = q.ListLaneWorks(ctx, lane); err != nil {
			return err
		}
		if err := q.DeleteLaneItems(ctx, lane); err != nil {
			return err
		}
		canceled, err = q.CancelLane(ctx, db.CancelLaneParams{ChangedAt: storage.Now(), Lane: lane})
		return err
	})
	if err != nil {
		return 0, err
	}
	changed := make(map[works.WorkID]bool, len(ws))
	for _, w := range ws {
		changed[w] = true
	}
	gr.queueChanged(ctx, changed)
	return int(canceled), nil
}
