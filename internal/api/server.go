package api

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/api/handlers"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/downloader"
	downloadermock "github.com/slipstream/slipstream/internal/downloader/mock"
	"github.com/slipstream/slipstream/internal/filesystem"
	fsmock "github.com/slipstream/slipstream/internal/filesystem/mock"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/status"
	indexerTypes "github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/metadata/mock"
	"github.com/slipstream/slipstream/internal/metadata/omdb"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
	"github.com/slipstream/slipstream/internal/metadata/tvdb"
	"github.com/slipstream/slipstream/internal/missing"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"
	"github.com/slipstream/slipstream/internal/scheduler"
	"github.com/slipstream/slipstream/internal/scheduler/tasks"
	"github.com/slipstream/slipstream/internal/websocket"
)

// Server handles HTTP requests for the SlipStream API.
type Server struct {
	echo      *echo.Echo
	dbManager *database.Manager
	hub       *websocket.Hub
	logger    zerolog.Logger
	cfg       *config.Config

	// startupDB is the database connection captured at startup time.
	// Used for handlers that need a *sql.DB reference.
	startupDB *sql.DB

	// Real metadata clients (stored for switching back from mock)
	realTMDBClient metadata.TMDBClient
	realTVDBClient metadata.TVDBClient
	realOMDBClient metadata.OMDBClient

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
	healthService       *health.Service
	importService       *importer.Service
	organizerService    *organizer.Service
	mediainfoService    *mediainfo.Service
	slotsService        *slots.Service
	notificationService *notification.Service
}

// NewServer creates a new API server instance.
func NewServer(dbManager *database.Manager, hub *websocket.Hub, cfg *config.Config, logger zerolog.Logger) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	db := dbManager.Conn()

	s := &Server{
		echo:      e,
		dbManager: dbManager,
		hub:       hub,
		logger:    logger,
		cfg:       cfg,
		startupDB: db,
	}

	// Store real metadata clients for later switching
	s.realTMDBClient = tmdb.NewClient(cfg.Metadata.TMDB, logger)
	s.realTVDBClient = tvdb.NewClient(cfg.Metadata.TVDB, logger)
	s.realOMDBClient = omdb.NewClient(cfg.Metadata.OMDB, logger)

	// Register WebSocket handler for dev mode toggle
	if hub != nil {
		hub.SetDevModeHandler(func(enabled bool) error {
			if err := dbManager.SetDevMode(enabled); err != nil {
				return err
			}
			// Update all services to use the new database connection
			s.updateServicesDB()
			// Switch metadata clients based on dev mode
			s.switchMetadataClients(enabled)
			// Create/clear mock indexer based on dev mode
			s.switchIndexer(enabled)
			// Create/clear mock download client based on dev mode
			s.switchDownloadClient(enabled)
			// Create mock notification based on dev mode
			s.switchNotification(enabled)
			// Create mock root folders based on dev mode
			s.switchRootFolders(enabled)
			// Copy quality profiles and populate mock media
			if enabled {
				s.copyQualityProfilesToDevDB()
				s.populateMockMedia()
			}
			return nil
		})
	}

	// Initialize health service first (no dependencies)
	s.healthService = health.NewService(logger)
	s.healthService.SetBroadcaster(hub)

	// Initialize services
	s.scannerService = scanner.NewService(logger)
	s.movieService = movies.NewService(db, hub, logger)
	s.tvService = tv.NewService(db, hub, logger)
	s.qualityService = quality.NewService(db, logger)
	s.slotsService = slots.NewService(db, s.qualityService, logger)

	// Wire up slot-related file deletion handling (Req 12.1.1, 12.1.2)
	s.movieService.SetFileDeleteHandler(s.slotsService)
	s.tvService.SetFileDeleteHandler(s.slotsService)

	// Wire up file deleter for slot disable operations (Req 12.2.2)
	s.slotsService.SetFileDeleter(&slotFileDeleterAdapter{
		movieSvc: s.movieService,
		tvSvc:    s.tvService,
	})

	s.defaultsService = defaults.NewService(sqlc.New(db))
	s.rootFolderService = rootfolder.NewService(db, logger, s.defaultsService)
	s.rootFolderService.SetHealthService(s.healthService)

	// Wire up root folder provider for slot-level root folders (Req 22.1.1-22.1.4)
	s.slotsService.SetRootFolderProvider(&slotRootFolderAdapter{
		rootFolderSvc: s.rootFolderService,
	})

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

	// Set up cookie store for persistent indexer sessions
	cookieStore := indexer.NewCookieStore(s.statusService)
	cardigannManager.SetCookieStore(cookieStore)

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
	s.importService.SetSlotsService(s.slotsService)

	// Initialize notification service
	s.notificationService = notification.NewService(db, logger)

	// Wire up notification service to health service for health alerts
	s.healthService.SetNotifier(s.notificationService)

	// Wire up notification service to grab service
	s.grabService.SetNotificationService(&grabNotificationAdapter{s.notificationService})

	// Wire up notification service to movies service
	s.movieService.SetNotificationDispatcher(&movieNotificationAdapter{s.notificationService})

	// Wire up notification service to TV service
	s.tvService.SetNotificationDispatcher(&tvNotificationAdapter{s.notificationService})

	// Wire up notification service to import service
	s.importService.SetNotificationDispatcher(&importNotificationAdapter{s.notificationService})

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
	s.libraryManagerService.SetSlotsService(s.slotsService)

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

	// Version slots routes (multi-version support)
	slotsHandlers := slots.NewHandlers(s.slotsService)
	slotsHandlers.RegisterRoutes(api.Group("/slots"))

	// Slots debug routes (gated behind developerMode)
	slotsDebugHandlers := slots.NewDebugHandlers(s.slotsService, s.dbManager.IsDevMode)
	slotsDebugHandlers.RegisterDebugRoutes(api.Group("/slots/debug"))

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
	autosearchSettings := autosearch.NewSettingsHandler(sqlc.New(s.startupDB), &s.cfg.AutoSearch)
	if s.scheduler != nil {
		autosearchSettings.SetScheduler(s.scheduler, s.scheduledSearcher, tasks.UpdateAutoSearchTask)
	}
	settings.GET("/autosearch", autosearchSettings.GetSettings)
	settings.PUT("/autosearch", autosearchSettings.UpdateSettings)

	// Import routes
	importHandlers := importer.NewHandlers(s.importService, s.startupDB)
	importHandlers.RegisterRoutes(api.Group("/import"))

	// Import settings routes
	importSettings := importer.NewSettingsHandlers(s.startupDB, s.importService)
	importSettings.RegisterSettingsRoutes(settings)

	// Notifications routes
	notificationHandlers := notification.NewHandlers(s.notificationService)
	notificationHandlers.RegisterRoutes(api.Group("/notifications"))

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
		"developerMode": s.dbManager.IsDevMode(),
		"tmdb": map[string]interface{}{
			"disableSearchOrdering": s.cfg.Metadata.TMDB.DisableSearchOrdering,
		},
	})
}

