package works

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Also runs inside Delete's Write before any row goes, with the ids of ws
// a Work has, in id order. It answers those to remove: another owner's
// rows for them go in the same Write, and a Work it leaves out stays
// whole.
type Also func(ctx context.Context, tx *storage.Tx[file], present []WorkID) ([]WorkID, error)

// Delete removes Works in one Write: each of ws that also answers, with
// its levels, Items, title rows and every row another owner keeps for it,
// which go by cascade. It publishes one works.removed naming them, none
// when it removed none. An id no Work has is left out, so a repeat
// converges.
func (w *Works) Delete(ctx context.Context, _ identity.Admin, ws []WorkID, also Also) ([]WorkID, error) {
	var removed []WorkID
	err := w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		present, err := presentIn(ctx, q, ws)
		if err != nil {
			return err
		}
		if removed, err = also(ctx, tx, present); err != nil || len(removed) == 0 {
			return err
		}
		return q.DeleteWorks(ctx, removed)
	})
	if err != nil {
		return nil, err
	}
	if len(removed) > 0 {
		w.cfg.Removed.Publish(ctx, RemovedOf(removed...))
	}
	return removed, nil
}
