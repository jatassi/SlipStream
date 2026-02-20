package importer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/websocket"
)

var (
	ErrNoProbeToolAvailable = errors.New("no media probe tool available")
	ErrFileNotFound         = errors.New("file not found")
	ErrFileTooSmall         = errors.New("file too small")
	ErrInvalidExtension     = errors.New("invalid file extension")
	ErrSampleFile           = errors.New("file appears to be a sample")
	ErrNoMatch              = errors.New("could not match file to library")
	ErrMatchConflict        = errors.New("match conflict between queue and parse")
	ErrAlreadyImporting     = errors.New("file is already being imported")
	ErrImportFailed         = errors.New("import failed after retries")
	ErrPathTooLong          = errors.New("destination path exceeds maximum length")
	ErrFileAlreadyInLibrary = errors.New("file already exists in library")
	ErrNotAnUpgrade         = errors.New("candidate file is not a quality upgrade")
)

// HealthService defines the interface for health tracking.
type HealthService interface {
	RegisterItemStr(category, id, name string)
	UnregisterItemStr(category, id string)
	SetErrorStr(category, id, message string)
	SetWarningStr(category, id, message string)
	ClearStatusStr(category, id string)
}

// HistoryService defines the interface for history logging.
type HistoryService interface {
	Create(ctx context.Context, input *HistoryInput) error
}

// HistoryInput is a simplified version of history.CreateInput to avoid import cycles.
type HistoryInput struct {
	EventType string
	MediaType string
	MediaID   int64
	Source    string
	Quality   string
	Data      map[string]any
}

// NotificationDispatcher defines the interface for import notifications.
type NotificationDispatcher interface {
	DispatchImport(ctx context.Context, event *ImportNotificationEvent)
	DispatchUpgrade(ctx context.Context, event *UpgradeNotificationEvent)
}

// StatusTrackerService defines the interface for request status tracking.
type StatusTrackerService interface {
	OnMovieAvailable(ctx context.Context, movieID int64) error
	OnEpisodeAvailable(ctx context.Context, episodeID int64) error
	OnDownloadFailed(ctx context.Context, mediaType string, mediaID int64) error
}

// ImportNotificationEvent contains import event data for notifications.
type ImportNotificationEvent struct {
	MediaType       string // "movie" or "episode"
	MovieID         *int64
	MovieTitle      string
	MovieYear       int
	SeriesID        *int64
	SeriesTitle     string
	EpisodeID       *int64
	SeasonNumber    int
	EpisodeNumber   int
	EpisodeTitle    string
	Quality         string
	SourcePath      string
	DestinationPath string
	ReleaseName     string
	SlotID          *int64
	SlotName        string
}

// UpgradeNotificationEvent contains upgrade event data for notifications.
type UpgradeNotificationEvent struct {
	MediaType     string // "movie" or "episode"
	MovieID       *int64
	MovieTitle    string
	MovieYear     int
	SeriesID      *int64
	SeriesTitle   string
	EpisodeID     *int64
	SeasonNumber  int
	EpisodeNumber int
	EpisodeTitle  string
	OldQuality    string
	NewQuality    string
	OldPath       string
	NewPath       string
	ReleaseName   string
	SlotID        *int64
	SlotName      string
}

// Config holds import service configuration.
type Config struct {
	WorkerCount int // Number of concurrent import workers (default: 1)
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		WorkerCount: 1, // Sequential processing as per spec
	}
}

// Service orchestrates the import pipeline.
type Service struct {
	db            *sql.DB
	queries       *sqlc.Queries
	downloader    *downloader.Service
	movies        *movies.Service
	tv            *tv.Service
	rootfolder    *rootfolder.Service
	organizer     *organizer.Service
	renamer       *renamer.Resolver
	mediainfo     *mediainfo.Service
	quality       *quality.Service
	slots         *slots.Service
	health        HealthService
	history       HistoryService
	notifier      NotificationDispatcher
	statusTracker StatusTrackerService
	hub           *websocket.Hub
	logger        *zerolog.Logger
	config        Config

	// Import queue
	importQueue chan ImportJob
	wg          sync.WaitGroup

	// Processing state
	mu         sync.Mutex
	processing map[string]bool // Track in-progress imports by path
	shutdown   chan struct{}
}

// ImportJob represents a single import task.
type ImportJob struct {
	SourcePath      string           // Path to the source file
	DownloadMapping *DownloadMapping // Queue metadata (nil for manual imports)
	QueueMedia      *QueueMedia      // Per-file status for season packs (nil for single files)
	Manual          bool             // Whether this is a manual import
	ConfirmedMatch  *LibraryMatch    // Pre-confirmed match for manual imports
	TargetSlotID    *int64           // Req 5.2.3: User-specified target slot (nil = auto-detect)
}

// DownloadMapping represents the queue-to-library mapping.
type DownloadMapping struct {
	ID               int64
	DownloadClientID int64
	DownloadID       string
	MediaType        string // "movie", "episode", "season", or "series"
	MovieID          *int64
	SeriesID         *int64
	SeasonNumber     *int
	EpisodeID        *int64
	IsSeasonPack     bool
	IsCompleteSeries bool
	TargetSlotID     *int64
	Source           string // "auto-search", "manual-search", "portal-request"
}

