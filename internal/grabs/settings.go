package grabs

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// The Automatic search section's own settings, stored in grabs_settings.
// Backlog search's and RSS sync's cadences are tasks' (BacklogEvery and
// RSSEvery), and the wait for each Indexer is indexers.SearchWait, which
// the section shows and sets in the same Write.
var (
	BacklogOn       = setting.Switch{Key: "search.backlogOn", Label: "Backlog search", Default: true}
	BacklogUpgrades = setting.Switch{Key: "search.backlogUpgrades", Label: "Backlog search looks for upgrades", Default: true}
	RSSOn           = setting.Switch{Key: "search.rssOn", Label: "RSS sync", Default: true}
	LongestGap      = setting.Duration{
		Key: "search.longestGap", Label: "Search a Want that keeps coming up empty at least every",
		Default: 7 * day, Min: day, Max: 90 * day, Step: time.Hour,
	}
	FullPace = setting.Duration{
		Key: "search.fullPace", Label: "Don't space out searches for Items released in the last",
		Default: 14 * day, Min: 0, Max: 90 * day, Step: day,
	}
	StopCount = setting.Count{
		Key: "search.stopCount", Label: "Stop searching again after",
		Default: 3, Min: 1, Max: 20, Unit: setting.Noun{One: "failed Grab", Many: "failed Grabs"},
	}
	StopWindow = setting.Duration{
		Key: "search.stopWindow", Label: "Count failed Grabs in the last",
		Default: 7 * day, Min: time.Hour, Max: 30 * day, Step: time.Hour,
	}
)

// Settings is the Automatic search section.
func (gr *Grabs) Settings() setting.Panel { return panel{gr} }

type panel struct{ gr *Grabs }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var row db.GrabsSetting
	var wait *int64
	err := p.gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if row, err = db.New(r).GetSettings(ctx); err != nil {
			return err
		}
		wait, err = indexers.SearchWaitIn(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	return []setting.Field{
		BacklogOn.Field(row.BacklogOn), BacklogUpgrades.Field(row.BacklogUpgrades), RSSOn.Field(row.RssOn),
		LongestGap.Field(row.LongestGapSeconds), FullPace.Field(row.FullPaceSeconds),
		StopCount.Field(row.StopCount), StopWindow.Field(row.StopWindowSeconds),
		indexers.SearchWait.Field(wait),
	}, nil
}

// Apply stores the patch in one Write, all or none: grabs_settings and
// indexers.SearchWait together.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	if len(patch) == 0 {
		return nil
	}
	return p.gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		next := db.SetSettingsParams{
			BacklogOn: row.BacklogOn, BacklogUpgrades: row.BacklogUpgrades, RssOn: row.RssOn,
			LongestGapSeconds: row.LongestGapSeconds, FullPaceSeconds: row.FullPaceSeconds,
			StopCount: row.StopCount, StopWindowSeconds: row.StopWindowSeconds,
		}
		var wait *setting.Stored[int64]
		stored := map[string]func(json.RawMessage) error{
			BacklogOn.Key:       into(BacklogOn.ParseJSON, &next.BacklogOn),
			BacklogUpgrades.Key: into(BacklogUpgrades.ParseJSON, &next.BacklogUpgrades),
			RSSOn.Key:           into(RSSOn.ParseJSON, &next.RssOn),
			LongestGap.Key:      into(LongestGap.ParseJSON, &next.LongestGapSeconds),
			FullPace.Key:        into(FullPace.ParseJSON, &next.FullPaceSeconds),
			StopCount.Key:       into(StopCount.ParseJSON, &next.StopCount),
			StopWindow.Key:      into(StopWindow.ParseJSON, &next.StopWindowSeconds),
			indexers.SearchWait.Key: func(raw json.RawMessage) error {
				v, err := indexers.SearchWait.ParseJSON(raw)
				wait = &v
				return err
			},
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
		if err := q.SetSettings(ctx, next); err != nil {
			return err
		}
		if wait == nil {
			return nil
		}
		return indexers.SetSearchWaitIn(ctx, tx, *wait)
	})
}

// into parses a value with parse and keeps its override in to.
func into[T any](parse func(json.RawMessage) (setting.Stored[T], error), to **T) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := parse(raw)
		*to = v.Override()
		return err
	}
}