// UpdateTMDBSearchOrdering toggles search ordering for TMDB.
// POST /api/v1/metadata/tmdb/search-ordering
func (s *Server) updateTMDBSearchOrdering(c echo.Context) error {
	if !s.dbManager.IsDevMode() {
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

// Queue handlers
func (s *Server) getQueue(c echo.Context) error {
	ctx := c.Request().Context()

	items, err := s.downloaderService.GetQueue(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Check if any items are completed/seeding - if so, trigger import processing
	// This provides faster import triggering than waiting for the scheduled task
	for _, item := range items {
		if item.Status == "completed" || item.Status == "seeding" {
			s.logger.Debug().
				Str("downloadId", item.ID).
				Str("status", item.Status).
				Msg("Detected completed download, triggering import check")
			// Trigger import check asynchronously to not block the response
			go func() {
				if err := s.importService.CheckAndProcessCompletedDownloads(context.Background()); err != nil {
					s.logger.Warn().Err(err).Msg("Failed to process completed downloads")
				}
			}()
			break // Only need to trigger once
		}
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

// slotFileDeleterAdapter adapts movie and TV services to slots.FileDeleter interface.
// Req 12.2.2: Delete files when disabling a slot with delete action.
type slotFileDeleterAdapter struct {
	movieSvc *movies.Service
	tvSvc    *tv.Service
}

// DeleteFile implements slots.FileDeleter.
func (a *slotFileDeleterAdapter) DeleteFile(ctx context.Context, mediaType string, fileID int64) error {
	switch mediaType {
	case "movie":
		return a.movieSvc.RemoveFile(ctx, fileID)
	case "episode":
		return a.tvSvc.RemoveEpisodeFile(ctx, fileID)
	default:
		return nil
	}
}

// slotRootFolderAdapter adapts rootfolder.Service to slots.RootFolderProvider.
type slotRootFolderAdapter struct {
	rootFolderSvc *rootfolder.Service
}

// Get implements slots.RootFolderProvider.
func (a *slotRootFolderAdapter) Get(ctx context.Context, id int64) (*slots.RootFolder, error) {
	rf, err := a.rootFolderSvc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &slots.RootFolder{
		ID:        rf.ID,
		Path:      rf.Path,
		Name:      rf.Name,
		MediaType: rf.MediaType,
	}, nil
}

// grabNotificationAdapter adapts notification.Service to grab.NotificationService interface.
type grabNotificationAdapter struct {
	svc *notification.Service
}

// OnGrab implements grab.NotificationService.
func (a *grabNotificationAdapter) OnGrab(ctx context.Context, release *indexerTypes.ReleaseInfo, clientName string, clientID int64, downloadID string, slotID *int64, slotName string) {
	event := notification.GrabEvent{
		Release: notification.ReleaseInfo{
			ReleaseName: release.Title,
			Quality:     release.Quality,
			Size:        release.Size,
			Indexer:     release.IndexerName,
		},
		DownloadClient: notification.DownloadClientInfo{
			ID:         clientID,
			Name:       clientName,
			DownloadID: downloadID,
		},
		GrabbedAt: time.Now(),
	}

	// Add slot info if provided
	if slotID != nil {
		event.Slot = &notification.SlotInfo{
			ID:   *slotID,
			Name: slotName,
		}
	}

	// TODO: Add movie/episode info from release if available

	a.svc.Dispatch(ctx, notification.EventGrab, event)
}

// movieNotificationAdapter adapts the notification service for movies.
type movieNotificationAdapter struct {
	svc *notification.Service
}

// DispatchMovieAdded implements movies.NotificationDispatcher.
func (a *movieNotificationAdapter) DispatchMovieAdded(ctx context.Context, movie movies.MovieNotificationInfo, addedAt time.Time) {
	event := notification.MovieAddedEvent{
		Movie: notification.MediaInfo{
			ID:       movie.ID,
			Title:    movie.Title,
			Year:     movie.Year,
			TMDbID:   int64(movie.TmdbID),
			IMDbID:   movie.ImdbID,
			Overview: movie.Overview,
		},
		AddedAt: addedAt,
	}
	a.svc.DispatchMovieAdded(ctx, event)
}

// DispatchMovieDeleted implements movies.NotificationDispatcher.
func (a *movieNotificationAdapter) DispatchMovieDeleted(ctx context.Context, movie movies.MovieNotificationInfo, deletedFiles bool, deletedAt time.Time) {
	event := notification.MovieDeletedEvent{
		Movie: notification.MediaInfo{
			ID:       movie.ID,
			Title:    movie.Title,
			Year:     movie.Year,
			TMDbID:   int64(movie.TmdbID),
			IMDbID:   movie.ImdbID,
			Overview: movie.Overview,
		},
		DeletedFiles: deletedFiles,
		DeletedAt:    deletedAt,
	}
	a.svc.DispatchMovieDeleted(ctx, event)
}

// tvNotificationAdapter adapts the notification service for TV series.
type tvNotificationAdapter struct {
	svc *notification.Service
}

// DispatchSeriesAdded implements tv.NotificationDispatcher.
func (a *tvNotificationAdapter) DispatchSeriesAdded(ctx context.Context, series tv.SeriesNotificationInfo, addedAt time.Time) {
	event := notification.SeriesAddedEvent{
		Series: notification.SeriesInfo{
			MediaInfo: notification.MediaInfo{
				ID:       series.ID,
				Title:    series.Title,
				Year:     series.Year,
				TMDbID:   int64(series.TmdbID),
				IMDbID:   series.ImdbID,
				Overview: series.Overview,
			},
			TVDbID: int64(series.TvdbID),
		},
		AddedAt: addedAt,
	}
	a.svc.DispatchSeriesAdded(ctx, event)
}

// DispatchSeriesDeleted implements tv.NotificationDispatcher.
func (a *tvNotificationAdapter) DispatchSeriesDeleted(ctx context.Context, series tv.SeriesNotificationInfo, deletedFiles bool, deletedAt time.Time) {
	event := notification.SeriesDeletedEvent{
		Series: notification.SeriesInfo{
			MediaInfo: notification.MediaInfo{
				ID:       series.ID,
				Title:    series.Title,
				Year:     series.Year,
				TMDbID:   int64(series.TmdbID),
				IMDbID:   series.ImdbID,
				Overview: series.Overview,
			},
			TVDbID: int64(series.TvdbID),
		},
		DeletedFiles: deletedFiles,
		DeletedAt:    deletedAt,
	}
	a.svc.DispatchSeriesDeleted(ctx, event)
}

// importNotificationAdapter adapts the notification service for imports.
type importNotificationAdapter struct {
	svc *notification.Service
}

// DispatchDownload implements importer.NotificationDispatcher.
func (a *importNotificationAdapter) DispatchDownload(ctx context.Context, event importer.DownloadNotificationEvent) {
	notifEvent := notification.DownloadEvent{
		Quality:         event.Quality,
		SourcePath:      event.SourcePath,
		DestinationPath: event.DestinationPath,
		ReleaseName:     event.ReleaseName,
		ImportedAt:      time.Now(),
	}

	if event.MediaType == "movie" && event.MovieID != nil {
		notifEvent.Movie = &notification.MediaInfo{
			ID:    *event.MovieID,
			Title: event.MovieTitle,
			Year:  event.MovieYear,
		}
	} else if event.MediaType == "episode" {
		notifEvent.Episode = &notification.EpisodeInfo{
			SeriesTitle:   event.SeriesTitle,
			SeasonNumber:  event.SeasonNumber,
			EpisodeNumber: event.EpisodeNumber,
			EpisodeTitle:  event.EpisodeTitle,
		}
		if event.SeriesID != nil {
			notifEvent.Episode.SeriesID = *event.SeriesID
		}
	}

	if event.SlotID != nil {
		notifEvent.Slot = &notification.SlotInfo{
			ID:   *event.SlotID,
			Name: event.SlotName,
		}
	}

	a.svc.DispatchDownload(ctx, notifEvent)
}

// DispatchUpgrade implements importer.NotificationDispatcher.
func (a *importNotificationAdapter) DispatchUpgrade(ctx context.Context, event importer.UpgradeNotificationEvent) {
	notifEvent := notification.UpgradeEvent{
		OldQuality:  event.OldQuality,
		NewQuality:  event.NewQuality,
		OldPath:     event.OldPath,
		NewPath:     event.NewPath,
		ReleaseName: event.ReleaseName,
		UpgradedAt:  time.Now(),
	}

	if event.MediaType == "movie" && event.MovieID != nil {
		notifEvent.Movie = &notification.MediaInfo{
			ID:    *event.MovieID,
			Title: event.MovieTitle,
			Year:  event.MovieYear,
		}
	} else if event.MediaType == "episode" {
		notifEvent.Episode = &notification.EpisodeInfo{
			SeriesTitle:   event.SeriesTitle,
			SeasonNumber:  event.SeasonNumber,
			EpisodeNumber: event.EpisodeNumber,
			EpisodeTitle:  event.EpisodeTitle,
		}
		if event.SeriesID != nil {
			notifEvent.Episode.SeriesID = *event.SeriesID
		}
	}

	if event.SlotID != nil {
		notifEvent.Slot = &notification.SlotInfo{
			ID:   *event.SlotID,
			Name: event.SlotName,
		}
	}

	a.svc.DispatchUpgrade(ctx, notifEvent)
}

// switchMetadataClients switches between real and mock metadata clients based on dev mode.
func (s *Server) switchMetadataClients(devMode bool) {
	if devMode {
		s.logger.Info().Msg("Switching to mock metadata providers")
		s.metadataService.SetClients(mock.NewTMDBClient(), mock.NewTVDBClient(), mock.NewOMDBClient())
	} else {
		s.logger.Info().Msg("Switching to real metadata providers")
		s.metadataService.SetClients(s.realTMDBClient, s.realTVDBClient, s.realOMDBClient)
	}
}

// switchIndexer creates or removes the mock indexer based on dev mode.
func (s *Server) switchIndexer(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock indexer already exists
		indexers, err := s.indexerService.List(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to list indexers for dev mode")
			return
		}

		// Look for existing mock indexer
		for _, idx := range indexers {
			if idx.DefinitionID == indexer.MockDefinitionID {
				s.logger.Info().Int64("id", idx.ID).Msg("Mock indexer already exists")
				return
			}
		}

		// Create mock indexer
		_, err = s.indexerService.Create(ctx, indexer.CreateIndexerInput{
			Name:           "Mock Indexer",
			DefinitionID:   indexer.MockDefinitionID,
			SupportsMovies: true,
			SupportsTV:     true,
			Enabled:        true,
			Priority:       1,
		})
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to create mock indexer")
			return
		}
		s.logger.Info().Msg("Created mock indexer for dev mode")
	} else {
		s.logger.Info().Msg("Dev mode disabled - mock indexer will remain until manually deleted")
	}
}

// updateServicesDB updates all services to use the current database connection.
// This must be called after switching databases (e.g., when toggling dev mode).
func (s *Server) updateServicesDB() {
	db := s.dbManager.Conn()
	s.downloaderService.SetDB(db)
	s.notificationService.SetDB(db)
	s.indexerService.SetDB(db)
	s.grabService.SetDB(db)
	s.rateLimiter.SetDB(db)
	s.rootFolderService.SetDB(db)
	s.movieService.SetDB(db)
	s.tvService.SetDB(db)
	s.qualityService.SetDB(db)
	s.libraryManagerService.SetDB(db)
	s.historyService.SetDB(db)
	s.slotsService.SetDB(db)
	s.autosearchService.SetDB(db)
	s.importService.SetDB(db)
	s.defaultsService.SetDB(db)
	s.preferencesService.SetDB(db)
	s.calendarService.SetDB(db)
	s.availabilityService.SetDB(db)
	s.missingService.SetDB(db)
	s.statusService.SetDB(db)
	s.logger.Info().Msg("Updated all services with new database connection")
}

// switchDownloadClient creates or removes the mock download client based on dev mode.
func (s *Server) switchDownloadClient(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock client already exists
		clients, err := s.downloaderService.List(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to list download clients for dev mode")
			return
		}

		// Look for existing mock client
		for _, c := range clients {
			if c.Type == "mock" {
				s.logger.Info().Int64("id", c.ID).Msg("Mock download client already exists")
				return
			}
		}

		// Create mock download client
		_, err = s.downloaderService.Create(ctx, downloader.CreateClientInput{
			Name:     "Mock Download Client",
			Type:     "mock",
			Host:     "localhost",
			Port:     9999,
			Enabled:  true,
			Priority: 1,
		})
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to create mock download client")
			return
		}
		s.logger.Info().Msg("Created mock download client for dev mode")
	} else {
		// Clear mock downloads when disabling dev mode
		downloadermock.GetInstance().Clear()
		s.logger.Info().Msg("Cleared mock downloads")
	}
}

// switchNotification creates mock notification based on dev mode.
func (s *Server) switchNotification(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock notification already exists
		notifications, err := s.notificationService.List(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to list notifications for dev mode")
			return
		}

		// Look for existing mock notification
		for _, n := range notifications {
			if n.Type == notification.NotifierMock {
				s.logger.Info().Int64("id", n.ID).Msg("Mock notification already exists")
				return
			}
		}

		// Create mock notification (subscribed to all events)
		_, err = s.notificationService.Create(ctx, notification.CreateInput{
			Name:             "Mock Notification",
			Type:             notification.NotifierMock,
			Enabled:          true,
			OnGrab:           true,
			OnDownload:       true,
			OnUpgrade:        true,
			OnMovieAdded:     true,
			OnMovieDeleted:   true,
			OnSeriesAdded:    true,
			OnSeriesDeleted:  true,
			OnHealthIssue:    true,
			OnHealthRestored: true,
			OnAppUpdate:      true,
		})
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to create mock notification")
			return
		}
		s.logger.Info().Msg("Created mock notification for dev mode")
	} else {
		s.logger.Info().Msg("Dev mode disabled - mock notification will remain until manually deleted")
	}
}