// QueueMedia represents per-file status within a download.
type QueueMedia struct {
	ID                int64
	DownloadMappingID int64
	EpisodeID         *int64
	MovieID           *int64
	FilePath          string
	FileStatus        string // "pending", "downloading", "ready", "importing", "imported", "failed"
	ErrorMessage      string
	ImportAttempts    int
}

// LibraryMatch represents a matched library item.
type LibraryMatch struct {
	MediaType          string  // "movie" or "episode"
	MovieID            *int64  // Set for movies
	SeriesID           *int64  // Set for episodes
	SeasonNum          *int    // Set for episodes
	EpisodeID          *int64  // Set for episodes
	EpisodeIDs         []int64 // Set for multi-episode files
	Confidence         float64 // Match confidence 0.0 - 1.0
	Source             string  // "queue", "parse", "manual"
	RootFolder         string  // Root folder path
	IsUpgrade          bool    // Whether this replaces an existing file
	ExistingFile       string  // Path to existing file if upgrade
	ExistingFileID     *int64  // ID of existing file for database cleanup
	CandidateQualityID int     // Quality ID of the candidate file
	ExistingQualityID  int     // Quality ID of the existing file
	QualityProfileID   int64   // Quality profile used for evaluation
}

// ImportResult contains the result of an import operation.
type ImportResult struct {
	Success         bool
	SourcePath      string
	DestinationPath string
	Match           *LibraryMatch
	MediaInfo       *mediainfo.MediaInfo
	LinkMode        organizer.LinkMode
	Error           error
	IsUpgrade       bool
	PreviousFile    string

	// Slot information (Req 5.2.1-5.2.3)
	RequiresSlotSelection bool             // True if user must select a slot
	SlotAssignments       []SlotAssignment // Available slot options when selection required
	RecommendedSlotID     *int64           // Recommended slot based on best match
	AssignedSlotID        *int64           // Slot file was assigned to after import
}

// SlotAssignment represents a potential slot for a file.
type SlotAssignment struct {
	SlotID     int64   `json:"slotId"`
	SlotNumber int     `json:"slotNumber"`
	SlotName   string  `json:"slotName"`
	MatchScore float64 `json:"matchScore"`
	IsUpgrade  bool    `json:"isUpgrade"`
	IsNewFill  bool    `json:"isNewFill"`
}

// NewService creates a new import service.
func NewService(
	db *sql.DB,
	downloaderSvc *downloader.Service,
	moviesSvc *movies.Service,
	tvSvc *tv.Service,
	rootfolderSvc *rootfolder.Service,
	organizerSvc *organizer.Service,
	mediainfoSvc *mediainfo.Service,
	hub *websocket.Hub,
	config Config,
	logger *zerolog.Logger,
) *Service {
	subLogger := logger.With().Str("component", "import").Logger()
	s := &Service{
		db:          db,
		queries:     sqlc.New(db),
		downloader:  downloaderSvc,
		movies:      moviesSvc,
		tv:          tvSvc,
		rootfolder:  rootfolderSvc,
		organizer:   organizerSvc,
		mediainfo:   mediainfoSvc,
		hub:         hub,
		logger:      &subLogger,
		config:      config,
		importQueue: make(chan ImportJob, 100),
		processing:  make(map[string]bool),
		shutdown:    make(chan struct{}),
	}

	// Initialize renamer with default settings
	// Settings will be loaded from database on first use
	defaultSettings := renamer.DefaultSettings()
	s.renamer = renamer.NewResolver(&defaultSettings)

	return s
}

// SetHealthService sets the health service for tracking import health.
func (s *Service) SetHealthService(hs HealthService) {
	s.health = hs
}

// SetHistoryService sets the history service for logging import events.
func (s *Service) SetHistoryService(hs HistoryService) {
	s.history = hs
}

// SetQualityService sets the quality service for upgrade comparison during import.
func (s *Service) SetQualityService(qs *quality.Service) {
	s.quality = qs
}

// SetSlotsService sets the slots service for multi-version support.
func (s *Service) SetSlotsService(ss *slots.Service) {
	s.slots = ss
}

// SetNotificationDispatcher sets the notification dispatcher for import events.
func (s *Service) SetNotificationDispatcher(n NotificationDispatcher) {
	s.notifier = n
}

// SetStatusTracker sets the status tracker for portal request updates.
func (s *Service) SetStatusTracker(st StatusTrackerService) {
	s.statusTracker = st
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// UpdateRenamerSettings updates the renamer with new settings.
func (s *Service) UpdateRenamerSettings(settings *renamer.Settings) {
	s.renamer = renamer.NewResolver(settings)
}

// Start starts the import worker(s).
func (s *Service) Start(ctx context.Context) {
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}
	s.logger.Info().Int("workers", s.config.WorkerCount).Msg("Import service started")
}

