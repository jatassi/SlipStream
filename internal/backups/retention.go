package backups

import (
	"context"
	"embed"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/backups/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("backups", schemaFiles) }

const day = 24 * time.Hour

var KeepScheduled = setting.Duration{
	Key: "backups.keepScheduled", Label: "Keep scheduled Backups for",
	Default: 28 * day, Min: day, Max: 3650 * day, Step: day,
}

var KeepEach = setting.Count{
	Key: "backups.keepEach", Label: "Keep the newest Backups taken before updates and restores",
	Default: 5, Min: 1, Max: 100, Unit: setting.Noun{One: "Backup", Many: "Backups"},
}

// Retention is what Prune keeps.
type Retention struct {
	KeepScheduled time.Duration
	KeepEach      int
}

// ReadRetention is the stored retention, each NULL its default.
func ReadRetention(ctx context.Context, r storage.Reader[file]) (Retention, error) {
	row, err := db.New(r).GetSettings(ctx)
	if err != nil {
		return Retention{}, err
	}
	var seconds *int64
	if row.KeepScheduledDays != nil {
		seconds = new(*row.KeepScheduledDays * int64(day/time.Second))
	}
	return Retention{KeepScheduled: KeepScheduled.Value(seconds), KeepEach: int(KeepEach.Value(row.KeepEach))}, nil
}

// Prune is retention, pure, over the verified Backups: scheduled Backups
// taken more than KeepScheduled before the newest verified scheduled one,
// and each other automatic kind beyond its newest KeepEach. Manual Backups
// and the newest verified Backup are never in the answer, and an entry in
// any other state counts for nothing. The newest verified Backup is the
// newest of its kind, so neither rule reaches it.
func Prune(l Ledger, r Retention) []archive.ID {
	var verified []Entry
	for _, e := range l.Backups {
		if e.State == Verified {
			verified = append(verified, e)
		}
	}
	slices.SortStableFunc(verified, func(a, b Entry) int { return b.TakenAt.Compare(a.TakenAt) })
	var horizon time.Time
	if i := slices.IndexFunc(verified, func(e Entry) bool { return e.Trigger == archive.Scheduled }); i >= 0 {
		horizon = verified[i].TakenAt.Add(-r.KeepScheduled)
	}
	kept := map[archive.Trigger]int{}
	var out []archive.ID
	for _, e := range verified {
		if e.Trigger == archive.Manual {
			continue
		}
		if e.Trigger == archive.Scheduled {
			if e.TakenAt.Before(horizon) {
				out = append(out, e.ID)
			}
			continue
		}
		kept[e.Trigger]++
		if kept[e.Trigger] > r.KeepEach {
			out = append(out, e.ID)
		}
	}
	return out
}
