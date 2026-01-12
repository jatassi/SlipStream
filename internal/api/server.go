package api

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/api/handlers"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/history"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/filesystem"
	"github.com/slipstream/slipstream/internal/health"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/status"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/missing"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"
	"github.com/slipstream/slipstream/internal/scheduler"
	"github.com/slipstream/slipstream/internal/scheduler/tasks"
	"github.com/slipstream/slipstream/internal/websocket"
)

// Server handles HTTP requests for the SlipStream API.
type Server struct {
	echo   *echo.Echo
	db     *sql.DB
	hub    *websocket.Hub
	logger zerolog.Logger
	cfg    *config.Config

	// Services
	scannerService        *scanner.Service
	movieService          *movies.Service
	tvService             *tv.Service
	qualityService        *quality.Service
	rootFolderService     *rootfolder.Service
	metadataService       *metadata.Service
	artworkDownloader     *metadata.ArtworkDownloader
	filesystemService     *filesystem.Service
	storageService        *filesystem.StorageService
	libraryManagerService *librarymanager.Service
	progressManager       *progress.Manager
	downloaderService     *downloader.Service
	indexerService        *indexer.Service
	searchService         *search.Service
	statusService         *status.Service
	rateLimiter           *ratelimit.Limiter
	grabService           *grab.Service
	defaultsService       *defaults.Service
	calendarService       *calendar.Service
	scheduler             *scheduler.Scheduler
	availabilityService   *availability.Service
	missingService        *missing.Service
	autosearchService     *autosearch.Service
	scheduledSearcher     *autosearch.ScheduledSearcher
	preferencesService    *preferences.Service
	historyService        *history.Service
	healthService         *health.Service
	importService         *importer.Service
	organizerService      *organizer.Service
	mediainfoService      *mediainfo.Service
}

