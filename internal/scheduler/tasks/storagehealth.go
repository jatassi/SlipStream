package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// StorageHealthTask handles scheduled storage health checks.
type StorageHealthTask struct {
	checker *health.StorageChecker
	logger  zerolog.Logger
}

// NewStorageHealthTask creates a new storage health check task.
func NewStorageHealthTask(
	checker *health.StorageChecker,
	logger zerolog.Logger,
) *StorageHealthTask {
	return &StorageHealthTask{
		checker: checker,
		logger:  logger.With().Str("task", "storage-health").Logger(),
	}
}

// Run executes the storage health check.
func (t *StorageHealthTask) Run(ctx context.Context) error {
	t.logger.Info().Msg("Starting storage health check")

	if err := t.checker.CheckAllStorage(ctx); err != nil {
		t.logger.Error().Err(err).Msg("Storage health check failed")
		return err
	}

	t.logger.Info().Msg("Storage health check completed")
	return nil
}

// RegisterStorageHealthTask registers the storage health check task with the scheduler.
func RegisterStorageHealthTask(
	sched *scheduler.Scheduler,
	checker *health.StorageChecker,
	cfg *config.HealthConfig,
	logger zerolog.Logger,
) error {
	task := NewStorageHealthTask(checker, logger)

	interval := cfg.StorageCheckInterval
	if interval == 0 {
		interval = 1 * time.Hour
	}

	// Convert interval to cron expression using @every directive
	cronExpr := fmt.Sprintf("@every %s", interval.String())

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          "storage-health",
		Name:        "Storage Health Check",
		Description: "Monitors disk space usage on volumes with root folders",
		Cron:        cronExpr,
		RunOnStart:  true, // Run immediately to detect storage issues
		Func:        task.Run,
	})
}
