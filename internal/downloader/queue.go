package downloader

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader/types"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

var mockQueueMu sync.Mutex

// QueueItem represents a download in the queue with parsed metadata.
type QueueItem struct {
	ID             string   `json:"id"`
	ClientID       int64    `json:"clientId"`
	ClientName     string   `json:"clientName"`
	Title          string   `json:"title"`
	MediaType      string   `json:"mediaType"` // "movie" or "series"
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
}

// GetQueue returns all items from all enabled download clients.
func (s *Service) GetQueue(ctx context.Context) ([]QueueItem, error) {
	// Get all enabled clients
	clients, err := s.queries.ListEnabledDownloadClients(ctx)
	if err != nil {
		return nil, err
	}

	var items []QueueItem

	// Include mock items if in developer mode
	if s.developerMode {
		mockItems := s.GetMockQueueItems()
		items = append(items, mockItems...)
	}

	for _, dbClient := range clients {
		// Skip unimplemented clients
		if !IsClientTypeImplemented(dbClient.Type) {
			continue
		}

		clientItems, err := s.getClientQueue(ctx, dbClient.ID, dbClient.Name)
		if err != nil {
			s.logger.Warn().Err(err).Int64("clientId", dbClient.ID).Str("name", dbClient.Name).Msg("Failed to get queue from client")
			continue
		}
		items = append(items, clientItems...)
	}

	// Enrich items with library IDs from download mappings
	s.enrichQueueItemsWithMappings(ctx, items)

	return items, nil
}

// enrichQueueItemsWithMappings populates library IDs on queue items from download_mappings.
func (s *Service) enrichQueueItemsWithMappings(ctx context.Context, items []QueueItem) {
	if len(items) == 0 {
		return
	}

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

	// Enrich each item
	for i := range items {
		key := fmt.Sprintf("%d:%s", items[i].ClientID, items[i].ID)
		if mapping, ok := mappingLookup[key]; ok {
			if mapping.MovieID.Valid {
				movieID := mapping.MovieID.Int64
				items[i].MovieID = &movieID
			}
			if mapping.SeriesID.Valid {
				seriesID := mapping.SeriesID.Int64
				items[i].SeriesID = &seriesID
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
		}
	}
}

// getClientQueue fetches queue items from any download client using the unified interface.
func (s *Service) getClientQueue(ctx context.Context, clientID int64, clientName string) ([]QueueItem, error) {
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
		item := s.downloadItemToQueueItem(d, clientID, clientName)
		items = append(items, item)
	}

	return items, nil
}

// downloadItemToQueueItem converts a DownloadItem to a QueueItem.
func (s *Service) downloadItemToQueueItem(d types.DownloadItem, clientID int64, clientName string) QueueItem {
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
		Title:          title,
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
	// Check if this is a mock item
	if strings.HasPrefix(downloadID, "mock-") {
		return s.removeMockItem(downloadID)
	}

	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return err
	}

	return client.Remove(ctx, downloadID, deleteFiles)
}

