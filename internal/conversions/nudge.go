package conversions

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

// Queue writes one nudge for each ask and wakes the reconciler, whose
// drain waits on tx's commit. Asks of a backfill share one backfill.
func (s Seam) Queue(ctx context.Context, tx *storage.Tx[storage.Library], asks ...files.Ask) error {
	q := db.New(tx)
	var backfill BackfillID
	for _, a := range asks {
		var into BackfillID
		if a.Backfill {
			if backfill.IsZero() {
				var err error
				if backfill, err = newBackfill(ctx, q, "backfill", targets.TargetID{}, a); err != nil {
					return err
				}
			}
			into = backfill
		}
		if err := nudge(ctx, q, a, into); err != nil {
			return err
		}
	}
	s.Poke.Wake()
	return nil
}

func nudge(ctx context.Context, q *db.Queries, a files.Ask, into BackfillID) error {
	at := storage.FromTime(a.Since)
	if err := q.InsertNudge(ctx, db.InsertNudgeParams{WorkID: a.Work, Why: string(a.Why), Since: at, BackfillID: into, CreatedAt: at}); err != nil {
		return fmt.Errorf("nudge the Work's Conversions: %w", err)
	}
	return nil
}

func newBackfill(ctx context.Context, q *db.Queries, kind string, target targets.TargetID, a files.Ask) (BackfillID, error) {
	id := BackfillID{UUID: storage.NewUUID()}
	if err := q.InsertBackfill(ctx, db.InsertBackfillParams{ID: id, TargetID: target, Kind: kind, CreatedAt: storage.FromTime(a.Since)}); err != nil {
		return BackfillID{}, fmt.Errorf("record the backfill: %w", err)
	}
	return id, nil
}
