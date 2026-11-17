package history

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Retention is the History step: entries older than Keep history for, and
// everything at or below the Clear floor, except what the Outbox owes.
func (s *Service) Retention() housekeeping.Step {
	return housekeeping.Aged("history", "History", setting.Noun{One: "entry", Many: "entries"}, "that are still being sent",
		func(ctx context.Context, before time.Time, after EventID, limit int) (housekeeping.Swept[EventID], error) {
			return s.sweep(ctx, before, after, limit)
		})
}

// sweep is one Write: up to limit entries after after, at or below the
// newer of the last entry captured before before and the Clear floor,
// less what the Outbox owes. Deleting an entry deletes its Works, its
// Health items and notifications' rows of it.
func (s *Service) sweep(ctx context.Context, before time.Time, after EventID, limit int) (housekeeping.Swept[EventID], error) {
	var out housekeeping.Swept[EventID]
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		out = housekeeping.Swept[EventID]{}
		q := db.New(tx)
		ceiling, err := q.LastIDBefore(ctx, storage.FromTime(before))
		if err != nil && !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		cleared, err := q.GetCleared(ctx)
		if err != nil {
			return err
		}
		ids, err := q.ListSweep(ctx, db.ListSweepParams{After: after, Ceiling: max(ceiling, cleared), Limit: int64(limit)})
		if err != nil || len(ids) == 0 {
			return err
		}
		owed, err := s.cfg.Outbox.Owed(ctx, tx, ids)
		if err != nil {
			return err
		}
		gone := slices.DeleteFunc(slices.Clone(ids), func(id EventID) bool { return owed[id] })
		if len(gone) > 0 {
			if err := q.DeleteEvents(ctx, gone); err != nil {
				return err
			}
		}
		out = housekeeping.Swept[EventID]{Seen: len(ids), Deleted: len(gone), Last: ids[len(ids)-1]}
		return nil
	})
	if err != nil || out.Deleted == 0 {
		return out, err
	}
	s.cfg.Log.Info("pruned History entries", "deleted", out.Deleted, "kept", out.Seen-out.Deleted)
	ring(s.swept)
	ring(s.bell)
	return out, nil
}
