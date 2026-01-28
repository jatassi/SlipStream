package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/api/handlers"
	apimw "github.com/slipstream/slipstream/internal/api/middleware"
	authratelimit "github.com/slipstream/slipstream/internal/api/ratelimit"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/downloader"
	downloadermock "github.com/slipstream/slipstream/internal/downloader/mock"
	"github.com/slipstream/slipstream/internal/firewall"
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
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/auth"
	"github.com/slipstream/slipstream/internal/portal/admin"
	"github.com/slipstream/slipstream/internal/portal/autoapprove"
	"github.com/slipstream/slipstream/internal/portal/invitations"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	portalnotifs "github.com/slipstream/slipstream/internal/portal/notifications"
	"github.com/slipstream/slipstream/internal/portal/quota"
	portalratelimit "github.com/slipstream/slipstream/internal/portal/ratelimit"
	"github.com/slipstream/slipstream/internal/portal/requests"
	portalsearch "github.com/slipstream/slipstream/internal/portal/search"
	"github.com/slipstream/slipstream/internal/portal/users"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"
	"github.com/slipstream/slipstream/internal/scheduler"
	"github.com/slipstream/slipstream/internal/scheduler/tasks"
	"github.com/slipstream/slipstream/internal/update"
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
	healthService            *health.Service
	importService            *importer.Service
	importSettingsHandlers   *importer.SettingsHandlers
	organizerService         *organizer.Service
	mediainfoService    *mediainfo.Service
	slotsService        *slots.Service
	notificationService *notification.Service
	queueBroadcaster    *downloader.QueueBroadcaster
	prowlarrService       *prowlarr.Service
	prowlarrModeManager   *prowlarr.ModeManager
	prowlarrSearchAdapter *prowlarr.SearchAdapter
	prowlarrGrabProvider  *prowlarr.GrabProvider
	searchRouter          *search.Router
	updateService         *update.Service
	firewallChecker       *firewall.Checker
	logsProvider          LogsProvider

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

	// Restart channel for triggering server restart
	restartChan chan<- struct{}

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

// serverDebugLog writes debug messages to the bootstrap log during server initialization.
func serverDebugLog(msg string) {
	var logDir string
	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			logDir = filepath.Join(localAppData, "SlipStream", "logs")
		}
	case "darwin":
		if home, _ := os.UserHomeDir(); home != "" {
			logDir = filepath.Join(home, "Library", "Logs", "SlipStream")
		}
	default:
		if home, _ := os.UserHomeDir(); home != "" {
			logDir = filepath.Join(home, ".config", "slipstream", "logs")
		}
	}
	if logDir == "" {
		logDir = "./logs"
	}

	logFile := filepath.Join(logDir, "bootstrap.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] [NewServer] %s\n", timestamp, msg)
}