// AddMockDownloads adds mock download items based on library content.
// This is only used in developer mode for testing the UI.
func (s *Service) AddMockDownloads(ctx context.Context, clientID int64, clientName string) ([]string, error) {
	mockQueueMu.Lock()
	defer mockQueueMu.Unlock()

	// Get library content to create realistic mock items
	movies, _ := s.queries.ListMovies(ctx)
	series, _ := s.queries.ListSeries(ctx)

	var addedIDs []string

	// Add a mock movie download
	if len(movies) > 0 {
		movie := movies[rand.Intn(len(movies))]
		mockID := fmt.Sprintf("mock-movie-%d-%d", movie.ID, time.Now().UnixNano())

		// Randomly select quality attributes
		qualities := []string{"1080p", "2160p"}
		sources := []string{"BluRay", "WEB-DL", "Remux"}
		codecs := []string{"x265", "x264"}
		hdrTypes := [][]string{{"DV", "HDR10"}, {"HDR10"}, {"HDR"}, {}}
		audioTypes := [][]string{{"Atmos", "TrueHD"}, {"DTS-HD"}, {"DD+"}, {}}

		quality := qualities[rand.Intn(len(qualities))]
		source := sources[rand.Intn(len(sources))]
		codec := codecs[rand.Intn(len(codecs))]

		var attrs []string
		if source == "Remux" {
			attrs = append(attrs, "REMUX")
		}
		if quality == "2160p" {
			attrs = append(attrs, hdrTypes[rand.Intn(len(hdrTypes))]...)
		}
		attrs = append(attrs, audioTypes[rand.Intn(len(audioTypes))]...)

		mockItem := MockQueueItem{
			ID:         mockID,
			ClientID:   clientID,
			ClientName: clientName,
			Title:      movie.Title,
			MediaType:  "movie",
			Quality:    quality,
			Source:     source,
			Codec:      codec,
			Attributes: attrs,
			Size:       int64(rand.Intn(40)+10) * 1024 * 1024 * 1024, // 10-50 GB
			Progress:   float64(rand.Intn(80) + 10),                  // 10-90%
			Status:     "downloading",
		}

		s.mockQueue = append(s.mockQueue, mockItem)
		addedIDs = append(addedIDs, mockID)
	}

	// Add a mock series episode download
	if len(series) > 0 {
		show := series[rand.Intn(len(series))]
		mockID := fmt.Sprintf("mock-series-%d-%d", show.ID, time.Now().UnixNano())

		qualities := []string{"1080p", "2160p", "720p"}
		sources := []string{"WEB-DL", "HDTV", "BluRay"}
		codecs := []string{"x265", "x264"}

		quality := qualities[rand.Intn(len(qualities))]
		source := sources[rand.Intn(len(sources))]
		codec := codecs[rand.Intn(len(codecs))]

		var attrs []string
		if quality == "2160p" {
			attrs = append(attrs, "HDR10")
		}
		if rand.Intn(2) == 0 {
			attrs = append(attrs, "DD+")
		}

		mockItem := MockQueueItem{
			ID:         mockID,
			ClientID:   clientID,
			ClientName: clientName,
			Title:      show.Title,
			MediaType:  "series",
			Quality:    quality,
			Source:     source,
			Codec:      codec,
			Attributes: attrs,
			Season:     rand.Intn(5) + 1,
			Episode:    rand.Intn(20) + 1,
			Size:       int64(rand.Intn(5)+1) * 1024 * 1024 * 1024, // 1-6 GB
			Progress:   float64(rand.Intn(80) + 10),                // 10-90%
			Status:     "downloading",
		}

		s.mockQueue = append(s.mockQueue, mockItem)
		addedIDs = append(addedIDs, mockID)
	}

	return addedIDs, nil
}

// GetMockQueueItems returns mock queue items as QueueItems.
func (s *Service) GetMockQueueItems() []QueueItem {
	mockQueueMu.Lock()
	defer mockQueueMu.Unlock()

	items := make([]QueueItem, 0, len(s.mockQueue))
	for _, m := range s.mockQueue {
		// Simulate download progress
		downloadedSize := int64(float64(m.Size) * m.Progress / 100)
		downloadSpeed := int64(rand.Intn(50)+10) * 1024 * 1024 // 10-60 MB/s
		remaining := m.Size - downloadedSize
		eta := int64(0)
		if downloadSpeed > 0 {
			eta = remaining / downloadSpeed
		}

		attrs := m.Attributes
		if attrs == nil {
			attrs = []string{}
		}

		items = append(items, QueueItem{
			ID:             m.ID,
			ClientID:       m.ClientID,
			ClientName:     m.ClientName,
			Title:          m.Title,
			MediaType:      m.MediaType,
			Status:         m.Status,
			Progress:       m.Progress,
			Size:           m.Size,
			DownloadedSize: downloadedSize,
			DownloadSpeed:  downloadSpeed,
			ETA:            eta,
			Quality:        m.Quality,
			Source:         m.Source,
			Codec:          m.Codec,
			Attributes:     attrs,
			Season:         m.Season,
			Episode:        m.Episode,
			DownloadPath:   "/mock/SlipStream/" + m.MediaType + "s",
		})
	}

	return items
}

// removeMockItem removes a mock item from the queue.
func (s *Service) removeMockItem(id string) error {
	mockQueueMu.Lock()
	defer mockQueueMu.Unlock()

	for i, item := range s.mockQueue {
		if item.ID == id {
			s.mockQueue = append(s.mockQueue[:i], s.mockQueue[i+1:]...)
			return nil
		}
	}
	return nil
}

// ClearMockQueue removes all mock items.
func (s *Service) ClearMockQueue() {
	mockQueueMu.Lock()
	defer mockQueueMu.Unlock()
	s.mockQueue = make([]MockQueueItem, 0)
}
