package importer

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
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
	Create(ctx context.Context, input HistoryInput) error
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
	db         *sql.DB
	queries    *sqlc.Queries
	downloader *downloader.Service
	movies     *movies.Service
	tv         *tv.Service
	rootfolder *rootfolder.Service
	organizer  *organizer.Service
	renamer    *renamer.Resolver
	mediainfo  *mediainfo.Service
	slots      *slots.Service
	health     HealthService
	history    HistoryService
	hub        *websocket.Hub
	logger     zerolog.Logger
	config     Config

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
	MediaType        string // "movie" or "episode"
	MovieID          *int64
	SeriesID         *int64
	SeasonNumber     *int
	EpisodeID        *int64
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
	MediaType   string  // "movie" or "episode"
	MovieID     *int64  // Set for movies
	SeriesID    *int64  // Set for episodes
	SeasonNum   *int    // Set for episodes
	EpisodeID   *int64  // Set for episodes
	EpisodeIDs  []int64 // Set for multi-episode files
	Confidence  float64 // Match confidence 0.0 - 1.0
	Source      string  // "queue", "parse", "manual"
	RootFolder  string  // Root folder path
	IsUpgrade      bool    // Whether this replaces an existing file
	ExistingFile   string  // Path to existing file if upgrade
	ExistingFileID *int64  // ID of existing file for database cleanup
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
	RequiresSlotSelection bool              // True if user must select a slot
	SlotAssignments       []SlotAssignment  // Available slot options when selection required
	RecommendedSlotID     *int64            // Recommended slot based on best match
	AssignedSlotID        *int64            // Slot file was assigned to after import
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
	downloader *downloader.Service,
	moviesSvc *movies.Service,
	tvSvc *tv.Service,
	rootfolderSvc *rootfolder.Service,
	organizerSvc *organizer.Service,
	mediainfoSvc *mediainfo.Service,
	hub *websocket.Hub,
	config Config,
	logger zerolog.Logger,
) *Service {
	s := &Service{
		db:          db,
		queries:     sqlc.New(db),
		downloader:  downloader,
		movies:      moviesSvc,
		tv:          tvSvc,
		rootfolder:  rootfolderSvc,
		organizer:   organizerSvc,
		mediainfo:   mediainfoSvc,
		hub:         hub,
		logger:      logger.With().Str("component", "import").Logger(),
		config:      config,
		importQueue: make(chan ImportJob, 100),
		processing:  make(map[string]bool),
		shutdown:    make(chan struct{}),
	}

	// Initialize renamer with default settings
	// Settings will be loaded from database on first use
	s.renamer = renamer.NewResolver(renamer.DefaultSettings())

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

// SetSlotsService sets the slots service for multi-version support.
func (s *Service) SetSlotsService(ss *slots.Service) {
	s.slots = ss
}

// UpdateRenamerSettings updates the renamer with new settings.
func (s *Service) UpdateRenamerSettings(settings renamer.Settings) {
	s.renamer = renamer.NewResolver(settings)
}

// Start starts the import worker(s).
func (s *Service) Start(ctx context.Context) {
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
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
func (s *Service) worker(ctx context.Context, id int) {
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

	// Apply import delay if configured for this client
	if job.DownloadMapping != nil && job.DownloadMapping.DownloadClientID > 0 {
		if err := s.applyImportDelay(ctx, job.DownloadMapping.DownloadClientID); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to apply import delay")
		}
	}

	result := s.processWithRetry(ctx, job)

	if result.Success {
		s.logger.Info().
			Str("source", result.SourcePath).
			Str("destination", result.DestinationPath).
			Str("linkMode", string(result.LinkMode)).
			Bool("upgrade", result.IsUpgrade).
			Msg("Import completed successfully")

		// Broadcast success event
		if s.hub != nil {
			s.hub.Broadcast("import:completed", map[string]any{
				"source":      result.SourcePath,
				"destination": result.DestinationPath,
				"mediaType":   result.Match.MediaType,
				"isUpgrade":   result.IsUpgrade,
			})
		}
	} else {
		s.logger.Error().
			Err(result.Error).
			Str("path", job.SourcePath).
			Msg("Import failed")

		// Register health warning
		if s.health != nil {
			s.health.SetWarningStr("import", job.SourcePath, result.Error.Error())
		}

		// Broadcast failure event
		if s.hub != nil {
			s.hub.Broadcast("import:failed", map[string]any{
				"source": result.SourcePath,
				"error":  result.Error.Error(),
			})
		}
	}
}

// GetQueueLength returns the current import queue length.
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
	// Get all active download mappings
	mappings, err := s.queries.ListActiveDownloadMappings(ctx)
	if err != nil {
		return nil, err
	}

	// Get download info from client
	client, err := s.downloader.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	items, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	// Try to match the file path to a download
	for _, item := range items {
		if item.DownloadDir == "" {
			continue
		}

		// Check if file is under this download's directory
		if strings.HasPrefix(filePath, item.DownloadDir) {
			// Find matching mapping
			for _, m := range mappings {
				if m.ClientID == clientID && m.DownloadID == item.ID {
					return s.convertMapping(m), nil
				}
			}
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
		return err
	}

	for _, cd := range completed {
		// Emit download:completed event
		if s.hub != nil {
			mediaType := "movie"
			if cd.SeriesID != nil {
				mediaType = "episode"
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

		// Create download mapping for processing
		mapping := &DownloadMapping{
			ID:               cd.MappingID,
			DownloadClientID: cd.ClientID,
			DownloadID:       cd.DownloadID,
			MovieID:          cd.MovieID,
			SeriesID:         cd.SeriesID,
			EpisodeID:        cd.EpisodeID,
		}
		if cd.SeriesID != nil {
			mapping.MediaType = "episode"
		} else {
			mapping.MediaType = "movie"
		}
		if cd.SeasonNumber != nil {
			mapping.SeasonNumber = cd.SeasonNumber
		}

		// Process the completed download
		if err := s.ProcessCompletedDownload(ctx, mapping); err != nil {
			s.logger.Warn().Err(err).
				Int64("clientId", cd.ClientID).
				Str("downloadId", cd.DownloadID).
				Msg("Failed to process completed download")
		}
	}

	return nil
}

// populateRootFolder determines and sets the root folder for a library match.
// Req 22.2.1-22.2.3: In multi-version mode, check target slot's root folder first.
func (s *Service) populateRootFolder(ctx context.Context, match *LibraryMatch, targetSlotID *int64) error {
	// In multi-version mode with a target slot, try slot's root folder first
	if s.slots != nil && targetSlotID != nil && s.slots.IsMultiVersionEnabled(ctx) {
		rootFolderPath, err := s.slots.GetRootFolderForSlot(ctx, *targetSlotID, match.MediaType)
		if err == nil && rootFolderPath != "" {
			match.RootFolder = rootFolderPath
			return nil
		}
		// Fall through to media item root folder if slot has no root folder set
	}

	// Existing logic: get root folder from movie/series
	var rootFolderID int64

	if match.MediaType == "movie" && match.MovieID != nil {
		movie, err := s.movies.Get(ctx, *match.MovieID)
		if err != nil {
			return err
		}
		rootFolderID = movie.RootFolderID
	} else if match.MediaType == "episode" && match.SeriesID != nil {
		series, err := s.tv.GetSeries(ctx, *match.SeriesID)
		if err != nil {
			return err
		}
		rootFolderID = series.RootFolderID
	}

	if rootFolderID == 0 {
		return errors.New("media has no root folder assigned")
	}

	// Look up the root folder path
	rf, err := s.rootfolder.Get(ctx, rootFolderID)
	if err != nil {
		return err
	}

	match.RootFolder = rf.Path
	return nil
}

// checkForExistingFile checks if there's an existing file that would be upgraded.
func (s *Service) checkForExistingFile(ctx context.Context, match *LibraryMatch) error {
	if match.MediaType == "movie" && match.MovieID != nil {
		file, err := s.movies.GetPrimaryFile(ctx, *match.MovieID)
		if err == nil && file != nil && file.Path != "" {
			match.IsUpgrade = true
			match.ExistingFile = file.Path
			match.ExistingFileID = &file.ID
		}
	} else if match.MediaType == "episode" && match.EpisodeID != nil {
		file, err := s.tv.GetEpisodeFile(ctx, *match.EpisodeID)
		if err == nil && file != nil && file.Path != "" {
			match.IsUpgrade = true
			match.ExistingFile = file.Path
			match.ExistingFileID = &file.ID
		}
	}
	return nil
}
