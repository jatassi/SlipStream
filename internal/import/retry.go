package importer

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const (
	// MaxRetries is the maximum number of import retry attempts.
	MaxRetries = 3

	// RetryDelayBase is the base delay between retries.
	RetryDelayBase = 5 * time.Second

	// RetryDelayMax is the maximum delay between retries.
	RetryDelayMax = 5 * time.Minute
)

// RetryPolicy defines retry behavior for different error types.
type RetryPolicy int

const (
	// RetryNever means don't retry this error.
	RetryNever RetryPolicy = iota
	// RetryImmediate means retry immediately.
	RetryImmediate
	// RetryWithBackoff means retry with exponential backoff.
	RetryWithBackoff
	// RetryAfterDelay means retry after a fixed delay.
	RetryAfterDelay
)

// classifyError determines the retry policy for an error.
func classifyError(err error) RetryPolicy {
	if err == nil {
		return RetryNever
	}

	// Permanent errors that shouldn't be retried
	permanentErrors := []error{
		ErrFileNotFound,
		ErrInvalidExtension,
		ErrSampleFile,
		ErrNoMatch,
		ErrMatchConflict,
		ErrPathTooLong,
		ErrFileAlreadyInLibrary,
		ErrNotAnUpgrade,
	}

	for _, permErr := range permanentErrors {
		if errors.Is(err, permErr) {
			return RetryNever
		}
	}

	// Transient errors that should be retried with backoff
	transientErrors := []error{
		ErrFileTooSmall, // File might still be copying
		ErrAlreadyImporting,
	}

	for _, transErr := range transientErrors {
		if errors.Is(err, transErr) {
			return RetryWithBackoff
		}
	}

	// Default to retry with backoff for unknown errors
	return RetryWithBackoff
}

// calculateRetryDelay calculates the delay before the next retry attempt.
func calculateRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return RetryDelayBase
	}

	// Exponential backoff: base * 2^attempt
	delay := RetryDelayBase * time.Duration(1<<uint(attempt))

	if delay > RetryDelayMax {
		delay = RetryDelayMax
	}

	return delay
}

// processWithRetry processes an import job with retry logic.
func (s *Service) processWithRetry(ctx context.Context, job ImportJob) *ImportResult {
	var lastResult *ImportResult
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			policy := classifyError(lastErr)

			switch policy {
			case RetryNever:
				s.logger.Debug().
					Err(lastErr).
					Str("path", job.SourcePath).
					Msg("Error is not retryable")
				return lastResult

			case RetryWithBackoff:
				delay := calculateRetryDelay(attempt)
				s.logger.Info().
					Err(lastErr).
					Str("path", job.SourcePath).
					Int("attempt", attempt).
					Dur("delay", delay).
					Msg("Retrying import after delay")

				select {
				case <-ctx.Done():
					return &ImportResult{
						SourcePath: job.SourcePath,
						Error:      ctx.Err(),
					}
				case <-time.After(delay):
				}

			case RetryImmediate:
				s.logger.Info().
					Err(lastErr).
					Str("path", job.SourcePath).
					Int("attempt", attempt).
					Msg("Retrying import immediately")
			}
		}

		result, err := s.processImport(ctx, job)
		lastResult = result
		lastErr = err

		if err == nil {
			return result
		}

		// Log retry attempt
		s.logger.Warn().
			Err(err).
			Str("path", job.SourcePath).
			Int("attempt", attempt).
			Int("maxRetries", MaxRetries).
			Msg("Import attempt failed")
	}

	// All retries exhausted
	if lastResult == nil {
		lastResult = &ImportResult{
			SourcePath: job.SourcePath,
		}
	}

	if lastResult.Error == nil {
		lastResult.Error = ErrImportFailed
	}

	s.logger.Error().
		Err(lastResult.Error).
		Str("path", job.SourcePath).
		Int("attempts", MaxRetries+1).
		Msg("Import failed after all retry attempts")

	return lastResult
}

// RetryInfo contains information about retry attempts.
type RetryInfo struct {
	Attempt     int           `json:"attempt"`
	MaxAttempts int           `json:"maxAttempts"`
	LastError   string        `json:"lastError,omitempty"`
	NextRetry   *time.Time    `json:"nextRetry,omitempty"`
	Policy      string        `json:"policy"`
}

