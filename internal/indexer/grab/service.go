// Package grab provides release grabbing functionality.
package grab

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/indexer/status"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

var (
	ErrNoDownloadClient  = errors.New("no suitable download client available")
	ErrInvalidRelease    = errors.New("invalid release")
	ErrDownloadFailed    = errors.New("download failed")
	ErrGrabLimitExceeded = errors.New("grab limit exceeded for indexer")
	ErrIndexerDisabled   = errors.New("indexer is temporarily disabled")
)

// Broadcaster interface for sending events to clients.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{}) error
}

// QueueTrigger interface for triggering immediate queue broadcasts.
type QueueTrigger interface {
	Trigger()
}

// NotificationService interface for sending external notifications.
type NotificationService interface {
	OnGrab(ctx context.Context, release *types.ReleaseInfo, clientName string, clientID int64, downloadID string, slotID *int64, slotName string)
}

// PortalStatusTracker tracks download status for portal request mirroring.
type PortalStatusTracker interface {
	OnDownloadStarted(ctx context.Context, mediaType string, mediaID int64) error
	OnDownloadFailed(ctx context.Context, mediaType string, mediaID int64) error
}

// Service handles grabbing releases and sending them to download clients.
type Service struct {
	queries              *sqlc.Queries
	downloaderService    *downloader.Service
	indexerService       IndexerClientProvider
	statusService        *status.Service
	rateLimiter          *ratelimit.Limiter
	broadcaster          Broadcaster
	queueTrigger         QueueTrigger
	notificationService  NotificationService
	portalStatusTracker  PortalStatusTracker
	logger               zerolog.Logger
}

// IndexerClientProvider provides access to indexer clients for downloading torrents.
type IndexerClientProvider interface {
	GetClient(ctx context.Context, id int64) (indexer.Indexer, error)
}

// NewService creates a new grab service.
func NewService(
	db *sql.DB,
	downloaderService *downloader.Service,
	logger zerolog.Logger,
) *Service {
	return &Service{
		queries:           sqlc.New(db),
		downloaderService: downloaderService,
		logger:            logger.With().Str("component", "grab").Logger(),
	}
}

// SetIndexerService sets the indexer service for downloading torrents with authentication.
func (s *Service) SetIndexerService(indexerService IndexerClientProvider) {
	s.indexerService = indexerService
}

// SetStatusService sets the status service for tracking indexer health.
func (s *Service) SetStatusService(statusService *status.Service) {
	s.statusService = statusService
}

// SetRateLimiter sets the rate limiter for controlling grab rates.
func (s *Service) SetRateLimiter(limiter *ratelimit.Limiter) {
	s.rateLimiter = limiter
}

// SetBroadcaster sets the WebSocket broadcaster for real-time events.
func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

// SetNotificationService sets the notification service for external notifications.
func (s *Service) SetNotificationService(notificationService NotificationService) {
	s.notificationService = notificationService
}

// SetPortalStatusTracker sets the portal status tracker for request status mirroring.
func (s *Service) SetPortalStatusTracker(tracker PortalStatusTracker) {
	s.portalStatusTracker = tracker
}