// switchRootFolders creates mock root folders based on dev mode.
func (s *Server) switchRootFolders(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Reset virtual filesystem to initial state
		fsmock.ResetInstance()

		// Check if mock root folders already exist
		folders, err := s.rootFolderService.List(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to list root folders for dev mode")
			return
		}

		hasMovieRoot := false
		hasTVRoot := false
		for _, f := range folders {
			if f.Path == fsmock.MockMoviesPath {
				hasMovieRoot = true
			}
			if f.Path == fsmock.MockTVPath {
				hasTVRoot = true
			}
		}

		// Create mock movie root folder
		if !hasMovieRoot {
			_, err = s.rootFolderService.Create(ctx, rootfolder.CreateRootFolderInput{
				Path:      fsmock.MockMoviesPath,
				Name:      "Mock Movies",
				MediaType: "movie",
			})
			if err != nil {
				s.logger.Error().Err(err).Msg("Failed to create mock movies root folder")
			} else {
				s.logger.Info().Str("path", fsmock.MockMoviesPath).Msg("Created mock movies root folder")
			}
		}

		// Create mock TV root folder
		if !hasTVRoot {
			_, err = s.rootFolderService.Create(ctx, rootfolder.CreateRootFolderInput{
				Path:      fsmock.MockTVPath,
				Name:      "Mock TV",
				MediaType: "tv",
			})
			if err != nil {
				s.logger.Error().Err(err).Msg("Failed to create mock TV root folder")
			} else {
				s.logger.Info().Str("path", fsmock.MockTVPath).Msg("Created mock TV root folder")
			}
		}
	} else {
		s.logger.Info().Msg("Dev mode disabled - mock root folders will remain until manually deleted")
	}
}

