package downloader

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// QueueMediaStatus represents the status of an individual file within a download.
type QueueMediaStatus string

const (
	QueueMediaStatusPending    QueueMediaStatus = "pending"
	QueueMediaStatusDownloading QueueMediaStatus = "downloading"
	QueueMediaStatusReady      QueueMediaStatus = "ready"
	QueueMediaStatusImporting  QueueMediaStatus = "importing"
	QueueMediaStatusImported   QueueMediaStatus = "imported"
	QueueMediaStatusFailed     QueueMediaStatus = "failed"
)

// QueueMediaEntry represents a file within a download that needs to be imported.
type QueueMediaEntry struct {
	ID                int64
	DownloadMappingID int64
	EpisodeID         *int64
	MovieID           *int64
	FilePath          string
	FileStatus        QueueMediaStatus
	ErrorMessage      string
	ImportAttempts    int
}

// CreateDownloadMappingInput contains the input for creating a download mapping.
type CreateDownloadMappingInput struct {
	ClientID         int64
	DownloadID       string
	MovieID          *int64
	SeriesID         *int64
	SeasonNumber     *int
	EpisodeID        *int64
	IsSeasonPack     bool
	IsCompleteSeries bool
	// Req 10.1.2: Target slot for multi-version tracking
	TargetSlotID *int64
}

// CreateDownloadMapping creates a download mapping record when a download is initiated.
// This links the download client's download ID to the library item(s).
func (s *Service) CreateDownloadMapping(ctx context.Context, input CreateDownloadMappingInput) (*sqlc.DownloadMapping, error) {
	params := sqlc.CreateDownloadMappingParams{
		ClientID:         input.ClientID,
		DownloadID:       input.DownloadID,
		IsSeasonPack:     boolToInt64(input.IsSeasonPack),
		IsCompleteSeries: boolToInt64(input.IsCompleteSeries),
	}

	if input.MovieID != nil {
		params.MovieID = sql.NullInt64{Int64: *input.MovieID, Valid: true}
	}
	if input.SeriesID != nil {
		params.SeriesID = sql.NullInt64{Int64: *input.SeriesID, Valid: true}
	}
	if input.SeasonNumber != nil {
		params.SeasonNumber = sql.NullInt64{Int64: int64(*input.SeasonNumber), Valid: true}
	}
	if input.EpisodeID != nil {
		params.EpisodeID = sql.NullInt64{Int64: *input.EpisodeID, Valid: true}
	}
	// Req 10.1.2: Set target slot for multi-version tracking
	if input.TargetSlotID != nil {
		params.TargetSlotID = sql.NullInt64{Int64: *input.TargetSlotID, Valid: true}
	}

	mapping, err := s.queries.CreateDownloadMapping(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create download mapping: %w", err)
	}

	s.logger.Debug().
		Int64("mappingId", mapping.ID).
		Int64("clientId", input.ClientID).
		Str("downloadId", input.DownloadID).
		Msg("Created download mapping")

	return mapping, nil
}

// GetDownloadMapping retrieves a download mapping by client ID and download ID.
func (s *Service) GetDownloadMapping(ctx context.Context, clientID int64, downloadID string) (*sqlc.DownloadMapping, error) {
	mapping, err := s.queries.GetDownloadMapping(ctx, sqlc.GetDownloadMappingParams{
		ClientID:   clientID,
		DownloadID: downloadID,
	})
	if err != nil {
		return nil, err
	}
	return mapping, nil
}

// DeleteDownloadMapping removes a download mapping after import or cancellation.
func (s *Service) DeleteDownloadMapping(ctx context.Context, clientID int64, downloadID string) error {
	if err := s.queries.DeleteDownloadMapping(ctx, sqlc.DeleteDownloadMappingParams{
		ClientID:   clientID,
		DownloadID: downloadID,
	}); err != nil {
		return fmt.Errorf("failed to delete download mapping: %w", err)
	}
	return nil
}

// ClearDownloadMappingSlot clears the target slot from a download mapping.
// Req 10.2.1: If download fails or rejected, slot reverts to "empty" immediately
// Req 10.2.2: No pending/retry state; waits for next search or manual action
func (s *Service) ClearDownloadMappingSlot(ctx context.Context, clientID int64, downloadID string) error {
	if err := s.queries.ClearDownloadMappingSlot(ctx, sqlc.ClearDownloadMappingSlotParams{
		ClientID:   clientID,
		DownloadID: downloadID,
	}); err != nil {
		return fmt.Errorf("failed to clear download mapping slot: %w", err)
	}

	s.logger.Debug().
		Int64("clientId", clientID).
		Str("downloadId", downloadID).
		Msg("Cleared slot assignment from failed download mapping")

	return nil
}

