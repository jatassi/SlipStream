package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// IndexerHealthTask handles scheduled health checks for indexers.
type IndexerHealthTask struct {
	indexer *indexer.Service
	health  *health.Service
	logger  zerolog.Logger
}

// NewIndexerHealthTask creates a new indexer health check task.
func NewIndexerHealthTask(
	indexerSvc *indexer.Service,
	healthSvc *health.Service,
	logger zerolog.Logger,
) *IndexerHealthTask {
	return &IndexerHealthTask{
		indexer: indexerSvc,
		health:  healthSvc,
		logger:  logger.With().Str("task", "indexer-health").Logger(),
	}
}

// Run executes the indexer health check.
func (t *IndexerHealthTask) Run(ctx context.Context) error {
	t.logger.Info().Msg("Starting indexer health check")

	indexers, err := t.indexer.ListEnabled(ctx)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to list indexers")
		return err
	}

	if len(indexers) == 0 {
		t.logger.Info().Msg("No enabled indexers configured, skipping health check")
		return nil
	}

	checkedCount := 0
	for _, idx := range indexers {
		idStr := fmt.Sprintf("%d", idx.ID)

		// Test the connection
		result, err := t.indexer.Test(ctx, idx.ID)
		if err != nil {
			t.health.SetError(health.CategoryIndexers, idStr, err.Error())
			t.logger.Warn().Err(err).Int64("indexerId", idx.ID).Str("name", idx.Name).Msg("Indexer health check failed")
			continue
		}

		if result.Success {
			t.health.ClearStatus(health.CategoryIndexers, idStr)
			t.logger.Debug().Int64("indexerId", idx.ID).Str("name", idx.Name).Msg("Indexer health check passed")
		} else {
			t.health.SetError(health.CategoryIndexers, idStr, result.Message)
			t.logger.Warn().Int64("indexerId", idx.ID).Str("name", idx.Name).Str("message", result.Message).Msg("Indexer health check failed")
		}
		checkedCount++
	}

	t.logger.Info().Int("checked", checkedCount).Int("total", len(indexers)).Msg("Indexer health check completed")
	return nil
}

// RegisterIndexerHealthTask registers the indexer health check task with the scheduler.
func RegisterIndexerHealthTask(
	sched *scheduler.Scheduler,
	indexerSvc *indexer.Service,
	healthSvc *health.Service,
	cfg *config.HealthConfig,
	logger zerolog.Logger,
) error {
	task := NewIndexerHealthTask(indexerSvc, healthSvc, logger)

	interval := cfg.IndexerCheckInterval
	if interval == 0 {
		interval = 6 * time.Hour
	}

	// Convert interval to cron expression using @every directive
	cronExpr := fmt.Sprintf("@every %s", interval.String())

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          "indexer-health",
		Name:        "Indexer Health Check",
		Description: "Tests connectivity to all enabled indexers",
		Cron:        cronExpr,
		RunOnStart:  true,
		Func:        task.Run,
	})
}
