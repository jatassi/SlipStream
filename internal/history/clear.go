package history

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

// Clear raises the floor to the newest entry in one Write, so History
// lists nothing older at once, and rings Run to sweep it. Running it twice
// is the same as once, and a later Clear raises the floor further.
func (s *Service) Clear(ctx context.Context, _ identity.Admin) error {
	var through EventID
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		id, err := q.NewestID(ctx)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		through = id
		return q.RaiseCleared(ctx, db.RaiseClearedParams{Through: id, At: storage.Now()})
	})
	if err != nil {
		return err
	}
	s.cfg.Log.Info("cleared History", "through", int64(through))
	ring(s.clear)
	ring(s.swept)
	ring(s.bell)
	return nil
}