// SetQueueTrigger sets the queue trigger for immediate queue broadcasts when downloads are added.
func (s *Service) SetQueueTrigger(trigger QueueTrigger) {
	s.queueTrigger = trigger
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

// GrabRequest represents a request to grab a release.
type GrabRequest struct {
	Release          *types.ReleaseInfo `json:"release"`
	ClientID         int64              `json:"clientId,omitempty"`      // Optional: specific client
	MediaType        string             `json:"mediaType,omitempty"`     // movie, episode, season
	MediaID          int64              `json:"mediaId,omitempty"`       // movie_id, episode_id, or series_id (for season packs)
	SeriesID         int64              `json:"seriesId,omitempty"`      // For episodes
	SeasonNumber     int                `json:"seasonNumber,omitempty"`
	IsSeasonPack     bool               `json:"isSeasonPack,omitempty"`
	IsCompleteSeries bool               `json:"isCompleteSeries,omitempty"`
	TargetSlotID     *int64             `json:"targetSlotId,omitempty"`  // Target slot for multi-version mode
	Source           string             `json:"source,omitempty"`        // "auto-search", "manual-search", "portal-request"
}

// GrabResult contains the result of a grab operation.
type GrabResult struct {
	Success    bool   `json:"success"`
	DownloadID string `json:"downloadId,omitempty"`
	ClientID   int64  `json:"clientId,omitempty"`
	ClientName string `json:"clientName,omitempty"`
	Error      string `json:"error,omitempty"`
}

// BulkGrabRequest represents a request to grab multiple releases.
type BulkGrabRequest struct {
	Releases         []*types.ReleaseInfo `json:"releases"`
	ClientID         int64                `json:"clientId,omitempty"`
	MediaType        string               `json:"mediaType,omitempty"`
	MediaID          int64                `json:"mediaId,omitempty"`
	SeriesID         int64                `json:"seriesId,omitempty"`
	SeasonNumber     int                  `json:"seasonNumber,omitempty"`
	IsSeasonPack     bool                 `json:"isSeasonPack,omitempty"`
	IsCompleteSeries bool                 `json:"isCompleteSeries,omitempty"`
	// Req 18.2.1: Target slot for multi-version mode
	TargetSlotID *int64 `json:"targetSlotId,omitempty"`
	Source       string `json:"source,omitempty"`
}

// BulkGrabResult contains results from grabbing multiple releases.
type BulkGrabResult struct {
	TotalRequested int           `json:"totalRequested"`
	Successful     int           `json:"successful"`
	Failed         int           `json:"failed"`
	Results        []*GrabResult `json:"results"`
}

// Grab downloads a release and sends it to a download client.
func (s *Service) Grab(ctx context.Context, req GrabRequest) (*GrabResult, error) {
	if req.Release == nil {
		return &GrabResult{
			Success: false,
			Error:   "release is required",
		}, ErrInvalidRelease
	}

	// Broadcast grab started event
	s.broadcastGrabStarted(req.Release)

	s.logger.Info().
		Str("title", req.Release.Title).
		Int64("indexerId", req.Release.IndexerID).
		Str("protocol", string(req.Release.Protocol)).
		Msg("Grabbing release")

	// Check if indexer is disabled
	if s.statusService != nil {
		disabled, _, err := s.statusService.IsDisabled(ctx, req.Release.IndexerID)
		if err != nil {
			s.logger.Warn().Err(err).Msg("Failed to check indexer status")
		} else if disabled {
			s.broadcastGrabCompleted(req.Release, nil, "indexer is temporarily disabled due to failures")
			return &GrabResult{
				Success: false,
				Error:   "indexer is temporarily disabled due to failures",
			}, ErrIndexerDisabled
		}
	}

	// Check grab rate limit
	if s.rateLimiter != nil {
		limited, err := s.rateLimiter.CheckGrabLimit(ctx, req.Release.IndexerID)
		if err != nil {
			s.logger.Warn().Err(err).Msg("Failed to check grab limit")
		} else if limited {
			s.broadcastGrabCompleted(req.Release, nil, "grab limit exceeded for this indexer")
			return &GrabResult{
				Success: false,
				Error:   "grab limit exceeded for this indexer",
			}, ErrGrabLimitExceeded
		}
	}

	// Select download client
	client, err := s.selectDownloadClient(ctx, req.Release.Protocol, req.ClientID)
	if err != nil {
		s.broadcastGrabCompleted(req.Release, nil, fmt.Sprintf("no suitable download client: %v", err))
		return &GrabResult{
			Success: false,
			Error:   fmt.Sprintf("no suitable download client: %v", err),
		}, err
	}

	// Send the download URL to the client
	// The download client will fetch the torrent/NZB directly from the indexer
	downloadID, err := s.sendToClient(ctx, client, req.Release, req.MediaType)
	if err != nil {
		s.recordFailure(ctx, req.Release.IndexerID, err)
		s.broadcastGrabCompleted(req.Release, &GrabResult{
			ClientID:   client.ID,
			ClientName: client.Name,
		}, fmt.Sprintf("failed to send to client: %v", err))
		return &GrabResult{
			Success:    false,
			ClientID:   client.ID,
			ClientName: client.Name,
			Error:      fmt.Sprintf("failed to send to client: %v", err),
		}, err
	}

	// Record successful grab
	s.recordSuccess(ctx, req.Release.IndexerID)
	if s.rateLimiter != nil {
		s.rateLimiter.RecordGrab(ctx, req.Release.IndexerID)
	}

	// Record in history
	s.recordGrabHistory(ctx, req, client, downloadID)

	// Create download mapping for tracking what's downloading
	s.createDownloadMapping(ctx, req, client.ID, downloadID)

	result := &GrabResult{
		Success:    true,
		DownloadID: downloadID,
		ClientID:   client.ID,
		ClientName: client.Name,
	}

	// Broadcast grab completed event
	s.broadcastGrabCompleted(req.Release, result, "")

	// Broadcast queue updated so frontends refresh their download status
	s.broadcastQueueUpdated()

	// Send external notifications
	if s.notificationService != nil {
		s.notificationService.OnGrab(ctx, req.Release, client.Name, client.ID, downloadID, req.TargetSlotID, "")
	}

	s.logger.Info().
		Str("title", req.Release.Title).
		Str("downloadId", downloadID).
		Str("clientName", client.Name).
		Msg("Successfully grabbed release")

	return result, nil
}

// broadcastGrabStarted sends a grab started event.
func (s *Service) broadcastGrabStarted(release *types.ReleaseInfo) {
	if s.broadcaster == nil {
		return
	}
	s.broadcaster.Broadcast(indexer.EventGrabStarted, indexer.GrabStartedPayload{
		Title:     release.Title,
		IndexerID: release.IndexerID,
		Protocol:  string(release.Protocol),
	})
}

// broadcastGrabCompleted sends a grab completed event.
func (s *Service) broadcastGrabCompleted(release *types.ReleaseInfo, result *GrabResult, errMsg string) {
	if s.broadcaster == nil {
		return
	}
	payload := indexer.GrabCompletedPayload{
		Title:     release.Title,
		IndexerID: release.IndexerID,
		Success:   result != nil && result.Success,
		Error:     errMsg,
	}
	if result != nil {
		payload.DownloadID = result.DownloadID
		payload.ClientName = result.ClientName
	}
	s.broadcaster.Broadcast(indexer.EventGrabCompleted, payload)
}

// broadcastQueueUpdated triggers an immediate queue state broadcast.
// The QueueBroadcaster will send queue:state which directly updates
// frontend state without triggering an API refetch.
func (s *Service) broadcastQueueUpdated() {
	if s.queueTrigger != nil {
		s.queueTrigger.Trigger()
	}
}

// GrabBulk grabs multiple releases.
func (s *Service) GrabBulk(ctx context.Context, req BulkGrabRequest) (*BulkGrabResult, error) {
	result := &BulkGrabResult{
		TotalRequested: len(req.Releases),
		Results:        make([]*GrabResult, 0, len(req.Releases)),
	}

	for _, release := range req.Releases {
		// Req 18.2.1: Pass target slot through to individual grabs
		grabResult, _ := s.Grab(ctx, GrabRequest{
			Release:          release,
			ClientID:         req.ClientID,
			MediaType:        req.MediaType,
			MediaID:          req.MediaID,
			SeriesID:         req.SeriesID,
			SeasonNumber:     req.SeasonNumber,
			IsSeasonPack:     req.IsSeasonPack,
			IsCompleteSeries: req.IsCompleteSeries,
			TargetSlotID:     req.TargetSlotID,
			Source:           req.Source,
		})

		result.Results = append(result.Results, grabResult)
		if grabResult.Success {
			result.Successful++
		} else {
			result.Failed++
		}
	}

	return result, nil
}

// selectDownloadClient selects an appropriate download client.
func (s *Service) selectDownloadClient(ctx context.Context, protocol types.Protocol, preferredClientID int64) (*downloader.DownloadClient, error) {
	// If preferred client specified, try to use it
	if preferredClientID > 0 {
		client, err := s.downloaderService.Get(ctx, preferredClientID)
		if err == nil && client.Enabled {
			// Check if client supports the protocol
			if s.clientSupportsProtocol(client, protocol) {
				return client, nil
			}
		}
	}

	// Get all enabled clients
	clients, err := s.downloaderService.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list download clients: %w", err)
	}

	// Filter by protocol support and enabled status
	for _, client := range clients {
		if !client.Enabled {
			continue
		}
		if s.clientSupportsProtocol(client, protocol) {
			return client, nil
		}
	}

	return nil, ErrNoDownloadClient
}

