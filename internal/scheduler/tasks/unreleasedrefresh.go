package tasks

import (
	"context"

	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// UnreleasedRefreshTaskID identifies the daily unreleased-metadata refresh task.
const UnreleasedRefreshTaskID = "unreleased-refresh"

// RegisterUnreleasedRefreshTask registers a daily task that refreshes metadata
// for library items in 'unreleased' status, so release dates, episode lists,
// and artwork stay current as items approach release.
func RegisterUnreleasedRefreshTask(sched *scheduler.Scheduler, lm *librarymanager.Service) error {
	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          UnreleasedRefreshTaskID,
		Name:        "Refresh Unreleased Metadata",
		Description: "Re-fetches metadata for unreleased movies and series so release dates and episode lists stay current",
		Cron:        "0 4 * * *", // 4:00 AM daily
		RunOnStart:  false,
		Func: func(ctx context.Context) error {
			return lm.RefreshUnreleasedMetadata(ctx)
		},
	})
}