// Stop stops the import service and waits for pending jobs.
func (s *Service) Stop() {
	close(s.shutdown)
	s.wg.Wait()
	s.logger.Info().Msg("Import service stopped")
}

// worker processes import jobs from the queue.
func (s *Service) worker(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.shutdown:
			return
		case job := <-s.importQueue:
			s.processJob(ctx, job)
		}
	}
}

// QueueImport queues a file for import.
func (s *Service) QueueImport(job ImportJob) error {
	s.mu.Lock()
	if s.processing[job.SourcePath] {
		s.mu.Unlock()
		return ErrAlreadyImporting
	}
	s.processing[job.SourcePath] = true
	s.mu.Unlock()

	select {
	case s.importQueue <- job:
		return nil
	default:
		s.mu.Lock()
		delete(s.processing, job.SourcePath)
		s.mu.Unlock()
		return errors.New("import queue is full")
	}
}

// IsProcessing returns whether a file is currently being imported.
func (s *Service) IsProcessing(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.processing[path]
}

// markComplete removes a path from the processing set.
func (s *Service) markComplete(path string) {
	s.mu.Lock()
	delete(s.processing, path)
	s.mu.Unlock()
}

// processJob handles a single import job with retry logic.
func (s *Service) processJob(ctx context.Context, job ImportJob) {
	defer s.markComplete(job.SourcePath)

	s.logger.Info().
		Str("path", job.SourcePath).
		Bool("manual", job.Manual).
		Msg("Processing import job")

	if job.DownloadMapping != nil && job.DownloadMapping.DownloadClientID > 0 {
		if err := s.applyImportDelay(ctx, job.DownloadMapping.DownloadClientID); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to apply import delay")
		}
	}

	result := s.processWithRetry(ctx, job)

	if result.Success {
		s.handleSuccessfulImport(ctx, result)
	} else {
		s.handleFailedImport(ctx, job, result)
	}
}

func (s *Service) handleSuccessfulImport(ctx context.Context, result *ImportResult) {
	s.logger.Info().
		Str("source", result.SourcePath).
		Str("destination", result.DestinationPath).
		Str("linkMode", string(result.LinkMode)).
		Bool("upgrade", result.IsUpgrade).
		Msg("Import completed successfully")

	s.updatePortalRequestStatus(ctx, result)
	s.broadcastImportSuccess(result)
	s.dispatchImportNotification(ctx, result)
}

func (s *Service) updatePortalRequestStatus(ctx context.Context, result *ImportResult) {
	if s.statusTracker == nil || result.Match == nil {
		return
	}

	if result.Match.MediaType == mediaTypeMovie && result.Match.MovieID != nil {
		if err := s.statusTracker.OnMovieAvailable(ctx, *result.Match.MovieID); err != nil {
			s.logger.Warn().Err(err).Int64("movieId", *result.Match.MovieID).Msg("Failed to update request status")
		}
	} else if result.Match.MediaType == mediaTypeEpisode && result.Match.EpisodeID != nil {
		if err := s.statusTracker.OnEpisodeAvailable(ctx, *result.Match.EpisodeID); err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", *result.Match.EpisodeID).Msg("Failed to update request status")
		}
	}
}

func (s *Service) broadcastImportSuccess(result *ImportResult) {
	if s.hub == nil {
		return
	}

	s.hub.Broadcast("import:completed", map[string]any{
		"source":      result.SourcePath,
		"destination": result.DestinationPath,
		"mediaType":   result.Match.MediaType,
		"isUpgrade":   result.IsUpgrade,
	})

	if result.Match.MediaType == mediaTypeMovie && result.Match.MovieID != nil {
		s.hub.Broadcast("movie:updated", map[string]any{"movieId": *result.Match.MovieID})
	} else if result.Match.MediaType == mediaTypeEpisode && result.Match.SeriesID != nil {
		s.hub.Broadcast("series:updated", map[string]any{"id": *result.Match.SeriesID})
	}
}

func (s *Service) handleFailedImport(ctx context.Context, job ImportJob, result *ImportResult) {
	s.logger.Error().
		Err(result.Error).
		Str("path", job.SourcePath).
		Msg("Import failed")

	s.updateMediaStatusToFailed(ctx, job, result)
	s.registerImportFailureHealth(job, result)
	s.broadcastImportFailure(result)
}

func (s *Service) updateMediaStatusToFailed(ctx context.Context, job ImportJob, result *ImportResult) {
	if job.DownloadMapping == nil {
		return
	}

	statusMsg := sql.NullString{String: result.Error.Error(), Valid: true}

	switch {
	case job.DownloadMapping.MovieID != nil:
		s.setMovieStatusFailed(ctx, *job.DownloadMapping.MovieID, statusMsg)
	case job.DownloadMapping.EpisodeID != nil:
		s.setEpisodeStatusFailed(ctx, job.DownloadMapping, statusMsg)
	}
}