// GetRetryInfo returns retry information for an error.
func GetRetryInfo(err error, currentAttempt int) *RetryInfo {
	policy := classifyError(err)

	info := &RetryInfo{
		Attempt:     currentAttempt,
		MaxAttempts: MaxRetries + 1,
	}

	if err != nil {
		info.LastError = err.Error()
	}

	switch policy {
	case RetryNever:
		info.Policy = "never"
	case RetryImmediate:
		info.Policy = "immediate"
	case RetryWithBackoff:
		info.Policy = "backoff"
		if currentAttempt < MaxRetries {
			delay := calculateRetryDelay(currentAttempt + 1)
			nextRetry := time.Now().Add(delay)
			info.NextRetry = &nextRetry
		}
	case RetryAfterDelay:
		info.Policy = "delay"
	}

	return info
}

// ShouldRetry returns whether an error should be retried.
func ShouldRetry(err error) bool {
	return classifyError(err) != RetryNever
}

// IsRetryableError returns true if the error is retryable.
func IsRetryableError(err error) bool {
	return classifyError(err) != RetryNever
}

// IsPermanentError returns true if the error should not be retried.
func IsPermanentError(err error) bool {
	return classifyError(err) == RetryNever
}

// MarkForRetry updates the queue media status to indicate retry is needed.
func (s *Service) MarkForRetry(ctx context.Context, queueMedia *QueueMedia, err error) error {
	if queueMedia == nil {
		return nil
	}

	queueMedia.ImportAttempts++
	queueMedia.ErrorMessage = err.Error()

	if queueMedia.ImportAttempts >= MaxRetries {
		queueMedia.FileStatus = "failed"
	} else if IsRetryableError(err) {
		queueMedia.FileStatus = "ready" // Will be retried
	} else {
		queueMedia.FileStatus = "failed"
	}

	// Increment attempts
	_, updateErr := s.queries.IncrementQueueMediaImportAttempts(ctx, queueMedia.ID)
	if updateErr != nil {
		return updateErr
	}

	// Update status and error
	_, updateErr = s.queries.UpdateQueueMediaStatusWithError(ctx, sqlc.UpdateQueueMediaStatusWithErrorParams{
		FileStatus:   queueMedia.FileStatus,
		ErrorMessage: sql.NullString{String: queueMedia.ErrorMessage, Valid: queueMedia.ErrorMessage != ""},
		ID:           queueMedia.ID,
	})
	return updateErr
}

// GetPendingRetries returns queue media items that are pending retry.
func (s *Service) GetPendingRetries(ctx context.Context) ([]*QueueMedia, error) {
	rows, err := s.queries.ListPendingImports(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*QueueMedia, 0, len(rows))
	for _, row := range rows {
		item := &QueueMedia{
			ID:                row.ID,
			DownloadMappingID: row.DownloadMappingID,
			FileStatus:        row.FileStatus,
			ImportAttempts:    int(row.ImportAttempts),
		}
		if row.FilePath.Valid {
			item.FilePath = row.FilePath.String
		}
		if row.ErrorMessage.Valid {
			item.ErrorMessage = row.ErrorMessage.String
		}
		if row.EpisodeID.Valid {
			id := row.EpisodeID.Int64
			item.EpisodeID = &id
		}
		if row.MovieID.Valid {
			id := row.MovieID.Int64
			item.MovieID = &id
		}
		items = append(items, item)
	}

	return items, nil
}

// ProcessPendingRetries processes all items that are pending retry.
func (s *Service) ProcessPendingRetries(ctx context.Context) error {
	items, err := s.GetPendingRetries(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		// Skip if max retries reached
		if item.ImportAttempts >= MaxRetries {
			continue
		}

		job := ImportJob{
			SourcePath: item.FilePath,
			QueueMedia: item,
		}

		if err := s.QueueImport(job); err != nil {
			s.logger.Warn().Err(err).Str("path", item.FilePath).Msg("Failed to queue retry")
		}
	}

	return nil
}