// copyQualityProfilesToDevDB copies quality profiles from production to dev database.
func (s *Server) copyQualityProfilesToDevDB() {
	ctx := context.Background()

	// Check if dev database already has profiles
	devProfiles, err := s.qualityService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list dev quality profiles")
		return
	}
	if len(devProfiles) > 0 {
		s.logger.Info().Int("count", len(devProfiles)).Msg("Dev database already has quality profiles")
		return
	}

	// Get profiles from production database
	prodQueries := sqlc.New(s.dbManager.ProdConn())
	prodRows, err := prodQueries.ListQualityProfiles(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list production quality profiles")
		return
	}

	if len(prodRows) == 0 {
		s.logger.Warn().Msg("No quality profiles in production database to copy")
		// Create default profiles in dev database
		if err := s.qualityService.EnsureDefaults(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Failed to create default quality profiles")
		}
		return
	}

	// Copy each profile to dev database
	devQueries := sqlc.New(s.dbManager.Conn())
	for _, row := range prodRows {
		_, err := devQueries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
			Name:                 row.Name,
			Cutoff:               row.Cutoff,
			Items:                row.Items,
			HdrSettings:          row.HdrSettings,
			VideoCodecSettings:   row.VideoCodecSettings,
			AudioCodecSettings:   row.AudioCodecSettings,
			AudioChannelSettings: row.AudioChannelSettings,
			UpgradesEnabled:      row.UpgradesEnabled,
		})
		if err != nil {
			s.logger.Error().Err(err).Str("name", row.Name).Msg("Failed to copy quality profile")
			continue
		}
		s.logger.Debug().Str("name", row.Name).Msg("Copied quality profile to dev database")
	}

	s.logger.Info().Int("count", len(prodRows)).Msg("Copied quality profiles to dev database")
}