func (s *Service) setMovieStatusFailed(ctx context.Context, movieID int64, statusMsg sql.NullString) {
	_ = s.queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		Status:           "failed",
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    statusMsg,
		ID:               movieID,
	})
	if s.hub != nil {
		s.hub.Broadcast("movie:updated", map[string]any{"movieId": movieID})
	}
	if s.statusTracker != nil {
		_ = s.statusTracker.OnDownloadFailed(ctx, "movie", movieID)
	}
}

func (s *Service) setEpisodeStatusFailed(ctx context.Context, mapping *DownloadMapping, statusMsg sql.NullString) {
	_ = s.queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		Status:           "failed",
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    statusMsg,
		ID:               *mapping.EpisodeID,
	})
	if s.hub != nil && mapping.SeriesID != nil {
		s.hub.Broadcast("series:updated", map[string]any{"id": *mapping.SeriesID})
	}
	if s.statusTracker != nil {
		_ = s.statusTracker.OnDownloadFailed(ctx, "episode", *mapping.EpisodeID)
	}
}

func (s *Service) registerImportFailureHealth(job ImportJob, result *ImportResult) {
	if s.health != nil {
		s.health.SetWarningStr("import", job.SourcePath, result.Error.Error())
	}
}

func (s *Service) broadcastImportFailure(result *ImportResult) {
	if s.hub != nil {
		s.hub.Broadcast("import:failed", map[string]any{
			"source": result.SourcePath,
			"error":  result.Error.Error(),
		})
	}
}

func (s *Service) GetQueueLength() int {
	return len(s.importQueue)
}

// HandleDownloadWatcherEvent is the handler for download watcher events.
// It's called when a video file is detected as complete in a download folder.
func (s *Service) HandleDownloadWatcherEvent(ctx context.Context, path string, clientID int64) error {
	s.logger.Debug().
		Str("path", path).
		Int64("clientId", clientID).
		Msg("Received download watcher event")

	// Check if we should skip stalled downloads
	// Try to find a mapping for this file's parent directory
	mapping, err := s.findMappingForPath(ctx, path, clientID)
	if err != nil {
		s.logger.Debug().Err(err).Str("path", path).Msg("No mapping found for file, proceeding with unmapped import")
	}

	// If we have a mapping, check if download is stalled
	if mapping != nil {
		if s.ShouldSkipStalledDownload(ctx, clientID, mapping.DownloadID) {
			s.logger.Info().Str("path", path).Msg("Skipping file from stalled download")
			return nil
		}
	}

	// Check for archive extraction needs
	parentDir := filepath.Dir(path)
	needsWait, err := s.NeedsExtractionWait(ctx, parentDir)
	if err != nil {
		s.logger.Warn().Err(err).Str("path", parentDir).Msg("Failed to check extraction status")
	} else if needsWait {
		s.logger.Info().Str("path", parentDir).Msg("Archive extraction in progress, skipping for now")
		return nil
	}

	// Queue the import
	job := ImportJob{
		SourcePath:      path,
		DownloadMapping: mapping,
		Manual:          false,
	}

	return s.QueueImport(job)
}

// findMappingForPath attempts to find a download mapping for a file path.
func (s *Service) findMappingForPath(ctx context.Context, filePath string, clientID int64) (*DownloadMapping, error) {
	mappings, err := s.queries.ListActiveDownloadMappings(ctx)
	if err != nil {
		return nil, err
	}

	client, err := s.downloader.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	items, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	downloadID := s.findDownloadIDForPath(filePath, items)
	if downloadID == "" {
		return nil, ErrNoMatch
	}

	return s.findMappingByClientAndDownload(mappings, clientID, downloadID)
}

func (s *Service) findDownloadIDForPath(filePath string, items []downloader.DownloadItem) string {
	for i := range items {
		item := &items[i]
		if item.DownloadDir != "" && strings.HasPrefix(filePath, item.DownloadDir) {
			return item.ID
		}
	}
	return ""
}

func (s *Service) findMappingByClientAndDownload(mappings []*sqlc.DownloadMapping, clientID int64, downloadID string) (*DownloadMapping, error) {
	for _, m := range mappings {
		if m.ClientID == clientID && m.DownloadID == downloadID {
			return s.convertMapping(m), nil
		}
	}
	return nil, ErrNoMatch
}

// GetProcessingCount returns the number of files currently being processed.
func (s *Service) GetProcessingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.processing)
}

// CheckAndProcessCompletedDownloads checks for completed downloads and triggers imports.
// This is called by the scheduler and emits WebSocket events.
func (s *Service) CheckAndProcessCompletedDownloads(ctx context.Context) error {
	completed, err := s.downloader.CheckForCompletedDownloads(ctx)
	if err != nil {
		s.logger.Debug().Err(err).Msg("CheckForCompletedDownloads returned error")
		return err
	}

	s.logger.Debug().Int("count", len(completed)).Msg("CheckForCompletedDownloads found downloads")

	for i := range completed {
		s.processCompletedEntry(ctx, &completed[i])
	}

	return nil
}

