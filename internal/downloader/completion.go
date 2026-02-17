package downloader

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

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
// Clients are polled concurrently, each with its own 5-second timeout.
func (s *Service) CheckForCompletedDownloads(ctx context.Context) ([]CompletedDownload, error) {
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	s.logger.Debug().Int("clientCount", len(clients)).Msg("CheckForCompletedDownloads: found clients")

	mappings, err := s.queries.ListActiveDownloadMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list mappings: %w", err)
	}

	s.logger.Debug().Int("mappingCount", len(mappings)).Msg("CheckForCompletedDownloads: found mappings")

	mappingLookup := buildMappingLookup(mappings, s.logger)

	var mu sync.Mutex
	var completed []CompletedDownload
	var wg sync.WaitGroup

	for _, dbClient := range clients {
		if !IsClientTypeImplemented(dbClient.Type) {
			continue
		}
		wg.Add(1)
		go func(dc *sqlc.DownloadClient) {
			defer wg.Done()
			clientCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			completedFromClient := s.checkClientForCompletions(clientCtx, dc, mappingLookup)
			if len(completedFromClient) > 0 {
				mu.Lock()
				completed = append(completed, completedFromClient...)
				mu.Unlock()
			}
		}(dbClient)
	}

	wg.Wait()
	return completed, nil
}

func buildMappingLookup(mappings []*sqlc.DownloadMapping, logger *zerolog.Logger) map[string]*sqlc.DownloadMapping {
	lookup := make(map[string]*sqlc.DownloadMapping)
	for _, m := range mappings {
		key := fmt.Sprintf("%d:%s", m.ClientID, m.DownloadID)
		lookup[key] = m
		logger.Debug().Str("key", key).Msg("CheckForCompletedDownloads: mapping key")
	}
	return lookup
}

func (s *Service) checkClientForCompletions(ctx context.Context, dbClient *sqlc.DownloadClient, mappingLookup map[string]*sqlc.DownloadMapping) []CompletedDownload {
	if !IsClientTypeImplemented(dbClient.Type) {
		s.logger.Debug().Str("type", dbClient.Type).Msg("CheckForCompletedDownloads: skipping unimplemented client type")
		return nil
	}

	client, err := s.GetClient(ctx, dbClient.ID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Msg("Failed to get client")
		return nil
	}

	downloads, err := client.List(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Msg("Failed to list downloads")
		return nil
	}

	s.logger.Debug().Int64("clientId", dbClient.ID).Int("downloadCount", len(downloads)).Msg("CheckForCompletedDownloads: got downloads from client")

	var completed []CompletedDownload
	for i := range downloads {
		d := &downloads[i]
		if cd := s.processDownload(d, dbClient, mappingLookup); cd != nil {
			completed = append(completed, *cd)
		}
	}

	return completed
}

func (s *Service) processDownload(d *types.DownloadItem, dbClient *sqlc.DownloadClient, mappingLookup map[string]*sqlc.DownloadMapping) *CompletedDownload {
	s.logger.Debug().
		Str("downloadId", d.ID).
		Str("status", string(d.Status)).
		Msg("CheckForCompletedDownloads: checking download")

	if !isDownloadComplete(d) {
		return nil
	}

	key := fmt.Sprintf("%d:%s", dbClient.ID, d.ID)
	mapping, hasMapping := mappingLookup[key]
	if !hasMapping {
		s.logger.Debug().Str("key", key).Msg("CheckForCompletedDownloads: no mapping found for completed download")
		return nil
	}

	s.logger.Debug().Str("key", key).Msg("CheckForCompletedDownloads: found mapping for completed download")

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

	populateCompletedDownloadFields(&cd, mapping)
	return &cd
}

func isDownloadComplete(d *types.DownloadItem) bool {
	if d.Status == types.StatusCompleted || d.Status == types.StatusSeeding {
		return true
	}
	if d.Status == types.StatusPaused && d.Progress >= 100 {
		return true
	}
	return false
}

func populateCompletedDownloadFields(cd *CompletedDownload, mapping *sqlc.DownloadMapping) {
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
}

// CheckForDisappearedDownloads detects media items with status "downloading" whose
// active download no longer exists in any download client, and marks them as failed.
func (s *Service) CheckForDisappearedDownloads(ctx context.Context) error {
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clients: %w", err)
	}

	activeDownloadIDs := s.collectActiveDownloadIDs(ctx, clients)

	if err := s.checkDisappearedMovies(ctx, activeDownloadIDs); err != nil {
		return err
	}

	if err := s.checkDisappearedEpisodes(ctx, activeDownloadIDs); err != nil {
		return err
	}

	return nil
}

func (s *Service) collectActiveDownloadIDs(ctx context.Context, clients []*sqlc.DownloadClient) map[string]struct{} {
	var mu sync.Mutex
	activeDownloadIDs := make(map[string]struct{})
	var wg sync.WaitGroup

	for _, dbClient := range clients {
		if !IsClientTypeImplemented(dbClient.Type) {
			continue
		}
		wg.Add(1)
		go func(dc *sqlc.DownloadClient) {
			defer wg.Done()
			clientCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			client, err := s.GetClient(clientCtx, dc.ID)
			if err != nil {
				s.logger.Warn().Err(err).Int64("clientId", dc.ID).Msg("Failed to get client for disappearance check")
				return
			}

			downloads, err := client.List(clientCtx)
			if err != nil {
				s.logger.Warn().Err(err).Int64("clientId", dc.ID).Msg("Failed to list downloads for disappearance check")
				return
			}

			mu.Lock()
			for i := range downloads {
				activeDownloadIDs[downloads[i].ID] = struct{}{}
			}
			mu.Unlock()
		}(dbClient)
	}

	wg.Wait()
	return activeDownloadIDs
}