// populateMockMedia creates mock movies and series in the dev database.
func (s *Server) populateMockMedia() {
	ctx := context.Background()

	// Get the mock root folders
	folders, err := s.rootFolderService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list root folders for mock media")
		return
	}

	var movieRootID, tvRootID int64
	for _, f := range folders {
		if f.Path == fsmock.MockMoviesPath {
			movieRootID = f.ID
		}
		if f.Path == fsmock.MockTVPath {
			tvRootID = f.ID
		}
	}

	if movieRootID == 0 || tvRootID == 0 {
		s.logger.Warn().Msg("Mock root folders not found, skipping media population")
		return
	}

	// Get a default quality profile
	profiles, err := s.qualityService.List(ctx)
	if err != nil || len(profiles) == 0 {
		s.logger.Warn().Msg("No quality profiles available for mock media")
		return
	}
	defaultProfileID := profiles[0].ID

	// Check if we already have mock movies
	existingMovies, _ := s.movieService.List(ctx, movies.ListMoviesOptions{PageSize: 1})
	if len(existingMovies) > 0 {
		s.logger.Info().Int("count", len(existingMovies)).Msg("Dev database already has movies")
		return
	}

	s.populateMockMovies(ctx, movieRootID, defaultProfileID)
	s.populateMockSeries(ctx, tvRootID, defaultProfileID)
}

