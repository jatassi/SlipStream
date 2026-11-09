package series

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Timings answers each episode's air date plus the air delay as the day
// it counts as out, unknown while it has no air date, and its runtime,
// else its series' average.
func (*Series) Timings(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.ItemID]targets.Timing, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.ListEveryTimingRow, error) { return q.ListEveryTiming(ctx) }, func(ids []works.WorkID) ([]db.ListEveryTimingRow, error) {
		rows, err := q.ListTimingsOf(ctx, ids)
		return as(rows, func(r db.ListTimingsOfRow) db.ListEveryTimingRow { return db.ListEveryTimingRow(r) }), err
	})
	if err != nil {
		return nil, err
	}
	settings, err := q.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	delay := AirDelay.Value(settings.AirDelaySeconds)
	basis := "Air date"
	if delay > 0 {
		basis += " plus " + setting.Words(delay)
	}
	out := make(map[works.ItemID]targets.Timing, len(rows))
	for _, row := range rows {
		var t targets.Timing
		if row.AirDate != nil {
			aired, err := time.Parse(time.DateOnly, *row.AirDate)
			if err != nil {
				return nil, fmt.Errorf("episode of item %s: air date %q is not a day", row.ItemID, *row.AirDate)
			}
			on := aired.AddDate(0, 0, int(delay/day))
			t.Released, t.Basis = release.Known(works.Date{Year: on.Year(), Month: int(on.Month()), Day: on.Day()}), basis
		}
		if row.RuntimeSeconds != nil {
			t.Runtime = release.Known(time.Duration(*row.RuntimeSeconds) * time.Second)
		}
		out[row.ItemID] = t
	}
	return out, nil
}

// Originals answers no series yet: every original language reads as
// unknown.
func (*Series) Originals(context.Context, storage.Reader[file], works.Among[works.WorkID]) (map[works.WorkID]release.Fact[release.Language], error) {
	return nil, nil
}

// Cuts answers none: every cut a series names is built in.
func (*Series) Cuts() []release.Cut { return nil }
