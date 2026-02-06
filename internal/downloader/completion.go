package downloader

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader/types"
)

// CompletedDownload represents a download that is ready for import.
type CompletedDownload struct {
	DownloadID   string
	ClientID     int64
	ClientName   string
	DownloadPath string
	MediaType    string
	Size         int64

	// Library mapping
	MovieID          *int64
	SeriesID         *int64
	SeasonNumber     *int
	EpisodeID        *int64
	IsSeasonPack     bool
	IsCompleteSeries bool
	TargetSlotID     *int64

	// Internal mapping reference
	MappingID int64
}

// CheckForCompletedDownloads finds all downloads that are complete and ready for import.
func (s *Service) CheckForCompletedDownloads(ctx context.Context) ([]CompletedDownload, error) {
	// Get all enabled clients
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	s.logger.Debug().Int("clientCount", len(clients)).Msg("CheckForCompletedDownloads: found clients")

	// Get all active download mappings
	mappings, err := s.queries.ListActiveDownloadMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list mappings: %w", err)
	}

	s.logger.Debug().Int("mappingCount", len(mappings)).Msg("CheckForCompletedDownloads: found mappings")

	// Build lookup map: clientID:downloadID -> mapping
	mappingLookup := make(map[string]*sqlc.DownloadMapping)
	for _, m := range mappings {
		key := fmt.Sprintf("%d:%s", m.ClientID, m.DownloadID)
		mappingLookup[key] = m
		s.logger.Debug().Str("key", key).Msg("CheckForCompletedDownloads: mapping key")
	}

	var completed []CompletedDownload

	for _, dbClient := range clients {
		if !IsClientTypeImplemented(dbClient.Type) {
			s.logger.Debug().Str("type", dbClient.Type).Msg("CheckForCompletedDownloads: skipping unimplemented client type")
			continue
		}

		client, err := s.GetClient(ctx, dbClient.ID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Msg("Failed to get client")
			continue
		}

		downloads, err := client.List(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Msg("Failed to list downloads")
			continue
		}

		s.logger.Debug().Int64("clientId", dbClient.ID).Int("downloadCount", len(downloads)).Msg("CheckForCompletedDownloads: got downloads from client")

		for _, d := range downloads {
			s.logger.Debug().
				Str("downloadId", d.ID).
				Str("status", string(d.Status)).
				Msg("CheckForCompletedDownloads: checking download")

			// Check if download is complete (completed, seeding, or paused/stopped with full progress).
			// Transmission maps "stopped" to StatusPaused — a torrent that finishes and hits its
			// seed ratio limit will transition downloading→seeding→stopped so fast that the
			// 2-second poll may never observe the brief seeding window.
			isComplete := d.Status == types.StatusCompleted || d.Status == types.StatusSeeding
			if !isComplete && d.Status == types.StatusPaused && d.Progress >= 100 {
				isComplete = true
			}
			if !isComplete {
				continue
			}

			// Look up mapping
			key := fmt.Sprintf("%d:%s", dbClient.ID, d.ID)
			mapping, hasMapping := mappingLookup[key]
			if !hasMapping {
				// No mapping = untracked download, skip
				s.logger.Debug().Str("key", key).Msg("CheckForCompletedDownloads: no mapping found for completed download")
				continue
			}

			s.logger.Debug().Str("key", key).Msg("CheckForCompletedDownloads: found mapping for completed download")

			// Compute the actual content path by joining DownloadDir with the torrent/download name
			// DownloadDir is the save location (e.g., D:\Downloads\Movies)
			// Name is the torrent name (e.g., "Movie.2024.1080p" for folders, or "Movie.2024.1080p.mkv" for single files)
			// The content path is where we should look for video files to import
			contentPath := filepath.Join(d.DownloadDir, d.Name)

			cd := CompletedDownload{
				DownloadID:   d.ID,
				ClientID:     dbClient.ID,
				ClientName:   dbClient.Name,
				DownloadPath: contentPath,
				MediaType:    detectMediaType(d.DownloadDir),
				Size:         d.Size,
				MappingID:    mapping.ID,
			}

			if mapping.MovieID.Valid {
				id := mapping.MovieID.Int64
				cd.MovieID = &id
			}
			if mapping.SeriesID.Valid {
				id := mapping.SeriesID.Int64
				cd.SeriesID = &id
			}
			if mapping.SeasonNumber.Valid {
				num := int(mapping.SeasonNumber.Int64)
				cd.SeasonNumber = &num
			}
			if mapping.EpisodeID.Valid {
				id := mapping.EpisodeID.Int64
				cd.EpisodeID = &id
			}
			cd.IsSeasonPack = mapping.IsSeasonPack == 1
			cd.IsCompleteSeries = mapping.IsCompleteSeries == 1
			if mapping.TargetSlotID.Valid {
				id := mapping.TargetSlotID.Int64
				cd.TargetSlotID = &id
			}

			completed = append(completed, cd)
		}
	}

	return completed, nil
}

