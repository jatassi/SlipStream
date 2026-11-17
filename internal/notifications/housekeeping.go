package notifications

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// pruneBound is how many events one prune Write deletes.
const pruneBound = 1000

// Housekeeping is the step that deletes events older than Keep
// notifications for. Told rows stay, so a later clear still pairs.
func (s *Service) Housekeeping() housekeeping.Step {
	return housekeeping.Plain("notifications", "Notifications", s.prune)
}

func (s *Service) prune(ctx context.Context) error {
	var events, gaveUp int
	for {
		n, g, err := s.pruneSome(ctx, time.Now())
		if err != nil {
			return err
		}
		events, gaveUp = events+n, gaveUp+g
		if n < pruneBound {
			break
		}
	}
	if events == 0 {
		return nil
	}
	s.cfg.Log.Info("pruned old notifications", "events", events, "givenUp", gaveUp)
	s.ring()
	s.cfg.Changed.Publish(ctx, Changed{Feed: true, Connections: gaveUp > 0})
	return nil
}

// pruneSome deletes up to pruneBound events older than Keep notifications
// for. A message still due on one, which happens when Keep is shorter than
// Give up, is given up first and counted on its connection.
func (s *Service) pruneSome(ctx context.Context, now time.Time) (events, gaveUp int, err error) {
	err = s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		settings, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		ids, err := q.ListPrunable(ctx, db.ListPrunableParams{Before: storage.FromTime(now.Add(-KeepFor.Value(settings.KeepSeconds))), Limit: pruneBound})
		if err != nil || len(ids) == 0 {
			return err
		}
		due, err := q.ListDueIn(ctx, ids)
		if err != nil {
			return err
		}
		if err := countGivenUp(ctx, tx, due); err != nil {
			return err
		}
		for _, id := range ids {
			if err := q.DeleteEvent(ctx, id); err != nil {
				return err
			}
		}
		events, gaveUp = len(ids), len(due)
		return nil
	})
	return events, gaveUp, err
}

func countGivenUp(ctx context.Context, tx *storage.Tx[file], due []db.ListDueInRow) error {
	if len(due) == 0 {
		return nil
	}
	q := db.New(tx)
	conns, err := connectionsIn(ctx, tx)
	if err != nil {
		return err
	}
	lines, err := linesIn(ctx, tx)
	if err != nil {
		return err
	}
	counts := map[ConnectionID]int{}
	for _, d := range due {
		counts[d.ConnectionID]++
	}
	for _, c := range conns {
		if n := counts[c.ID]; n > 0 {
			l := lineFor(lines, c)
			l.GivenUp += n
			if err := q.UpsertLine(ctx, l.params(c.ID)); err != nil {
				return err
			}
		}
	}
	return nil
}
