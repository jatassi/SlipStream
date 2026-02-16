package tasks

import (
	"github.com/slipstream/slipstream/internal/history"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const HistoryCleanupTaskID = "history-cleanup"

// RegisterHistoryCleanupTask registers the history cleanup task with the scheduler.
// The task runs daily at 2 AM to delete entries older than the configured retention period.
func RegisterHistoryCleanupTask(sched *scheduler.Scheduler, historyService *history.Service) error {
	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          HistoryCleanupTaskID,
		Name:        "History Cleanup",
		Description: "Deletes history entries older than the configured retention period",
		Cron:        "0 2 * * *",
		RunOnStart:  false,
		Func:        historyService.CleanupOldEntries,
	})
}