func (s *Service) processCompletedEntry(ctx context.Context, cd *downloader.CompletedDownload) {
	if cd.NextImportRetryAt != nil && time.Now().Before(*cd.NextImportRetryAt) {
		s.logger.Debug().
			Int64("clientId", cd.ClientID).
			Str("downloadId", cd.DownloadID).
			Int64("attempt", cd.ImportAttempts).
			Time("retryAt", *cd.NextImportRetryAt).
			Msg("Skipping import attempt, backoff not elapsed")
		return
	}

	s.broadcastDownloadCompleted(cd)

	mapping := s.completedDownloadToMapping(cd)

	if err := s.ProcessCompletedDownload(ctx, mapping); err != nil {
		s.handleCompletedImportFailure(ctx, cd, err)
		return
	}

	if err := s.downloader.DeleteDownloadMapping(ctx, cd.ClientID, cd.DownloadID); err != nil {
		s.logger.Warn().Err(err).
			Int64("clientId", cd.ClientID).
			Str("downloadId", cd.DownloadID).
			Msg("Failed to delete download mapping after processing")
	} else {
		s.logger.Debug().
			Int64("clientId", cd.ClientID).
			Str("downloadId", cd.DownloadID).
			Msg("Deleted download mapping after processing completed download")
	}
}

func (s *Service) handleCompletedImportFailure(ctx context.Context, cd *downloader.CompletedDownload, importErr error) {
	attempts, err := s.downloader.IncrementMappingImportAttempts(ctx, cd.ClientID, cd.DownloadID, importErr.Error())
	if err != nil {
		s.logger.Warn().Err(err).
			Int64("clientId", cd.ClientID).
			Str("downloadId", cd.DownloadID).
			Msg("Failed to increment import attempt counter")
		return
	}

	if attempts >= downloader.MaxCompletionRetries {
		s.logger.Error().Err(importErr).
			Int64("clientId", cd.ClientID).
			Str("downloadId", cd.DownloadID).
			Int64("attempts", attempts).
			Msg("Import permanently failed after max retries, cleaning up mapping")

		s.markCompletionMediaFailed(ctx, cd, importErr)

		_ = s.downloader.DeleteDownloadMapping(ctx, cd.ClientID, cd.DownloadID)

		if s.health != nil {
			s.health.SetWarningStr("import",
				cd.DownloadID,
				fmt.Sprintf("Import failed after %d attempts: %s", attempts, importErr.Error()))
		}
	} else {
		backoff := downloader.ImportRetryBackoff(attempts)
		s.logger.Warn().Err(importErr).
			Int64("clientId", cd.ClientID).
			Str("downloadId", cd.DownloadID).
			Int64("attempt", attempts).
			Int64("maxAttempts", downloader.MaxCompletionRetries).
			Dur("retryIn", backoff).
			Msg("Import failed, will retry after backoff")
	}
}

func (s *Service) markCompletionMediaFailed(ctx context.Context, cd *downloader.CompletedDownload, importErr error) {
	statusMsg := sql.NullString{String: importErr.Error(), Valid: true}

	switch {
	case cd.MovieID != nil:
		s.setMovieStatusFailed(ctx, *cd.MovieID, statusMsg)
	case cd.EpisodeID != nil:
		mapping := &DownloadMapping{
			EpisodeID: cd.EpisodeID,
			SeriesID:  cd.SeriesID,
		}
		s.setEpisodeStatusFailed(ctx, mapping, statusMsg)
	case cd.SeriesID != nil && cd.IsSeasonPack:
		s.setSeasonStatusFailed(ctx, cd, statusMsg)
	}
}

func (s *Service) setSeasonStatusFailed(ctx context.Context, cd *downloader.CompletedDownload, statusMsg sql.NullString) {
	if cd.SeriesID == nil || cd.SeasonNumber == nil {
		return
	}
	episodes, err := s.tv.ListEpisodes(ctx, *cd.SeriesID, cd.SeasonNumber)
	if err != nil {
		s.logger.Warn().Err(err).
			Int64("seriesId", *cd.SeriesID).
			Int("season", *cd.SeasonNumber).
			Msg("Failed to get episodes for season pack failure marking")
		return
	}
	for _, ep := range episodes {
		_ = s.queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
			Status:           "failed",
			ActiveDownloadID: sql.NullString{},
			StatusMessage:    statusMsg,
			ID:               ep.ID,
		})
	}
	if s.hub != nil {
		s.hub.Broadcast("series:updated", map[string]any{"id": *cd.SeriesID})
	}
}

func (s *Service) broadcastDownloadCompleted(cd *downloader.CompletedDownload) {
	if s.hub == nil {
		return
	}

	mediaType := mediaTypeMovie
	if cd.SeriesID != nil {
		mediaType = mediaTypeEpisode
	}
	s.hub.Broadcast("download:completed", map[string]any{
		"clientId":     cd.ClientID,
		"downloadId":   cd.DownloadID,
		"downloadPath": cd.DownloadPath,
		"mediaType":    mediaType,
		"movieId":      cd.MovieID,
		"seriesId":     cd.SeriesID,
		"episodeId":    cd.EpisodeID,
		"isSeasonPack": cd.IsSeasonPack,
	})
}

