package indexers

import (
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Prune is Housekeeping's step for Indexers, in one Write: it completes
// as lost each query row whose job hasn't answered Timings.Unanswered
// after its reservation, deletes the query rows older than
// Timings.KeepQueries, and deletes the sessions of every Node that no
// longer holds Searching. The Controller keeps its sessions, since it
// covers Searching for a holder that lapsed.
func (x *Indexers) Prune(ctx context.Context) error {
	holding, self := x.cfg.Roster.Holding(role.Searching), x.cfg.Roster.Self()
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		now := storage.Now()
		if _, err := q.LoseUnanswered(ctx, db.LoseUnansweredParams{Now: now, Before: storage.FromTime(now.Time().Add(-x.cfg.Timings.Unanswered))}); err != nil {
			return err
		}
		if _, err := q.PruneQueries(ctx, storage.FromTime(now.Time().Add(-x.cfg.Timings.KeepQueries))); err != nil {
			return err
		}
		held, err := q.ListSessionNodes(ctx)
		if err != nil {
			return err
		}
		for _, node := range held {
			if node == self || slices.Contains(holding, node) {
				continue
			}
			if err := q.DeleteSessionsOn(ctx, node); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("prune the Indexers' queries and sessions: %w", err)
	}
	return nil
}
