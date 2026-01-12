package tasks

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/scheduler"
)

// ImportScanner defines the interface for scanning for pending imports.
type ImportScanner interface {
	ScanForPendingImports(ctx context.Context) error
	ProcessPendingRetries(ctx context.Context) error
	CheckAndProcessCompletedDownloads(ctx context.Context) error
}

// importScanTask wraps the import scanner for scheduled execution.
type importScanTask struct {
	scanner ImportScanner
	logger  zerolog.Logger
}

// newImportScanTask creates a new import scan task.
func newImportScanTask(scanner ImportScanner, logger zerolog.Logger) *importScanTask {
	return &importScanTask{
		scanner: scanner,
		logger:  logger.With().Str("task", "import-scan").Logger(),
	}
}

// run executes the import scan.
func (t *importScanTask) run(ctx context.Context) error {
	t.logger.Debug().Msg("Running import scan")

	// Check for completed downloads and emit WebSocket events
	if err := t.scanner.CheckAndProcessCompletedDownloads(ctx); err != nil {
		t.logger.Warn().Err(err).Msg("Check completed downloads failed")
	}

	if err := t.scanner.ScanForPendingImports(ctx); err != nil {
		t.logger.Error().Err(err).Msg("Import scan failed")
		return err
	}

	if err := t.scanner.ProcessPendingRetries(ctx); err != nil {
		t.logger.Warn().Err(err).Msg("Processing pending retries failed")
	}

	t.logger.Debug().Msg("Import scan completed")
	return nil
}

// RegisterImportScanTask registers the import scan task with the scheduler.
func RegisterImportScanTask(sched *scheduler.Scheduler, scanner ImportScanner, logger zerolog.Logger) error {
	if scanner == nil {
		return nil
	}

	task := newImportScanTask(scanner, logger)

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          "import-scan",
		Name:        "Import Scan",
		Description: "Scans download folders for completed files ready to import",
		Cron:        "*/5 * * * *", // Every 5 minutes
		RunOnStart:  true,          // Scan on startup
		Func:        task.run,
	})
}