// NewServer creates a new API server instance.
func NewServer(dbManager *database.Manager, hub *websocket.Hub, cfg *config.Config, logger zerolog.Logger, restartChan chan<- struct{}) *Server {
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
	cardigannManager, err := cardigann.NewManager(cardigannCfg, logger)
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
		if err := tasks.RegisterIndexerHealthTask(s.scheduler, s.indexerService, s.prowlarrService, s.prowlarrModeManager, s.healthService, &cfg.Health, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register indexer health task")
		}
		// Register Prowlarr health check task
		if err := tasks.RegisterProwlarrHealthTask(s.scheduler, s.prowlarrService, s.prowlarrModeManager, s.healthService, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register Prowlarr health task")
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

	// Initialize update service (after scheduler so we can register the task)
	s.updateService = update.NewService(db, logger, restartChan)
	s.updateService.SetBroadcaster(hub)

	// Initialize firewall checker
	s.firewallChecker = firewall.NewChecker()

	// Register update check task
	if s.scheduler != nil {
		if err := tasks.RegisterUpdateCheckTask(s.scheduler, s.updateService, logger); err != nil {
			logger.Error().Err(err).Msg("Failed to register update check task")
		}
	}

	serverDebugLog("Setting up middleware...")
	s.setupMiddleware()
	serverDebugLog("Setting up routes...")
	s.setupRoutes()
	serverDebugLog("NewServer complete")

	return s
}

// setupMiddleware configures Echo middleware.
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.echo.Use(middleware.Recover())

	// Request ID
	s.echo.Use(middleware.RequestID())

	// Security headers
	s.echo.Use(apimw.SecurityHeaders())

	// Request body size limit (2MB)
	s.echo.Use(middleware.BodyLimit("2M"))

	// CORS - allow same-origin only (origin hostname must match request hostname)
	s.echo.Use(apimw.SameOriginCORS())

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

	// Public routes (no auth required)
	api.GET("/status", s.getStatus)

	// Auth routes (public, no auth required) with rate limiting
	authGroup := api.Group("/auth")
	authGroup.Use(s.authLimiter.Middleware())
	authGroup.GET("/status", s.getAuthStatus)
	authGroup.POST("/setup", s.adminSetup)
	authGroup.DELETE("/admin", s.deleteAdmin)

	// Protected routes (admin auth required)
	protected := api.Group("")
	protected.Use(s.adminAuthMiddleware())

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
	healthHandlers.RegisterRoutes(protected.Group("/system/health"))

	// Note: System logs routes are registered in SetLogsProvider() since the
	// logs provider is set after NewServer() returns

	// Movies routes - use new handlers
	movieHandlers := movies.NewHandlers(s.movieService)
	movieHandlers.RegisterRoutes(protected.Group("/movies"))

	// Series routes - use new handlers
	tvHandlers := tv.NewHandlers(s.tvService)
	tvHandlers.RegisterRoutes(protected.Group("/series"))

	// Library manager routes (scanning and refresh) - initialized here for refresh endpoints
	libraryManagerHandlers := librarymanager.NewHandlers(s.libraryManagerService)

	// Refresh metadata endpoints (need to be on the movies/series groups)
	protected.POST("/movies/:id/refresh", libraryManagerHandlers.RefreshMovie)
	protected.POST("/series/:id/refresh", libraryManagerHandlers.RefreshSeries)

	// Library add endpoints (creates item + downloads artwork)
	libraryGroup := protected.Group("/library")
	libraryGroup.POST("/movies", libraryManagerHandlers.AddMovie)
	libraryGroup.POST("/series", libraryManagerHandlers.AddSeries)

	// Quality profiles routes
	qualityHandlers := quality.NewHandlers(s.qualityService)
	qualityHandlers.RegisterRoutes(protected.Group("/qualityprofiles"))

	// Version slots routes (multi-version support)
	slotsHandlers := slots.NewHandlers(s.slotsService)
	slotsHandlers.RegisterRoutes(protected.Group("/slots"))

	// Slots debug routes (gated behind developerMode)
	slotsDebugHandlers := slots.NewDebugHandlers(s.slotsService, s.dbManager.IsDevMode)
	slotsDebugHandlers.RegisterDebugRoutes(protected.Group("/slots/debug"))

	// Root folders routes
	rootFolderHandlers := rootfolder.NewHandlers(s.rootFolderService)
	rootFolderHandlers.RegisterRoutes(protected.Group("/rootfolders"))

	// Wire up auto-scan when root folder is created
	rootFolderHandlers.SetOnFolderCreated(func(folderID int64) {
		ctx := context.Background()
		_, err := s.libraryManagerService.ScanRootFolder(ctx, folderID)
		if err != nil {
			s.logger.Error().Err(err).Int64("rootFolderId", folderID).Msg("Auto-scan failed for new root folder")
		}
	})

	rootFoldersGroup := protected.Group("/rootfolders")
	rootFoldersGroup.POST("/:id/scan", libraryManagerHandlers.ScanRootFolder)
	rootFoldersGroup.GET("/:id/scan", libraryManagerHandlers.GetScanStatus)
	rootFoldersGroup.DELETE("/:id/scan", libraryManagerHandlers.CancelScan)
	protected.GET("/scans", libraryManagerHandlers.GetAllScanStatuses)
	protected.POST("/scans", libraryManagerHandlers.ScanAllRootFolders)

	// Metadata handlers
	metadataHandlers := metadata.NewHandlers(s.metadataService, s.artworkDownloader)

	// Artwork serving (public - no auth required, images loaded via <img> tags)
	// Must be registered before protected routes so it takes precedence
	metadataHandlers.RegisterArtworkRoutes(api.Group("/metadata"))

	// Other metadata routes (accessible to both admin and portal users)
	metadataGroup := api.Group("/metadata")
	metadataGroup.Use(s.portalAuthMiddleware.AnyAuth())
	metadataHandlers.RegisterRoutes(metadataGroup)

	// TMDB configuration endpoints (admin only)
	protected.POST("/metadata/tmdb/search-ordering", s.updateTMDBSearchOrdering)

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
	filesystemHandlers.RegisterRoutes(protected.Group("/filesystem"))

	// Settings routes
	settings := protected.Group("/settings")
	settings.GET("", s.getSettings)
	settings.PUT("", s.updateSettings)
	settings.POST("/apikey", s.regenerateApiKey)

	// System routes
	protected.POST("/system/restart", s.restart)
	protected.GET("/system/firewall", s.checkFirewall)

	// Indexers routes
	indexersGroup := protected.Group("/indexers")
	indexerHandlers := indexer.NewHandlers(s.indexerService)
	indexerHandlers.SetStatusService(s.statusService)
	indexerHandlers.RegisterRoutes(indexersGroup)

	// Prowlarr routes (under /indexers/prowlarr and /indexers/mode)
	prowlarrHandlers := prowlarr.NewHandlers(s.prowlarrService, s.prowlarrModeManager)
	prowlarrHandlers.RegisterRoutes(indexersGroup)

	// Download clients routes
	clients := protected.Group("/downloadclients")
	clients.GET("", s.listDownloadClients)
	clients.POST("", s.addDownloadClient)
	clients.POST("/test", s.testNewDownloadClient)
	clients.GET("/:id", s.getDownloadClient)
	clients.PUT("/:id", s.updateDownloadClient)
	clients.DELETE("/:id", s.deleteDownloadClient)
	clients.POST("/:id/test", s.testDownloadClient)

	// Queue/Downloads routes
	protected.GET("/queue", s.getQueue)
	protected.POST("/queue/:id/pause", s.pauseDownload)
	protected.POST("/queue/:id/resume", s.resumeDownload)
	protected.POST("/queue/:id/fastforward", s.fastForwardDownload)
	protected.DELETE("/queue/:id", s.removeFromQueue)

	// History routes
	historyHandlers := history.NewHandlers(s.historyService)
	historyHandlers.RegisterRoutes(protected.Group("/history"))
	protected.GET("/history/indexer", s.getIndexerHistory)

	// Search routes (with search router for mode-aware search routing)
	searchHandlers := search.NewHandlers(s.searchRouter, s.qualityService)
	searchHandlers.RegisterRoutes(protected.Group("/search"))

	// Grab routes (under /search for grabbing search results)
	grabHandlers := grab.NewHandlers(s.grabService)
	grabHandlers.RegisterRoutes(protected.Group("/search"))

	// Defaults routes
	defaultsHandlers := defaults.NewHandlers(s.defaultsService)
	defaultsHandlers.RegisterRoutes(protected.Group("/defaults"))

	// Preferences routes
	preferencesHandlers := preferences.NewHandlers(s.preferencesService)
	preferencesHandlers.RegisterRoutes(protected.Group("/preferences"))

	// Calendar routes
	calendarHandlers := calendar.NewHandlers(s.calendarService)
	calendarHandlers.RegisterRoutes(protected.Group("/calendar"))

	// Missing routes
	missingHandlers := missing.NewHandlers(s.missingService)
	missingHandlers.RegisterRoutes(protected.Group("/missing"))

	// Autosearch routes
	autosearchHandlers := autosearch.NewHandlers(s.autosearchService)
	autosearchHandlers.SetScheduledSearcher(s.scheduledSearcher)
	autosearchHandlers.RegisterRoutes(protected.Group("/autosearch"))

	// Autosearch settings routes
	autosearchSettings := autosearch.NewSettingsHandler(sqlc.New(s.startupDB), &s.cfg.AutoSearch)
	if s.scheduler != nil {
		autosearchSettings.SetScheduler(s.scheduler, s.scheduledSearcher, tasks.UpdateAutoSearchTask)
	}
	settings.GET("/autosearch", autosearchSettings.GetSettings)
	settings.PUT("/autosearch", autosearchSettings.UpdateSettings)

	// Import routes
	importHandlers := importer.NewHandlers(s.importService, s.startupDB)
	importHandlers.RegisterRoutes(protected.Group("/import"))

	// Import settings routes
	s.importSettingsHandlers = importer.NewSettingsHandlers(s.startupDB, s.importService)
	s.importSettingsHandlers.RegisterSettingsRoutes(settings)

	// Notifications routes (admin-only for CRUD)
	notificationHandlers := notification.NewHandlers(s.notificationService)
	notificationHandlers.RegisterRoutes(protected.Group("/notifications"))

	// Notifications shared routes (schema and test - accessible to both admin and portal users)
	notificationsShared := api.Group("/notifications")
	notificationsShared.Use(s.portalAuthMiddleware.AnyAuth())
	notificationHandlers.RegisterSharedRoutes(notificationsShared)

	// Scheduler routes
	if s.scheduler != nil {
		schedulerHandler := handlers.NewSchedulerHandler(s.scheduler)
		schedulerGroup := protected.Group("/scheduler")
		schedulerGroup.GET("/tasks", schedulerHandler.ListTasks)
		schedulerGroup.GET("/tasks/:id", schedulerHandler.GetTask)
		schedulerGroup.POST("/tasks/:id/run", schedulerHandler.RunTask)
	}

	// Update routes
	updateHandlers := update.NewHandlers(s.updateService)
	updateHandlers.RegisterRoutes(protected.Group("/update"))

	// Portal routes (External Requests feature) - has its own auth middleware
	s.setupPortalRoutes(api)
}