// CheckForDisappearedDownloads detects media items with status "downloading" whose
// active download no longer exists in any download client, and marks them as failed.
func (s *Service) CheckForDisappearedDownloads(ctx context.Context) error {
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	// Collect all active download IDs across all clients
	activeDownloadIDs := make(map[string]struct{})
	for _, dbClient := range clients {
		if !IsClientTypeImplemented(dbClient.Type) {
			continue
		}

		client, err := s.GetClient(ctx, dbClient.ID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Msg("Failed to get client for disappearance check")
			continue
		}

		downloads, err := client.List(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Msg("Failed to list downloads for disappearance check")
			continue
		}

		for _, d := range downloads {
			activeDownloadIDs[d.ID] = struct{}{}
		}
	}

	failedParams := sqlc.UpdateMovieStatusWithDetailsParams{
		Status:           "failed",
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    sql.NullString{String: "Download removed from client", Valid: true},
	}

	// Check downloading movies
	movies, err := s.queries.ListDownloadingMovies(ctx)
	if err != nil {
		return fmt.Errorf("failed to list downloading movies: %w", err)
	}

	for _, m := range movies {
		if !m.ActiveDownloadID.Valid {
			continue
		}
		downloadID := m.ActiveDownloadID.String
		if strings.HasPrefix(downloadID, "mock-") {
			continue
		}
		if _, exists := activeDownloadIDs[downloadID]; !exists {
			failedParams.ID = m.ID
			if err := s.queries.UpdateMovieStatusWithDetails(ctx, failedParams); err != nil {
				s.logger.Warn().Err(err).Int64("movieId", m.ID).Msg("Failed to mark disappeared movie download as failed")
				continue
			}
			s.logger.Info().Int64("movieId", m.ID).Str("downloadId", downloadID).Msg("Download disappeared from client, marked movie as failed")
			if s.statusChangeLogger != nil {
				_ = s.statusChangeLogger.LogStatusChanged(ctx, "movie", m.ID, "downloading", "failed", "Download removed from client")
			}
			if s.broadcaster != nil {
				_ = s.broadcaster.Broadcast("movie:updated", map[string]any{"movieId": m.ID})
			}
			if s.portalStatusTracker != nil {
				_ = s.portalStatusTracker.OnDownloadFailed(ctx, "movie", m.ID)
			}
		}
	}

	// Check downloading episodes
	episodes, err := s.queries.ListDownloadingEpisodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list downloading episodes: %w", err)
	}

	epFailedParams := sqlc.UpdateEpisodeStatusWithDetailsParams{
		Status:           "failed",
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    sql.NullString{String: "Download removed from client", Valid: true},
	}

	for _, ep := range episodes {
		if !ep.ActiveDownloadID.Valid {
			continue
		}
		downloadID := ep.ActiveDownloadID.String
		if strings.HasPrefix(downloadID, "mock-") {
			continue
		}
		if _, exists := activeDownloadIDs[downloadID]; !exists {
			epFailedParams.ID = ep.ID
			if err := s.queries.UpdateEpisodeStatusWithDetails(ctx, epFailedParams); err != nil {
				s.logger.Warn().Err(err).Int64("episodeId", ep.ID).Msg("Failed to mark disappeared episode download as failed")
				continue
			}
			s.logger.Info().Int64("episodeId", ep.ID).Str("downloadId", downloadID).Msg("Download disappeared from client, marked episode as failed")
			if s.statusChangeLogger != nil {
				_ = s.statusChangeLogger.LogStatusChanged(ctx, "episode", ep.ID, "downloading", "failed", "Download removed from client")
			}
			if s.broadcaster != nil {
				_ = s.broadcaster.Broadcast("series:updated", map[string]any{"id": ep.SeriesID})
			}
			if s.portalStatusTracker != nil {
				_ = s.portalStatusTracker.OnDownloadFailed(ctx, "episode", ep.ID)
			}
		}
	}

	return nil
}

// GetDownloadPath returns the file path for a specific download.
func (s *Service) GetDownloadPath(ctx context.Context, clientID int64, downloadID string) (string, error) {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return "", err
	}

	downloads, err := client.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list downloads: %w", err)
	}

	for _, d := range downloads {
		if d.ID == downloadID {
			return d.DownloadDir, nil
		}
	}

	return "", fmt.Errorf("download %s not found in client %d", downloadID, clientID)
}

// GetDownloadStatus returns the status of a specific download.
func (s *Service) GetDownloadStatus(ctx context.Context, clientID int64, downloadID string) (types.Status, error) {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return "", err
	}

	downloads, err := client.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list downloads: %w", err)
	}

	for _, d := range downloads {
		if d.ID == downloadID {
			return d.Status, nil
		}
	}

	return "", fmt.Errorf("download %s not found in client %d", downloadID, clientID)
}

// IsDownloadComplete checks if a specific download is complete.
func (s *Service) IsDownloadComplete(ctx context.Context, clientID int64, downloadID string) (bool, error) {
	status, err := s.GetDownloadStatus(ctx, clientID, downloadID)
	if err != nil {
		return false, err
	}

	return status == types.StatusCompleted || status == types.StatusSeeding, nil
}

// IsDownloadStalled checks if a download is stalled (paused or error).
func (s *Service) IsDownloadStalled(ctx context.Context, clientID int64, downloadID string) (bool, error) {
	status, err := s.GetDownloadStatus(ctx, clientID, downloadID)
	if err != nil {
		return false, err
	}

	return status == types.StatusPaused || status == types.StatusError, nil
}

// GetQueueItemCount returns the number of active items in the queue across all clients.
// Completed/seeding downloads are excluded from the count.
func (s *Service) GetQueueItemCount(ctx context.Context) (int, error) {
	count := 0

	// Get all enabled clients
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return 0, err
	}

	for _, dbClient := range clients {
		if !IsClientTypeImplemented(dbClient.Type) {
			continue
		}

		client, err := s.GetClient(ctx, dbClient.ID)
		if err != nil {
			continue
		}

		downloads, err := client.List(ctx)
		if err != nil {
			continue
		}

		for _, d := range downloads {
			if d.Status == types.StatusCompleted || d.Status == types.StatusSeeding {
				continue
			}
			if d.Progress >= 100 {
				continue
			}
			count++
		}
	}

	return count, nil
}

// HasQueueItems returns true if there are any items in the queue.
func (s *Service) HasQueueItems(ctx context.Context) (bool, error) {
	count, err := s.GetQueueItemCount(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
