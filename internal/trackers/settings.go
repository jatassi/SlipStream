package trackers

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
)

var (
	// Margin is added to every rule's seeding time, for what a Tracker
	// counts that nobody can see from outside.
	Margin = setting.Duration{
		Key: "trackers.margin", Label: "Margin past a rule",
		Default: 2 * time.Hour, Min: 0, Max: 7 * 24 * time.Hour, Step: time.Minute,
	}
	// WarnAfter is how long a torrent that owes may go without announcing
	// before Health warns.
	WarnAfter = setting.Duration{
		Key: "trackers.warnAfter", Label: "Warn when a torrent that owes hasn't announced for",
		Default: 6 * time.Hour, Min: time.Minute, Max: 7 * 24 * time.Hour, Step: time.Minute,
	}
)

// Settings is the margin and the warning delay. The default goal is edited
// whole through SetDefaultGoal, with the same editor as a Tracker's own.
func (t *Trackers) Settings() setting.Panel { return panel{t} }

type panel struct{ t *Trackers }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var s db.TrackersSetting
	err := p.t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		s, err = db.New(r).GetSettings(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return []setting.Field{Margin.Field(s.MarginSeconds), WarnAfter.Field(s.WarnAfterSeconds)}, nil
}

// stored is a panel setting and the query that stores it.
type stored struct {
	setting.Duration
	set func(*db.Queries, context.Context, *int64) error
}

var panelSettings = []stored{
	{Margin, (*db.Queries).SetMargin},
	{WarnAfter, (*db.Queries).SetWarnAfter},
}

func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	var sets []func(context.Context, *db.Queries) error
	for key, raw := range patch {
		i := slices.IndexFunc(panelSettings, func(s stored) bool { return s.Key == key })
		if i < 0 {
			return &setting.UnknownKeyError{Key: key}
		}
		v, err := panelSettings[i].ParseJSON(raw)
		if err != nil {
			return err
		}
		set := panelSettings[i].set
		sets = append(sets, func(ctx context.Context, q *db.Queries) error { return set(q, ctx, v.Override()) })
	}
	if len(sets) == 0 {
		return nil
	}
	err := p.t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		for _, set := range sets {
			if err := set(ctx, q); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	p.t.saved(ctx, nil)
	return nil
}
