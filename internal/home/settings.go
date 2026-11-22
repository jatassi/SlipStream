package home

import (
	"context"

	"github.com/jatassi/slipstream/internal/home/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Rows caps Needs you, Jobs and Recent on Home.
var Rows = setting.Count{
	Key: "home.rows", Label: "Rows on Home", Default: 8, Min: 3, Max: 50,
	Unit: setting.Noun{One: "entry", Many: "entries"},
}

func (s *Service) Fields(ctx context.Context) ([]setting.Field, error) {
	rows, err := s.stored(ctx)
	if err != nil {
		return nil, err
	}
	return []setting.Field{Rows.Field(rows)}, nil
}

func (s *Service) stored(ctx context.Context) (*int64, error) {
	var rows *int64
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, err := db.New(r).GetSettings(ctx)
		rows = row.RowsShown
		return err
	})
	return rows, err
}

// Apply saves p and tells every Admin's Home.
func (s *Service) Apply(ctx context.Context, p setting.Patch) error {
	var set *setting.Stored[int64]
	for key, raw := range p {
		if key != Rows.Key {
			return &setting.UnknownKeyError{Key: key}
		}
		v, err := Rows.ParseJSON(raw)
		if err != nil {
			return err
		}
		set = &v
	}
	if set == nil {
		return nil
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetRows(ctx, set.Override())
	})
	if err != nil {
		return err
	}
	s.cfg.Changed.Publish(ctx, SettingsChanged{})
	return nil
}
