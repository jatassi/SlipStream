package history

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// publishGap is the least time between two history.changed: a burst of
// captures is one.
const publishGap = time.Second

// Run publishes history.changed after commits that recorded a newer entry
// and after a sweep that deleted rows, at most once a publishGap. It
// returns when ctx ends.
func (s *Service) Run(ctx context.Context) error {
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
		newest, err := s.newest(ctx)
		if err != nil {
			if ctx.Err() == nil {
				s.cfg.Log.Error("read the newest History entry", "err", err.Error())
			}
			continue
		}
		swept := false
		select {
		case <-s.swept:
			swept = true
		default:
		}
		if newest <= last && !swept {
			continue
		}
		last = max(last, newest)
		s.cfg.Changed.Publish(ctx, Changed{})
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(publishGap):
		}
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
