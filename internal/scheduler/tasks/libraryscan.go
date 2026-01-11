package tasks

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// LibraryScanTask handles scheduled library scanning.
type LibraryScanTask struct {
	libraryManager *librarymanager.Service
	rootFolders    *rootfolder.Service
	logger         zerolog.Logger
}

// NewLibraryScanTask creates a new library scan task.
func NewLibraryScanTask(lm *librarymanager.Service, rf *rootfolder.Service, logger zerolog.Logger) *LibraryScanTask {
	return &LibraryScanTask{
		libraryManager: lm,
		rootFolders:    rf,
		logger:         logger.With().Str("task", "library-scan").Logger(),
	}
}

// Run executes the library scan task, scanning all root folders.
func (t *LibraryScanTask) Run(ctx context.Context) error {
	t.logger.Info().Msg("Starting scheduled library scan")

	folders, err := t.rootFolders.List(ctx)
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to list root folders")
		return err
	}

	if len(folders) == 0 {
		t.logger.Info().Msg("No root folders configured, skipping scan")
		return nil
	}

	var lastErr error
	scannedCount := 0

	for _, folder := range folders {
		if t.libraryManager.IsScanActive(folder.ID) {
			t.logger.Info().Int64("rootFolderId", folder.ID).Str("path", folder.Path).Msg("Scan already active, skipping")
			continue
		}

		t.logger.Info().Int64("rootFolderId", folder.ID).Str("path", folder.Path).Msg("Scanning root folder")

		result, err := t.libraryManager.ScanRootFolder(ctx, folder.ID)
		if err != nil {
			t.logger.Error().Err(err).Int64("rootFolderId", folder.ID).Msg("Failed to scan root folder")
			lastErr = err
			continue
		}

		t.logger.Info().
			Int64("rootFolderId", folder.ID).
			Int("totalFiles", result.TotalFiles).
			Int("moviesAdded", result.MoviesAdded).
			Int("seriesAdded", result.SeriesAdded).
			Int("filesLinked", result.FilesLinked).
			Msg("Root folder scan completed")

		scannedCount++
	}

	t.logger.Info().Int("scannedFolders", scannedCount).Int("totalFolders", len(folders)).Msg("Scheduled library scan completed")

	return lastErr
}

// RegisterLibraryScanTask registers the library scan task with the scheduler.
func RegisterLibraryScanTask(sched *scheduler.Scheduler, lm *librarymanager.Service, rf *rootfolder.Service, logger zerolog.Logger) error {
	task := NewLibraryScanTask(lm, rf, logger)

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          "library-scan",
		Name:        "Library Scan",
		Description: "Scans all root folders for new media files",
		Cron:        "30 23 * * *", // 11:30 PM daily
		RunOnStart:  false,
		Func:        task.Run,
	})
}