// setupPortalRoutes configures External Requests portal routes.
func (s *Server) setupPortalRoutes(api *echo.Group) {
	// Portal auth routes (login, signup, profile) - require portal to be enabled
	authGroup := api.Group("/requests/auth")
	authGroup.Use(s.portalAuthMiddleware.PortalEnabled())
	authGroup.Use(s.authLimiter.Middleware()) // Rate limit auth endpoints
	portalAuthHandlers := auth.NewHandlers(
		s.portalAuthService,
		s.portalUsersService,
		s.portalInvitationsService,
	)
	portalAuthHandlers.SetLockoutChecker(s.authLimiter)
	portalAuthHandlers.RegisterRoutes(authGroup, s.portalAuthMiddleware)

	// Passkey routes
	passkeyHandlers := auth.NewPasskeyHandlers(
		s.portalPasskeyService,
		s.portalAuthService,
		s.portalUsersService,
	)
	passkeyHandlers.RegisterRoutes(authGroup, s.portalAuthMiddleware)

	// Portal user routes (authenticated portal users) - require portal to be enabled
	requestsGroup := api.Group("/requests")
	requestsGroup.Use(s.portalAuthMiddleware.PortalEnabled())

	// Portal search with rate limiting
	searchHandlers := portalsearch.NewHandlers(
		s.metadataService,
		s.portalLibraryChecker,
		s.portalUsersService,
	)
	searchGroup := requestsGroup.Group("/search")
	searchGroup.Use(s.portalAuthMiddleware.AnyAuth())
	searchGroup.Use(s.portalSearchLimiter.Middleware())
	searchHandlers.RegisterRoutes(searchGroup, s.portalAuthMiddleware)

	// Portal request handlers
	requestHandlers := requests.NewHandlers(
		s.portalRequestsService,
		requests.NewWatchersService(sqlc.New(s.startupDB), s.logger),
		s.portalUsersService,
		&portalAutoApproveAdapter{svc: s.portalAutoApproveService},
		&portalQueueGetterAdapter{downloaderSvc: s.downloaderService},
		&portalMediaLookupAdapter{queries: sqlc.New(s.startupDB)},
		s.logger,
	)
	requestHandlers.RegisterRoutes(requestsGroup, s.portalAuthMiddleware)

	// Portal user notifications
	portalNotifHandlers := portalnotifs.NewHandlers(s.portalNotificationsService)
	portalNotifHandlers.RegisterRoutes(requestsGroup.Group("/notifications"), s.portalAuthMiddleware)

	// Portal notification inbox (in-app notifications)
	portalInboxHandlers := portalnotifs.NewInboxHandlers(s.portalNotificationsService)
	portalInboxHandlers.RegisterRoutes(requestsGroup.Group("/inbox"), s.portalAuthMiddleware)

	// Admin routes (admin users only)
	adminGroup := api.Group("/admin/requests")

	// Admin user management
	adminUserHandlers := admin.NewUsersHandlers(s.portalUsersService, s.portalQuotaService)
	adminUserHandlers.RegisterRoutes(adminGroup.Group("/users"), s.portalAuthMiddleware)

	// Admin invitations
	adminInvitationHandlers := admin.NewInvitationsHandlers(s.portalInvitationsService)
	adminInvitationHandlers.RegisterRoutes(adminGroup.Group("/invitations"), s.portalAuthMiddleware)

	// Admin request management
	s.portalMediaProvisioner = &portalMediaProvisionerAdapter{
		queries:        sqlc.New(s.startupDB),
		movieService:   s.movieService,
		tvService:      s.tvService,
		libraryManager: s.libraryManagerService,
		logger:         s.logger,
	}
	s.portalRequestSearcher = requests.NewRequestSearcher(
		sqlc.New(s.startupDB),
		s.portalRequestsService,
		s.autosearchService,
		s.portalMediaProvisioner,
		s.logger,
	)
	s.portalAutoApproveService.SetRequestSearcher(s.portalRequestSearcher)
	adminRequestHandlers := admin.NewRequestsHandlers(
		s.portalRequestsService,
		&portalRequestSearcherAdapter{searcher: s.portalRequestSearcher},
	)
	adminRequestHandlers.RegisterRoutes(adminGroup, s.portalAuthMiddleware)

	// Admin settings
	s.adminSettingsHandlers = admin.NewSettingsHandlers(s.portalQuotaService, sqlc.New(s.startupDB))
	s.adminSettingsHandlers.RegisterRoutes(adminGroup.Group("/settings"), s.portalAuthMiddleware)
}