// HandleFailedDownload handles a failed download by clearing its slot assignment
// and removing the mapping. Call this when a download is detected as failed or rejected.
// Req 10.2.1: Slot reverts to "empty" immediately on failure
func (s *Service) HandleFailedDownload(ctx context.Context, clientID int64, downloadID string) error {
	// Get the mapping to check if it has a slot assigned
	mapping, err := s.GetDownloadMapping(ctx, clientID, downloadID)
	if err != nil {
		// Mapping may not exist, that's OK
		return nil
	}

	// If the mapping had a slot, log the failure
	if mapping.TargetSlotID.Valid {
		s.logger.Info().
			Int64("clientId", clientID).
			Str("downloadId", downloadID).
			Int64("slotId", mapping.TargetSlotID.Int64).
			Msg("Download failed, clearing slot assignment")
	}

	// Delete the mapping (which implicitly clears the slot)
	return s.DeleteDownloadMapping(ctx, clientID, downloadID)
}

// CreateQueueMediaInput contains the input for creating a queue media entry.
type CreateQueueMediaInput struct {
	DownloadMappingID int64
	EpisodeID         *int64
	MovieID           *int64
	FilePath          string
	FileStatus        QueueMediaStatus
	// Req 16.2.3: Per-episode slot assignment for season packs
	TargetSlotID *int64
}

// CreateQueueMedia creates a queue_media entry for tracking individual file import status.
func (s *Service) CreateQueueMedia(ctx context.Context, input CreateQueueMediaInput) (*sqlc.QueueMedium, error) {
	params := sqlc.CreateQueueMediaParams{
		DownloadMappingID: input.DownloadMappingID,
		FileStatus:        string(input.FileStatus),
	}

	if input.FilePath != "" {
		params.FilePath = sql.NullString{String: input.FilePath, Valid: true}
	}
	if input.EpisodeID != nil {
		params.EpisodeID = sql.NullInt64{Int64: *input.EpisodeID, Valid: true}
	}
	if input.MovieID != nil {
		params.MovieID = sql.NullInt64{Int64: *input.MovieID, Valid: true}
	}
	// Req 16.2.3: Per-episode slot assignment for season packs
	if input.TargetSlotID != nil {
		params.TargetSlotID = sql.NullInt64{Int64: *input.TargetSlotID, Valid: true}
	}

	entry, err := s.queries.CreateQueueMedia(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue media entry: %w", err)
	}

	s.logger.Debug().
		Int64("id", entry.ID).
		Int64("mappingId", input.DownloadMappingID).
		Str("status", string(input.FileStatus)).
		Msg("Created queue media entry")

	return entry, nil
}

// GetQueueMedia retrieves a queue media entry by ID.
func (s *Service) GetQueueMedia(ctx context.Context, id int64) (*sqlc.QueueMedium, error) {
	return s.queries.GetQueueMedia(ctx, id)
}

// GetQueueMediaByDownloadMapping retrieves all queue media entries for a download mapping.
func (s *Service) GetQueueMediaByDownloadMapping(ctx context.Context, mappingID int64) ([]*sqlc.QueueMedium, error) {
	return s.queries.GetQueueMediaByDownloadMapping(ctx, mappingID)
}

// UpdateQueueMediaStatus updates the status of a queue media entry.
func (s *Service) UpdateQueueMediaStatus(ctx context.Context, id int64, status QueueMediaStatus) error {
	_, err := s.queries.UpdateQueueMediaStatus(ctx, sqlc.UpdateQueueMediaStatusParams{
		ID:         id,
		FileStatus: string(status),
	})
	if err != nil {
		return fmt.Errorf("failed to update queue media status: %w", err)
	}
	return nil
}