// NewServer creates a new API server instance.
func NewServer(db *sql.DB, hub *websocket.Hub, cfg *config.Config, logger zerolog.Logger) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	s := &Server{
		echo:   e,
		db:     db,
		hub:    hub,
		logger: logger,
		cfg:    cfg,
	}

	// Initialize health service first (no dependencies)
	s.healthService = health.NewService(logger)
	s.healthService.SetBroadcaster(hub)

	// Initialize services
	s.scannerService = scanner.NewService(logger)
	s.movieService = movies.NewService(db, hub, logger)
	s.tvService = tv.NewService(db, hub, logger)
	s.qualityService = quality.NewService(db, logger)
	s.defaultsService = defaults.NewService(sqlc.New(db))
	s.rootFolderService = rootfolder.NewService(db, logger, s.defaultsService)
	s.rootFolderService.SetHealthService(s.healthService)

	// Initialize metadata service and artwork downloader
	s.metadataService = metadata.NewService(cfg.Metadata, logger)
	s.metadataService.SetHealthService(s.healthService)
	s.artworkDownloader = metadata.NewArtworkDownloader(metadata.DefaultArtworkConfig(), logger)
	s.artworkDownloader.SetBroadcaster(hub)

	// Initialize filesystem service
	s.filesystemService = filesystem.NewService(logger)

	// Initialize storage service (combines filesystem and root folder data)
	s.storageService = filesystem.NewStorageService(s.filesystemService, s.rootFolderService, logger)

	// Initialize downloader service
	s.downloaderService = downloader.NewService(db, logger)
	s.downloaderService.SetDeveloperMode(cfg.DeveloperMode)
	s.downloaderService.SetHealthService(s.healthService)

	// Initialize Cardigann manager for indexer definitions
	// Note: Definitions are fetched lazily when the user opens the Add Indexer dialog
	cardigannManager, err := cardigann.NewManager(cardigann.DefaultManagerConfig(), logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize Cardigann manager")
	}

	// Initialize indexer service
	s.indexerService = indexer.NewService(db, cardigannManager, logger)
	s.indexerService.SetHealthService(s.healthService)

	// Initialize indexer status service
	s.statusService = status.NewService(db, logger)
	s.statusService.SetHealthService(s.healthService)

	// Initialize rate limiter
	s.rateLimiter = ratelimit.NewLimiter(db, ratelimit.DefaultConfig(), logger)

	// Initialize search service with status, rate limiting, and WebSocket events
	s.searchService = search.NewService(s.indexerService, logger)
	s.searchService.SetStatusService(s.statusService)
	s.searchService.SetRateLimiter(s.rateLimiter)
	s.searchService.SetBroadcaster(hub)

	// Initialize grab service with status, rate limiting, and WebSocket events
	s.grabService = grab.NewService(db, s.downloaderService, logger)
	s.grabService.SetIndexerService(s.indexerService)
	s.grabService.SetStatusService(s.statusService)
	s.grabService.SetRateLimiter(s.rateLimiter)
	s.grabService.SetBroadcaster(hub)

	// Initialize defaults service
	s.defaultsService = defaults.NewService(sqlc.New(db))

	// Initialize calendar service
	s.calendarService = calendar.NewService(db, logger)

	// Initialize availability service
	s.availabilityService = availability.NewService(db, logger)

	// Initialize missing service
	s.missingService = missing.NewService(db, logger)

	// Initialize preferences service
	s.preferencesService = preferences.NewService(sqlc.New(db))

	// Initialize history service
	s.historyService = history.NewService(db, logger)

	// Initialize organizer service (for file operations)
	s.organizerService = organizer.NewService(organizer.DefaultNamingConfig(), logger)

	// Initialize mediainfo service (for probing media files)
	s.mediainfoService = mediainfo.NewService(mediainfo.DefaultConfig(), logger)

	// Initialize import service (for processing completed downloads)
	s.importService = importer.NewService(
		db,
		s.downloaderService,
		s.movieService,
		s.tvService,
		s.rootFolderService,
		s.organizerService,
		s.mediainfoService,
		hub,
		importer.DefaultConfig(),
		logger,
	)
	s.importService.SetHealthService(s.healthService)
	s.importService.SetHistoryService(&importHistoryAdapter{s.historyService})

	// Initialize autosearch service
	s.autosearchService = autosearch.NewService(db, s.searchService, s.grabService, s.qualityService, logger)
	s.autosearchService.SetBroadcaster(hub)
	s.autosearchService.SetHistoryService(s.historyService)

	// Load saved autosearch settings into config before creating scheduler
	if err := autosearch.LoadSettingsIntoConfig(context.Background(), sqlc.New(db), &cfg.AutoSearch); err != nil {
		logger.Warn().Err(err).Msg("Failed to load autosearch settings, using defaults")
	}

	// Initialize scheduled searcher for automatic background searches
	s.scheduledSearcher = autosearch.NewScheduledSearcher(s.autosearchService, &cfg.AutoSearch, logger)

	// Initialize scheduler
	sched, err := scheduler.New(logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize scheduler")
	} else {
		s.scheduler = sched
		// Register availability refresh task
		if err := tasks.RegisterAvailabilityTask(s.scheduler, s.availabilityService); err != nil {
			logger.Error().Err(err).Msg("Failed to register availability task")
		}
		// Register automatic search task
		if err := tasks.RegisterAutoSearchTask(s.scheduler, s.scheduledSearcher, &cfg.AutoSearch); err != nil {
			logger.Error().Err(err).Msg("Failed to register autosearch task")
		}
	}

	// Initialize progress manager for tracking activities
	s.progressManager = progress.NewManager(hub, logger)

	// Initialize library manager service (orchestrates scanning and file matching)
	s.libraryManagerService = librarymanager.NewService(
		db,
		s.scannerService,
		s.movieService,
		s.tvService,
		s.metadataService,
		s.artworkDownloader,
		s.rootFolderService,
		s.qualityService,
		s.progressManager,
		logger,
	)
	// Wire up optional services for search-on-add functionality
	s.libraryManagerService.SetAutosearchService(s.autosearchService)
	s.libraryManagerService.SetPreferencesService(s.preferencesService)

	// Register library-dependent scheduled tasks (after library manager is initialized)
	if s.scheduler != nil {
		// Register library scan task (runs daily at 11:30 PM)
		if err := tasks.RegisterLibraryScanTask(s.scheduler, s.libraryManagerService, s.rootFolderService, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register library scan task")
		}
		// Register metadata refresh task (runs daily at 11:30 PM)
		if err := tasks.RegisterMetadataRefreshTask(s.scheduler, s.libraryManagerService, s.movieService, s.tvService, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register metadata refresh task")
		}
		// Register download client health check task
		if err := tasks.RegisterDownloadClientHealthTask(s.scheduler, s.downloaderService, s.healthService, &cfg.Health, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register download client health task")
		}
		// Register indexer health check task
		if err := tasks.RegisterIndexerHealthTask(s.scheduler, s.indexerService, s.healthService, &cfg.Health, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register indexer health task")
		}
		// Register storage health check task
		storageAdapter := health.NewStorageServiceAdapter(s.storageService)
		storageChecker := health.NewStorageChecker(s.healthService, storageAdapter, &cfg.Health, logger)
		if err := tasks.RegisterStorageHealthTask(s.scheduler, storageChecker, &cfg.Health, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register storage health task")
		}
		// Register import scan task (scans for completed downloads ready to import)
		if err := tasks.RegisterImportScanTask(s.scheduler, s.importService, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register import scan task")
		}
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures Echo middleware.
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.echo.Use(middleware.Recover())

	// Request ID
	s.echo.Use(middleware.RequestID())

	// CORS
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Request logging
	s.echo.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogLatency:  true,
		LogMethod:   true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error != nil {
				s.logger.Error().
					Str("method", v.Method).
					Str("uri", v.URI).
					Int("status", v.Status).
					Dur("latency", v.Latency).
					Err(v.Error).
					Msg("request error")
			} else {
				s.logger.Info().
					Str("method", v.Method).
					Str("uri", v.URI).
					Int("status", v.Status).
					Dur("latency", v.Latency).
					Msg("request")
			}
			return nil
		},
	}))

	// Gzip compression
	s.echo.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
		Skipper: func(c echo.Context) bool {
			// Skip compression for WebSocket
			return c.Request().Header.Get("Upgrade") == "websocket"
		},
	}))
}