func (s *Service) checkDisappearedMovies(ctx context.Context, activeDownloadIDs map[string]struct{}) error {
	movies, err := s.queries.ListDownloadingMovies(ctx)
	if err != nil {
		return fmt.Errorf("failed to list downloading movies: %w", err)
	}

	failedParams := sqlc.UpdateMovieStatusWithDetailsParams{
		Status:           "failed",
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    sql.NullString{String: "Download removed from client", Valid: true},
	}

	for _, m := range movies {
		if shouldMarkMovieAsDisappeared(m, activeDownloadIDs) {
			s.markMovieAsDisappeared(ctx, m, failedParams)
		}
	}

	return nil
}

func shouldMarkMovieAsDisappeared(m *sqlc.ListDownloadingMoviesRow, activeDownloadIDs map[string]struct{}) bool {
	if !m.ActiveDownloadID.Valid {
		return false
	}
	downloadID := m.ActiveDownloadID.String
	if strings.HasPrefix(downloadID, "mock-") {
		return false
	}
	_, exists := activeDownloadIDs[downloadID]
	return !exists
}

func (s *Service) markMovieAsDisappeared(ctx context.Context, m *sqlc.ListDownloadingMoviesRow, failedParams sqlc.UpdateMovieStatusWithDetailsParams) {
	failedParams.ID = m.ID
	if err := s.queries.UpdateMovieStatusWithDetails(ctx, failedParams); err != nil {
		s.logger.Warn().Err(err).Int64("movieId", m.ID).Msg("Failed to mark disappeared movie download as failed")
		return
	}

	downloadID := m.ActiveDownloadID.String
	s.logger.Info().Int64("movieId", m.ID).Str("downloadId", downloadID).Msg("Download disappeared from client, marked movie as failed")

	if s.statusChangeLogger != nil {
		_ = s.statusChangeLogger.LogStatusChanged(ctx, "movie", m.ID, "downloading", "failed", "Download removed from client")
	}
	if s.broadcaster != nil {
		s.broadcaster.Broadcast("movie:updated", map[string]any{"movieId": m.ID})
	}
	if s.portalStatusTracker != nil {
		_ = s.portalStatusTracker.OnDownloadFailed(ctx, "movie", m.ID)
	}
}

func (s *Service) checkDisappearedEpisodes(ctx context.Context, activeDownloadIDs map[string]struct{}) error {
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
		if shouldMarkEpisodeAsDisappeared(ep, activeDownloadIDs) {
			s.markEpisodeAsDisappeared(ctx, ep, epFailedParams)
		}
	}

	return nil
}

func shouldMarkEpisodeAsDisappeared(ep *sqlc.ListDownloadingEpisodesRow, activeDownloadIDs map[string]struct{}) bool {
	if !ep.ActiveDownloadID.Valid {
		return false
	}
	downloadID := ep.ActiveDownloadID.String
	if strings.HasPrefix(downloadID, "mock-") {
		return false
	}
	_, exists := activeDownloadIDs[downloadID]
	return !exists
}

func (s *Service) markEpisodeAsDisappeared(ctx context.Context, ep *sqlc.ListDownloadingEpisodesRow, epFailedParams sqlc.UpdateEpisodeStatusWithDetailsParams) {
	epFailedParams.ID = ep.ID
	if err := s.queries.UpdateEpisodeStatusWithDetails(ctx, epFailedParams); err != nil {
		s.logger.Warn().Err(err).Int64("episodeId", ep.ID).Msg("Failed to mark disappeared episode download as failed")
		return
	}

	downloadID := ep.ActiveDownloadID.String
	s.logger.Info().Int64("episodeId", ep.ID).Str("downloadId", downloadID).Msg("Download disappeared from client, marked episode as failed")

	if s.statusChangeLogger != nil {
		_ = s.statusChangeLogger.LogStatusChanged(ctx, "episode", ep.ID, "downloading", "failed", "Download removed from client")
	}
	if s.broadcaster != nil {
		s.broadcaster.Broadcast("series:updated", map[string]any{"id": ep.SeriesID})
	}
	if s.portalStatusTracker != nil {
		_ = s.portalStatusTracker.OnDownloadFailed(ctx, "episode", ep.ID)
	}
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

	for i := range downloads {
		d := &downloads[i]
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

	for i := range downloads {
		d := &downloads[i]
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

	return status == types.StatusPaused || status == types.StatusError || status == types.StatusWarning, nil
}

// GetQueueItemCount returns the number of active items in the queue across all clients.
func (s *Service) GetQueueItemCount(ctx context.Context) (int, error) {
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, dbClient := range clients {
		clientCount := s.countQueueItemsForClient(ctx, dbClient)
		count += clientCount
	}

	return count, nil
}

func (s *Service) countQueueItemsForClient(ctx context.Context, dbClient *sqlc.DownloadClient) int {
	if !IsClientTypeImplemented(dbClient.Type) {
		return 0
	}

	client, err := s.GetClient(ctx, dbClient.ID)
	if err != nil {
		return 0
	}

	downloads, err := client.List(ctx)
	if err != nil {
		return 0
	}

	count := 0
	for i := range downloads {
		d := &downloads[i]
		if isActiveQueueItem(d) {
			count++
		}
	}

	return count
}

func isActiveQueueItem(d *types.DownloadItem) bool {
	if d.Status == types.StatusCompleted || d.Status == types.StatusSeeding {
		return false
	}
	if d.Progress >= 100 {
		return false
	}
	return true
}

// HasQueueItems returns true if there are any items in the queue.
func (s *Service) HasQueueItems(ctx context.Context) (bool, error) {
	count, err := s.GetQueueItemCount(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
