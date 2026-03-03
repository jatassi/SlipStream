package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	authratelimit "github.com/slipstream/slipstream/internal/api/ratelimit"
	"github.com/slipstream/slipstream/internal/arrimport"
	"github.com/slipstream/slipstream/internal/auth"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/filesystem"
	"github.com/slipstream/slipstream/internal/firewall"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/status"
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
	"github.com/slipstream/slipstream/internal/metadata/omdb"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
	"github.com/slipstream/slipstream/internal/metadata/tvdb"
	"github.com/slipstream/slipstream/internal/missing"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/notification/plex"
	"github.com/slipstream/slipstream/internal/portal/admin"
	"github.com/slipstream/slipstream/internal/portal/autoapprove"
	"github.com/slipstream/slipstream/internal/portal/invitations"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	portalnotifs "github.com/slipstream/slipstream/internal/portal/notifications"
	"github.com/slipstream/slipstream/internal/portal/quota"
	portalratelimit "github.com/slipstream/slipstream/internal/portal/ratelimit"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/rsssync"
	"github.com/slipstream/slipstream/internal/scheduler"
	"github.com/slipstream/slipstream/internal/scheduler/tasks"
	"github.com/slipstream/slipstream/internal/update"
	"github.com/slipstream/slipstream/internal/websocket"
)

const (
	mediaTypeMovie   = "movie"
	mediaTypeEpisode = "episode"
	queryTrue        = "true"
)

// Server handles HTTP requests for the SlipStream API.
type Server struct {
	echo      *echo.Echo
	dbManager *database.Manager
	hub       *websocket.Hub
	logger    *zerolog.Logger
	cfg       *config.Config

	startupDB *sql.DB

	library      LibraryGroup
	metadata     MetadataGroup
	filesystem   FilesystemGroup
	download     DownloadGroup
	search       SearchGroup
	automation   AutomationGroup
	system       SystemGroup
	notification NotificationGroup
	portal       PortalGroup
	security     SecurityGroup

	registry ServiceRegistry

	restartChan    chan<- bool
	configuredPort int
}

// SetConfiguredPort sets the original configured port before any conflict resolution.
func (s *Server) SetConfiguredPort(port int) {
	s.configuredPort = port
}

// SetLogsProvider sets the provider for log streaming and retrieval.
// This must be called after NewServer to register the logs routes.
func (s *Server) SetLogsProvider(provider LogsProvider) {
	s.system.Logs = provider

	// Register logs routes (called after NewServer, so we register here)
	logsGroup := s.echo.Group("/api/v1/system/logs")
	logsGroup.Use(s.adminAuthMiddleware())
	logsHandlers := NewLogsHandlers(provider)
	logsHandlers.RegisterRoutes(logsGroup)
}

