package api

import (
	"context"
	"database/sql"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/scheduler/tasks"
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
	switchable   SwitchableServices
	registry     *module.Registry
	devMode      *DevModeManager

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

	serverDebugLog("Building services via Wire...")
	services, err := BuildServices(dbManager, hub, cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to build services")
	}
	serverDebugLog("Services built")

	apiLogger := logger.With().Str("component", "api").Logger()
	s := &Server{
		echo:         e,
		dbManager:    dbManager,
		hub:          hub,
		logger:       &apiLogger,
		cfg:          cfg,
		startupDB:    dbManager.Conn(),
		library:      services.Library,
		metadata:     services.Metadata,
		filesystem:   services.Filesystem,
		download:     services.Download,
		search:       services.Search,
		automation:   services.Automation,
		system:       services.System,
		notification: services.Notification,
		portal:       services.Portal,
		security:     services.Security,
		switchable:   services.Switchable,
		registry:     services.Registry,
		restartChan:  restartChan,
	}

	s.devMode = NewDevModeManager(
		&s.library, &s.metadata, &s.search, &s.download,
		&s.notification, &s.switchable, s.dbManager, s.logger,
	)

	serverDebugLog("Wiring circular dependencies...")
	wireCircularDeps(s)

	serverDebugLog("Validating switchable services registry...")
	if err := s.switchable.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("Switchable services validation failed")
	}

	serverDebugLog("Wiring late bindings...")
	wireLateBindings(s)

	serverDebugLog("Setting up middleware...")
	s.setupMiddleware()
	serverDebugLog("Setting up routes...")
	s.setupRoutes()
	serverDebugLog("NewServer complete")

	return s
}

func schedulerBroadcaster(hub *websocket.Hub) func(taskID string, running bool) {
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

// registerLibraryDependentTasks registers scheduler tasks that depend on library services.
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

// Start begins listening for HTTP requests.
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

	// Stop progress manager timers
	if s.system.Progress != nil {
		s.system.Progress.Close()
	}

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
	for _, mt := range []string{"movie", "tv"} {
		if err := s.library.Quality.EnsureDefaults(ctx, mt); err != nil {
			s.logger.Warn().Err(err).Str("moduleType", mt).Msg("Failed to ensure default quality profiles")
		}
	}
	return nil
}
