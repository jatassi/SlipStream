package tasks

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/scheduler"
	"github.com/slipstream/slipstream/internal/update"
)

type UpdateCheckTask struct {
	updateService *update.Service
	logger        *zerolog.Logger
}

func NewUpdateCheckTask(updateService *update.Service, logger *zerolog.Logger) *UpdateCheckTask {
	subLogger := logger.With().Str("task", "update-check").Logger()
	return &UpdateCheckTask{
		updateService: updateService,
		logger:        &subLogger,
	}
}

func (t *UpdateCheckTask) Run(ctx context.Context) error {
	t.logger.Info().Msg("Starting scheduled update check")

	if err := t.updateService.CheckAndAutoInstall(ctx); err != nil {
		t.logger.Error().Err(err).Msg("Update check failed")
		return err
	}

	t.logger.Info().Msg("Scheduled update check completed")
	return nil
}

func RegisterUpdateCheckTask(sched *scheduler.Scheduler, updateService *update.Service, logger *zerolog.Logger) error {
	task := NewUpdateCheckTask(updateService, logger)

	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          "update-check",
		Name:        "Update Check",
		Description: "Checks for new SlipStream versions and optionally auto-installs",
		Cron:        "0 0 * * *", // Daily at midnight
		RunOnStart:  true,
		Func:        task.Run,
	})
}