// UpdateQueueMediaStatusWithError updates the status and sets an error message.
func (s *Service) UpdateQueueMediaStatusWithError(ctx context.Context, id int64, status QueueMediaStatus, errorMsg string) error {
	_, err := s.queries.UpdateQueueMediaStatusWithError(ctx, sqlc.UpdateQueueMediaStatusWithErrorParams{
		ID:           id,
		FileStatus:   string(status),
		ErrorMessage: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to update queue media status with error: %w", err)
	}
	return nil
}

// UpdateQueueMediaFilePath updates the file path of a queue media entry.
func (s *Service) UpdateQueueMediaFilePath(ctx context.Context, id int64, filePath string) error {
	_, err := s.queries.UpdateQueueMediaFilePath(ctx, sqlc.UpdateQueueMediaFilePathParams{
		ID:       id,
		FilePath: sql.NullString{String: filePath, Valid: filePath != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to update queue media file path: %w", err)
	}
	return nil
}

// IncrementQueueMediaImportAttempts increments the import attempt counter.
func (s *Service) IncrementQueueMediaImportAttempts(ctx context.Context, id int64) (int, error) {
	entry, err := s.queries.IncrementQueueMediaImportAttempts(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("failed to increment import attempts: %w", err)
	}
	return int(entry.ImportAttempts), nil
}

// ResetQueueMediaForRetry resets a failed queue media entry for retry.
func (s *Service) ResetQueueMediaForRetry(ctx context.Context, id int64) error {
	_, err := s.queries.ResetQueueMediaForRetry(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to reset queue media for retry: %w", err)
	}
	return nil
}

// DeleteQueueMedia removes a queue media entry.
func (s *Service) DeleteQueueMedia(ctx context.Context, id int64) error {
	return s.queries.DeleteQueueMedia(ctx, id)
}

// DeleteQueueMediaByDownloadMapping removes all queue media entries for a download mapping.
func (s *Service) DeleteQueueMediaByDownloadMapping(ctx context.Context, mappingID int64) error {
	return s.queries.DeleteQueueMediaByDownloadMapping(ctx, mappingID)
}

// ListPendingQueueMedia returns all queue media entries that need attention.
func (s *Service) ListPendingQueueMedia(ctx context.Context) ([]*sqlc.QueueMedium, error) {
	return s.queries.ListPendingQueueMedia(ctx)
}

// ListQueueMediaByStatus returns queue media entries with a specific status.
func (s *Service) ListQueueMediaByStatus(ctx context.Context, status QueueMediaStatus) ([]*sqlc.QueueMedium, error) {
	return s.queries.ListQueueMediaByStatus(ctx, string(status))
}

// CountQueueMediaByStatus returns the count of queue media entries with a specific status.
func (s *Service) CountQueueMediaByStatus(ctx context.Context, status QueueMediaStatus) (int64, error) {
	return s.queries.CountQueueMediaByStatus(ctx, string(status))
}

// GetReadyQueueMediaWithMapping returns ready queue media entries with their download mapping info.
func (s *Service) GetReadyQueueMediaWithMapping(ctx context.Context) ([]*sqlc.GetReadyQueueMediaWithMappingRow, error) {
	return s.queries.GetReadyQueueMediaWithMapping(ctx)
}

// GetFailedQueueMediaWithMapping returns failed queue media entries with their download mapping info.
func (s *Service) GetFailedQueueMediaWithMapping(ctx context.Context) ([]*sqlc.GetFailedQueueMediaWithMappingRow, error) {
	return s.queries.GetFailedQueueMediaWithMapping(ctx)
}

// AddTorrentWithMapping adds a torrent and creates the download mapping in a single operation.
// Returns the torrent ID and mapping ID.
func (s *Service) AddTorrentWithMapping(ctx context.Context, clientID int64, url string, input CreateDownloadMappingInput) (string, int64, error) {
	// Add the torrent first
	mediaType := "movie"
	if input.SeriesID != nil {
		mediaType = "series"
	}

	torrentID, err := s.AddTorrent(ctx, clientID, url, mediaType)
	if err != nil {
		return "", 0, fmt.Errorf("failed to add torrent: %w", err)
	}

	// Create the download mapping
	input.ClientID = clientID
	input.DownloadID = torrentID

	mapping, err := s.CreateDownloadMapping(ctx, input)
	if err != nil {
		s.logger.Error().Err(err).Str("torrentId", torrentID).Msg("Failed to create download mapping after adding torrent")
		return torrentID, 0, nil
	}

	return torrentID, mapping.ID, nil
}

// AddTorrentContentWithMapping adds a torrent from content and creates the download mapping.
// Returns the torrent ID and mapping ID.
func (s *Service) AddTorrentContentWithMapping(ctx context.Context, clientID int64, content []byte, input CreateDownloadMappingInput) (string, int64, error) {
	// Add the torrent first
	mediaType := "movie"
	if input.SeriesID != nil {
		mediaType = "series"
	}

	torrentID, err := s.AddTorrentWithContent(ctx, clientID, content, mediaType)
	if err != nil {
		return "", 0, fmt.Errorf("failed to add torrent: %w", err)
	}

	// Create the download mapping
	input.ClientID = clientID
	input.DownloadID = torrentID

	mapping, err := s.CreateDownloadMapping(ctx, input)
	if err != nil {
		s.logger.Error().Err(err).Str("torrentId", torrentID).Msg("Failed to create download mapping after adding torrent")
		return torrentID, 0, nil
	}

	return torrentID, mapping.ID, nil
}

// CreateQueueMediaForMovie creates a queue_media entry for a movie download.
func (s *Service) CreateQueueMediaForMovie(ctx context.Context, mappingID int64, movieID int64) (*sqlc.QueueMedium, error) {
	return s.CreateQueueMedia(ctx, CreateQueueMediaInput{
		DownloadMappingID: mappingID,
		MovieID:           &movieID,
		FileStatus:        QueueMediaStatusPending,
	})
}

// CreateQueueMediaForEpisode creates a queue_media entry for an episode download.
func (s *Service) CreateQueueMediaForEpisode(ctx context.Context, mappingID int64, episodeID int64) (*sqlc.QueueMedium, error) {
	return s.CreateQueueMedia(ctx, CreateQueueMediaInput{
		DownloadMappingID: mappingID,
		EpisodeID:         &episodeID,
		FileStatus:        QueueMediaStatusPending,
	})
}

// CreateQueueMediaForEpisodes creates queue_media entries for multiple episodes (season pack).
func (s *Service) CreateQueueMediaForEpisodes(ctx context.Context, mappingID int64, episodeIDs []int64) ([]*sqlc.QueueMedium, error) {
	entries := make([]*sqlc.QueueMedium, 0, len(episodeIDs))
	for _, epID := range episodeIDs {
		entry, err := s.CreateQueueMediaForEpisode(ctx, mappingID, epID)
		if err != nil {
			return entries, fmt.Errorf("failed to create queue media for episode %d: %w", epID, err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
