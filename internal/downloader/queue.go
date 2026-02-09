package downloader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader/mock"
	"github.com/slipstream/slipstream/internal/downloader/types"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// QueueItem represents a download in the queue with parsed metadata.
type QueueItem struct {
	ID             string   `json:"id"`
	ClientID       int64    `json:"clientId"`
	ClientName     string   `json:"clientName"`
	ClientType     string   `json:"clientType"`
	Title          string   `json:"title"`
	ReleaseName    string   `json:"releaseName"` // Original release name from download client
	MediaType      string   `json:"mediaType"`   // "movie" or "series"
	Status         string   `json:"status"`
	Progress       float64  `json:"progress"`       // 0-100
	Size           int64    `json:"size"`           // bytes
	DownloadedSize int64    `json:"downloadedSize"` // bytes
	DownloadSpeed  int64    `json:"downloadSpeed"`  // bytes/sec
	ETA            int64    `json:"eta"`            // seconds, -1 if unavailable
	Quality        string   `json:"quality,omitempty"`
	Source         string   `json:"source,omitempty"`
	Codec          string   `json:"codec,omitempty"`
	Attributes     []string `json:"attributes"` // HDR, Atmos, REMUX, etc.
	Season         int      `json:"season,omitempty"`
	Episode        int      `json:"episode,omitempty"`
	DownloadPath   string   `json:"downloadPath"`
	// Library mapping - populated from download_mappings table
	MovieID          *int64 `json:"movieId,omitempty"`
	SeriesID         *int64 `json:"seriesId,omitempty"`
	SeasonNumber     *int   `json:"seasonNumber,omitempty"`
	EpisodeID        *int64 `json:"episodeId,omitempty"`
	IsSeasonPack     bool   `json:"isSeasonPack,omitempty"`
	IsCompleteSeries bool   `json:"isCompleteSeries,omitempty"`
	// Req 10.1.2: Target slot info shown inline with each queue item
	TargetSlotID   *int64 `json:"targetSlotId,omitempty"`
	TargetSlotName string `json:"targetSlotName,omitempty"`
}

// ClientError reports a per-client failure when fetching the queue.
type ClientError struct {
	ClientID   int64  `json:"clientId"`
	ClientName string `json:"clientName"`
	Message    string `json:"message"`
}

// QueueResponse wraps queue items together with any per-client errors.
// When a client is unreachable, its last known items are served as stale data
// and the error is reported so the frontend can display a warning.
type QueueResponse struct {
	Items  []QueueItem   `json:"items"`
	Errors []ClientError `json:"errors,omitempty"`
}

// GetQueue returns all items from all enabled download clients.
// On per-client failure, cached (stale) items are served and the error is reported.
func (s *Service) GetQueue(ctx context.Context) (*QueueResponse, error) {
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return nil, err
	}

	var items []QueueItem
	var clientErrors []ClientError

	for _, dbClient := range clients {
		if !IsClientTypeImplemented(dbClient.Type) {
			continue
		}

		clientItems, err := s.getClientQueue(ctx, dbClient.ID, dbClient.Name, dbClient.Type)
		if err != nil {
			s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Str("name", dbClient.Name).Msg("Failed to get queue from client, serving cached data")
			clientErrors = append(clientErrors, ClientError{
				ClientID:   dbClient.ID,
				ClientName: dbClient.Name,
				Message:    err.Error(),
			})
			items = append(items, s.getCachedQueue(dbClient.ID)...)
			continue
		}
		s.setCachedQueue(dbClient.ID, clientItems)
		items = append(items, clientItems...)
	}

	s.enrichQueueItemsWithMappings(ctx, items)

	return &QueueResponse{Items: items, Errors: clientErrors}, nil
}

// enrichQueueItemsWithMappings populates library IDs on queue items from download_mappings.
// Req 10.1.1: Queue shows raw downloads with mapped media
// Req 10.1.2: Target slot info shown inline with each queue item
func (s *Service) enrichQueueItemsWithMappings(_ context.Context, items []QueueItem) {
	if len(items) == 0 {
		return
	}

	// Use a fresh context with its own timeout for the database queries.
	// The parent context may be nearly exhausted from slow download client responses,
	// but this enrichment is non-critical and shouldn't fail the entire queue fetch.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all active download mappings
	mappings, err := s.queries.ListActiveDownloadMappings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get download mappings")
		return
	}

	// Build lookup map: clientID:downloadID -> mapping
	mappingLookup := make(map[string]*sqlc.DownloadMapping)
	for _, m := range mappings {
		key := fmt.Sprintf("%d:%s", m.ClientID, m.DownloadID)
		mappingLookup[key] = m
	}

	// Build slot lookup map: slotID -> slot name
	slotLookup := s.buildSlotLookup(ctx)

	// Enrich each item
	for i := range items {
		key := fmt.Sprintf("%d:%s", items[i].ClientID, items[i].ID)
		if mapping, ok := mappingLookup[key]; ok {
			if mapping.MovieID.Valid {
				movieID := mapping.MovieID.Int64
				items[i].MovieID = &movieID
				items[i].MediaType = "movie"
			}
			if mapping.SeriesID.Valid {
				seriesID := mapping.SeriesID.Int64
				items[i].SeriesID = &seriesID
				items[i].MediaType = "series"
			}
			if mapping.SeasonNumber.Valid {
				seasonNum := int(mapping.SeasonNumber.Int64)
				items[i].SeasonNumber = &seasonNum
			}
			if mapping.EpisodeID.Valid {
				episodeID := mapping.EpisodeID.Int64
				items[i].EpisodeID = &episodeID
			}
			items[i].IsSeasonPack = mapping.IsSeasonPack == 1
			items[i].IsCompleteSeries = mapping.IsCompleteSeries == 1

			// Req 10.1.2: Populate target slot info
			if mapping.TargetSlotID.Valid {
				slotID := mapping.TargetSlotID.Int64
				items[i].TargetSlotID = &slotID
				if slotName, ok := slotLookup[slotID]; ok {
					items[i].TargetSlotName = slotName
				}
			}
		}
	}
}

