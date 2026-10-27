package indexers

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// LongBackoff is how long every request to an Indexer may fail before
// Health warns.
var LongBackoff = setting.Duration{
	Key: "indexers.longBackoff", Label: "Warn when every request has failed for",
	Default: 6 * time.Hour, Min: time.Hour, Max: 7 * 24 * time.Hour, Step: time.Hour,
}

// Settings is the Indexer settings an Admin sets in Settings > Indexers.
func (x *Indexers) Settings() setting.Panel { return panel{x} }

type panel struct{ x *Indexers }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var s db.IndexersSetting
	err := p.x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		s, err = db.New(r).GetSettings(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return []setting.Field{LongBackoff.Field(s.LongBackoffSeconds)}, nil
}

func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	for key := range patch {
		if key != LongBackoff.Key {
			return &setting.UnknownKeyError{Key: key}
		}
	}
	raw, ok := patch[LongBackoff.Key]
	if !ok {
		return nil
	}
	v, err := LongBackoff.ParseJSON(raw)
	if err != nil {
		return err
	}
	return p.x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetLongBackoff(ctx, v.Override())
	})
}
