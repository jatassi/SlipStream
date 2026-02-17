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

const (
	mediaTypeMovie  = "movie"
	mediaTypeSeries = "series"
	statusQueued    = "queued"
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
	IsSeasonPack     bool   `json:"isSeasonPack"`
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
type QueueResponse struct {
	Items  []QueueItem   `json:"items"`
	Errors []ClientError `json:"errors,omitempty"`
}

// GetQueue returns all items from all enabled download clients.
// Clients are polled concurrently, each with its own 5-second timeout.
func (s *Service) GetQueue(ctx context.Context) (*QueueResponse, error) {
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return nil, err
	}

	type clientResult struct {
		clientID int64
		items    []QueueItem
		err      error
		name     string
	}

	var count int
	for _, c := range clients {
		if IsClientTypeImplemented(c.Type) {
			count++
		}
	}

	results := make(chan clientResult, count)
	for _, dbClient := range clients {
		if !IsClientTypeImplemented(dbClient.Type) {
			continue
		}
		go func(dc *sqlc.DownloadClient) {
			clientCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			items, err := s.getClientQueue(clientCtx, dc.ID, dc.Name, dc.Type)
			results <- clientResult{clientID: dc.ID, items: items, err: err, name: dc.Name}
		}(dbClient)
	}

	items := []QueueItem{}
	var clientErrors []ClientError
	for range count {
		r := <-results
		if r.err != nil {
			s.logger.Warn().Err(r.err).Int64("clientId", r.clientID).Str("name", r.name).Msg("Failed to get queue from client, serving cached data")
			clientErrors = append(clientErrors, ClientError{
				ClientID:   r.clientID,
				ClientName: r.name,
				Message:    r.err.Error(),
			})
			items = append(items, s.getCachedQueue(r.clientID)...)
			continue
		}
		s.setCachedQueue(r.clientID, r.items)
		items = append(items, r.items...)
	}

	s.enrichQueueItemsWithMappings(ctx, items)

	return &QueueResponse{Items: items, Errors: clientErrors}, nil
}

// enrichQueueItemsWithMappings populates library IDs on queue items from download_mappings.
func (s *Service) enrichQueueItemsWithMappings(_ context.Context, items []QueueItem) {
	if len(items) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mappings, err := s.queries.ListActiveDownloadMappings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get download mappings")
		return
	}

	mappingLookup := buildDownloadMappingLookup(mappings)
	slotLookup := s.buildSlotLookup(ctx)

	for i := range items {
		s.enrichSingleQueueItem(&items[i], mappingLookup, slotLookup)
	}
}

func buildDownloadMappingLookup(mappings []*sqlc.DownloadMapping) map[string]*sqlc.DownloadMapping {
	lookup := make(map[string]*sqlc.DownloadMapping)
	for _, m := range mappings {
		key := fmt.Sprintf("%d:%s", m.ClientID, m.DownloadID)
		lookup[key] = m
	}
	return lookup
}

func (s *Service) enrichSingleQueueItem(item *QueueItem, mappingLookup map[string]*sqlc.DownloadMapping, slotLookup map[int64]string) {
	key := fmt.Sprintf("%d:%s", item.ClientID, item.ID)
	mapping, ok := mappingLookup[key]
	if !ok {
		return
	}

	populateQueueItemFromMapping(item, mapping)
	populateQueueItemSlot(item, mapping, slotLookup)
}

func populateQueueItemFromMapping(item *QueueItem, mapping *sqlc.DownloadMapping) {
	if mapping.MovieID.Valid {
		movieID := mapping.MovieID.Int64
		item.MovieID = &movieID
		item.MediaType = mediaTypeMovie
	}
	if mapping.SeriesID.Valid {
		seriesID := mapping.SeriesID.Int64
		item.SeriesID = &seriesID
		item.MediaType = mediaTypeSeries
	}
	if mapping.SeasonNumber.Valid {
		seasonNum := int(mapping.SeasonNumber.Int64)
		item.SeasonNumber = &seasonNum
	}
	if mapping.EpisodeID.Valid {
		episodeID := mapping.EpisodeID.Int64
		item.EpisodeID = &episodeID
	}
	item.IsSeasonPack = mapping.IsSeasonPack == 1
	item.IsCompleteSeries = mapping.IsCompleteSeries == 1
}

func populateQueueItemSlot(item *QueueItem, mapping *sqlc.DownloadMapping, slotLookup map[int64]string) {
	if !mapping.TargetSlotID.Valid {
		return
	}

	slotID := mapping.TargetSlotID.Int64
	item.TargetSlotID = &slotID

	if slotName, ok := slotLookup[slotID]; ok {
		item.TargetSlotName = slotName
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
	for i := range downloads {
		d := &downloads[i]
		if shouldIncludeInQueue(d) {
			item := s.downloadItemToQueueItem(d, clientID, clientName, clientType)
			items = append(items, item)
		}
	}

	return items, nil
}

func shouldIncludeInQueue(d *types.DownloadItem) bool {
	if d.Status == types.StatusCompleted || d.Status == types.StatusSeeding {
		return false
	}
	if d.Progress >= 100 {
		return false
	}
	return true
}

// downloadItemToQueueItem converts a DownloadItem to a QueueItem.
func (s *Service) downloadItemToQueueItem(d *types.DownloadItem, clientID int64, clientName, clientType string) QueueItem {
	parsed := scanner.ParseFilename(d.Name)
	mediaType := detectMediaType(d.DownloadDir)

	title := parsed.Title
	if title == "" {
		title = d.Name
	}

	status := mapDownloadStatus(d.Status)

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
		return mediaTypeMovie
	}
	if strings.Contains(pathLower, "slipstream/series") || strings.Contains(pathLower, "slipstream\\series") {
		return mediaTypeSeries
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
		return statusQueued
	case types.StatusWarning:
		return "warning"
	case types.StatusError:
		return string(QueueMediaStatusFailed)
	default:
		return statusQueued
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

	for i := range downloads {
		d := &downloads[i]
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