// clientSupportsProtocol checks if a client supports the given protocol.
func (s *Service) clientSupportsProtocol(client *downloader.DownloadClient, protocol types.Protocol) bool {
	switch protocol {
	case types.ProtocolTorrent:
		return client.Type == "transmission" || client.Type == "qbittorrent" || client.Type == "deluge" || client.Type == "rtorrent" || client.Type == "mock"
	case types.ProtocolUsenet:
		return client.Type == "sabnzbd" || client.Type == "nzbget"
	default:
		return false
	}
}

// sendToClient sends the release to a download client.
func (s *Service) sendToClient(ctx context.Context, client *downloader.DownloadClient, release *types.ReleaseInfo, mediaType string) (string, error) {
	// For torrent protocol, download the torrent file using the authenticated indexer client,
	// then pass the file content to the download client. This is necessary because private
	// trackers require authentication cookies that the download client doesn't have.
	if release.Protocol == types.ProtocolTorrent && s.indexerService != nil {
		indexerClient, err := s.indexerService.GetClient(ctx, release.IndexerID)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Int64("indexerId", release.IndexerID).
				Msg("Failed to get indexer client, falling back to direct URL")
		} else {
			torrentData, err := indexerClient.Download(ctx, release.DownloadURL)
			if err != nil {
				s.logger.Warn().
					Err(err).
					Str("url", release.DownloadURL).
					Msg("Failed to download torrent via indexer, falling back to direct URL")
			} else {
				downloadID, err := s.downloaderService.AddTorrentWithContent(ctx, client.ID, torrentData, mediaType, release.Title)
				if err != nil {
					return "", fmt.Errorf("failed to add download: %w", err)
				}
				return downloadID, nil
			}
		}
	}

	// Fallback: Use direct URL (works for public trackers or magnet links)
	downloadID, err := s.downloaderService.AddTorrent(ctx, client.ID, release.DownloadURL, mediaType, release.Title)
	if err != nil {
		return "", fmt.Errorf("failed to add download: %w", err)
	}

	return downloadID, nil
}

