package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// DownloadClientHealthTask handles scheduled health checks for download clients.
type DownloadClientHealthTask struct {
	downloader *downloader.Service
	health     *health.Service
	logger     *zerolog.Logger
}

// NewDownloadClientHealthTask creates a new download client health check task.
func NewDownloadClientHealthTask(
	downloaderSvc *downloader.Service,
	healthSvc *health.Service,
	logger *zerolog.Logger,
) *DownloadClientHealthTask {
	subLogger := logger.With().Str("task", "download-client-health").Logger()
	return &DownloadClientHealthTask{
		downloader: downloaderSvc,
		health:     healthSvc,
		logger:     &subLogger,
	}
}

// Run executes the download client health check.
func (t *DownloadClientHealthTask) Run(ctx context.Context) error {
	t.logger.Info().Msg("Starting download client health check")

	clients, err := t.downloader.List(ctx)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to list download clients")
		return err
	}

	if len(clients) == 0 {
		t.logger.Info().Msg("No download clients configured, skipping health check")
		return nil
	}

	checkedCount := 0
	skippedCount := 0
	for _, client := range clients {
		if !client.Enabled {
			continue
		}

		clientIDStr := fmt.Sprintf("%d", client.ID)

		// Check if downloads are in progress - skip test and mark OK if so
		hasActive, err := t.downloader.HasActiveDownloads(ctx, client.ID)
		if err != nil {
			t.logger.Debug().Err(err).Int64("clientId", client.ID).Str("name", client.Name).Msg("Could not check for active downloads, proceeding with test")
		} else if hasActive {
			// Client is working since downloads are active - mark OK and skip test
			t.health.ClearStatus(health.CategoryDownloadClients, clientIDStr)
			t.logger.Debug().Int64("clientId", client.ID).Str("name", client.Name).Msg("Skipping health check - downloads in progress")
			skippedCount++
			continue
		}

		// Test the connection
		result, err := t.downloader.Test(ctx, client.ID)
		if err != nil {
			t.health.SetError(health.CategoryDownloadClients, clientIDStr, err.Error())
			t.logger.Warn().Err(err).Int64("clientId", client.ID).Str("name", client.Name).Msg("Download client health check failed")
			continue
		}

		if result.Success {
			t.health.ClearStatus(health.CategoryDownloadClients, clientIDStr)
			t.logger.Debug().Int64("clientId", client.ID).Str("name", client.Name).Msg("Download client health check passed")
		} else {
			t.health.SetError(health.CategoryDownloadClients, clientIDStr, result.Message)
			t.logger.Warn().Int64("clientId", client.ID).Str("name", client.Name).Str("message", result.Message).Msg("Download client health check failed")
		}
		checkedCount++
	}

	t.logger.Info().Int("checked", checkedCount).Int("skipped", skippedCount).Int("total", len(clients)).Msg("Download client health check completed")
	return nil
}

// RegisterDownloadClientHealthTask registers the download client health check task with the scheduler.
func RegisterDownloadClientHealthTask(
	sched *scheduler.Scheduler,
	downloaderSvc *downloader.Service,
	healthSvc *health.Service,
	cfg *config.HealthConfig,
	logger *zerolog.Logger,
) error {
	task := NewDownloadClientHealthTask(downloaderSvc, healthSvc, logger)

	interval := cfg.DownloadClientCheckInterval
	if interval == 0 {
		interval = 6 * time.Hour
	}

	// Convert interval to cron expression using @every directive
	cronExpr := fmt.Sprintf("@every %s", interval.String())

	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          "download-client-health",
		Name:        "Download Client Health Check",
		Description: "Tests connectivity to all download clients",
		Cron:        cronExpr,
		RunOnStart:  true,
		Func:        task.Run,
	})
}