func (s *Server) populateMockMovies(ctx context.Context, rootFolderID, qualityProfileID int64) {
	// Mock movies with TMDB IDs - metadata will be fetched from mock provider
	mockMovieIDs := []struct {
		tmdbID   int
		hasFiles bool
	}{
		{603, true},      // The Matrix
		{27205, true},    // Inception
		{438631, true},   // Dune
		{680, true},      // Pulp Fiction
		{550, true},      // Fight Club
		{693134, false},  // Dune: Part Two
		{872585, false},  // Oppenheimer
		{346698, false},  // Barbie
	}

	for _, m := range mockMovieIDs {
		// Fetch full metadata from mock provider
		movieMeta, err := s.metadataService.GetMovie(ctx, m.tmdbID)
		if err != nil {
			s.logger.Error().Err(err).Int("tmdbID", m.tmdbID).Msg("Failed to fetch mock movie metadata")
			continue
		}

		path := fsmock.MockMoviesPath + "/" + movieMeta.Title + " (" + itoa(movieMeta.Year) + ")"

		input := movies.CreateMovieInput{
			Title:            movieMeta.Title,
			Year:             movieMeta.Year,
			TmdbID:           movieMeta.ID, // ID is the TMDB ID
			ImdbID:           movieMeta.ImdbID,
			Overview:         movieMeta.Overview,
			Runtime:          movieMeta.Runtime,
			RootFolderID:     rootFolderID,
			QualityProfileID: qualityProfileID,
			Path:             path,
			Monitored:        true,
		}

		movie, err := s.movieService.Create(ctx, input)
		if err != nil {
			s.logger.Error().Err(err).Str("title", movieMeta.Title).Msg("Failed to create mock movie")
			continue
		}

		// Download artwork from mock metadata URLs
		go s.artworkDownloader.DownloadMovieArtwork(ctx, movieMeta)

		// If the movie has files in the VFS, create file entries
		if m.hasFiles {
			s.createMockMovieFiles(ctx, movie.ID, path)
		}

		s.logger.Debug().Str("title", movieMeta.Title).Bool("hasFiles", m.hasFiles).Msg("Created mock movie")
	}

	s.logger.Info().Int("count", len(mockMovieIDs)).Msg("Populated mock movies")
}