// portalAutoApproveAdapter adapts autoapprove.Service to requests.AutoApproveProcessor interface.
type portalAutoApproveAdapter struct {
	svc *autoapprove.Service
}

func (a *portalAutoApproveAdapter) ProcessAutoApprove(req *requests.Request, user *users.User) error {
	ctx := context.Background()
	_, err := a.svc.ProcessAutoApprove(ctx, req)
	return err
}

// portalRequestSearcherAdapter adapts requests.RequestSearcher to admin.RequestSearcher interface.
type portalRequestSearcherAdapter struct {
	searcher *requests.RequestSearcher
}

func (a *portalRequestSearcherAdapter) SearchForRequestAsync(requestID int64) {
	a.searcher.SearchForRequestAsync(context.Background(), requestID)
}

// portalMediaProvisionerAdapter implements requests.MediaProvisioner to find or create media in library.
type portalMediaProvisionerAdapter struct {
	queries        *sqlc.Queries
	movieService   *movies.Service
	tvService      *tv.Service
	libraryManager *librarymanager.Service
	logger         zerolog.Logger
}

func (a *portalMediaProvisionerAdapter) EnsureMovieInLibrary(ctx context.Context, input requests.MediaProvisionInput) (int64, error) {
	// First, try to find existing movie by TMDB ID
	existing, err := a.movieService.GetByTmdbID(ctx, int(input.TmdbID))
	if err == nil && existing != nil {
		a.logger.Debug().Int64("tmdbID", input.TmdbID).Int64("movieID", existing.ID).Msg("found existing movie in library")
		return existing.ID, nil
	}

	// Movie not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := a.getDefaultSettings(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	movie, err := a.movieService.Create(ctx, movies.CreateMovieInput{
		Title:            input.Title,
		Year:             input.Year,
		TmdbID:           int(input.TmdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create movie: %w", err)
	}

	a.logger.Info().Int64("tmdbID", input.TmdbID).Int64("movieID", movie.ID).Str("title", input.Title).Msg("created movie in library from request")
	return movie.ID, nil
}

func (a *portalMediaProvisionerAdapter) EnsureSeriesInLibrary(ctx context.Context, input requests.MediaProvisionInput) (int64, error) {
	// First, try to find existing series by TVDB ID
	existing, err := a.tvService.GetSeriesByTvdbID(ctx, int(input.TvdbID))
	if err == nil && existing != nil {
		a.logger.Debug().Int64("tvdbID", input.TvdbID).Int64("seriesID", existing.ID).Msg("found existing series in library")
		return existing.ID, nil
	}

	// Series not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := a.getDefaultSettings(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	series, err := a.tvService.CreateSeries(ctx, tv.CreateSeriesInput{
		Title:            input.Title,
		TvdbID:           int(input.TvdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create series: %w", err)
	}

	a.logger.Info().Int64("tvdbID", input.TvdbID).Int64("seriesID", series.ID).Str("title", input.Title).Msg("created series in library from request")

	// Fetch metadata including seasons and episodes
	if a.libraryManager != nil {
		if _, err := a.libraryManager.RefreshSeriesMetadata(ctx, series.ID); err != nil {
			a.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to refresh series metadata, series created without episodes")
		} else {
			a.logger.Info().Int64("seriesID", series.ID).Msg("fetched series metadata with seasons and episodes")
		}
	}

	return series.ID, nil
}

func (a *portalMediaProvisionerAdapter) getDefaultSettings(ctx context.Context) (rootFolderID, qualityProfileID int64, err error) {
	// Get default root folder from settings
	if setting, err := a.queries.GetSetting(ctx, "requests_default_root_folder_id"); err == nil && setting.Value != "" {
		if v, parseErr := strconv.ParseInt(setting.Value, 10, 64); parseErr == nil {
			rootFolderID = v
		}
	}

	// If no default root folder set, try to get the first available one
	if rootFolderID == 0 {
		rootFolders, err := a.queries.ListRootFolders(ctx)
		if err != nil || len(rootFolders) == 0 {
			return 0, 0, errors.New("no root folder configured - please set a default root folder in request settings")
		}
		rootFolderID = rootFolders[0].ID
	}

	// Get first quality profile as default
	profiles, err := a.queries.ListQualityProfiles(ctx)
	if err != nil || len(profiles) == 0 {
		return 0, 0, errors.New("no quality profile configured")
	}
	qualityProfileID = profiles[0].ID

	return rootFolderID, qualityProfileID, nil
}

