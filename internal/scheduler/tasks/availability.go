package tasks

import (
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// RegisterAvailabilityTask registers the availability refresh task with the scheduler.
func RegisterAvailabilityTask(sched *scheduler.Scheduler, svc *availability.Service) error {
	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          "availability-refresh",
		Name:        "Refresh Media Availability",
		Description: "Updates released status for movies, episodes, seasons, and series based on release/air dates",
		Cron:        "0 0 * * *", // Midnight daily
		RunOnStart:  true,        // Run immediately on startup to catch up
		Func:        svc.RefreshAll,
	})
}
