package tasks

import (
	"fmt"

	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// RegisterAutoSearchTask registers the automatic search task with the scheduler.
func RegisterAutoSearchTask(sched *scheduler.Scheduler, searcher *autosearch.ScheduledSearcher, cfg *config.AutoSearchConfig) error {
	if !cfg.Enabled {
		return nil // Task disabled, don't register
	}

	// Build cron expression from interval hours
	// Run at minute 0 of every Nth hour
	cronExpr := fmt.Sprintf("0 */%d * * *", cfg.IntervalHours)
	if cfg.IntervalHours == 1 {
		cronExpr = "0 * * * *" // Every hour at minute 0
	} else if cfg.IntervalHours >= 24 {
		cronExpr = "0 0 * * *" // Daily at midnight
	}

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          "autosearch",
		Name:        "Automatic Search",
		Description: "Searches for missing monitored items and downloads best available releases",
		Cron:        cronExpr,
		RunOnStart:  false, // Don't search immediately on startup
		Func:        searcher.Run,
	})
}
