package tasks

import (
	"fmt"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/rsssync"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const RssSyncTaskID = "rss-sync"

func buildRssSyncCronExpr(intervalMin int) string {
	if intervalMin <= 0 {
		intervalMin = 15
	}
	return fmt.Sprintf("*/%d * * * *", intervalMin)
}

// RegisterRssSyncTask registers the RSS sync task with the scheduler.
func RegisterRssSyncTask(sched *scheduler.Scheduler, service *rsssync.Service, cfg *config.RssSyncConfig) error {
	if !cfg.Enabled {
		return nil
	}

	cronExpr := buildRssSyncCronExpr(cfg.IntervalMin)

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          RssSyncTaskID,
		Name:        "RSS Sync",
		Description: "Fetch recent releases from RSS feeds and grab matching items",
		Cron:        cronExpr,
		RunOnStart:  true,
		Func:        service.Run,
	})
}

// UpdateRssSyncTask updates the RSS sync task based on new settings.
func UpdateRssSyncTask(sched *scheduler.Scheduler, service *rsssync.Service, cfg *config.RssSyncConfig) error {
	if err := sched.UnregisterTask(RssSyncTaskID); err != nil {
		return fmt.Errorf("failed to unregister rss-sync task: %w", err)
	}

	if !cfg.Enabled {
		return nil
	}

	return RegisterRssSyncTask(sched, service, cfg)
}
