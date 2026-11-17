package history

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/storage"
)

// burst is how long Run waits after a ring before it publishes, so a
// burst of captures is one history.changed.
const burst = time.Second

// Run publishes history.changed once per burst of commits that recorded a
// newer entry, and once after a sweep that deleted rows or a Clear. It
// sweeps to the Clear floor at start and after each Clear, so a sweep a
// restart stopped finishes. It returns when ctx ends.
func (s *Service) Run(ctx context.Context) error {
	var sweeper sync.WaitGroup
	sweeper.Go(func() { s.sweepCleared(ctx) })
	defer sweeper.Wait()
	last, err := s.newest(ctx)
	if err != nil && ctx.Err() == nil {
		s.cfg.Log.Error("read the newest History entry", "err", err.Error())
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.bell:
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(burst):
		}
		drain(s.bell)
		newest, err := s.newest(ctx)
		if err != nil {
			if ctx.Err() == nil {
				s.cfg.Log.Error("read the newest History entry", "err", err.Error())
			}
			continue
		}
		if swept := drain(s.swept); newest <= last && !swept {
			continue
		}
		last = max(last, newest)
		s.cfg.Changed.Publish(ctx, Changed{})
	}
}

// drain is whether c had rung, emptying it.
func drain(c chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}

// newest is the newest entry's id, 0 for none. It reads in a Write, so a
// capture that rang the bell has committed or rolled back by then.
func (s *Service) newest(ctx context.Context) (EventID, error) {
	var id EventID
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		id, err = db.New(tx).NewestID(ctx)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		return err
	})
	return id, err
}

// sweepCleared deletes every entry at or below the Clear floor that the
// Outbox doesn't owe, once now and again each time Clear rings.
func (s *Service) sweepCleared(ctx context.Context) {
	for {
		_, err := housekeeping.Drain(ctx, s.cfg.Batch, func(ctx context.Context, after EventID, limit int) (housekeeping.Swept[EventID], error) {
			return s.sweep(ctx, time.Time{}, after, limit)
		})
		if err != nil && ctx.Err() == nil {
			s.cfg.Log.Error("sweep cleared History", "err", err.Error())
		}
		select {
		case <-ctx.Done():
			return
		case <-s.clear:
		}
	}
}