func (s *Service) completedDownloadToMapping(cd *downloader.CompletedDownload) *DownloadMapping {
	mapping := &DownloadMapping{
		ID:               cd.MappingID,
		DownloadClientID: cd.ClientID,
		DownloadID:       cd.DownloadID,
		MovieID:          cd.MovieID,
		SeriesID:         cd.SeriesID,
		EpisodeID:        cd.EpisodeID,
		SeasonNumber:     cd.SeasonNumber,
		IsSeasonPack:     cd.IsSeasonPack,
		IsCompleteSeries: cd.IsCompleteSeries,
		TargetSlotID:     cd.TargetSlotID,
	}
	mapping.MediaType = determineMappingMediaType(mapping)
	return mapping
}

// populateRootFolder determines and sets the root folder for a library match.
// Req 22.2.1-22.2.3: In multi-version mode, check target slot's root folder first.
func (s *Service) populateRootFolder(ctx context.Context, match *LibraryMatch, targetSlotID *int64) error {
	if path, ok := s.trySlotRootFolder(ctx, targetSlotID, match.MediaType); ok {
		match.RootFolder = path
		return nil
	}

	rootFolderID, err := s.getMediaRootFolderID(ctx, match)
	if err != nil {
		return err
	}

	rf, err := s.rootfolder.Get(ctx, rootFolderID)
	if err != nil {
		return err
	}

	match.RootFolder = rf.Path
	return nil
}

func (s *Service) trySlotRootFolder(ctx context.Context, targetSlotID *int64, mediaType string) (string, bool) {
	if s.slots == nil || targetSlotID == nil || !s.slots.IsMultiVersionEnabled(ctx) {
		return "", false
	}
	path, err := s.slots.GetRootFolderForSlot(ctx, *targetSlotID, mediaType)
	if err != nil || path == "" {
		return "", false
	}
	return path, true
}

func (s *Service) getMediaRootFolderID(ctx context.Context, match *LibraryMatch) (int64, error) {
	switch {
	case match.MediaType == mediaTypeMovie && match.MovieID != nil:
		movie, err := s.movies.Get(ctx, *match.MovieID)
		if err != nil {
			return 0, err
		}
		return movie.RootFolderID, nil
	case match.MediaType == mediaTypeEpisode && match.SeriesID != nil:
		series, err := s.tv.GetSeries(ctx, *match.SeriesID)
		if err != nil {
			return 0, err
		}
		return series.RootFolderID, nil
	default:
		return 0, errors.New("media has no root folder assigned")
	}
}

// checkForExistingFile checks if there's an existing file and performs quality comparison.
// In multi-version mode, this is skipped entirely — slot evaluation handles per-slot upgrades.
// Returns ErrNotAnUpgrade if a file exists but the candidate is not a quality upgrade.
func (s *Service) checkForExistingFile(ctx context.Context, match *LibraryMatch, sourcePath string) error {
	if s.slots != nil && s.slots.IsMultiVersionEnabled(ctx) {
		return nil
	}

	existingFile, qualityProfileID, err := s.fetchExistingFileInfo(ctx, match)
	if err != nil {
		return err
	}
	if existingFile == nil {
		return nil
	}

	match.ExistingFile = existingFile.path
	match.ExistingFileID = &existingFile.id
	match.QualityProfileID = qualityProfileID

	return s.evaluateUpgradeEligibility(ctx, match, existingFile, qualityProfileID, sourcePath)
}

type existingFileInfo struct {
	id        int64
	path      string
	qualityID sql.NullInt64
}

func (s *Service) fetchExistingFileInfo(ctx context.Context, match *LibraryMatch) (*existingFileInfo, int64, error) {
	switch {
	case match.MediaType == mediaTypeMovie && match.MovieID != nil:
		return s.fetchMovieFileInfo(ctx, *match.MovieID)
	case match.MediaType == mediaTypeEpisode && match.EpisodeID != nil:
		return s.fetchEpisodeFileInfo(ctx, match)
	default:
		return nil, 0, nil
	}
}

func (s *Service) fetchMovieFileInfo(ctx context.Context, movieID int64) (*existingFileInfo, int64, error) {
	file, err := s.movies.GetPrimaryFile(ctx, movieID)
	if err != nil {
		return nil, 0, nil //nolint:nilerr // no file found is not an error
	}
	if file.Path == "" {
		return nil, 0, nil
	}

	movie, err := s.movies.Get(ctx, movieID)
	if err != nil {
		return nil, 0, err
	}

	info := &existingFileInfo{
		id:   file.ID,
		path: file.Path,
	}

	if dbFile, dbErr := s.queries.GetMovieFile(ctx, file.ID); dbErr == nil {
		info.qualityID = dbFile.QualityID
	}

	return info, movie.QualityProfileID, nil
}