func (a *portalMediaProvisionerAdapter) SetDB(db *sql.DB) {
	a.queries = sqlc.New(db)
}

// portalQueueGetterAdapter adapts downloader.Service to requests.QueueGetter interface.
type portalQueueGetterAdapter struct {
	downloaderSvc *downloader.Service
}

func (a *portalQueueGetterAdapter) GetQueue(ctx context.Context) ([]requests.QueueItem, error) {
	queue, err := a.downloaderSvc.GetQueue(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]requests.QueueItem, len(queue))
	for i, item := range queue {
		result[i] = requests.QueueItem{
			ID:             item.ID,
			ClientID:       item.ClientID,
			ClientName:     item.ClientName,
			Title:          item.Title,
			MediaType:      item.MediaType,
			Status:         item.Status,
			Progress:       item.Progress,
			Size:           item.Size,
			DownloadedSize: item.DownloadedSize,
			DownloadSpeed:  item.DownloadSpeed,
			ETA:            item.ETA,
			Season:         item.Season,
			Episode:        item.Episode,
			MovieID:        item.MovieID,
			SeriesID:       item.SeriesID,
			SeasonNumber:   item.SeasonNumber,
			IsSeasonPack:   item.IsSeasonPack,
		}
	}
	return result, nil
}

// portalMediaLookupAdapter adapts sqlc.Queries to requests.MediaLookup interface.
type portalMediaLookupAdapter struct {
	queries *sqlc.Queries
}