// setupRoutes configures API routes.
func (s *Server) setupRoutes() {
	// Health check
	s.echo.GET("/health", s.healthCheck)

	// API v1 group
	api := s.echo.Group("/api/v1")

	// System routes
	api.GET("/status", s.getStatus)

	// System health routes (under /api/v1/system/health to avoid conflict with /health endpoint)
	healthHandlers := health.NewHandlers(s.healthService, &health.TestFunctions{
		TestDownloadClient: func(ctx context.Context, id int64) (bool, string) {
			result, err := s.downloaderService.Test(ctx, id)
			if err != nil {
				return false, err.Error()
			}
			return result.Success, result.Message
		},
		TestIndexer: func(ctx context.Context, id int64) (bool, string) {
			result, err := s.indexerService.Test(ctx, id)
			if err != nil {
				return false, err.Error()
			}
			return result.Success, result.Message
		},
		GetRootFolderPath: func(ctx context.Context, id int64) (string, error) {
			folder, err := s.rootFolderService.Get(ctx, id)
			if err != nil {
				return "", err
			}
			return folder.Path, nil
		},
		IsTMDBConfigured: s.metadataService.IsTMDBConfigured,
		IsTVDBConfigured: s.metadataService.IsTVDBConfigured,
		TestTMDB:         s.metadataService.TestTMDB,
		TestTVDB:         s.metadataService.TestTVDB,
	})
	healthHandlers.RegisterRoutes(api.Group("/system/health"))

	// Auth routes
	auth := api.Group("/auth")
	auth.POST("/login", s.login)
	auth.POST("/logout", s.logout)
	auth.GET("/status", s.authStatus)

	// Movies routes - use new handlers
	movieHandlers := movies.NewHandlers(s.movieService)
	movieHandlers.RegisterRoutes(api.Group("/movies"))

	// Series routes - use new handlers
	tvHandlers := tv.NewHandlers(s.tvService)
	tvHandlers.RegisterRoutes(api.Group("/series"))

	// Library manager routes (scanning and refresh) - initialized here for refresh endpoints
	libraryManagerHandlers := librarymanager.NewHandlers(s.libraryManagerService)

	// Refresh metadata endpoints (need to be on the movies/series groups)
	api.POST("/movies/:id/refresh", libraryManagerHandlers.RefreshMovie)
	api.POST("/series/:id/refresh", libraryManagerHandlers.RefreshSeries)

	// Library add endpoints (creates item + downloads artwork)
	libraryGroup := api.Group("/library")
	libraryGroup.POST("/movies", libraryManagerHandlers.AddMovie)
	libraryGroup.POST("/series", libraryManagerHandlers.AddSeries)

	// Quality profiles routes
	qualityHandlers := quality.NewHandlers(s.qualityService)
	qualityHandlers.RegisterRoutes(api.Group("/qualityprofiles"))

	// Root folders routes
	rootFolderHandlers := rootfolder.NewHandlers(s.rootFolderService)
	rootFolderHandlers.RegisterRoutes(api.Group("/rootfolders"))

	// Wire up auto-scan when root folder is created
	rootFolderHandlers.SetOnFolderCreated(func(folderID int64) {
		ctx := context.Background()
		_, err := s.libraryManagerService.ScanRootFolder(ctx, folderID)
		if err != nil {
			s.logger.Error().Err(err).Int64("rootFolderId", folderID).Msg("Auto-scan failed for new root folder")
		}
	})

	rootFoldersGroup := api.Group("/rootfolders")
	rootFoldersGroup.POST("/:id/scan", libraryManagerHandlers.ScanRootFolder)
	rootFoldersGroup.GET("/:id/scan", libraryManagerHandlers.GetScanStatus)
	rootFoldersGroup.DELETE("/:id/scan", libraryManagerHandlers.CancelScan)
	api.GET("/scans", libraryManagerHandlers.GetAllScanStatuses)
	api.POST("/scans", libraryManagerHandlers.ScanAllRootFolders)

	// Metadata routes
	metadataHandlers := metadata.NewHandlers(s.metadataService, s.artworkDownloader)
	metadataHandlers.RegisterRoutes(api.Group("/metadata"))

	// TMDB configuration endpoints
	api.POST("/metadata/tmdb/search-ordering", s.updateTMDBSearchOrdering)

	// Filesystem routes (for folder browsing)
	filesystemHandlers := filesystem.NewHandlersWithStorage(s.filesystemService, s.storageService)
	filesystemHandlers.SetMediaParser(func(filename string) *filesystem.ParsedInfo {
		parsed := scanner.ParseFilename(filename)
		if parsed == nil {
			return nil
		}
		return &filesystem.ParsedInfo{
			Title:            parsed.Title,
			Year:             parsed.Year,
			Season:           parsed.Season,
			Episode:          parsed.Episode,
			EndEpisode:       parsed.EndEpisode,
			IsSeasonPack:     parsed.IsSeasonPack,
			IsCompleteSeries: parsed.IsCompleteSeries,
			Quality:          parsed.Quality,
			Source:           parsed.Source,
			Codec:            parsed.Codec,
			IsTV:             parsed.IsTV,
		}
	})
	filesystemHandlers.RegisterRoutes(api.Group("/filesystem"))

	// Settings routes
	settings := api.Group("/settings")
	settings.GET("", s.getSettings)
	settings.PUT("", s.updateSettings)

	// Indexers routes
	indexerHandlers := indexer.NewHandlers(s.indexerService)
	indexerHandlers.SetStatusService(s.statusService)
	indexerHandlers.RegisterRoutes(api.Group("/indexers"))

	// Download clients routes
	clients := api.Group("/downloadclients")
	clients.GET("", s.listDownloadClients)
	clients.POST("", s.addDownloadClient)
	clients.POST("/test", s.testNewDownloadClient)
	clients.GET("/:id", s.getDownloadClient)
	clients.PUT("/:id", s.updateDownloadClient)
	clients.DELETE("/:id", s.deleteDownloadClient)
	clients.POST("/:id/test", s.testDownloadClient)
	clients.POST("/:id/debug/addtorrent", s.debugAddTorrent)

	// Queue/Downloads routes
	api.GET("/queue", s.getQueue)
	api.POST("/queue/:id/pause", s.pauseDownload)
	api.POST("/queue/:id/resume", s.resumeDownload)
	api.DELETE("/queue/:id", s.removeFromQueue)

	// History routes
	historyHandlers := history.NewHandlers(s.historyService)
	historyHandlers.RegisterRoutes(api.Group("/history"))
	api.GET("/history/indexer", s.getIndexerHistory)

	// Search routes (with quality service for scored search endpoints)
	searchHandlers := search.NewHandlers(s.searchService, s.qualityService)
	searchHandlers.RegisterRoutes(api.Group("/search"))

	// Grab routes (under /search for grabbing search results)
	grabHandlers := grab.NewHandlers(s.grabService)
	grabHandlers.RegisterRoutes(api.Group("/search"))

	// Defaults routes
	defaultsHandlers := defaults.NewHandlers(s.defaultsService)
	defaultsHandlers.RegisterRoutes(api.Group("/defaults"))

	// Preferences routes
	preferencesHandlers := preferences.NewHandlers(s.preferencesService)
	preferencesHandlers.RegisterRoutes(api.Group("/preferences"))

	// Calendar routes
	calendarHandlers := calendar.NewHandlers(s.calendarService)
	calendarHandlers.RegisterRoutes(api.Group("/calendar"))

	// Missing routes
	missingHandlers := missing.NewHandlers(s.missingService)
	missingHandlers.RegisterRoutes(api.Group("/missing"))

	// Autosearch routes
	autosearchHandlers := autosearch.NewHandlers(s.autosearchService)
	autosearchHandlers.SetScheduledSearcher(s.scheduledSearcher)
	autosearchHandlers.RegisterRoutes(api.Group("/autosearch"))

	// Autosearch settings routes
	autosearchSettings := autosearch.NewSettingsHandler(sqlc.New(s.db), &s.cfg.AutoSearch)
	settings.GET("/autosearch", autosearchSettings.GetSettings)
	settings.PUT("/autosearch", autosearchSettings.UpdateSettings)

	// Import routes
	importHandlers := importer.NewHandlers(s.importService, s.db)
	importHandlers.RegisterRoutes(api.Group("/import"))

	// Import settings routes
	importSettings := importer.NewSettingsHandlers(s.db, s.importService)
	importSettings.RegisterSettingsRoutes(settings)

	// Scheduler routes
	if s.scheduler != nil {
		schedulerHandler := handlers.NewSchedulerHandler(s.scheduler)
		schedulerGroup := api.Group("/scheduler")
		schedulerGroup.GET("/tasks", schedulerHandler.ListTasks)
		schedulerGroup.GET("/tasks/:id", schedulerHandler.GetTask)
		schedulerGroup.POST("/tasks/:id/run", schedulerHandler.RunTask)
	}
}

