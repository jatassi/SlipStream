package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const prowlarrHealthID = "prowlarr-connection"

// ProwlarrHealthTask handles scheduled health checks for Prowlarr connection.
type ProwlarrHealthTask struct {
	prowlarrService *prowlarr.Service
	modeManager     *prowlarr.ModeManager
	health          *health.Service
	logger          *zerolog.Logger
}

// NewProwlarrHealthTask creates a new Prowlarr health check task.
func NewProwlarrHealthTask(
	prowlarrService *prowlarr.Service,
	modeManager *prowlarr.ModeManager,
	healthService *health.Service,
	logger *zerolog.Logger,
) *ProwlarrHealthTask {
	subLogger := logger.With().Str("task", "prowlarr-health").Logger()
	return &ProwlarrHealthTask{
		prowlarrService: prowlarrService,
		modeManager:     modeManager,
		health:          healthService,
		logger:          &subLogger,
	}
}

// Run executes the Prowlarr health check.
func (t *ProwlarrHealthTask) Run(ctx context.Context) error {
	// Check if Prowlarr mode is active
	isProwlarr, err := t.modeManager.IsProwlarrMode(ctx)
	if err != nil {
		t.logger.Warn().Err(err).Msg("Failed to check Prowlarr mode")
		return nil
	}

	if !isProwlarr {
		// Not in Prowlarr mode, ensure item is unregistered
		t.health.UnregisterItem(health.CategoryProwlarr, prowlarrHealthID)
		t.logger.Debug().Msg("Prowlarr mode not active, skipping health check")
		return nil
	}

	t.logger.Info().Msg("Starting Prowlarr health check")

	// Register the Prowlarr item if not already registered
	t.health.RegisterItem(health.CategoryProwlarr, prowlarrHealthID, "Prowlarr")

	// Get connection status
	status, err := t.prowlarrService.GetConnectionStatus(ctx)
	if err != nil {
		t.health.SetError(health.CategoryProwlarr, prowlarrHealthID, err.Error())
		t.logger.Warn().Err(err).Msg("Prowlarr health check failed")
		return nil
	}

	if status.Connected {
		t.health.ClearStatus(health.CategoryProwlarr, prowlarrHealthID)
		t.logger.Debug().
			Str("version", status.Version).
			Msg("Prowlarr health check passed")
	} else {
		message := "Connection failed"
		if status.Error != "" {
			message = status.Error
		}
		t.health.SetError(health.CategoryProwlarr, prowlarrHealthID, message)
		t.logger.Warn().Str("error", status.Error).Msg("Prowlarr health check failed")
	}

	// Also refresh indexer data and capabilities on successful connection
	if status.Connected {
		if _, err := t.prowlarrService.RefreshCapabilities(ctx); err != nil {
			t.logger.Warn().Err(err).Msg("Failed to refresh Prowlarr capabilities")
		}
		if _, err := t.prowlarrService.RefreshIndexers(ctx); err != nil {
			t.logger.Warn().Err(err).Msg("Failed to refresh Prowlarr indexers")
		}
	}

	t.logger.Info().Msg("Prowlarr health check completed")
	return nil
}

// RegisterProwlarrHealthTask registers the Prowlarr health check task with the scheduler.
func RegisterProwlarrHealthTask(
	sched *scheduler.Scheduler,
	prowlarrService *prowlarr.Service,
	modeManager *prowlarr.ModeManager,
	healthService *health.Service,
	logger *zerolog.Logger,
) error {
	task := NewProwlarrHealthTask(prowlarrService, modeManager, healthService, logger)

	// Per spec 9.1.1: 15-minute check interval
	interval := 15 * time.Minute
	cronExpr := fmt.Sprintf("@every %s", interval.String())

	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          "prowlarr-health",
		Name:        "Prowlarr Health Check",
		Description: "Tests connectivity to Prowlarr and refreshes indexer data",
		Cron:        cronExpr,
		RunOnStart:  true,
		Func:        task.Run,
	})
}