func (s *Server) createMockMovieFiles(ctx context.Context, movieID int64, moviePath string) {
	vfs := fsmock.GetInstance()
	files, err := vfs.ListDirectory(moviePath)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.Type != fsmock.FileTypeVideo {
			continue
		}

		// Parse quality info from filename
		quality := parseQualityFromFilename(f.Name)

		_, err := s.movieService.AddFile(ctx, movieID, movies.CreateMovieFileInput{
			Path:    f.Path,
			Size:    f.Size,
			Quality: quality,
		})
		if err != nil {
			s.logger.Debug().Err(err).Str("path", f.Path).Msg("Failed to create movie file")
		}
	}
}

func (s *Server) populateMockSeries(ctx context.Context, rootFolderID, qualityProfileID int64) {
	// Mock series with TVDB IDs - metadata will be fetched from mock provider
	mockSeriesIDs := []struct {
		tvdbID   int
		hasFiles bool
	}{
		{81189, true},   // Breaking Bad
		{121361, true},  // Game of Thrones
		{305288, true},  // Stranger Things
		{361753, true},  // The Mandalorian
		{355567, false}, // The Boys
	}

	for _, s2 := range mockSeriesIDs {
		// Fetch full series metadata from mock provider
		seriesMeta, err := s.metadataService.GetSeriesByTVDB(ctx, s2.tvdbID)
		if err != nil {
			s.logger.Error().Err(err).Int("tvdbID", s2.tvdbID).Msg("Failed to fetch mock series metadata")
			continue
		}

		// Fetch seasons and episodes from mock provider
		seasonsMeta, err := s.metadataService.GetSeriesSeasons(ctx, seriesMeta.TmdbID, seriesMeta.TvdbID)
		if err != nil {
			s.logger.Warn().Err(err).Str("title", seriesMeta.Title).Msg("Failed to fetch seasons, using empty")
			seasonsMeta = nil
		}

		path := fsmock.MockTVPath + "/" + seriesMeta.Title

		// Convert season metadata to SeasonInput
		var seasons []tv.SeasonInput
		for _, sm := range seasonsMeta {
			var episodes []tv.EpisodeInput
			for _, ep := range sm.Episodes {
				var airDate *time.Time
				if ep.AirDate != "" {
					if t, err := time.Parse("2006-01-02", ep.AirDate); err == nil {
						airDate = &t
					}
				}
				episodes = append(episodes, tv.EpisodeInput{
					EpisodeNumber: ep.EpisodeNumber,
					Title:         ep.Title,
					Overview:      ep.Overview,
					AirDate:       airDate,
					Monitored:     true,
				})
			}
			seasons = append(seasons, tv.SeasonInput{
				SeasonNumber: sm.SeasonNumber,
				Monitored:    true,
				Episodes:     episodes,
			})
		}

		input := tv.CreateSeriesInput{
			Title:            seriesMeta.Title,
			Year:             seriesMeta.Year,
			TvdbID:           seriesMeta.TvdbID,
			TmdbID:           seriesMeta.TmdbID,
			ImdbID:           seriesMeta.ImdbID,
			Overview:         seriesMeta.Overview,
			Runtime:          seriesMeta.Runtime,
			RootFolderID:     rootFolderID,
			QualityProfileID: qualityProfileID,
			Path:             path,
			Monitored:        true,
			SeasonFolder:     true,
			Seasons:          seasons,
		}

		series, err := s.tvService.CreateSeries(ctx, input)
		if err != nil {
			s.logger.Error().Err(err).Str("title", seriesMeta.Title).Msg("Failed to create mock series")
			continue
		}

		// Download artwork from mock metadata URLs
		go s.artworkDownloader.DownloadSeriesArtwork(ctx, seriesMeta)

		// Create episode files if the series has files in VFS
		if s2.hasFiles {
			s.createMockEpisodeFiles(ctx, series.ID, path)
		}

		s.logger.Debug().Str("title", seriesMeta.Title).Bool("hasFiles", s2.hasFiles).Int("seasons", len(seasons)).Msg("Created mock series")
	}

	s.logger.Info().Int("count", len(mockSeriesIDs)).Msg("Populated mock series")
}