func (s *Service) fetchEpisodeFileInfo(ctx context.Context, match *LibraryMatch) (*existingFileInfo, int64, error) {
	file, err := s.tv.GetEpisodeFile(ctx, *match.EpisodeID)
	if err != nil {
		return nil, 0, nil //nolint:nilerr // no file found is not an error
	}
	if file.Path == "" {
		return nil, 0, nil
	}

	info := &existingFileInfo{
		id:   file.ID,
		path: file.Path,
	}

	if dbFile, dbErr := s.queries.GetEpisodeFile(ctx, file.ID); dbErr == nil {
		info.qualityID = dbFile.QualityID
	}

	var qualityProfileID int64
	if match.SeriesID != nil {
		series, err := s.tv.GetSeries(ctx, *match.SeriesID)
		if err != nil {
			return nil, 0, err
		}
		qualityProfileID = series.QualityProfileID
	}

	return info, qualityProfileID, nil
}

func (s *Service) evaluateUpgradeEligibility(ctx context.Context, match *LibraryMatch, existingFile *existingFileInfo, qualityProfileID int64, sourcePath string) error {
	if s.quality == nil || qualityProfileID == 0 {
		match.IsUpgrade = true
		return nil
	}

	profile, err := s.quality.Get(ctx, qualityProfileID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("profileId", qualityProfileID).Msg("Failed to load quality profile, assuming upgrade")
		match.IsUpgrade = true
		return nil
	}

	if !profile.UpgradesEnabled {
		match.IsUpgrade = false
		return ErrNotAnUpgrade
	}

	return s.compareQualityForUpgrade(match, existingFile, profile, sourcePath)
}

func (s *Service) compareQualityForUpgrade(match *LibraryMatch, existingFile *existingFileInfo, profile *quality.Profile, sourcePath string) error {
	parsed := scanner.ParsePath(sourcePath)
	candidateMatch := quality.MatchQuality(parsed.Quality, parsed.Source, profile)

	if candidateMatch.Matches {
		match.CandidateQualityID = candidateMatch.MatchedQualityID
	}

	existingQualityID := 0
	if existingFile.qualityID.Valid {
		existingQualityID = int(existingFile.qualityID.Int64)
	}
	match.ExistingQualityID = existingQualityID

	if existingQualityID == 0 {
		match.IsUpgrade = true
		return nil
	}

	if match.CandidateQualityID == 0 {
		match.IsUpgrade = false
		return ErrNotAnUpgrade
	}

	if profile.IsAtOrAboveCutoff(existingQualityID) {
		match.IsUpgrade = false
		return ErrNotAnUpgrade
	}

	if profile.IsUpgrade(existingQualityID, match.CandidateQualityID) {
		match.IsUpgrade = true
		return nil
	}

	match.IsUpgrade = false
	return ErrNotAnUpgrade
}

// getSlotFileID returns the file ID currently assigned to a slot for a given media item.
func (s *Service) getSlotFileID(ctx context.Context, match *LibraryMatch, slotID int64) *int64 {
	if s.slots == nil {
		return nil
	}
	var mediaID int64
	switch {
	case match.MediaType == mediaTypeMovie && match.MovieID != nil:
		mediaID = *match.MovieID
	case match.MediaType == mediaTypeEpisode && match.EpisodeID != nil:
		mediaID = *match.EpisodeID
	default:
		return nil
	}
	fileID, err := s.slots.GetSlotFileID(ctx, match.MediaType, mediaID, slotID)
	if err != nil || fileID == 0 {
		return nil
	}
	return &fileID
}

// getFilePath returns the file path for a given file ID and media type.
func (s *Service) getFilePath(ctx context.Context, mediaType string, fileID int64) string {
	switch mediaType {
	case "movie":
		f, err := s.queries.GetMovieFile(ctx, fileID)
		if err == nil {
			return f.Path
		}
	case "episode":
		f, err := s.queries.GetEpisodeFile(ctx, fileID)
		if err == nil {
			return f.Path
		}
	}
	return ""
}

// recordImportDecision records a rejection decision for a source file.
func (s *Service) recordImportDecision(ctx context.Context, sourcePath, decision string, match *LibraryMatch) {
	var mediaID int64
	mediaType := match.MediaType
	switch {
	case match.MediaType == mediaTypeMovie && match.MovieID != nil:
		mediaID = *match.MovieID
	case match.MediaType == mediaTypeEpisode && match.EpisodeID != nil:
		mediaID = *match.EpisodeID
	default:
		return
	}

	params := sqlc.UpsertImportDecisionParams{
		SourcePath: sourcePath,
		Decision:   decision,
		MediaType:  mediaType,
		MediaID:    mediaID,
	}

	if match.CandidateQualityID > 0 {
		params.CandidateQualityID = sql.NullInt64{Int64: int64(match.CandidateQualityID), Valid: true}
	}
	if match.ExistingQualityID > 0 {
		params.ExistingQualityID = sql.NullInt64{Int64: int64(match.ExistingQualityID), Valid: true}
	}
	if match.ExistingFileID != nil {
		params.ExistingFileID = sql.NullInt64{Int64: *match.ExistingFileID, Valid: true}
	}
	if match.QualityProfileID > 0 {
		params.QualityProfileID = sql.NullInt64{Int64: match.QualityProfileID, Valid: true}
	}

	if _, err := s.queries.UpsertImportDecision(ctx, params); err != nil {
		s.logger.Warn().Err(err).Str("path", sourcePath).Msg("Failed to record import decision")
	}
}

