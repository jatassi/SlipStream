package tasks

import (
	"github.com/slipstream/slipstream/internal/portal/quota"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const QuotaResetTaskID = "quota-reset"

// RegisterQuotaResetTask registers the quota reset task with the scheduler.
// The task runs every Monday at midnight local server time.
func RegisterQuotaResetTask(sched *scheduler.Scheduler, quotaService *quota.Service) error {
	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          QuotaResetTaskID,
		Name:        "Quota Reset",
		Description: "Resets weekly quotas for external request users",
		Cron:        "0 0 * * 1", // Every Monday at midnight
		RunOnStart:  false,
		Func:        quotaService.ResetAllQuotas,
	})
}
