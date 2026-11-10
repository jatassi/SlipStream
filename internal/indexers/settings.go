package indexers

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
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

// SearchWait is how long a search waits for each Indexer's job, from when
// it first asked. Fetch, Test and captcha jobs keep Timings.Job. Its step
// is a second, since a Duration steps by a whole unit.
var SearchWait = setting.Duration{
	Key: "indexers.searchWait", Label: "Wait for each Indexer up to",
	Default: 2 * time.Minute, Min: 10 * time.Second, Max: 10 * time.Minute, Step: time.Second,
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
	return []setting.Field{SearchWait.Field(s.SearchWaitSeconds), LongBackoff.Field(s.LongBackoffSeconds)}, nil
}

func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	if len(patch) == 0 {
		return nil
	}
	err := p.x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		s, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		next := db.SetSettingsParams{LongBackoffSeconds: s.LongBackoffSeconds, SearchWaitSeconds: s.SearchWaitSeconds}
		stored := map[string]func(json.RawMessage) error{
			LongBackoff.Key: into(LongBackoff, &next.LongBackoffSeconds),
			SearchWait.Key:  into(SearchWait, &next.SearchWaitSeconds),
		}
		for _, key := range slices.Sorted(maps.Keys(patch)) {
			parse, ok := stored[key]
			if !ok {
				return &setting.UnknownKeyError{Key: key}
			}
			if err := parse(patch[key]); err != nil {
				return err
			}
		}
		return q.SetSettings(ctx, next)
	})
	if err != nil {
		return err
	}
	p.x.cfg.Settings.Publish(ctx, SettingsChanged{})
	return nil
}

// into parses a value of d and keeps its override in to.
func into(d setting.Duration, to **int64) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := d.ParseJSON(raw)
		*to = v.Override()
		return err
	}
}

// searchWait is the stored SearchWait, read in r.
func searchWait(ctx context.Context, r storage.Reader[file]) (time.Duration, error) {
	s, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return 0, err
	}
	return SearchWait.Value(s.SearchWaitSeconds), nil
}