func (s *Server) createMockEpisodeFiles(ctx context.Context, seriesID int64, seriesPath string) {
	vfs := fsmock.GetInstance()
	seasonDirs, err := vfs.ListDirectory(seriesPath)
	if err != nil {
		return
	}

	// Get all episodes for this series
	episodes, err := s.tvService.ListEpisodes(ctx, seriesID, nil)
	if err != nil {
		return
	}

	// Create a map of season:episode -> episodeID
	episodeMap := make(map[string]int64)
	for _, ep := range episodes {
		key := itoa(ep.SeasonNumber) + ":" + itoa(ep.EpisodeNumber)
		episodeMap[key] = ep.ID
	}

	for _, seasonDir := range seasonDirs {
		if seasonDir.Type != fsmock.FileTypeDirectory {
			continue
		}

		seasonNum := parseSeasonNumber(seasonDir.Name)
		if seasonNum == 0 {
			continue
		}

		episodeFiles, err := vfs.ListDirectory(seasonDir.Path)
		if err != nil {
			continue
		}

		for _, f := range episodeFiles {
			if f.Type != fsmock.FileTypeVideo {
				continue
			}

			epNum := parseEpisodeNumber(f.Name)
			if epNum == 0 {
				continue
			}

			key := itoa(seasonNum) + ":" + itoa(epNum)
			episodeID, ok := episodeMap[key]
			if !ok {
				continue
			}

			quality := parseQualityFromFilename(f.Name)
			_, _ = s.tvService.AddEpisodeFile(ctx, episodeID, tv.CreateEpisodeFileInput{
				Path:    f.Path,
				Size:    f.Size,
				Quality: quality,
			})
		}
	}
}

func parseQualityFromFilename(filename string) string {
	filename = strings.ToLower(filename)
	if strings.Contains(filename, "2160p") {
		if strings.Contains(filename, "remux") {
			return "Remux-2160p"
		}
		return "Bluray-2160p"
	}
	if strings.Contains(filename, "1080p") {
		if strings.Contains(filename, "web") {
			return "WEBDL-1080p"
		}
		return "Bluray-1080p"
	}
	if strings.Contains(filename, "720p") {
		if strings.Contains(filename, "web") {
			return "WEBDL-720p"
		}
		return "Bluray-720p"
	}
	return "Unknown"
}

func parseSeasonNumber(name string) int {
	// Handle "Season 01", "Season 1", "S01", etc.
	name = strings.ToLower(name)
	name = strings.TrimPrefix(name, "season ")
	name = strings.TrimPrefix(name, "s")
	name = strings.TrimSpace(name)
	num, _ := strconv.Atoi(name)
	return num
}

func parseEpisodeNumber(filename string) int {
	// Find SxxExx pattern
	filename = strings.ToLower(filename)
	idx := strings.Index(filename, "e")
	if idx == -1 {
		return 0
	}
	// Extract number after 'e'
	numStr := ""
	for i := idx + 1; i < len(filename) && i < idx+4; i++ {
		if filename[i] >= '0' && filename[i] <= '9' {
			numStr += string(filename[i])
		} else {
			break
		}
	}
	num, _ := strconv.Atoi(numStr)
	return num
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