// buildSlotLookup builds a map of slot IDs to slot names.
func (s *Service) buildSlotLookup(ctx context.Context) map[int64]string {
	slots, err := s.queries.ListVersionSlots(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get version slots for queue enrichment")
		return nil
	}

	lookup := make(map[int64]string, len(slots))
	for _, slot := range slots {
		lookup[slot.ID] = slot.Name
	}
	return lookup
}

// getClientQueue fetches queue items from any download client using the unified interface.
// Completed/seeding downloads are filtered out - use CheckForCompletedDownloads for import processing.
func (s *Service) getClientQueue(ctx context.Context, clientID int64, clientName, clientType string) ([]QueueItem, error) {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	downloads, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]QueueItem, 0, len(downloads))
	for _, d := range downloads {
		// Filter out completed/seeding downloads - they should not appear in the queue.
		if d.Status == types.StatusCompleted || d.Status == types.StatusSeeding {
			continue
		}
		// Filter out fully-downloaded items regardless of reported status.
		// Covers: stopped after seeding (StatusPaused at 100%), seeding with
		// tracker/piece errors (StatusError at 100% — Transmission overrides
		// seeding status to error when torrent has issues), etc.
		if d.Progress >= 100 {
			continue
		}
		item := s.downloadItemToQueueItem(d, clientID, clientName, clientType)
		items = append(items, item)
	}

	return items, nil
}

// downloadItemToQueueItem converts a DownloadItem to a QueueItem.
func (s *Service) downloadItemToQueueItem(d types.DownloadItem, clientID int64, clientName, clientType string) QueueItem {
	// Parse the download name to extract metadata
	parsed := scanner.ParseFilename(d.Name)

	// Determine media type from download path
	mediaType := detectMediaType(d.DownloadDir)

	// Use parsed title or fall back to the download name
	title := parsed.Title
	if title == "" {
		title = d.Name
	}

	// Map status
	status := mapDownloadStatus(d.Status)

	// Ensure attributes is not nil
	attributes := parsed.Attributes
	if attributes == nil {
		attributes = []string{}
	}

	return QueueItem{
		ID:             d.ID,
		ClientID:       clientID,
		ClientName:     clientName,
		ClientType:     clientType,
		Title:          title,
		ReleaseName:    d.Name,
		MediaType:      mediaType,
		Status:         status,
		Progress:       d.Progress,
		Size:           d.Size,
		DownloadedSize: d.DownloadedSize,
		DownloadSpeed:  d.DownloadSpeed,
		ETA:            d.ETA,
		Quality:        parsed.Quality,
		Source:         parsed.Source,
		Codec:          parsed.Codec,
		Attributes:     attributes,
		Season:         parsed.Season,
		Episode:        parsed.Episode,
		DownloadPath:   d.DownloadDir,
	}
}

// detectMediaType determines if the download is a movie or series based on the path.
func detectMediaType(path string) string {
	pathLower := strings.ToLower(path)
	if strings.Contains(pathLower, "slipstream/movies") || strings.Contains(pathLower, "slipstream\\movies") {
		return "movie"
	}
	if strings.Contains(pathLower, "slipstream/series") || strings.Contains(pathLower, "slipstream\\series") {
		return "series"
	}
	return "unknown"
}

// mapDownloadStatus maps types.Status to our queue status string.
func mapDownloadStatus(status types.Status) string {
	switch status {
	case types.StatusDownloading:
		return "downloading"
	case types.StatusPaused:
		return "paused"
	case types.StatusCompleted, types.StatusSeeding:
		return "completed"
	case types.StatusQueued:
		return "queued"
	case types.StatusError:
		return "error"
	default:
		return "queued"
	}
}

// HasActiveDownloads checks if a specific client has any active downloads (downloading or queued).
func (s *Service) HasActiveDownloads(ctx context.Context, clientID int64) (bool, error) {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return false, err
	}

	downloads, err := client.List(ctx)
	if err != nil {
		return false, err
	}

	for _, d := range downloads {
		if d.Status == types.StatusDownloading || d.Status == types.StatusQueued {
			return true, nil
		}
	}

	return false, nil
}

// PauseDownload pauses a download.
func (s *Service) PauseDownload(ctx context.Context, clientID int64, downloadID string) error {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return err
	}

	return client.Pause(ctx, downloadID)
}

// ResumeDownload resumes a paused download.
func (s *Service) ResumeDownload(ctx context.Context, clientID int64, downloadID string) error {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return err
	}

	return client.Resume(ctx, downloadID)
}

// RemoveDownload removes a download from the client.
func (s *Service) RemoveDownload(ctx context.Context, clientID int64, downloadID string, deleteFiles bool) error {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return err
	}

	return client.Remove(ctx, downloadID, deleteFiles)
}

// FastForwardMockDownload instantly completes a mock download.
// Returns an error if the client is not a mock client.
func (s *Service) FastForwardMockDownload(ctx context.Context, clientID int64, downloadID string) error {
	cfg, err := s.Get(ctx, clientID)
	if err != nil {
		return err
	}

	if cfg.Type != "mock" {
		return fmt.Errorf("fast forward is only available for mock download clients")
	}

	mockClient := mock.GetInstance()
	return mockClient.FastForward(downloadID)
}
