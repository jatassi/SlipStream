package tasks

import (
	"fmt"

	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const AutoSearchTaskID = "autosearch"

// buildAutoSearchCronExpr builds a cron expression from interval hours.
func buildAutoSearchCronExpr(intervalHours int) string {
	if intervalHours == 1 {
		return "0 * * * *" // Every hour at minute 0
	} else if intervalHours >= 24 {
		return "0 0 * * *" // Daily at midnight
	}
	return fmt.Sprintf("0 */%d * * *", intervalHours)
}

// RegisterAutoSearchTask registers the automatic search task with the scheduler.
func RegisterAutoSearchTask(sched *scheduler.Scheduler, searcher *autosearch.ScheduledSearcher, cfg *config.AutoSearchConfig) error {
	if !cfg.Enabled {
		return nil // Task disabled, don't register
	}

	cronExpr := buildAutoSearchCronExpr(cfg.IntervalHours)

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          AutoSearchTaskID,
		Name:        "Automatic Search",
		Description: "Searches for missing monitored items and downloads best available releases",
		Cron:        cronExpr,
		RunOnStart:  false,
		Func:        searcher.Run,
	})
}

// UpdateAutoSearchTask updates the automatic search task based on new settings.
// It unregisters the existing task (if any) and registers a new one if enabled.
func UpdateAutoSearchTask(sched *scheduler.Scheduler, searcher *autosearch.ScheduledSearcher, cfg *config.AutoSearchConfig) error {
	// Always unregister first (safe even if task doesn't exist)
	if err := sched.UnregisterTask(AutoSearchTaskID); err != nil {
		return fmt.Errorf("failed to unregister autosearch task: %w", err)
	}

	// If disabled, we're done
	if !cfg.Enabled {
		return nil
	}

	// Register with new settings
	return RegisterAutoSearchTask(sched, searcher, cfg)
}