// recordSuccess records a successful grab operation.
func (s *Service) recordSuccess(ctx context.Context, indexerID int64) {
	if s.statusService == nil {
		return
	}
	if err := s.statusService.RecordSuccess(ctx, indexerID); err != nil {
		s.logger.Warn().Err(err).Int64("indexerId", indexerID).Msg("Failed to record success")
	}
}

// recordFailure records a failed grab operation.
func (s *Service) recordFailure(ctx context.Context, indexerID int64, opError error) {
	if s.statusService == nil {
		return
	}
	if err := s.statusService.RecordFailure(ctx, indexerID, opError); err != nil {
		s.logger.Warn().Err(err).Int64("indexerId", indexerID).Msg("Failed to record failure")
	}
}

// recordGrabHistory records the grab in indexer history.
func (s *Service) recordGrabHistory(ctx context.Context, req GrabRequest, client *downloader.DownloadClient, downloadID string) {
	// Check if the indexer exists locally before recording history.
	// In Prowlarr mode, indexer IDs come from Prowlarr and don't exist in the local database.
	if _, err := s.queries.GetIndexer(ctx, req.Release.IndexerID); err != nil {
		s.logger.Debug().Int64("indexerId", req.Release.IndexerID).Msg("Skipping grab history - indexer not in local database (likely Prowlarr mode)")
		return
	}

	_, err := s.queries.CreateIndexerHistoryEvent(ctx, sqlc.CreateIndexerHistoryEventParams{
		IndexerID:    req.Release.IndexerID,
		EventType:    "grab",
		Successful:   1,
		Query:        sql.NullString{String: req.Release.Title, Valid: true},
		Categories:   sql.NullString{},
		ResultsCount: sql.NullInt64{Int64: 1, Valid: true},
		ElapsedMs:    sql.NullInt64{},
		Data: sql.NullString{
			String: fmt.Sprintf(`{"downloadId":"%s","clientId":%d,"clientName":"%s","mediaType":"%s","mediaId":%d}`,
				downloadID, client.ID, client.Name, req.MediaType, req.MediaID),
			Valid: true,
		},
	})
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to record grab history")
	}
}