// Start begins listening for HTTP requests.
func (s *Server) Start(address string) error {
	s.logger.Info().Str("address", address).Msg("starting HTTP server")

	// Register existing items with health service
	ctx := context.Background()
	if err := s.downloaderService.RegisterExistingClients(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to register existing download clients with health service")
	}
	if err := s.indexerService.RegisterExistingIndexers(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to register existing indexers with health service")
	}
	if err := s.rootFolderService.RegisterExistingRootFolders(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to register existing root folders with health service")
	}
	s.metadataService.RegisterMetadataProviders()

	// Start the import service workers
	if s.importService != nil {
		s.importService.Start(context.Background())
	}

	// Start the scheduler
	if s.scheduler != nil {
		if err := s.scheduler.Start(); err != nil {
			s.logger.Error().Err(err).Msg("Failed to start scheduler")
		}
	}

	return s.echo.Start(address)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")

	// Stop the scheduler
	if s.scheduler != nil {
		if err := s.scheduler.Stop(); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to stop scheduler")
		}
	}

	// Stop the import service workers
	if s.importService != nil {
		s.importService.Stop()
	}

	return s.echo.Shutdown(ctx)
}

// Echo returns the underlying Echo instance (for serving static files).
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// EnsureDefaults creates default data like quality profiles.
func (s *Server) EnsureDefaults(ctx context.Context) error {
	return s.qualityService.EnsureDefaults(ctx)
}

