package tasks

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks/internal/db"
)

var (
	MaintenanceWindow = setting.Window{Key: "tasks.window", Label: "Maintenance window", Default: setting.Span{Start: 4 * 60, End: 6 * 60}}
	JitterPercent     = setting.Count{Key: "tasks.jitterPercent", Label: "Jitter", Default: 10, Min: 0, Max: 50, Unit: setting.Noun{One: "%", Many: "%"}}
	JitterCap         = setting.Duration{Key: "tasks.jitterCap", Label: "Longest jitter", Default: 5 * time.Minute, Min: 0, Max: time.Hour, Step: time.Second}
	RunsKept          = setting.Count{Key: "tasks.runsKept", Label: "Runs kept per task", Default: 50, Min: 1, Max: 1000, Unit: setting.Noun{One: "run", Many: "runs"}}
	HeaderDelay       = setting.Duration{Key: "tasks.headerDelay", Label: "Show a scheduled run in the header after", Default: 5 * time.Second, Min: 0, Max: 5 * time.Minute, Step: time.Second}
	PartialWarnAfter  = setting.Count{Key: "tasks.partialWarnIntervals", Label: "Warn when a unit has not succeeded for", Default: 3, Min: 1, Max: 100, Unit: setting.Noun{One: "interval", Many: "intervals"}}
	OverdueRule       = setting.Choice[Overdue]{Key: "tasks.overdue", Label: "Overdue tasks", Default: CatchUp, Options: overdueOptions}
)

var overdueOptions = []setting.Option[Overdue]{{Value: CatchUp, Label: "Catch up"}, {Value: Wait, Label: "Wait"}}

// Schedule is the schedule settings' overrides, each nil at its default.
type Schedule struct {
	Window                                                            *setting.Span
	JitterPercent, JitterCap, RunsKept, HeaderDelay, PartialWarnAfter *int64
	Overdue                                                           *Overdue
}

// ScheduleChange is the schedule settings to store, each nil to leave.
type ScheduleChange struct {
	Window                                                            *setting.Stored[setting.Span]
	JitterPercent, JitterCap, RunsKept, HeaderDelay, PartialWarnAfter *setting.Stored[int64]
	Overdue                                                           *setting.Stored[Overdue]
}

func scheduleOf(row db.TasksSetting) Schedule {
	s := Schedule{
		JitterPercent: row.JitterPercent, JitterCap: row.JitterCapSeconds, RunsKept: row.RunsKept,
		HeaderDelay: row.HeaderDelaySeconds, PartialWarnAfter: row.PartialWarnIntervals, Overdue: row.Overdue,
	}
	if row.WindowStart != nil && row.WindowEnd != nil {
		s.Window = &setting.Span{Start: int(*row.WindowStart), End: int(*row.WindowEnd)}
	}
	return s
}

func (s *Scheduler) Schedule(ctx context.Context) (Schedule, error) {
	row, err := db.New(s.cfg.Library).GetSettings(ctx)
	return scheduleOf(row), err
}

// SetSchedule stores the change and plans again.
func (s *Scheduler) SetSchedule(ctx context.Context, c ScheduleChange) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		if c.Window != nil {
			row.WindowStart, row.WindowEnd = nil, nil
			if w := c.Window.Override(); w != nil {
				row.WindowStart, row.WindowEnd = new(int64(w.Start)), new(int64(w.End))
			}
		}
		for _, f := range []struct {
			change *setting.Stored[int64]
			column **int64
		}{
			{c.JitterPercent, &row.JitterPercent},
			{c.JitterCap, &row.JitterCapSeconds},
			{c.RunsKept, &row.RunsKept},
			{c.HeaderDelay, &row.HeaderDelaySeconds},
			{c.PartialWarnAfter, &row.PartialWarnIntervals},
		} {
			if f.change != nil {
				*f.column = f.change.Override()
			}
		}
		if c.Overdue != nil {
			row.Overdue = c.Overdue.Override()
		}
		return q.SetSettings(ctx, db.SetSettingsParams{
			WindowStart: row.WindowStart, WindowEnd: row.WindowEnd, JitterPercent: row.JitterPercent,
			JitterCapSeconds: row.JitterCapSeconds, RunsKept: row.RunsKept, HeaderDelaySeconds: row.HeaderDelaySeconds,
			PartialWarnIntervals: row.PartialWarnIntervals, Overdue: row.Overdue,
		})
	})
	if err != nil {
		return err
	}
	s.changed(ctx, "")
	s.Wake()
	return nil
}

// SetCadence stores the task's cadence override in seconds, which the
// next plan reads.
func (s *Scheduler) SetCadence(ctx context.Context, id TaskID, v setting.Stored[int64]) error {
	return s.setTask(ctx, id, func(ctx context.Context, q *db.Queries) error {
		return q.SetTaskEvery(ctx, db.SetTaskEveryParams{ID: id, EverySeconds: v.Override()})
	})
}

func (s *Scheduler) SetOverdue(ctx context.Context, id TaskID, v setting.Stored[Overdue]) error {
	return s.setTask(ctx, id, func(ctx context.Context, q *db.Queries) error {
		return q.SetTaskOverdue(ctx, db.SetTaskOverdueParams{ID: id, Overdue: v.Override()})
	})
}

func (s *Scheduler) setTask(ctx context.Context, id TaskID, set func(context.Context, *db.Queries) error) error {
	if _, ok := s.byID[id]; !ok {
		return ErrUnknownTask
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return set(ctx, db.New(tx))
	})
	if err != nil {
		return err
	}
	s.changed(ctx, id)
	s.Wake()
	return nil
}

// PruneRuns keeps the newest runs_kept finished runs of each task and
// never deletes an open one.
func PruneRuns(ctx context.Context, lib *storage.DB[file]) error {
	return lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		return q.PruneRuns(ctx, RunsKept.Value(row.RunsKept))
	})
}

func sinceOf(units []db.TasksUnit) []unitSince {
	since := make([]unitSince, 0, len(units))
	for _, u := range units {
		at := u.LastSuccessAt
		if at.IsZero() {
			at = u.FirstSeenAt
		}
		since = append(since, unitSince{label: u.Label, at: at.Time()})
	}
	return since
}
