package backups

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/backups/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

var WarnAfter = setting.Count{
	Key: "backups.warnAfter", Label: "Warn after this many missed Backups",
	Default: 2, Min: 1, Max: 30, Unit: noun,
}

// Cadences is the Backup task's cadence as tasks stores it.
type Cadences interface {
	Every(ctx context.Context, id tasks.TaskID) (*int64, error)
	// EveryIn is Every read in r, inside the caller's Read or Write.
	EveryIn(ctx context.Context, r storage.Reader[file], id tasks.TaskID) (*int64, error)
	SetCadence(ctx context.Context, id tasks.TaskID, v setting.Stored[int64]) error
}

// Settings is the Backups section: Back up every, which tasks stores, and
// backups_settings.
func (b *Backups) Settings() setting.Panel { return panel{b} }

type panel struct{ b *Backups }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var row db.GetSettingsRow
	err := p.b.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		row, err = db.New(r).GetSettings(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	every, err := p.b.cfg.Tasks.Every(ctx, TaskID)
	if err != nil {
		return nil, err
	}
	return []setting.Field{
		Cadence.Field(every),
		KeepScheduled.Field(seconds(row.KeepScheduledDays, day)),
		KeepEach.Field(row.KeepEach),
		WarnAfter.Field(row.WarnAfter),
		CopyWaitLimit.Field(seconds(row.CopyWaitLimitHours, time.Hour)),
		CheckCopies.Field(seconds(row.CheckCopiesDays, day)),
	}, nil
}

// Apply stores the patch, all or none: backups_settings in one Write, then
// Back up every through tasks.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	var cadence *setting.Stored[int64]
	if raw, ok := patch[Cadence.Key]; ok {
		v, err := Cadence.ParseJSON(raw)
		if err != nil {
			return err
		}
		cadence = &v
	}
	err := p.b.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return store(ctx, db.New(tx), patch)
	})
	if err != nil {
		return err
	}
	if cadence != nil {
		if err := p.b.cfg.Tasks.SetCadence(ctx, TaskID, *cadence); err != nil {
			return err
		}
	}
	p.b.cfg.Changed.Publish(ctx, Changed{})
	return nil
}

func store(ctx context.Context, q *db.Queries, patch setting.Patch) error {
	row, err := q.GetSettings(ctx)
	if err != nil {
		return err
	}
	next := db.SetSettingsParams(row)
	stored := map[string]func(json.RawMessage) error{
		Cadence.Key:       func(json.RawMessage) error { return nil },
		KeepScheduled.Key: inUnits(KeepScheduled, day, &next.KeepScheduledDays),
		KeepEach.Key:      count(KeepEach, &next.KeepEach),
		WarnAfter.Key:     count(WarnAfter, &next.WarnAfter),
		CopyWaitLimit.Key: inUnits(CopyWaitLimit, time.Hour, &next.CopyWaitLimitHours),
		CheckCopies.Key:   inUnits(CheckCopies, day, &next.CheckCopiesDays),
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
}

func count(c setting.Count, to **int64) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := c.ParseJSON(raw)
		*to = v.Override()
		return err
	}
}

// inUnits keeps d's override as a whole count of unit, the column's unit.
// d's Step is unit, so the count is whole.
func inUnits(d setting.Duration, unit time.Duration, to **int64) func(json.RawMessage) error {
	return func(raw json.RawMessage) error {
		v, err := d.ParseJSON(raw)
		if o := v.Override(); o != nil {
			*to = new(*o / int64(unit/time.Second))
			return err
		}
		*to = nil
		return err
	}
}

// warnings is how long the Controller's Backups may go without a good one
// before Health errs: Warn after this many missed Backups times Back up
// every.
func (b *Backups) warnings(ctx context.Context) (time.Duration, error) {
	var warn, every *int64
	err := b.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, err := db.New(r).GetSettings(ctx)
		if err != nil {
			return err
		}
		warn = row.WarnAfter
		every, err = b.cfg.Tasks.EveryIn(ctx, r, TaskID)
		return err
	})
	if err != nil {
		return 0, err
	}
	return time.Duration(WarnAfter.Value(warn)) * Cadence.Value(every), nil
}
