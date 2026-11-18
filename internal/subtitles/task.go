package subtitles

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Task is Subtitle search: one unit on the Controller that reconciles the
// wants, queues every due one and ends. The drain works the queue within
// each provider's allowance, so a long drain never fails a run.
func (s *Subtitles) Task() tasks.Task {
	return tasks.Task{
		ID:      TaskID,
		Name:    "Subtitle search",
		About:   "Searches for the wanted subtitles that are due and keeps each one only when its timing matches the file.",
		Group:   tasks.GroupOf(role.Subtitles.Group()),
		Cadence: tasks.Every{Interval: Every},
		Noun:    setting.Noun{One: "step", Many: "steps"},
		Units: func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "due", Label: "Due subtitles", Place: tasks.OnController()}}, nil
		},
		Run:    s.search,
		Health: string(health.SubtitlesID),
		Shown: func(ctx context.Context, r storage.Reader[storage.Library]) (bool, error) {
			wanted, err := db.New(r).ListWantedTypes(ctx)
			return len(wanted) > 0, err
		},
		Off: func(ctx context.Context, r storage.Reader[storage.Library]) (bool, error) {
			row, err := db.New(r).GetSettings(ctx)
			return !SearchOn.Value(row.SearchOn), err
		},
	}
}

func (s *Subtitles) search(ctx context.Context, _ tasks.Unit, p *tasks.Progress) error {
	if _, err := s.Reconcile(ctx); err != nil {
		return err
	}
	var on bool
	var n int
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		halt, err := s.cfg.Reconciling(ctx, tx)
		if err != nil || halt {
			return err
		}
		q := db.New(tx)
		row, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		if on = SearchOn.Value(row.SearchOn); !on {
			return nil
		}
		n, err = s.queue(ctx, q, true, SearchUpgrades.Value(row.SearchUpgrades))
		return err
	})
	if err != nil {
		return err
	}
	if !on {
		return tasks.SkipOwned("Subtitle search is off in Settings.")
	}
	s.drain.soon()
	p.Note(queuedWords(n))
	return nil
}

func queuedWords(n int) string {
	if n == 1 {
		return "Queued 1 subtitle."
	}
	return "Queued " + setting.Grouped(n) + " subtitles."
}

// Retention is the Housekeeping step "Subtitle tries": Tried before rows
// older than Keep history for, oldest first. A Mark bad stays, so no
// search takes that subtitle again.
func (s *Subtitles) Retention() housekeeping.Step {
	return housekeeping.Aged("subtitle-tries", "Subtitle tries", setting.Noun{One: "try", Many: "tries"}, "that are marked bad",
		func(ctx context.Context, before time.Time, _ struct{}, limit int) (housekeeping.Swept[struct{}], error) {
			var n int64
			err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				var err error
				n, err = db.New(tx).PruneTried(ctx, db.PruneTriedParams{Before: storage.FromTime(before), N: int64(limit)})
				return err
			})
			return housekeeping.Swept[struct{}]{Seen: int(n), Deleted: int(n)}, err
		})
}