func (a *portalMediaLookupAdapter) GetMovieTmdbID(ctx context.Context, movieID int64) (*int64, error) {
	movie, err := a.queries.GetMovie(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if movie.TmdbID.Valid {
		return &movie.TmdbID.Int64, nil
	}
	return nil, nil
}

func (a *portalMediaLookupAdapter) GetSeriesTvdbID(ctx context.Context, seriesID int64) (*int64, error) {
	series, err := a.queries.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.TvdbID.Valid {
		return &series.TvdbID.Int64, nil
	}
	return nil, nil
}

// portalEnabledChecker checks if the external requests portal is enabled.
type portalEnabledChecker struct {
	queries *sqlc.Queries
}

func (c *portalEnabledChecker) IsPortalEnabled(ctx context.Context) bool {
	setting, err := c.queries.GetSetting(ctx, "requests_portal_enabled")
	if err != nil {
		return true // Default to enabled if setting not found
	}
	return setting.Value != "0" && setting.Value != "false"
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

// InitializeNetworkServices initializes services that require network connectivity.
// This should be called with retry logic to handle network unavailability at startup.
func (s *Server) InitializeNetworkServices(ctx context.Context) error {
	return s.indexerService.InitializeDefinitions(ctx)
}

// --- Handler implementations ---

func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getStatus(c echo.Context) error {
	ctx := c.Request().Context()

	movieCount, _ := s.movieService.Count(ctx)
	seriesCount, _ := s.tvService.Count(ctx)

	adminExists, _ := s.portalUsersService.AdminExists(ctx)

	// Check if portal is enabled (defaults to true if not set)
	portalEnabled := true
	queries := sqlc.New(s.startupDB)
	if setting, err := queries.GetSetting(ctx, "requests_portal_enabled"); err == nil {
		portalEnabled = setting.Value != "0" && setting.Value != "false"
	}

	response := map[string]interface{}{
		"version":       config.Version,
		"startTime":     time.Now().Format(time.RFC3339),
		"movieCount":    movieCount,
		"seriesCount":   seriesCount,
		"developerMode": s.dbManager.IsDevMode(),
		"portalEnabled": portalEnabled,
		"requiresSetup": !adminExists,
		"requiresAuth":  true,
		"actualPort":    s.cfg.Server.Port,
		"tmdb": map[string]interface{}{
			"disableSearchOrdering": s.cfg.Metadata.TMDB.DisableSearchOrdering,
		},
	}
	if s.configuredPort > 0 && s.configuredPort != s.cfg.Server.Port {
		response["configuredPort"] = s.configuredPort
	}
	return c.JSON(http.StatusOK, response)
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

func (s *Server) getSettings(c echo.Context) error {
	ctx := c.Request().Context()
	queries := sqlc.New(s.dbManager.Conn())

	serverPort := s.cfg.Server.Port
	if setting, err := queries.GetSetting(ctx, "server_port"); err == nil {
		if port, err := strconv.Atoi(setting.Value); err == nil {
			serverPort = port
		}
	}

	logLevel := s.cfg.Logging.Level
	if setting, err := queries.GetSetting(ctx, "log_level"); err == nil {
		logLevel = setting.Value
	}

	apiKey := ""
	if setting, err := queries.GetSetting(ctx, "api_key"); err == nil {
		apiKey = setting.Value
	}

	logPath := s.cfg.Logging.Path
	if logPath != "" {
		if absPath, err := filepath.Abs(logPath); err == nil {
			logPath = absPath
		}
	}

	logMaxSizeMB := s.cfg.Logging.MaxSizeMB
	if logMaxSizeMB <= 0 {
		logMaxSizeMB = 10
	}
	if setting, err := queries.GetSetting(ctx, "log_max_size_mb"); err == nil {
		if v, err := strconv.Atoi(setting.Value); err == nil {
			logMaxSizeMB = v
		}
	}

	logMaxBackups := s.cfg.Logging.MaxBackups
	if logMaxBackups <= 0 {
		logMaxBackups = 5
	}
	if setting, err := queries.GetSetting(ctx, "log_max_backups"); err == nil {
		if v, err := strconv.Atoi(setting.Value); err == nil {
			logMaxBackups = v
		}
	}

	logMaxAgeDays := s.cfg.Logging.MaxAgeDays
	if logMaxAgeDays <= 0 {
		logMaxAgeDays = 30
	}
	if setting, err := queries.GetSetting(ctx, "log_max_age_days"); err == nil {
		if v, err := strconv.Atoi(setting.Value); err == nil {
			logMaxAgeDays = v
		}
	}

	logCompress := s.cfg.Logging.Compress
	if setting, err := queries.GetSetting(ctx, "log_compress"); err == nil {
		logCompress = setting.Value == "true"
	}

	externalAccessEnabled := false
	if setting, err := queries.GetSetting(ctx, "external_access_enabled"); err == nil {
		externalAccessEnabled = setting.Value == "true"
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"serverPort":            serverPort,
		"logLevel":              logLevel,
		"apiKey":                apiKey,
		"logPath":               logPath,
		"logMaxSizeMB":          logMaxSizeMB,
		"logMaxBackups":         logMaxBackups,
		"logMaxAgeDays":         logMaxAgeDays,
		"logCompress":           logCompress,
		"externalAccessEnabled": externalAccessEnabled,
	})
}

func (s *Server) updateSettings(c echo.Context) error {
	ctx := c.Request().Context()
	queries := sqlc.New(s.dbManager.Conn())

	var input struct {
		ServerPort            *int    `json:"serverPort"`
		LogLevel              *string `json:"logLevel"`
		LogMaxSizeMB          *int    `json:"logMaxSizeMB"`
		LogMaxBackups         *int    `json:"logMaxBackups"`
		LogMaxAgeDays         *int    `json:"logMaxAgeDays"`
		LogCompress           *bool   `json:"logCompress"`
		ExternalAccessEnabled *bool   `json:"externalAccessEnabled"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if input.ServerPort != nil {
		if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   "server_port",
			Value: strconv.Itoa(*input.ServerPort),
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if input.LogLevel != nil {
		validLevels := map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[*input.LogLevel] {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid log level"})
		}
		if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   "log_level",
			Value: *input.LogLevel,
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if input.LogMaxSizeMB != nil {
		if *input.LogMaxSizeMB < 1 || *input.LogMaxSizeMB > 100 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "log max size must be between 1 and 100 MB"})
		}
		if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   "log_max_size_mb",
			Value: strconv.Itoa(*input.LogMaxSizeMB),
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if input.LogMaxBackups != nil {
		if *input.LogMaxBackups < 1 || *input.LogMaxBackups > 20 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "log max backups must be between 1 and 20"})
		}
		if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   "log_max_backups",
			Value: strconv.Itoa(*input.LogMaxBackups),
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if input.LogMaxAgeDays != nil {
		if *input.LogMaxAgeDays < 1 || *input.LogMaxAgeDays > 365 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "log max age must be between 1 and 365 days"})
		}
		if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   "log_max_age_days",
			Value: strconv.Itoa(*input.LogMaxAgeDays),
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if input.LogCompress != nil {
		if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   "log_compress",
			Value: strconv.FormatBool(*input.LogCompress),
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	if input.ExternalAccessEnabled != nil {
		if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   "external_access_enabled",
			Value: strconv.FormatBool(*input.ExternalAccessEnabled),
		}); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return s.getSettings(c)
}

func (s *Server) regenerateApiKey(c echo.Context) error {
	ctx := c.Request().Context()
	queries := sqlc.New(s.dbManager.Conn())

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate API key"})
	}
	apiKey := hex.EncodeToString(bytes)

	if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "api_key",
		Value: apiKey,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"apiKey": apiKey})
}

func (s *Server) restart(c echo.Context) error {
	s.logger.Info().Msg("Restart requested via API")

	select {
	case s.restartChan <- struct{}{}:
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Restart initiated",
		})
	default:
		return echo.NewHTTPError(http.StatusConflict, "Restart already in progress")
	}
}

func (s *Server) checkFirewall(c echo.Context) error {
	ctx := c.Request().Context()

	// Get the configured server port
	port := s.cfg.Server.Port
	queries := sqlc.New(s.dbManager.Conn())
	if setting, err := queries.GetSetting(ctx, "server_port"); err == nil {
		if p, err := strconv.Atoi(setting.Value); err == nil {
			port = p
		}
	}

	status, err := s.firewallChecker.CheckPort(ctx, port)
	if err != nil {
		s.logger.Warn().Err(err).Int("port", port).Msg("Failed to check firewall status")
	}

	return c.JSON(http.StatusOK, status)
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

	// Trigger import check asynchronously - provides faster import triggering than scheduled task
	// The import service is efficient and only processes newly completed downloads
	go func() {
		if err := s.importService.CheckAndProcessCompletedDownloads(context.Background()); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to process completed downloads")
		}
	}()

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

	// Trigger immediate broadcast of queue state
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

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

	// Trigger fast polling and immediate broadcast
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "resumed"})
}

func (s *Server) fastForwardDownload(c echo.Context) error {
	ctx := c.Request().Context()
	downloadID := c.Param("id")

	var body struct {
		ClientID int64 `json:"clientId"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.downloaderService.FastForwardMockDownload(ctx, body.ClientID, downloadID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Trigger immediate broadcast of queue state
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "completed"})
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

	// Trigger immediate broadcast of queue state
	if s.queueBroadcaster != nil {
		s.queueBroadcaster.Trigger()
	}

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
	s.importSettingsHandlers.SetDB(db)
	s.defaultsService.SetDB(db)
	s.preferencesService.SetDB(db)
	s.calendarService.SetDB(db)
	s.availabilityService.SetDB(db)
	s.missingService.SetDB(db)
	s.statusService.SetDB(db)
	s.prowlarrService.SetDB(db)

	// Update portal services
	queries := sqlc.New(db)
	s.portalUsersService.SetDB(queries)
	s.portalInvitationsService.SetDB(queries)
	s.portalQuotaService.SetDB(queries)
	s.portalNotificationsService.SetDB(queries)
	s.portalAutoApproveService.SetDB(queries)
	s.portalRequestsService.SetDB(queries)
	s.portalAuthService.SetDB(queries)
	if s.portalPasskeyService != nil {
		s.portalPasskeyService.SetDB(queries)
	}
	s.portalMediaProvisioner.SetDB(db)
	s.portalRequestSearcher.SetDB(db)
	s.portalWatchersService.SetDB(db)
	s.portalStatusTracker.SetDB(db)
	s.portalLibraryChecker.SetDB(db)
	s.adminSettingsHandlers.SetDB(queries)

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

// copyJWTSecretToDevDB copies the JWT secret from production to dev database.
// This ensures tokens issued in production mode remain valid in dev mode.
func (s *Server) copyJWTSecretToDevDB() {
	ctx := context.Background()

	// Get JWT secret from production database
	prodQueries := sqlc.New(s.dbManager.ProdConn())
	setting, err := prodQueries.GetSetting(ctx, "portal_jwt_secret")
	if err != nil {
		s.logger.Debug().Err(err).Msg("No JWT secret in production database to copy")
		return
	}

	if setting.Value == "" {
		s.logger.Debug().Msg("Production JWT secret is empty, nothing to copy")
		return
	}

	// Copy to dev database
	devQueries := sqlc.New(s.dbManager.Conn())
	_, err = devQueries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "portal_jwt_secret",
		Value: setting.Value,
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to copy JWT secret to dev database")
		return
	}

	s.logger.Info().Msg("Copied JWT secret from production to dev database")
}

// copySettingsToDevDB copies application settings from production to dev database.
func (s *Server) copySettingsToDevDB() {
	ctx := context.Background()

	prodQueries := sqlc.New(s.dbManager.ProdConn())
	devQueries := sqlc.New(s.dbManager.Conn())

	settingKeys := []string{"server_port", "log_level", "api_key"}
	copied := 0

	for _, key := range settingKeys {
		setting, err := prodQueries.GetSetting(ctx, key)
		if err != nil {
			continue
		}

		if setting.Value == "" {
			continue
		}

		_, err = devQueries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   key,
			Value: setting.Value,
		})
		if err != nil {
			s.logger.Error().Err(err).Str("key", key).Msg("Failed to copy setting to dev database")
			continue
		}
		copied++
	}

	if copied > 0 {
		s.logger.Info().Int("count", copied).Msg("Copied settings to dev database")
	}
}