// GetGrabHistory returns recent grab history.
func (s *Service) GetGrabHistory(ctx context.Context, limit, offset int) ([]GrabHistoryItem, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.queries.ListRecentIndexerHistory(ctx, sqlc.ListRecentIndexerHistoryParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	// Filter for grab events only
	items := make([]GrabHistoryItem, 0)
	for _, row := range rows {
		if row.EventType != "grab" {
			continue
		}
		items = append(items, GrabHistoryItem{
			ID:         row.ID,
			IndexerID:  row.IndexerID,
			Title:      row.Query.String,
			Successful: row.Successful == 1,
			CreatedAt:  row.CreatedAt.Time,
			Data:       row.Data.String,
		})
	}

	return items, nil
}

// GrabHistoryItem represents a grab history entry.
type GrabHistoryItem struct {
	ID         int64     `json:"id"`
	IndexerID  int64     `json:"indexerId"`
	Title      string    `json:"title"`
	Successful bool      `json:"successful"`
	CreatedAt  time.Time `json:"createdAt"`
	Data       string    `json:"data,omitempty"`
}

// createDownloadMapping creates a mapping between a download and its library item.
func (s *Service) createDownloadMapping(ctx context.Context, req GrabRequest, clientID int64, downloadID string) {
	source := req.Source
	if source == "" {
		source = "auto-search"
	}
	params := sqlc.CreateDownloadMappingParams{
		ClientID:   clientID,
		DownloadID: downloadID,
		Source:     source,
	}

	// Set target slot ID if provided (for multi-version mode)
	if req.TargetSlotID != nil {
		params.TargetSlotID = sql.NullInt64{Int64: *req.TargetSlotID, Valid: true}
	}

	switch req.MediaType {
	case "movie":
		params.MovieID = sql.NullInt64{Int64: req.MediaID, Valid: true}
	case "episode":
		params.EpisodeID = sql.NullInt64{Int64: req.MediaID, Valid: true}
		if req.SeriesID > 0 {
			params.SeriesID = sql.NullInt64{Int64: req.SeriesID, Valid: true}
		}
		if req.SeasonNumber > 0 {
			params.SeasonNumber = sql.NullInt64{Int64: int64(req.SeasonNumber), Valid: true}
		}
	case "season":
		params.SeriesID = sql.NullInt64{Int64: req.MediaID, Valid: true}
		params.SeasonNumber = sql.NullInt64{Int64: int64(req.SeasonNumber), Valid: true}
		if req.IsSeasonPack {
			params.IsSeasonPack = 1
		}
		if req.IsCompleteSeries {
			params.IsCompleteSeries = 1
		}
	default:
		return
	}

	_, err := s.queries.CreateDownloadMapping(ctx, params)
	if err != nil {
		s.logger.Warn().Err(err).
			Str("downloadId", downloadID).
			Str("mediaType", req.MediaType).
			Int64("mediaId", req.MediaID).
			Msg("Failed to create download mapping")
		return
	}

	activeDownloadID := sql.NullString{String: downloadID, Valid: true}

	switch req.MediaType {
	case "movie":
		if err := s.queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
			Status:           "downloading",
			ActiveDownloadID: activeDownloadID,
			StatusMessage:    sql.NullString{},
			ID:               req.MediaID,
		}); err != nil {
			s.logger.Warn().Err(err).Int64("movieId", req.MediaID).Msg("Failed to set movie status to downloading")
		} else if s.broadcaster != nil {
			_ = s.broadcaster.Broadcast("movie:updated", map[string]any{"movieId": req.MediaID})
		}
	case "episode":
		if err := s.queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
			Status:           "downloading",
			ActiveDownloadID: activeDownloadID,
			StatusMessage:    sql.NullString{},
			ID:               req.MediaID,
		}); err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", req.MediaID).Msg("Failed to set episode status to downloading")
		} else if s.broadcaster != nil && req.SeriesID > 0 {
			_ = s.broadcaster.Broadcast("series:updated", map[string]any{"id": req.SeriesID})
		}
	}

	// Notify portal status tracker for request mirroring
	if s.portalStatusTracker != nil {
		_ = s.portalStatusTracker.OnDownloadStarted(ctx, req.MediaType, req.MediaID)
	}
}
