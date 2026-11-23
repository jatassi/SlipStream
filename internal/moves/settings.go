package moves

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	NeedsYouAfter = setting.Duration{
		Key: "moves.needsYouAfter", Label: "Show a Move waiting to switch in Needs you after",
		Default: time.Hour, Min: 5 * time.Minute, Max: 7 * 24 * time.Hour, Step: time.Minute,
	}
	PlexBound = setting.Duration{
		Key: "moves.plexBound", Label: "Delete the old files after Plex finds the new ones, or at most after",
		Default: 30 * time.Minute, Min: time.Minute, Max: 24 * time.Hour, Step: time.Minute,
	}
)

// Settings is the two stored settings, each its default while NULL.
type Settings struct {
	NeedsYouAfter, PlexBound time.Duration
}

func settingsIn(ctx context.Context, r storage.Reader[file]) (Settings, error) {
	row, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		NeedsYouAfter: NeedsYouAfter.Value(minutes(row.NeedsYouAfterMinutes)),
		PlexBound:     PlexBound.Value(minutes(row.PlexBoundMinutes)),
	}, nil
}

// minutes is a stored count of minutes as setting.Duration reads it.
func minutes(n *int64) *int64 {
	if n == nil {
		return nil
	}
	return new(*n * 60)
}

// Settings is the Moves heading of Settings > Library roots.
func (m *Moves) Settings() setting.Panel { return panel{m} }

type panel struct{ m *Moves }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var row db.GetSettingsRow
	err := p.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		row, err = db.New(r).GetSettings(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return []setting.Field{NeedsYouAfter.Field(minutes(row.NeedsYouAfterMinutes)), PlexBound.Field(minutes(row.PlexBoundMinutes))}, nil
}

// Apply stores the patch in one Write, all or none, and wakes the units
// that wait out the Plex bound so they judge it again.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	err := p.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		next := db.SetSettingsParams(row)
		stored := map[string]func(json.RawMessage) error{
			NeedsYouAfter.Key: inMinutes(NeedsYouAfter, &next.NeedsYouAfterMinutes),
			PlexBound.Key:     inMinutes(PlexBound, &next.PlexBoundMinutes),
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
		waiting := string(WaitMediaServers)
		return q.Wake(ctx, db.WakeParams{Now: storage.FromTime(p.m.now()), Waiting: &waiting})
	})
	if err != nil {
		return err
	}
	p.m.Poke()
	return nil
}

// inMinutes keeps d's override as whole minutes, the column's unit. d's
// Step is a minute, so the count is whole.
func inMinutes(d setting.Duration, to **int64) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := d.ParseJSON(raw)
		if o := v.Override(); o != nil {
			*to = new(*o / 60)
			return err
		}
		*to = nil
		return err
	}
}