// NewServer creates a new API server instance.
func NewServer(dbManager *database.Manager, hub *websocket.Hub, cfg *config.Config, logger *zerolog.Logger, restartChan chan<- bool) *Server {
	serverDebugLog("Creating Echo instance...")
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.IPExtractor = echo.ExtractIPDirect()
	serverDebugLog("Echo instance created")

	db := dbManager.Conn()

	serverDebugLog("Creating Server struct...")
	apiLogger := logger.With().Str("component", "api").Logger()
	s := &Server{
		echo:        e,
		dbManager:   dbManager,
		hub:         hub,
		logger:      &apiLogger,
		cfg:         cfg,
		startupDB:   db,
		restartChan: restartChan,
	}
	serverDebugLog("Server struct created")

	// Store real metadata clients for later switching
	serverDebugLog("Creating metadata clients...")
	s.metadata.RealTMDBClient = tmdb.NewClient(cfg.Metadata.TMDB, logger)
	s.metadata.RealTVDBClient = tvdb.NewClient(cfg.Metadata.TVDB, logger)
	s.metadata.RealOMDBClient = omdb.NewClient(cfg.Metadata.OMDB, logger)
	serverDebugLog("Metadata clients created")

	// Register WebSocket handler for dev mode toggle
	if hub != nil {
		hub.SetDevModeHandler(func(enabled bool) error {
			if err := dbManager.SetDevMode(enabled); err != nil {
				return err
			}
			// Copy settings to dev database before updating services
			if enabled {
				s.copyJWTSecretToDevDB()
				s.copySettingsToDevDB()
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
			// Copy data and populate mock media
			if enabled {
				profileIDMapping := s.copyQualityProfilesToDevDB()
				s.copyPortalUsersToDevDB(profileIDMapping)
				s.copyPortalUserNotificationsToDevDB()
				s.setupDevModeSlots()
				s.populateMockMedia()
			}
			return nil
		})
	}

	s.initSystemServices(db, hub, logger)
	s.initLibraryServices(db, hub, logger)
	s.initMetadataServices(db, hub, cfg, logger)
	s.initFilesystemServices(logger)
	s.initDownloadServices(db, hub, logger)
	s.initSearchServices(db, hub, cfg, logger)
	s.initAutomationServices(db, hub, cfg, logger)
	s.initNotificationServices(db, logger)
	s.initPortalServices(db, hub, cfg, logger)
	s.initSecurityServices()

	s.registerLibraryDependentTasks(cfg, logger)

	// Initialize update service (after scheduler so we can register the task)
	s.system.Update = update.NewService(db, logger, restartChan)
	s.system.Update.SetBroadcaster(hub)
	s.system.Update.SetPort(cfg.Server.Port)

	// Initialize firewall checker
	s.system.Firewall = firewall.NewChecker()

	// Register update check task
	queries := sqlc.New(db)
	if s.automation.Scheduler != nil {
		if err := tasks.RegisterUpdateCheckTask(s.automation.Scheduler, s.system.Update, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register update check task")
		}
		// Register Plex library refresh task
		if err := tasks.RegisterPlexRefreshTask(s.automation.Scheduler, queries, s.notification.Service, s.notification.PlexClient, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register Plex refresh task")
		}
	}

	serverDebugLog("Setting up middleware...")
	s.setupMiddleware()
	serverDebugLog("Setting up routes...")
	s.setupRoutes()
	serverDebugLog("NewServer complete")

	return s
}

// initSystemServices initializes the health and system-level services.
func (s *Server) initSystemServices(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger) {
	serverDebugLog("Initializing health service...")
	s.system.Health = health.NewService(logger)
	s.system.Health.SetBroadcaster(hub)

	s.system.Defaults = defaults.NewService(sqlc.New(db))
	s.system.Calendar = calendar.NewService(db, logger)
	s.system.Availability = availability.NewService(db, logger)
	s.system.Missing = missing.NewService(db, logger)
	s.system.Preferences = preferences.NewService(sqlc.New(db))
	s.system.History = history.NewService(db, logger)
	s.system.History.SetBroadcaster(hub)
	s.system.Progress = progress.NewManager(hub, logger)

	s.registry.RegisterDB(
		s.system.Defaults,
		s.system.Calendar,
		s.system.Availability,
		s.system.Missing,
		s.system.History,
		s.system.Preferences,
	)
}

// initLibraryServices initializes the library management services.
func (s *Server) initLibraryServices(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger) {
	serverDebugLog("Initializing core services (scanner, movie, tv, quality, slots)...")
	s.library.Scanner = scanner.NewService(logger)
	s.library.Movies = movies.NewService(db, hub, logger)
	s.library.TV = tv.NewService(db, hub, logger)
	s.library.Quality = quality.NewService(db, logger)
	s.library.Slots = slots.NewService(db, s.library.Quality, logger)
	serverDebugLog("Core services initialized")

	// Wire up quality profile service for status evaluation on file import
	s.library.Movies.SetQualityService(s.library.Quality)
	s.library.TV.SetQualityService(s.library.Quality)

	// Wire up slot-related file deletion handling (Req 12.1.1, 12.1.2)
	s.library.Movies.SetFileDeleteHandler(s.library.Slots)
	s.library.TV.SetFileDeleteHandler(s.library.Slots)

	// Wire up file deleter for slot disable operations (Req 12.2.2)
	s.library.Slots.SetFileDeleter(&slotFileDeleterAdapter{
		movieSvc: s.library.Movies,
		tvSvc:    s.library.TV,
	})

	s.library.RootFolder = rootfolder.NewService(db, logger, s.system.Defaults)
	s.library.RootFolder.SetHealthService(s.system.Health)

	// Wire up root folder provider for slot-level root folders (Req 22.1.1-22.1.4)
	s.library.Slots.SetRootFolderProvider(&slotRootFolderAdapter{
		rootFolderSvc: s.library.RootFolder,
	})

	// Initialize organizer service (for file operations)
	namingCfg := organizer.DefaultNamingConfig()
	s.library.Organizer = organizer.NewService(&namingCfg, logger)

	// Initialize mediainfo service (for probing media files)
	s.library.Mediainfo = mediainfo.NewService(mediainfo.DefaultConfig(), logger)

	s.registry.RegisterDB(
		s.library.Movies,
		s.library.TV,
		s.library.Quality,
		s.library.Slots,
		s.library.RootFolder,
	)

	// Wire status change loggers
	s.library.Movies.SetStatusChangeLogger(&statusChangeLoggerAdapter{s.system.History})
	s.library.TV.SetStatusChangeLogger(&statusChangeLoggerAdapter{s.system.History})
}

// initMetadataServices initializes metadata and artwork services.
func (s *Server) initMetadataServices(db *sql.DB, hub *websocket.Hub, cfg *config.Config, logger *zerolog.Logger) {
	s.metadata.Service = metadata.NewService(&cfg.Metadata, logger)
	s.metadata.Service.SetHealthService(s.system.Health)
	s.metadata.NetworkLogoStore = metadata.NewSQLNetworkLogoStore(db)
	s.metadata.Service.SetNetworkLogoStore(s.metadata.NetworkLogoStore)

	// Derive artwork directory from database path (uses same data directory)
	dataDir := filepath.Dir(cfg.Database.Path)
	artworkCfg := metadata.ArtworkConfig{
		BaseDir: filepath.Join(dataDir, "artwork"),
		Timeout: 30 * time.Second,
	}
	s.metadata.ArtworkDownloader = metadata.NewArtworkDownloader(artworkCfg, logger)
	s.metadata.ArtworkDownloader.SetBroadcaster(hub)

	s.registry.RegisterDB(
		s.metadata.NetworkLogoStore,
	)
}

// initFilesystemServices initializes filesystem and storage services.
func (s *Server) initFilesystemServices(logger *zerolog.Logger) {
	s.filesystem.Service = filesystem.NewService(logger)
	s.filesystem.Storage = filesystem.NewStorageService(s.filesystem.Service, s.library.RootFolder, logger)
}

// initDownloadServices initializes download management services.
func (s *Server) initDownloadServices(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger) {
	s.download.Service = downloader.NewService(db, logger)
	s.download.Service.SetHealthService(s.system.Health)
	s.download.Service.SetBroadcaster(hub)
	s.download.Service.SetStatusChangeLogger(&statusChangeLoggerAdapter{s.system.History})

	// Initialize queue broadcaster for real-time download progress updates
	if hub != nil {
		s.download.QueueBroadcaster = downloader.NewQueueBroadcaster(s.download.Service, hub, logger)
	}

	s.registry.RegisterDB(
		s.download.Service,
	)
}

// initSearchServices initializes search, indexer, and grab services.
func (s *Server) initSearchServices(db *sql.DB, hub *websocket.Hub, cfg *config.Config, logger *zerolog.Logger) {
	// Initialize Cardigann manager for indexer definitions
	// Note: Definitions are fetched lazily when the user opens the Add Indexer dialog
	serverDebugLog("Initializing Cardigann manager...")
	serverDebugLog(fmt.Sprintf("Cardigann definitions dir: %s", cfg.Indexer.Cardigann.DefinitionsDir))
	cardigannCfg := cardigann.ManagerConfig{
		Repository: cardigann.RepositoryConfig{
			BaseURL:        cfg.Indexer.Cardigann.RepositoryURL,
			Branch:         cfg.Indexer.Cardigann.Branch,
			Version:        cfg.Indexer.Cardigann.Version,
			RequestTimeout: cfg.Indexer.Cardigann.RequestTimeoutDuration(),
			UserAgent:      "SlipStream/1.0",
		},
		Cache: cardigann.CacheConfig{
			DefinitionsDir: cfg.Indexer.Cardigann.DefinitionsDir,
			CustomDir:      cfg.Indexer.Cardigann.CustomDir,
		},
		AutoUpdate:     cfg.Indexer.Cardigann.AutoUpdate,
		UpdateInterval: cfg.Indexer.Cardigann.UpdateIntervalDuration(),
	}
	cardigannManager, err := cardigann.NewManager(&cardigannCfg, logger)
	if err != nil {
		serverDebugLog(fmt.Sprintf("Failed to initialize Cardigann manager: %v", err))
		logger.Error().Err(err).Msg("Failed to initialize Cardigann manager")
	}
	serverDebugLog("Cardigann manager initialized")

	// Initialize indexer service
	serverDebugLog("Initializing indexer service...")
	s.search.Indexer = indexer.NewService(db, cardigannManager, logger)
	s.search.Indexer.SetHealthService(s.system.Health)
	serverDebugLog("Indexer service initialized")

	// Initialize indexer status service
	s.search.Status = status.NewService(db, logger)
	s.search.Status.SetHealthService(s.system.Health)

	// Set up cookie store for persistent indexer sessions
	cookieStore := indexer.NewCookieStore(s.search.Status)
	cardigannManager.SetCookieStore(cookieStore)

	// Initialize rate limiter
	s.search.RateLimiter = ratelimit.NewLimiter(db, ratelimit.DefaultConfig(), logger)

	// Initialize Prowlarr service
	s.search.Prowlarr = prowlarr.NewService(db, logger)
	s.search.ProwlarrMode = prowlarr.NewModeManager(s.search.Prowlarr, s.dbManager.IsDevMode)

	// Wire up mode check for Cardigann definition updates
	// Definitions should only be updated when in SlipStream mode
	if manager := s.search.Indexer.GetManager(); manager != nil {
		modeManager := s.search.ProwlarrMode
		manager.SetModeCheckFunc(func() bool {
			isSlipStream, err := modeManager.IsSlipStreamMode(context.Background())
			if err != nil {
				// Default to SlipStream mode on error
				return true
			}
			return isSlipStream
		})
	}

	// Initialize search service with status, rate limiting, and WebSocket events
	s.search.Search = search.NewService(s.search.Indexer, logger)
	s.search.Search.SetStatusService(s.search.Status)
	s.search.Search.SetRateLimiter(s.search.RateLimiter)
	s.search.Search.SetBroadcaster(hub)

	// Initialize Prowlarr search adapter and grab provider for mode-aware routing
	s.search.ProwlarrSearch = prowlarr.NewSearchAdapter(s.search.Prowlarr)
	s.search.ProwlarrGrab = prowlarr.NewGrabProvider(
		s.search.Prowlarr,
		s.search.ProwlarrMode,
		s.search.Indexer,
		logger,
	)

	// Initialize search router for mode-aware search routing
	s.search.Router = search.NewRouter(s.search.Search, logger)
	s.search.Router.SetProwlarrSearcher(s.search.ProwlarrSearch)
	s.search.Router.SetModeProvider(s.search.ProwlarrMode)

	// Initialize grab service with status, rate limiting, and WebSocket events
	// Uses prowlarrGrabProvider for mode-aware routing (Prowlarr vs internal indexers)
	s.search.Grab = grab.NewService(db, s.download.Service, logger)
	s.search.Grab.SetIndexerService(s.search.ProwlarrGrab)
	s.search.Grab.SetStatusService(s.search.Status)
	s.search.Grab.SetRateLimiter(s.search.RateLimiter)
	s.search.Grab.SetBroadcaster(hub)
	if s.download.QueueBroadcaster != nil {
		s.search.Grab.SetQueueTrigger(s.download.QueueBroadcaster)
	}

	// Initialize shared grab lock for concurrent grab protection
	s.search.GrabLock = decisioning.NewGrabLock()

	s.registry.RegisterDB(
		s.search.Indexer,
		s.search.Status,
		s.search.RateLimiter,
		s.search.Grab,
		s.search.Prowlarr,
	)
}

// initAutomationServices initializes automation, import, and scheduling services.
func (s *Server) initAutomationServices(db *sql.DB, hub *websocket.Hub, cfg *config.Config, logger *zerolog.Logger) {
	// Initialize import service (for processing completed downloads)
	s.automation.Import = importer.NewService(
		db,
		s.download.Service,
		s.library.Movies,
		s.library.TV,
		s.library.RootFolder,
		s.library.Organizer,
		s.library.Mediainfo,
		hub,
		importer.DefaultConfig(),
		logger,
	)
	s.automation.Import.SetHealthService(s.system.Health)
	s.automation.Import.SetHistoryService(&importHistoryAdapter{s.system.History})
	s.automation.Import.SetQualityService(s.library.Quality)
	s.automation.Import.SetSlotsService(s.library.Slots)
	s.library.Quality.SetImportDecisionCleaner(s.automation.Import)

	// Wire up import service to queue broadcaster for immediate import triggering
	if s.download.QueueBroadcaster != nil {
		s.download.QueueBroadcaster.SetCompletionHandler(s.automation.Import)
	}

	// Initialize arr import service (for migrating from Radarr/Sonarr)
	s.automation.ArrImport = arrimport.NewService(
		db,
		s.library.Movies,
		s.library.TV,
		&arrImportRootFolderAdapter{svc: s.library.RootFolder},
		&arrImportQualityAdapter{svc: s.library.Quality},
		s.system.Progress,
		&arrImportHubAdapter{hub: hub},
		logger,
	)
	s.automation.ArrImport.SetSlotsService(s.library.Slots)

	// Initialize autosearch service (uses search router for mode-aware search routing)
	s.automation.Autosearch = autosearch.NewService(db, s.search.Router, s.search.Grab, s.library.Quality, logger)
	s.automation.Autosearch.SetBroadcaster(hub)
	s.automation.Autosearch.SetHistoryService(s.system.History)
	s.automation.Autosearch.SetGrabLock(s.search.GrabLock)

	// Load saved autosearch settings into config before creating scheduler
	if err := autosearch.LoadSettingsIntoConfig(context.Background(), sqlc.New(db), &cfg.AutoSearch); err != nil {
		logger.Warn().Err(err).Msg("Failed to load autosearch settings, using defaults")
	}

	// Initialize scheduled searcher for automatic background searches
	s.automation.ScheduledSearcher = autosearch.NewScheduledSearcher(s.automation.Autosearch, &cfg.AutoSearch, logger)

	// Initialize RSS sync service
	rssFetcher := rsssync.NewFeedFetcher(s.search.Indexer, s.search.Prowlarr, s.search.ProwlarrMode, sqlc.New(db), logger)
	s.automation.RssSync = rsssync.NewService(sqlc.New(db), rssFetcher, s.search.Grab, s.library.Quality, s.system.History, s.search.GrabLock, s.system.Health, hub, logger)

	// Load saved RSS sync settings into config
	if err := rsssync.LoadSettingsIntoConfig(context.Background(), sqlc.New(db), &cfg.RssSync); err != nil {
		logger.Warn().Err(err).Msg("Failed to load RSS sync settings, using defaults")
	}

	// Initialize scheduler
	serverDebugLog("Initializing scheduler...")
	sched, err := scheduler.New(logger)
	if err != nil {
		serverDebugLog(fmt.Sprintf("Failed to initialize scheduler: %v", err))
		logger.Error().Err(err).Msg("Failed to initialize scheduler")
	} else {
		serverDebugLog("Scheduler initialized, registering tasks...")
		s.automation.Scheduler = sched
		s.automation.Scheduler.OnTaskStateChanged(schedulerBroadcaster(hub))
		// Register availability refresh task
		if err := tasks.RegisterAvailabilityTask(s.automation.Scheduler, s.system.Availability); err != nil {
			logger.Error().Err(err).Msg("Failed to register availability task")
		}
		// Register automatic search task
		if err := tasks.RegisterAutoSearchTask(s.automation.Scheduler, s.automation.ScheduledSearcher, &cfg.AutoSearch); err != nil {
			logger.Error().Err(err).Msg("Failed to register autosearch task")
		}
		// Register RSS sync task
		if err := tasks.RegisterRssSyncTask(s.automation.Scheduler, s.automation.RssSync, &cfg.RssSync); err != nil {
			logger.Error().Err(err).Msg("Failed to register RSS sync task")
		}
		serverDebugLog("Scheduler tasks registered")
	}

	// Initialize library manager service (orchestrates scanning and file matching)
	s.library.LibraryManager = librarymanager.NewService(
		db,
		s.library.Scanner,
		s.library.Movies,
		s.library.TV,
		s.metadata.Service,
		s.metadata.ArtworkDownloader,
		s.library.RootFolder,
		s.library.Quality,
		s.system.Progress,
		logger,
	)
	// Wire up optional services for search-on-add functionality
	s.library.LibraryManager.SetAutosearchService(s.automation.Autosearch)
	s.library.LibraryManager.SetPreferencesService(s.system.Preferences)
	s.library.LibraryManager.SetSlotsService(s.library.Slots)
	s.library.LibraryManager.SetHealthService(s.system.Health)

	// Wire up series refresher for pre-search metadata updates
	s.automation.ScheduledSearcher.SetSeriesRefresher(s.library.LibraryManager)

	// Wire up metadata refresh for arr-import (fetches artwork/metadata after creating items)
	s.automation.ArrImport.SetMetadataRefresher(&arrImportMetadataRefresherAdapter{svc: s.library.LibraryManager})

	s.registry.RegisterDB(
		s.automation.Autosearch,
		s.automation.Import,
		s.library.LibraryManager,
	)
	s.registry.RegisterQueries(
		s.automation.RssSync,
	)
}

// initNotificationServices initializes notification services.
func (s *Server) initNotificationServices(db *sql.DB, logger *zerolog.Logger) {
	s.notification.Service = notification.NewService(db, logger)

	// Initialize Plex client and handlers for OAuth/discovery endpoints
	plexHTTPClient := &http.Client{Timeout: 30 * time.Second}
	s.notification.PlexClient = plex.NewClient(plexHTTPClient, logger, config.Version)
	s.notification.PlexHandlers = plex.NewHandlers(s.notification.PlexClient, logger)

	// Wire up notification service to health service for health alerts
	s.system.Health.SetNotifier(s.notification.Service)

	// Wire up notification service to grab service
	s.search.Grab.SetNotificationService(&grabNotificationAdapter{
		svc:    s.notification.Service,
		movies: s.library.Movies,
		tv:     s.library.TV,
	})

	// Wire up notification service to movies service
	s.library.Movies.SetNotificationDispatcher(&movieNotificationAdapter{s.notification.Service})

	// Wire up notification service to TV service
	s.library.TV.SetNotificationDispatcher(&tvNotificationAdapter{s.notification.Service})

	// Wire up notification service to import service
	s.automation.Import.SetNotificationDispatcher(&importNotificationAdapter{s.notification.Service})

	// Wire up config import services for arr-import
	s.automation.ArrImport.SetConfigImportServices(
		s.download.Service,
		s.search.Indexer,
		s.notification.Service,
		s.library.Quality,
		s.automation.Import,
	)

	s.registry.RegisterDB(
		s.notification.Service,
	)
}

// initPortalServices initializes the external requests portal services.
func (s *Server) initPortalServices(db *sql.DB, hub *websocket.Hub, cfg *config.Config, logger *zerolog.Logger) {
	serverDebugLog("Initializing portal services...")
	queries := sqlc.New(db)
	s.portal.Users = users.NewService(queries, logger)
	s.portal.Invitations = invitations.NewService(queries, logger)
	s.portal.Quota = quota.NewService(queries, logger)
	s.portal.Requests = requests.NewService(queries, logger)
	s.portal.Requests.SetBroadcaster(requests.NewEventBroadcaster(hub))
	s.portal.Notifications = portalnotifs.NewService(queries, s.notification.Service, hub, logger)
	s.portal.Watchers = requests.NewWatchersService(queries, logger)
	s.portal.Requests.SetNotificationDispatcher(s.portal.Notifications)
	s.portal.Requests.SetWatchersService(s.portal.Watchers)
	s.portal.AutoApprove = autoapprove.NewService(
		queries,
		s.portal.Users,
		s.library.Quality,
		s.portal.Quota,
		s.portal.Requests,
		logger,
	)
	serverDebugLog("Portal services initialized")

	// Initialize status tracker for portal request status updates
	s.portal.StatusTracker = requests.NewStatusTracker(queries, s.portal.Requests, s.portal.Watchers, logger)
	s.portal.StatusTracker.SetMovieLookup(&statusTrackerMovieLookup{movieSvc: s.library.Movies})
	s.portal.StatusTracker.SetEpisodeLookup(&statusTrackerEpisodeLookup{tvSvc: s.library.TV})
	s.portal.StatusTracker.SetSeriesLookup(&statusTrackerSeriesLookup{tvSvc: s.library.TV})
	s.portal.StatusTracker.SetNotificationDispatcher(s.portal.Notifications)
	s.portal.LibraryChecker = requests.NewLibraryChecker(queries, logger)
	s.portal.AdminLibraryChecker = &adminRequestLibraryCheckerAdapter{queries: queries}
	s.automation.Import.SetStatusTracker(s.portal.StatusTracker)
	s.search.Grab.SetPortalStatusTracker(s.portal.StatusTracker)
	s.download.Service.SetPortalStatusTracker(s.portal.StatusTracker)

	// Initialize portal media provisioner
	s.portal.MediaProvisioner = &portalMediaProvisionerAdapter{
		queries:        queries,
		movieService:   s.library.Movies,
		tvService:      s.library.TV,
		libraryManager: s.library.LibraryManager,
		logger:         logger,
	}

	serverDebugLog("Initializing portal auth service...")
	authSvc, err := auth.NewService(queries, s.logger, cfg.Portal.JWTSecret)
	if err != nil {
		serverDebugLog(fmt.Sprintf("Failed to initialize portal auth service: %v", err))
		logger.Error().Err(err).Msg("Failed to initialize portal auth service")
	}
	s.portal.Auth = authSvc
	s.portal.AuthMiddleware = portalmw.NewAuthMiddleware(authSvc)
	s.portal.AuthMiddleware.SetEnabledChecker(&portalEnabledChecker{queries: queries})
	serverDebugLog("Portal auth service initialized")

	serverDebugLog("Initializing passkey service...")
	passkeySvc, err := auth.NewPasskeyService(queries, auth.PasskeyConfig{
		RPDisplayName: cfg.Portal.WebAuthn.RPDisplayName,
		RPID:          cfg.Portal.WebAuthn.RPID,
		RPOrigins:     cfg.Portal.WebAuthn.RPOrigins,
	})
	if err != nil {
		serverDebugLog(fmt.Sprintf("Failed to initialize passkey service: %v", err))
		logger.Error().Err(err).Msg("Failed to initialize passkey service")
	}
	s.portal.Passkey = passkeySvc
	serverDebugLog("Passkey service initialized")

	// Wire token validator to the WebSocket hub now that the auth service is ready.
	if hub != nil && authSvc != nil {
		hub.SetTokenValidator(func(token string) error {
			_, err := authSvc.ValidateAdminToken(token)
			return err
		})
	}

	s.portal.SearchLimiter = portalratelimit.NewSearchLimiter(func() int64 {
		if setting, err := queries.GetSetting(context.Background(), admin.SettingSearchRateLimit); err == nil && setting.Value != "" {
			if v, parseErr := strconv.ParseInt(setting.Value, 10, 64); parseErr == nil {
				return v
			}
		}
		return portalratelimit.DefaultRequestsPerMinute
	})
	s.portal.SearchLimiter.StartCleanup(5 * time.Minute)

	s.registry.RegisterDB(
		s.portal.MediaProvisioner,
		s.portal.StatusTracker,
		s.portal.LibraryChecker,
		s.portal.AdminLibraryChecker,
		s.portal.Watchers,
	)
	s.registry.RegisterQueries(
		s.portal.Users,
		s.portal.Invitations,
		s.portal.Quota,
		s.portal.Notifications,
		s.portal.AutoApprove,
		s.portal.Requests,
		s.portal.Auth,
	)
	if s.portal.Passkey != nil {
		s.registry.RegisterQueries(s.portal.Passkey)
	}
}

// initSecurityServices initializes rate limiting and security services.
func (s *Server) initSecurityServices() {
	// Initialize auth rate limiter (IP-based + account lockout)
	s.security.AuthLimiter = authratelimit.NewAuthLimiter()
	s.security.AuthLimiter.StartCleanup(5 * time.Minute)
}

func schedulerBroadcaster(hub *websocket.Hub) scheduler.TaskStateCallback {
	return func(taskID string, running bool) {
		eventType := "scheduler:task:started"
		if !running {
			eventType = "scheduler:task:completed"
		}
		hub.Broadcast(eventType, map[string]any{
			"taskId":  taskID,
			"running": running,
		})
	}
}

// Start begins listening for HTTP requests.
func (s *Server) registerLibraryDependentTasks(cfg *config.Config, logger *zerolog.Logger) {
	if s.automation.Scheduler == nil {
		return
	}

	if err := tasks.RegisterLibraryScanTask(s.automation.Scheduler, s.library.LibraryManager, s.library.RootFolder, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register library scan task")
	}
	if err := tasks.RegisterDownloadClientHealthTask(s.automation.Scheduler, s.download.Service, s.system.Health, &cfg.Health, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register download client health task")
	}
	if err := tasks.RegisterIndexerHealthTask(s.automation.Scheduler, s.search.Indexer, s.search.Prowlarr, s.search.ProwlarrMode, s.system.Health, &cfg.Health, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register indexer health task")
	}
	if err := tasks.RegisterProwlarrHealthTask(s.automation.Scheduler, s.search.Prowlarr, s.search.ProwlarrMode, s.system.Health, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register Prowlarr health task")
	}
	storageAdapter := health.NewStorageServiceAdapter(s.filesystem.Storage)
	storageChecker := health.NewStorageChecker(s.system.Health, storageAdapter, &cfg.Health, logger)
	if err := tasks.RegisterStorageHealthTask(s.automation.Scheduler, storageChecker, &cfg.Health, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register storage health task")
	}
	if err := tasks.RegisterImportScanTask(s.automation.Scheduler, s.automation.Import, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register import scan task")
	}
	if err := tasks.RegisterHistoryCleanupTask(s.automation.Scheduler, s.system.History); err != nil {
		logger.Error().Err(err).Msg("Failed to register history cleanup task")
	}
}

func (s *Server) Start(address string) error {
	s.logger.Info().Str("address", address).Msg("starting HTTP server")

	// Register existing items with health service
	ctx := context.Background()
	if err := s.download.Service.RegisterExistingClients(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to register existing download clients with health service")
	}
	if err := s.search.Indexer.RegisterExistingIndexers(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to register existing indexers with health service")
	}
	if err := s.library.RootFolder.RegisterExistingRootFolders(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to register existing root folders with health service")
	}
	s.metadata.Service.RegisterMetadataProviders()

	// Start queue broadcaster for real-time download progress
	if s.download.QueueBroadcaster != nil {
		s.download.QueueBroadcaster.Start()
	}

	// Start the import service workers
	if s.automation.Import != nil {
		s.automation.Import.Start(context.Background())
	}

	// Start the scheduler
	if s.automation.Scheduler != nil {
		if err := s.automation.Scheduler.Start(); err != nil {
			s.logger.Error().Err(err).Msg("Failed to start scheduler")
		}
	}

	return s.echo.Start(address)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")

	// Stop the queue broadcaster
	if s.download.QueueBroadcaster != nil {
		s.download.QueueBroadcaster.Stop()
	}

	// Stop the scheduler
	if s.automation.Scheduler != nil {
		if err := s.automation.Scheduler.Stop(); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to stop scheduler")
		}
	}

	// Stop the import service workers
	if s.automation.Import != nil {
		s.automation.Import.Stop()
	}

	return s.echo.Shutdown(ctx)
}

// Echo returns the underlying Echo instance (for serving static files).
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// EnsureDefaults creates default data like quality profiles.
func (s *Server) EnsureDefaults(ctx context.Context) error {
	return s.library.Quality.EnsureDefaults(ctx)
}
