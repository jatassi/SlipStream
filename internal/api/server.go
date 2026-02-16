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

	// startupDB is the database connection captured at startup time.
	// Used for handlers that need a *sql.DB reference.
	startupDB *sql.DB

	// Real metadata clients (stored for switching back from mock)
	realTMDBClient metadata.TMDBClient
	realTVDBClient metadata.TVDBClient
	realOMDBClient metadata.OMDBClient

	// Services
	scannerService         *scanner.Service
	movieService           *movies.Service
	tvService              *tv.Service
	qualityService         *quality.Service
	rootFolderService      *rootfolder.Service
	metadataService        *metadata.Service
	artworkDownloader      *metadata.ArtworkDownloader
	networkLogoStore       *metadata.SQLNetworkLogoStore
	filesystemService      *filesystem.Service
	storageService         *filesystem.StorageService
	libraryManagerService  *librarymanager.Service
	progressManager        *progress.Manager
	downloaderService      *downloader.Service
	indexerService         *indexer.Service
	searchService          *search.Service
	statusService          *status.Service
	rateLimiter            *ratelimit.Limiter
	grabService            *grab.Service
	defaultsService        *defaults.Service
	calendarService        *calendar.Service
	scheduler              *scheduler.Scheduler
	availabilityService    *availability.Service
	missingService         *missing.Service
	autosearchService      *autosearch.Service
	scheduledSearcher      *autosearch.ScheduledSearcher
	rssSyncService         *rsssync.Service
	rssSyncSettingsHandler *rsssync.SettingsHandler
	grabLock               *decisioning.GrabLock
	preferencesService     *preferences.Service
	historyService         *history.Service
	healthService          *health.Service
	importService          *importer.Service
	importSettingsHandlers *importer.SettingsHandlers
	organizerService       *organizer.Service
	mediainfoService       *mediainfo.Service
	slotsService           *slots.Service
	notificationService    *notification.Service
	plexHandlers           *plex.Handlers
	plexClient             *plex.Client
	queueBroadcaster       *downloader.QueueBroadcaster
	prowlarrService        *prowlarr.Service
	prowlarrModeManager    *prowlarr.ModeManager
	prowlarrSearchAdapter  *prowlarr.SearchAdapter
	prowlarrGrabProvider   *prowlarr.GrabProvider
	searchRouter           *search.Router
	updateService          *update.Service
	firewallChecker        *firewall.Checker
	logsProvider           LogsProvider

	// Portal services
	portalUsersService         *users.Service
	portalInvitationsService   *invitations.Service
	portalRequestsService      *requests.Service
	portalQuotaService         *quota.Service
	portalNotificationsService *portalnotifs.Service
	portalAutoApproveService   *autoapprove.Service
	portalAuthService          *auth.Service
	portalPasskeyService       *auth.PasskeyService
	portalAuthMiddleware       *portalmw.AuthMiddleware
	portalSearchLimiter        *portalratelimit.SearchLimiter
	portalRequestSearcher      *requests.RequestSearcher
	portalMediaProvisioner     *portalMediaProvisionerAdapter
	portalWatchersService      *requests.WatchersService
	portalStatusTracker        *requests.StatusTracker
	portalLibraryChecker       *requests.LibraryChecker
	adminSettingsHandlers      *admin.SettingsHandlers

	// Security
	authLimiter *authratelimit.AuthLimiter

	// Restart channel for triggering server restart (bool = spawn new process after shutdown)
	restartChan chan<- bool

	// Port tracking (configured vs actual after conflict resolution)
	configuredPort int
}

// SetConfiguredPort sets the original configured port before any conflict resolution.
func (s *Server) SetConfiguredPort(port int) {
	s.configuredPort = port
}

