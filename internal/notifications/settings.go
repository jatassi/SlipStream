package notifications

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

const day = 24 * time.Hour

// The Notifications section: one stored value each. Every other bound is
// in Timings or a constant.
var (
	HealthWait = setting.Duration{
		Key: "notifications.healthWait", Label: "Wait before sending a new Health item",
		Default: 5 * time.Minute, Min: 0, Max: time.Hour, Step: time.Minute,
	}
	GiveUp = setting.Duration{
		Key: "notifications.giveUpAfter", Label: "Give up on a message after",
		Default: day, Min: time.Hour, Max: 7 * day, Step: time.Hour,
	}
	WarnAfter = setting.Duration{
		Key: "notifications.warnAfter", Label: "Warn when a connection has failed for",
		Default: 15 * time.Minute, Min: time.Minute, Max: day, Step: time.Minute,
	}
)

// Settings is the Notifications section's panel. Apply stores the patch
// all or none in one Write, then wakes the dispatcher and tells screens,
// so Health asks again with the new warn time.
func (s *Service) Settings() setting.Panel { return panel{s} }

type panel struct{ s *Service }

type stored struct {
	setting.Duration
	get func(db.NotificationsSetting) *int64
	set func(*db.Queries, context.Context, *int64) error
}

var rowSettings = []stored{
	{HealthWait, func(r db.NotificationsSetting) *int64 { return r.HealthWaitSeconds }, (*db.Queries).SetHealthWait},
	{GiveUp, func(r db.NotificationsSetting) *int64 { return r.GiveUpSeconds }, (*db.Queries).SetGiveUp},
	{WarnAfter, func(r db.NotificationsSetting) *int64 { return r.WarnSeconds }, (*db.Queries).SetWarn},
}

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var row db.NotificationsSetting
	err := p.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		row, err = db.New(r).GetSettings(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := make([]setting.Field, len(rowSettings))
	for i, st := range rowSettings {
		out[i] = st.Field(st.get(row))
	}
	return out, nil
}

func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	var sets []func(context.Context, *db.Queries) error
	for key, raw := range patch {
		i := slices.IndexFunc(rowSettings, func(st stored) bool { return st.Key == key })
		if i < 0 {
			return &setting.UnknownKeyError{Key: key}
		}
		v, err := rowSettings[i].ParseJSON(raw)
		if err != nil {
			return err
		}
		set := rowSettings[i].set
		sets = append(sets, func(ctx context.Context, q *db.Queries) error { return set(q, ctx, v.Override()) })
	}
	if len(sets) == 0 {
		return nil
	}
	err := p.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
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
	p.s.ring()
	p.s.cfg.Changed.Publish(ctx, Changed{Connections: true})
	return nil
}