// --- Handler implementations ---

func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getStatus(c echo.Context) error {
	ctx := c.Request().Context()

	movieCount, _ := s.movieService.Count(ctx)
	seriesCount, _ := s.tvService.Count(ctx)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"version":       "0.0.1-dev",
		"startTime":     time.Now().Format(time.RFC3339),
		"movieCount":    movieCount,
		"seriesCount":   seriesCount,
		"developerMode": s.cfg.DeveloperMode,
		"tmdb": map[string]interface{}{
			"disableSearchOrdering": s.cfg.Metadata.TMDB.DisableSearchOrdering,
		},
	})
}

// UpdateTMDBSearchOrdering toggles search ordering for TMDB.
// POST /api/v1/metadata/tmdb/search-ordering
func (s *Server) updateTMDBSearchOrdering(c echo.Context) error {
	if !s.cfg.DeveloperMode {
		return echo.NewHTTPError(http.StatusForbidden, "debug features require developer mode")
	}

	var request struct {
		DisableSearchOrdering bool `json:"disableSearchOrdering"`
	}

	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Update configuration
	s.cfg.Metadata.TMDB.DisableSearchOrdering = request.DisableSearchOrdering

	s.logger.Info().
		Bool("disableSearchOrdering", request.DisableSearchOrdering).
		Msg("TMDB search ordering setting updated")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"disableSearchOrdering": s.cfg.Metadata.TMDB.DisableSearchOrdering,
	})
}