// ClearDecisionsForProfile clears import decisions that reference a specific quality profile.
// Implements quality.ImportDecisionCleaner.
func (s *Service) ClearDecisionsForProfile(ctx context.Context, profileID int64) error {
	return s.queries.DeleteImportDecisionsByProfile(ctx, sql.NullInt64{Int64: profileID, Valid: true})
}

// dispatchImportNotification sends a download or upgrade notification after successful import.
func (s *Service) dispatchImportNotification(ctx context.Context, result *ImportResult) {
	if s.notifier == nil || result.Match == nil {
		return
	}

	if result.IsUpgrade {
		s.dispatchUpgradeNotification(ctx, result)
	} else {
		s.dispatchImportNotificationEvent(ctx, result)
	}
}

func (s *Service) dispatchUpgradeNotification(ctx context.Context, result *ImportResult) {
	event := UpgradeNotificationEvent{
		MediaType:   result.Match.MediaType,
		OldQuality:  qualityNameFromID(result.Match.ExistingQualityID),
		OldPath:     result.PreviousFile,
		NewPath:     result.DestinationPath,
		NewQuality:  qualityNameFromID(result.Match.CandidateQualityID),
		ReleaseName: filepath.Base(result.SourcePath),
		SlotID:      result.AssignedSlotID,
	}

	s.populateUpgradeEventDetails(ctx, &event, result.Match)
	s.notifier.DispatchUpgrade(ctx, &event)
}

func (s *Service) populateUpgradeEventDetails(ctx context.Context, event *UpgradeNotificationEvent, match *LibraryMatch) {
	if match.MediaType == mediaTypeMovie && match.MovieID != nil {
		event.MovieID = match.MovieID
		if movie, err := s.movies.Get(ctx, *match.MovieID); err == nil {
			event.MovieTitle = movie.Title
			event.MovieYear = movie.Year
		}
	} else if match.MediaType == mediaTypeEpisode && match.EpisodeID != nil {
		s.populateUpgradeEpisodeDetails(ctx, event, match)
	}
}

func (s *Service) populateUpgradeEpisodeDetails(ctx context.Context, event *UpgradeNotificationEvent, match *LibraryMatch) {
	event.EpisodeID = match.EpisodeID
	event.SeriesID = match.SeriesID
	if match.SeasonNum != nil {
		event.SeasonNumber = *match.SeasonNum
	}

	if episode, err := s.tv.GetEpisode(ctx, *match.EpisodeID); err == nil {
		event.EpisodeNumber = episode.EpisodeNumber
		event.EpisodeTitle = episode.Title
	}

	if match.SeriesID != nil {
		if series, err := s.tv.GetSeries(ctx, *match.SeriesID); err == nil {
			event.SeriesTitle = series.Title
		}
	}
}

func (s *Service) dispatchImportNotificationEvent(ctx context.Context, result *ImportResult) {
	event := ImportNotificationEvent{
		MediaType:       result.Match.MediaType,
		Quality:         qualityNameFromID(result.Match.CandidateQualityID),
		SourcePath:      result.SourcePath,
		DestinationPath: result.DestinationPath,
		ReleaseName:     filepath.Base(result.SourcePath),
		SlotID:          result.AssignedSlotID,
	}

	s.populateImportEventDetails(ctx, &event, result.Match)
	s.notifier.DispatchImport(ctx, &event)
}

func (s *Service) populateImportEventDetails(ctx context.Context, event *ImportNotificationEvent, match *LibraryMatch) {
	if match.MediaType == mediaTypeMovie && match.MovieID != nil {
		event.MovieID = match.MovieID
		if movie, err := s.movies.Get(ctx, *match.MovieID); err == nil {
			event.MovieTitle = movie.Title
			event.MovieYear = movie.Year
		}
	} else if match.MediaType == mediaTypeEpisode && match.EpisodeID != nil {
		s.populateImportEpisodeDetails(ctx, event, match)
	}
}

func (s *Service) populateImportEpisodeDetails(ctx context.Context, event *ImportNotificationEvent, match *LibraryMatch) {
	event.EpisodeID = match.EpisodeID
	event.SeriesID = match.SeriesID
	if match.SeasonNum != nil {
		event.SeasonNumber = *match.SeasonNum
	}

	if episode, err := s.tv.GetEpisode(ctx, *match.EpisodeID); err == nil {
		event.EpisodeNumber = episode.EpisodeNumber
		event.EpisodeTitle = episode.Title
	}

	if match.SeriesID != nil {
		if series, err := s.tv.GetSeries(ctx, *match.SeriesID); err == nil {
			event.SeriesTitle = series.Title
		}
	}
}

// qualityNameFromID returns the quality name for a given ID, or empty string if unknown.
func qualityNameFromID(id int) string {
	if q, ok := quality.GetQualityByID(id); ok {
		return q.Name
	}
	return ""
}