// copyPortalUsersToDevDB copies portal users from production to dev database.
// This preserves user IDs so that JWTs issued against prod DB work in dev mode.
// profileIDMapping maps production quality profile IDs to dev database IDs.
func (s *Server) copyPortalUsersToDevDB(profileIDMapping map[int64]int64) {
	ctx := context.Background()

	// Get users from production database
	prodQueries := sqlc.New(s.dbManager.ProdConn())
	prodUsers, err := prodQueries.ListPortalUsers(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list production portal users")
		return
	}

	if len(prodUsers) == 0 {
		s.logger.Debug().Msg("No portal users in production database to copy")
		return
	}

	// Copy each user to dev database (skip if already exists)
	devQueries := sqlc.New(s.dbManager.Conn())
	copied := 0
	for _, user := range prodUsers {
		// Check if user already exists in dev DB
		_, err := devQueries.GetPortalUser(ctx, user.ID)
		if err == nil {
			continue // User already exists
		}

		// Map quality profile ID using the provided mapping
		var qualityProfileID sql.NullInt64
		if user.QualityProfileID.Valid {
			if newID, ok := profileIDMapping[user.QualityProfileID.Int64]; ok {
				qualityProfileID = sql.NullInt64{Int64: newID, Valid: true}
			}
		}

		_, err = devQueries.CreatePortalUserWithID(ctx, sqlc.CreatePortalUserWithIDParams{
			ID:               user.ID,
			Username:         user.Username,
			PasswordHash:     user.PasswordHash,
			DisplayName:      user.DisplayName,
			QualityProfileID: qualityProfileID,
			AutoApprove:      user.AutoApprove,
			Enabled:          user.Enabled,
			IsAdmin:          user.IsAdmin,
		})
		if err != nil {
			s.logger.Error().Err(err).Str("username", user.Username).Msg("Failed to copy portal user")
			continue
		}
		copied++
	}

	if copied > 0 {
		s.logger.Info().Int("count", copied).Msg("Copied portal users to dev database")
	}
}

// copyPortalUserNotificationsToDevDB copies portal user notification channels from production to dev database.
func (s *Server) copyPortalUserNotificationsToDevDB() {
	ctx := context.Background()

	prodQueries := sqlc.New(s.dbManager.ProdConn())
	devQueries := sqlc.New(s.dbManager.Conn())

	prodUsers, err := prodQueries.ListPortalUsers(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list production portal users for notification copy")
		return
	}

	copied := 0
	for _, user := range prodUsers {
		notifs, err := prodQueries.ListUserNotifications(ctx, user.ID)
		if err != nil {
			s.logger.Error().Err(err).Int64("user_id", user.ID).Msg("Failed to list user notifications")
			continue
		}

		for _, n := range notifs {
			_, err := devQueries.CreateUserNotification(ctx, sqlc.CreateUserNotificationParams{
				UserID:      n.UserID,
				Type:        n.Type,
				Name:        n.Name,
				Settings:    n.Settings,
				OnAvailable: n.OnAvailable,
				OnApproved:  n.OnApproved,
				OnDenied:    n.OnDenied,
				Enabled:     n.Enabled,
			})
			if err != nil {
				s.logger.Error().Err(err).Str("name", n.Name).Msg("Failed to copy user notification")
				continue
			}
			copied++
		}
	}

	if copied > 0 {
		s.logger.Info().Int("count", copied).Msg("Copied portal user notifications to dev database")
	}
}

