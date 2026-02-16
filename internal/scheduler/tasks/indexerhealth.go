package tasks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// IndexerHealthTask handles scheduled health checks for indexers.
type IndexerHealthTask struct {
	indexer         *indexer.Service
	prowlarrService *prowlarr.Service
	modeManager     *prowlarr.ModeManager
	health          *health.Service
	logger          *zerolog.Logger
}

// NewIndexerHealthTask creates a new indexer health check task.
func NewIndexerHealthTask(
	indexerSvc *indexer.Service,
	prowlarrSvc *prowlarr.Service,
	modeMgr *prowlarr.ModeManager,
	healthSvc *health.Service,
	logger *zerolog.Logger,
) *IndexerHealthTask {
	subLogger := logger.With().Str("task", "indexer-health").Logger()
	return &IndexerHealthTask{
		indexer:         indexerSvc,
		prowlarrService: prowlarrSvc,
		modeManager:     modeMgr,
		health:          healthSvc,
		logger:          &subLogger,
	}
}

// Run executes the indexer health check.
func (t *IndexerHealthTask) Run(ctx context.Context) error {
	t.logger.Info().Msg("Starting indexer health check")

	// Check if we're in Prowlarr mode
	isProwlarrMode, err := t.modeManager.IsProwlarrMode(ctx)
	if err != nil {
		t.logger.Warn().Err(err).Msg("Failed to check Prowlarr mode, falling back to internal indexers")
		isProwlarrMode = false
	}

	if isProwlarrMode {
		return t.runProwlarrHealthCheck(ctx)
	}
	return t.runInternalHealthCheck(ctx)
}

// runProwlarrHealthCheck checks health of Prowlarr indexers.
func (t *IndexerHealthTask) runProwlarrHealthCheck(ctx context.Context) error {
	t.logger.Debug().Msg("Using Prowlarr indexer health")

	// Clear any internal SlipStream indexers from health (they don't have "prowlarr-" prefix)
	t.clearNonProwlarrIndexers()

	// Fetch indexers from Prowlarr
	indexers, err := t.prowlarrService.GetIndexers(ctx)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to fetch Prowlarr indexers")
		return err
	}

	if len(indexers) == 0 {
		t.logger.Info().Msg("No Prowlarr indexers found")
		return nil
	}

	// Track which indexer IDs we've seen for cleanup
	seenIDs := make(map[string]bool)

	for i := range indexers {
		idx := &indexers[i]
		idStr := fmt.Sprintf("prowlarr-%d", idx.ID)
		seenIDs[idStr] = true

		// Register the indexer
		t.health.RegisterItem(health.CategoryIndexers, idStr, idx.Name)

		// Map Prowlarr status to health status
		switch idx.Status {
		case prowlarr.IndexerStatusHealthy:
			t.health.ClearStatus(health.CategoryIndexers, idStr)
		case prowlarr.IndexerStatusWarning:
			t.health.SetWarning(health.CategoryIndexers, idStr, "Indexer has warnings")
		case prowlarr.IndexerStatusDisabled:
			t.health.SetWarning(health.CategoryIndexers, idStr, "Indexer is disabled")
		case prowlarr.IndexerStatusFailed:
			t.health.SetError(health.CategoryIndexers, idStr, "Indexer has failed")
		default:
			t.health.ClearStatus(health.CategoryIndexers, idStr)
		}
	}

	t.logger.Info().Int("count", len(indexers)).Msg("Prowlarr indexer health check completed")
	return nil
}

// clearNonProwlarrIndexers removes any internal SlipStream indexers from health tracking.
func (t *IndexerHealthTask) clearNonProwlarrIndexers() {
	items := t.health.GetByCategory(health.CategoryIndexers)
	for _, item := range items {
		if !strings.HasPrefix(item.ID, "prowlarr-") {
			t.health.UnregisterItem(health.CategoryIndexers, item.ID)
			t.logger.Debug().Str("id", item.ID).Msg("Unregistered internal indexer from health (now in Prowlarr mode)")
		}
	}
}

// runInternalHealthCheck checks health of internal SlipStream indexers.
func (t *IndexerHealthTask) runInternalHealthCheck(ctx context.Context) error {
	t.logger.Debug().Msg("Using internal indexer health")

	// Clear any Prowlarr indexers from health (they have "prowlarr-" prefix)
	t.clearProwlarrIndexers()

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

// clearProwlarrIndexers removes any Prowlarr indexers from health tracking.
func (t *IndexerHealthTask) clearProwlarrIndexers() {
	items := t.health.GetByCategory(health.CategoryIndexers)
	for _, item := range items {
		if strings.HasPrefix(item.ID, "prowlarr-") {
			t.health.UnregisterItem(health.CategoryIndexers, item.ID)
			t.logger.Debug().Str("id", item.ID).Msg("Unregistered Prowlarr indexer from health (now in SlipStream mode)")
		}
	}
}

// RegisterIndexerHealthTask registers the indexer health check task with the scheduler.
func RegisterIndexerHealthTask(
	sched *scheduler.Scheduler,
	indexerSvc *indexer.Service,
	prowlarrSvc *prowlarr.Service,
	modeMgr *prowlarr.ModeManager,
	healthSvc *health.Service,
	cfg *config.HealthConfig,
	logger *zerolog.Logger,
) error {
	task := NewIndexerHealthTask(indexerSvc, prowlarrSvc, modeMgr, healthSvc, logger)

	interval := cfg.IndexerCheckInterval
	if interval == 0 {
		interval = 6 * time.Hour
	}

	// Convert interval to cron expression using @every directive
	cronExpr := fmt.Sprintf("@every %s", interval.String())

	return sched.RegisterTask(&scheduler.TaskConfig{
		ID:          "indexer-health",
		Name:        "Indexer Health Check",
		Description: "Tests connectivity to all enabled indexers",
		Cron:        cronExpr,
		RunOnStart:  true,
		Func:        task.Run,
	})
}
