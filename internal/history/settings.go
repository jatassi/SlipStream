package history

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

const day = 24 * time.Hour

// KeepFor bounds History and is Housekeeping's Tuning.
var KeepFor = setting.Duration{
	Key: "history.keepFor", Label: "Keep history for",
	Default: 90 * day, Min: 30 * day, Max: 1095 * day, Step: day,
}

// Keep is Keep history for's value now.
func (s *Service) Keep(ctx context.Context) (time.Duration, error) {
	keep, err := s.stored(ctx)
	return KeepFor.Value(keep), err
}

func (s *Service) Fields(ctx context.Context) ([]setting.Field, error) {
	keep, err := s.stored(ctx)
	if err != nil {
		return nil, err
	}
	return []setting.Field{KeepFor.Field(keep)}, nil
}

func (s *Service) stored(ctx context.Context) (*int64, error) {
	var keep *int64
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, err := db.New(r).GetSettings(ctx)
		keep = row.KeepSeconds
		return err
	})
	return keep, err
}

func (s *Service) Apply(ctx context.Context, p setting.Patch) error {
	var set *setting.Stored[int64]
	for key, raw := range p {
		if key != KeepFor.Key {
			return &setting.UnknownKeyError{Key: key}
		}
		v, err := KeepFor.ParseJSON(raw)
		if err != nil {
			return err
		}
		set = &v
	}
	if set == nil {
		return nil
	}
	return s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetKeep(ctx, set.Override())
	})
}
