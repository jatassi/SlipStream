package tasks

import (
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// RegisterAvailabilityTask registers the status refresh task with the scheduler.
func RegisterAvailabilityTask(sched *scheduler.Scheduler, svc *availability.Service) error {
	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          "status-refresh",
		Name:        "Refresh Media Status",
		Description: "Transitions unreleased movies and episodes to missing once their release/air date passes",
		Cron:        "0 * * * *", // Hourly
		RunOnStart:  true,        // Run immediately on startup to catch up
		Func:        svc.RefreshAll,
	})
}