// copyQualityProfilesToDevDB copies quality profiles from production to dev database.
// Returns a mapping of production profile IDs to dev profile IDs.
func (s *Server) copyQualityProfilesToDevDB() map[int64]int64 {
	ctx := context.Background()
	idMapping := make(map[int64]int64)

	// Check if dev database already has profiles
	devProfiles, err := s.qualityService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list dev quality profiles")
		return idMapping
	}
	if len(devProfiles) > 0 {
		s.logger.Info().Int("count", len(devProfiles)).Msg("Dev database already has quality profiles")
		return idMapping
	}

	// Get profiles from production database
	prodQueries := sqlc.New(s.dbManager.ProdConn())
	prodRows, err := prodQueries.ListQualityProfiles(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list production quality profiles")
		return idMapping
	}

	if len(prodRows) == 0 {
		s.logger.Warn().Msg("No quality profiles in production database to copy")
		// Create default profiles in dev database
		if err := s.qualityService.EnsureDefaults(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Failed to create default quality profiles")
		}
		return idMapping
	}

	// Copy each profile to dev database and track ID mapping
	devQueries := sqlc.New(s.dbManager.Conn())
	for _, row := range prodRows {
		newProfile, err := devQueries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
			Name:                 row.Name,
			Cutoff:               row.Cutoff,
			Items:                row.Items,
			HdrSettings:          row.HdrSettings,
			VideoCodecSettings:   row.VideoCodecSettings,
			AudioCodecSettings:   row.AudioCodecSettings,
			AudioChannelSettings: row.AudioChannelSettings,
			UpgradesEnabled:      row.UpgradesEnabled,
			AllowAutoApprove:     row.AllowAutoApprove,
		})
		if err != nil {
			s.logger.Error().Err(err).Str("name", row.Name).Msg("Failed to copy quality profile")
			continue
		}
		idMapping[row.ID] = newProfile.ID
		s.logger.Debug().Str("name", row.Name).Int64("prodID", row.ID).Int64("devID", newProfile.ID).Msg("Copied quality profile to dev database")
	}

	s.logger.Info().Int("count", len(prodRows)).Msg("Copied quality profiles to dev database")
	return idMapping
}

// setupDevModeSlots configures version slots for developer mode testing.
// Assigns quality profiles to slots and enables them so the dry run feature works.
func (s *Server) setupDevModeSlots() {
	ctx := context.Background()

	// Get all slots
	slotList, err := s.slotsService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list slots for dev mode setup")
		return
	}

	// Get all quality profiles
	profiles, err := s.qualityService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list quality profiles for dev mode setup")
		return
	}

	if len(profiles) == 0 {
		s.logger.Warn().Msg("No quality profiles available for dev mode slot setup")
		return
	}

	// Assign profiles to slots and enable them
	// Use different profiles for each slot if available
	for i, slot := range slotList {
		profileIdx := i % len(profiles)
		profileID := profiles[profileIdx].ID

		input := slots.UpdateSlotInput{
			Name:         slot.Name,
			Enabled:      true,
			DisplayOrder: slot.DisplayOrder,
		}
		input.QualityProfileID = &profileID

		_, err := s.slotsService.Update(ctx, slot.ID, input)
		if err != nil {
			s.logger.Error().Err(err).Int64("slotId", slot.ID).Msg("Failed to update slot for dev mode")
			continue
		}
		s.logger.Debug().Int64("slotId", slot.ID).Int64("profileId", profileID).Msg("Configured slot for dev mode")
	}

	s.logger.Info().Int("count", len(slotList)).Msg("Configured slots for dev mode")
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

// statusTrackerMovieLookup implements requests.MovieLookup
type statusTrackerMovieLookup struct {
	movieSvc *movies.Service
}

func (l *statusTrackerMovieLookup) GetTmdbIDByMovieID(ctx context.Context, movieID int64) (int64, error) {
	movie, err := l.movieSvc.Get(ctx, movieID)
	if err != nil {
		return 0, err
	}
	return int64(movie.TmdbID), nil
}

// statusTrackerEpisodeLookup implements requests.EpisodeLookup
type statusTrackerEpisodeLookup struct {
	tvSvc *tv.Service
}

func (l *statusTrackerEpisodeLookup) GetEpisodeInfo(ctx context.Context, episodeID int64) (tvdbID int64, seasonNum, episodeNum int, err error) {
	episode, err := l.tvSvc.GetEpisode(ctx, episodeID)
	if err != nil {
		return 0, 0, 0, err
	}

	series, err := l.tvSvc.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return 0, 0, 0, err
	}

	return int64(series.TvdbID), episode.SeasonNumber, episode.EpisodeNumber, nil
}

// statusTrackerSeriesLookup implements requests.SeriesLookup
type statusTrackerSeriesLookup struct {
	tvSvc *tv.Service
}

func (l *statusTrackerSeriesLookup) GetSeriesIDByTvdbID(ctx context.Context, tvdbID int64) (int64, error) {
	return l.tvSvc.GetSeriesIDByTvdbID(ctx, tvdbID)
}

func (l *statusTrackerSeriesLookup) AreSeasonsComplete(ctx context.Context, seriesID int64, seasonNumbers []int64) (bool, error) {
	return l.tvSvc.AreSeasonsComplete(ctx, seriesID, seasonNumbers)
}