// SetLogsProvider sets the provider for log streaming and retrieval.
// This must be called after NewServer to register the logs routes.
func (s *Server) SetLogsProvider(provider LogsProvider) {
	s.logsProvider = provider

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
	serverDebugLog("Echo instance created")

	db := dbManager.Conn()

	serverDebugLog("Creating Server struct...")
	s := &Server{
		echo:        e,
		dbManager:   dbManager,
		hub:         hub,
		logger:      logger,
		cfg:         cfg,
		startupDB:   db,
		restartChan: restartChan,
	}
	serverDebugLog("Server struct created")

	// Store real metadata clients for later switching
	serverDebugLog("Creating metadata clients...")
	s.realTMDBClient = tmdb.NewClient(cfg.Metadata.TMDB, logger)
	s.realTVDBClient = tvdb.NewClient(cfg.Metadata.TVDB, logger)
	s.realOMDBClient = omdb.NewClient(cfg.Metadata.OMDB, logger)
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

	// Initialize health service first (no dependencies)
	serverDebugLog("Initializing health service...")
	s.healthService = health.NewService(logger)
	s.healthService.SetBroadcaster(hub)

	// Initialize services
	serverDebugLog("Initializing core services (scanner, movie, tv, quality, slots)...")
	s.scannerService = scanner.NewService(logger)
	s.movieService = movies.NewService(db, hub, logger)
	s.tvService = tv.NewService(db, hub, logger)
	s.qualityService = quality.NewService(db, logger)
	s.slotsService = slots.NewService(db, s.qualityService, logger)
	serverDebugLog("Core services initialized")

	// Wire up quality profile service for status evaluation on file import
	s.movieService.SetQualityService(s.qualityService)
	s.tvService.SetQualityService(s.qualityService)

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
	s.metadataService = metadata.NewService(&cfg.Metadata, logger)
	s.metadataService.SetHealthService(s.healthService)
	s.networkLogoStore = metadata.NewSQLNetworkLogoStore(db)
	s.metadataService.SetNetworkLogoStore(s.networkLogoStore)
	// Derive artwork directory from database path (uses same data directory)
	dataDir := filepath.Dir(cfg.Database.Path)
	artworkCfg := metadata.ArtworkConfig{
		BaseDir: filepath.Join(dataDir, "artwork"),
		Timeout: 30 * time.Second,
	}
	s.artworkDownloader = metadata.NewArtworkDownloader(artworkCfg, logger)
	s.artworkDownloader.SetBroadcaster(hub)

	// Initialize filesystem service
	s.filesystemService = filesystem.NewService(logger)

	// Initialize storage service (combines filesystem and root folder data)
	s.storageService = filesystem.NewStorageService(s.filesystemService, s.rootFolderService, logger)

	// Initialize downloader service
	s.downloaderService = downloader.NewService(db, logger)
	s.downloaderService.SetHealthService(s.healthService)

	// Initialize queue broadcaster for real-time download progress updates
	if hub != nil {
		s.queueBroadcaster = downloader.NewQueueBroadcaster(s.downloaderService, hub, logger)
	}

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
	s.indexerService = indexer.NewService(db, cardigannManager, logger)
	s.indexerService.SetHealthService(s.healthService)
	serverDebugLog("Indexer service initialized")

	// Initialize indexer status service
	s.statusService = status.NewService(db, logger)
	s.statusService.SetHealthService(s.healthService)

	// Set up cookie store for persistent indexer sessions
	cookieStore := indexer.NewCookieStore(s.statusService)
	cardigannManager.SetCookieStore(cookieStore)

	// Initialize rate limiter
	s.rateLimiter = ratelimit.NewLimiter(db, ratelimit.DefaultConfig(), logger)

	// Initialize Prowlarr service
	s.prowlarrService = prowlarr.NewService(db, logger)
	s.prowlarrModeManager = prowlarr.NewModeManager(s.prowlarrService, dbManager.IsDevMode)

	// Wire up mode check for Cardigann definition updates
	// Definitions should only be updated when in SlipStream mode
	if manager := s.indexerService.GetManager(); manager != nil {
		modeManager := s.prowlarrModeManager
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
	s.searchService = search.NewService(s.indexerService, logger)
	s.searchService.SetStatusService(s.statusService)
	s.searchService.SetRateLimiter(s.rateLimiter)
	s.searchService.SetBroadcaster(hub)

	// Initialize Prowlarr search adapter and grab provider for mode-aware routing
	s.prowlarrSearchAdapter = prowlarr.NewSearchAdapter(s.prowlarrService)
	s.prowlarrGrabProvider = prowlarr.NewGrabProvider(
		s.prowlarrService,
		s.prowlarrModeManager,
		s.indexerService,
		logger,
	)

	// Initialize search router for mode-aware search routing
	s.searchRouter = search.NewRouter(s.searchService, logger)
	s.searchRouter.SetProwlarrSearcher(s.prowlarrSearchAdapter)
	s.searchRouter.SetModeProvider(s.prowlarrModeManager)

	// Initialize grab service with status, rate limiting, and WebSocket events
	// Uses prowlarrGrabProvider for mode-aware routing (Prowlarr vs internal indexers)
	s.grabService = grab.NewService(db, s.downloaderService, logger)
	s.grabService.SetIndexerService(s.prowlarrGrabProvider)
	s.grabService.SetStatusService(s.statusService)
	s.grabService.SetRateLimiter(s.rateLimiter)
	s.grabService.SetBroadcaster(hub)
	if s.queueBroadcaster != nil {
		s.grabService.SetQueueTrigger(s.queueBroadcaster)
	}

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
	namingCfg := organizer.DefaultNamingConfig()
	s.organizerService = organizer.NewService(&namingCfg, logger)

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
	s.importService.SetQualityService(s.qualityService)
	s.importService.SetSlotsService(s.slotsService)
	s.qualityService.SetImportDecisionCleaner(s.importService)

	// Initialize notification service
	s.notificationService = notification.NewService(db, logger)

	// Initialize Plex client and handlers for OAuth/discovery endpoints
	plexHTTPClient := &http.Client{Timeout: 30 * time.Second}
	s.plexClient = plex.NewClient(plexHTTPClient, logger, config.Version)
	s.plexHandlers = plex.NewHandlers(s.plexClient, logger)

	// Wire up notification service to health service for health alerts
	s.healthService.SetNotifier(s.notificationService)

	// Wire up notification service to grab service
	s.grabService.SetNotificationService(&grabNotificationAdapter{
		svc:    s.notificationService,
		movies: s.movieService,
		tv:     s.tvService,
	})

	// Wire up notification service to movies service
	s.movieService.SetNotificationDispatcher(&movieNotificationAdapter{s.notificationService})

	// Wire up notification service to TV service
	s.tvService.SetNotificationDispatcher(&tvNotificationAdapter{s.notificationService})

	// Wire up notification service to import service
	s.importService.SetNotificationDispatcher(&importNotificationAdapter{s.notificationService})

	// Wire up import service to queue broadcaster for immediate import triggering
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.SetCompletionHandler(s.importService)
	}

	// Initialize portal services
	serverDebugLog("Initializing portal services...")
	queries := sqlc.New(db)
	s.portalUsersService = users.NewService(queries, logger)
	s.portalInvitationsService = invitations.NewService(queries, logger)
	s.portalQuotaService = quota.NewService(queries, logger)
	s.portalRequestsService = requests.NewService(queries, logger)
	s.portalRequestsService.SetBroadcaster(requests.NewEventBroadcaster(hub))
	s.portalNotificationsService = portalnotifs.NewService(queries, s.notificationService, hub, logger)
	s.portalWatchersService = requests.NewWatchersService(queries, logger)
	s.portalRequestsService.SetNotificationDispatcher(s.portalNotificationsService)
	s.portalRequestsService.SetWatchersService(s.portalWatchersService)
	s.portalAutoApproveService = autoapprove.NewService(
		queries,
		s.portalUsersService,
		s.qualityService,
		s.portalQuotaService,
		s.portalRequestsService,
		logger,
	)
	serverDebugLog("Portal services initialized")

	// Initialize status tracker for portal request status updates
	s.portalStatusTracker = requests.NewStatusTracker(queries, s.portalRequestsService, s.portalWatchersService, logger)
	s.portalStatusTracker.SetMovieLookup(&statusTrackerMovieLookup{movieSvc: s.movieService})
	s.portalStatusTracker.SetEpisodeLookup(&statusTrackerEpisodeLookup{tvSvc: s.tvService})
	s.portalStatusTracker.SetSeriesLookup(&statusTrackerSeriesLookup{tvSvc: s.tvService})
	s.portalStatusTracker.SetNotificationDispatcher(s.portalNotificationsService)
	s.portalLibraryChecker = requests.NewLibraryChecker(queries, logger)
	s.importService.SetStatusTracker(s.portalStatusTracker)
	s.grabService.SetPortalStatusTracker(s.portalStatusTracker)
	s.downloaderService.SetPortalStatusTracker(s.portalStatusTracker)
	s.downloaderService.SetBroadcaster(hub)
	s.historyService.SetBroadcaster(hub)
	s.downloaderService.SetStatusChangeLogger(&statusChangeLoggerAdapter{s.historyService})
	s.movieService.SetStatusChangeLogger(&statusChangeLoggerAdapter{s.historyService})
	s.tvService.SetStatusChangeLogger(&statusChangeLoggerAdapter{s.historyService})

	serverDebugLog("Initializing portal auth service...")
	authSvc, err := auth.NewService(queries, cfg.Portal.JWTSecret)
	if err != nil {
		serverDebugLog(fmt.Sprintf("Failed to initialize portal auth service: %v", err))
		logger.Error().Err(err).Msg("Failed to initialize portal auth service")
	}
	s.portalAuthService = authSvc
	s.portalAuthMiddleware = portalmw.NewAuthMiddleware(authSvc)
	s.portalAuthMiddleware.SetEnabledChecker(&portalEnabledChecker{queries: queries})
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
	s.portalPasskeyService = passkeySvc
	serverDebugLog("Passkey service initialized")

	s.portalSearchLimiter = portalratelimit.NewSearchLimiter(func() int64 {
		if setting, err := queries.GetSetting(context.Background(), admin.SettingSearchRateLimit); err == nil && setting.Value != "" {
			if v, parseErr := strconv.ParseInt(setting.Value, 10, 64); parseErr == nil {
				return v
			}
		}
		return portalratelimit.DefaultRequestsPerMinute
	})
	s.portalSearchLimiter.StartCleanup(5 * time.Minute)

	// Initialize auth rate limiter (IP-based + account lockout)
	s.authLimiter = authratelimit.NewAuthLimiter()
	s.authLimiter.StartCleanup(5 * time.Minute)

	// Initialize autosearch service (uses search router for mode-aware search routing)
	s.autosearchService = autosearch.NewService(db, s.searchRouter, s.grabService, s.qualityService, logger)
	s.autosearchService.SetBroadcaster(hub)
	s.autosearchService.SetHistoryService(s.historyService)

	// Load saved autosearch settings into config before creating scheduler
	if err := autosearch.LoadSettingsIntoConfig(context.Background(), sqlc.New(db), &cfg.AutoSearch); err != nil {
		logger.Warn().Err(err).Msg("Failed to load autosearch settings, using defaults")
	}

	// Initialize scheduled searcher for automatic background searches
	s.scheduledSearcher = autosearch.NewScheduledSearcher(s.autosearchService, &cfg.AutoSearch, logger)

	// Initialize shared grab lock for concurrent grab protection
	s.grabLock = decisioning.NewGrabLock()
	s.autosearchService.SetGrabLock(s.grabLock)

	// Initialize RSS sync service
	rssFetcher := rsssync.NewFeedFetcher(s.indexerService, s.prowlarrService, s.prowlarrModeManager, sqlc.New(db), logger)
	s.rssSyncService = rsssync.NewService(sqlc.New(db), rssFetcher, s.grabService, s.qualityService, s.historyService, s.grabLock, s.healthService, hub, logger)

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
		s.scheduler = sched
		// Register availability refresh task
		if err := tasks.RegisterAvailabilityTask(s.scheduler, s.availabilityService); err != nil {
			logger.Error().Err(err).Msg("Failed to register availability task")
		}
		// Register automatic search task
		if err := tasks.RegisterAutoSearchTask(s.scheduler, s.scheduledSearcher, &cfg.AutoSearch); err != nil {
			logger.Error().Err(err).Msg("Failed to register autosearch task")
		}
		// Register RSS sync task
		if err := tasks.RegisterRssSyncTask(s.scheduler, s.rssSyncService, &cfg.RssSync); err != nil {
			logger.Error().Err(err).Msg("Failed to register RSS sync task")
		}
		serverDebugLog("Scheduler tasks registered")
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
	s.libraryManagerService.SetHealthService(s.healthService)

	// Wire up series refresher for pre-search metadata updates
	s.scheduledSearcher.SetSeriesRefresher(s.libraryManagerService)

	s.registerLibraryDependentTasks(cfg, logger)

	// Initialize update service (after scheduler so we can register the task)
	s.updateService = update.NewService(db, logger, restartChan)
	s.updateService.SetBroadcaster(hub)
	s.updateService.SetPort(cfg.Server.Port)

	// Initialize firewall checker
	s.firewallChecker = firewall.NewChecker()

	// Register update check task
	if s.scheduler != nil {
		if err := tasks.RegisterUpdateCheckTask(s.scheduler, s.updateService, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register update check task")
		}
		// Register Plex library refresh task
		if err := tasks.RegisterPlexRefreshTask(s.scheduler, queries, s.notificationService, s.plexClient, logger); err != nil {
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

// Start begins listening for HTTP requests.
func (s *Server) registerLibraryDependentTasks(cfg *config.Config, logger *zerolog.Logger) {
	if s.scheduler == nil {
		return
	}

	if err := tasks.RegisterLibraryScanTask(s.scheduler, s.libraryManagerService, s.rootFolderService, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register library scan task")
	}
	if err := tasks.RegisterDownloadClientHealthTask(s.scheduler, s.downloaderService, s.healthService, &cfg.Health, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register download client health task")
	}
	if err := tasks.RegisterIndexerHealthTask(s.scheduler, s.indexerService, s.prowlarrService, s.prowlarrModeManager, s.healthService, &cfg.Health, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register indexer health task")
	}
	if err := tasks.RegisterProwlarrHealthTask(s.scheduler, s.prowlarrService, s.prowlarrModeManager, s.healthService, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register Prowlarr health task")
	}
	storageAdapter := health.NewStorageServiceAdapter(s.storageService)
	storageChecker := health.NewStorageChecker(s.healthService, storageAdapter, &cfg.Health, logger)
	if err := tasks.RegisterStorageHealthTask(s.scheduler, storageChecker, &cfg.Health, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register storage health task")
	}
	if err := tasks.RegisterImportScanTask(s.scheduler, s.importService, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register import scan task")
	}
	if err := tasks.RegisterHistoryCleanupTask(s.scheduler, s.historyService); err != nil {
		logger.Error().Err(err).Msg("Failed to register history cleanup task")
	}
}

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

	// Start queue broadcaster for real-time download progress
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Start()
	}

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

	// Stop the queue broadcaster
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Stop()
	}

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