// Auth handlers (placeholders)
func (s *Server) login(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) logout(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) authStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"authenticated": false,
		"requiresAuth":  false,
	})
}

// Settings handlers (placeholders)
func (s *Server) getSettings(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{})
}

func (s *Server) updateSettings(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// Download client handlers
func (s *Server) listDownloadClients(c echo.Context) error {
	ctx := c.Request().Context()

	clients, err := s.downloaderService.List(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, clients)
}

func (s *Server) addDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	var input downloader.CreateClientInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	client, err := s.downloaderService.Create(ctx, input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, client)
}

func (s *Server) getDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	client, err := s.downloaderService.Get(ctx, id)
	if err != nil {
		if err == downloader.ErrClientNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, client)
}

func (s *Server) updateDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var input downloader.UpdateClientInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	client, err := s.downloaderService.Update(ctx, id, input)
	if err != nil {
		if err == downloader.ErrClientNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, client)
}

func (s *Server) deleteDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := s.downloaderService.Delete(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) testDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	result, err := s.downloaderService.Test(ctx, id)
	if err != nil {
		if err == downloader.ErrClientNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) testNewDownloadClient(c echo.Context) error {
	ctx := c.Request().Context()

	var input downloader.CreateClientInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	result, err := s.downloaderService.TestConfig(ctx, input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) debugAddTorrent(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// Only allow in developer mode
	if !s.cfg.DeveloperMode {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "debug features only available in developer mode"})
	}

	// Get client name for the mock items
	client, err := s.downloaderService.Get(ctx, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "client not found"})
	}

	// Add mock downloads using library content
	mockIDs, err := s.downloaderService.AddMockDownloads(ctx, id, client.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to add mock downloads: " + err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"mockIds": mockIDs,
		"message": "Mock downloads added based on library content",
	})
}

// Queue handlers
func (s *Server) getQueue(c echo.Context) error {
	ctx := c.Request().Context()

	items, err := s.downloaderService.GetQueue(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, items)
}

func (s *Server) pauseDownload(c echo.Context) error {
	ctx := c.Request().Context()
	torrentID := c.Param("id")

	var body struct {
		ClientID int64 `json:"clientId"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.downloaderService.PauseDownload(ctx, body.ClientID, torrentID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Broadcast queue update so UI refreshes immediately
	s.hub.Broadcast("queue:updated", nil)

	return c.JSON(http.StatusOK, map[string]string{"status": "paused"})
}

func (s *Server) resumeDownload(c echo.Context) error {
	ctx := c.Request().Context()
	torrentID := c.Param("id")

	var body struct {
		ClientID int64 `json:"clientId"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.downloaderService.ResumeDownload(ctx, body.ClientID, torrentID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Broadcast queue update so UI refreshes immediately
	s.hub.Broadcast("queue:updated", nil)

	return c.JSON(http.StatusOK, map[string]string{"status": "resumed"})
}

func (s *Server) removeFromQueue(c echo.Context) error {
	ctx := c.Request().Context()
	torrentID := c.Param("id")

	clientID, err := strconv.ParseInt(c.QueryParam("clientId"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid clientId"})
	}

	deleteFiles := c.QueryParam("deleteFiles") == "true"

	if err := s.downloaderService.RemoveDownload(ctx, clientID, torrentID, deleteFiles); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Broadcast queue update so UI refreshes immediately
	s.hub.Broadcast("queue:updated", nil)

	return c.NoContent(http.StatusNoContent)
}

// getIndexerHistory returns indexer search and grab history.
func (s *Server) getIndexerHistory(c echo.Context) error {
	limit := 50
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	history, err := s.grabService.GetGrabHistory(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, history)
}

// importHistoryAdapter adapts history.Service to importer.HistoryService interface.
type importHistoryAdapter struct {
	svc *history.Service
}

// Create implements importer.HistoryService.
func (a *importHistoryAdapter) Create(ctx context.Context, input importer.HistoryInput) error {
	_, err := a.svc.Create(ctx, history.CreateInput{
		EventType: history.EventType(input.EventType),
		MediaType: history.MediaType(input.MediaType),
		MediaID:   input.MediaID,
		Source:    input.Source,
		Quality:   input.Quality,
		Data:      input.Data,
	})
	return err
}
