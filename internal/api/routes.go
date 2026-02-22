package api

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/slipstream/slipstream/internal/api/handlers"
	apimw "github.com/slipstream/slipstream/internal/api/middleware"
	"github.com/slipstream/slipstream/internal/arrimport"
	"github.com/slipstream/slipstream/internal/auth"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/filesystem"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/missing"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/portal/admin"
	portallibrary "github.com/slipstream/slipstream/internal/portal/library"
	portalnotifs "github.com/slipstream/slipstream/internal/portal/notifications"
	"github.com/slipstream/slipstream/internal/portal/requests"
	portalsearch "github.com/slipstream/slipstream/internal/portal/search"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/rsssync"
	"github.com/slipstream/slipstream/internal/scheduler/tasks"
	"github.com/slipstream/slipstream/internal/update"
)

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

	// Block proxy probes (absolute URI requests like GET http://www.google.com/)
	s.echo.Use(apimw.ProxyRequestBlock())

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
	s.echo.GET("/health", s.healthCheck)

	api := s.echo.Group("/api/v1")
	api.GET("/status", s.getStatus)

	s.setupAuthRoutes(api)

	protected := api.Group("")
	protected.Use(s.adminAuthMiddleware())
	settings := protected.Group("/settings")
	settings.GET("", s.getSettings)
	settings.PUT("", s.updateSettings)
	settings.POST("/apikey", s.regenerateAPIKey)

	s.setupSystemRoutes(protected)
	s.setupLibraryRoutes(api, protected)
	s.setupIndexerRoutes(protected)
	s.setupDownloadRoutes(protected)
	s.setupMediaRoutes(api, protected)
	s.setupAutomationRoutes(protected, settings)
	s.setupNotificationRoutes(api, protected)
	s.setupSchedulerRoutes(protected)
	s.setupPortalRoutes(api)
}

func (s *Server) setupAuthRoutes(api *echo.Group) {
	authGroup := api.Group("/auth")
	authGroup.Use(s.authLimiter.Middleware())
	authGroup.GET("/status", s.getAuthStatus)
	authGroup.POST("/setup", s.adminSetup)
	authGroup.DELETE("/admin", s.deleteAdmin)
}

func (s *Server) setupSystemRoutes(protected *echo.Group) {
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

	protected.POST("/system/restart", s.restart)
	protected.GET("/system/firewall", s.checkFirewall)

	updateHandlers := update.NewHandlers(s.updateService)
	updateHandlers.RegisterRoutes(protected.Group("/update"))
}

func (s *Server) setupLibraryRoutes(api, protected *echo.Group) {
	movieHandlers := movies.NewHandlers(s.movieService)
	movieHandlers.RegisterRoutes(protected.Group("/movies"))

	tvHandlers := tv.NewHandlers(s.tvService)
	tvHandlers.RegisterRoutes(protected.Group("/series"))

	libraryManagerHandlers := librarymanager.NewHandlers(s.libraryManagerService)
	protected.POST("/movies/refresh", libraryManagerHandlers.RefreshAllMovies)
	protected.POST("/series/refresh", libraryManagerHandlers.RefreshAllSeries)
	protected.POST("/movies/:id/refresh", libraryManagerHandlers.RefreshMovie)
	protected.POST("/series/:id/refresh", libraryManagerHandlers.RefreshSeries)

	libraryGroup := protected.Group("/library")
	libraryGroup.POST("/movies", libraryManagerHandlers.AddMovie)
	libraryGroup.POST("/series", libraryManagerHandlers.AddSeries)

	qualityHandlers := quality.NewHandlers(s.qualityService)
	qualityHandlers.RegisterRoutes(protected.Group("/qualityprofiles"))

	slotsHandlers := slots.NewHandlers(s.slotsService)
	slotsHandlers.RegisterRoutes(protected.Group("/slots"))

	slotsDebugHandlers := slots.NewDebugHandlers(s.slotsService, s.dbManager.IsDevMode)
	slotsDebugHandlers.RegisterDebugRoutes(protected.Group("/slots/debug"))

	rootFolderHandlers := rootfolder.NewHandlers(s.rootFolderService)
	rootFolderHandlers.RegisterRoutes(protected.Group("/rootfolders"))
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

	// Metadata routes
	metadataHandlers := metadata.NewHandlers(s.metadataService, s.artworkDownloader)
	metadataHandlers.RegisterArtworkRoutes(api.Group("/metadata"))

	metadataGroup := api.Group("/metadata")
	metadataGroup.Use(s.portalAuthMiddleware.AnyAuth())
	metadataHandlers.RegisterRoutes(metadataGroup)

	protected.POST("/metadata/tmdb/search-ordering", s.updateTMDBSearchOrdering)

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
}

func (s *Server) setupIndexerRoutes(protected *echo.Group) {
	indexersGroup := protected.Group("/indexers")
	indexerHandlers := indexer.NewHandlers(s.indexerService)
	indexerHandlers.SetStatusService(s.statusService)
	indexerHandlers.RegisterRoutes(indexersGroup)

	prowlarrHandlers := prowlarr.NewHandlers(s.prowlarrService, s.prowlarrModeManager)
	prowlarrHandlers.RegisterRoutes(indexersGroup)
}

func (s *Server) setupDownloadRoutes(protected *echo.Group) {
	clients := protected.Group("/downloadclients")
	clients.GET("", s.listDownloadClients)
	clients.POST("", s.addDownloadClient)
	clients.POST("/test", s.testNewDownloadClient)
	clients.GET("/:id", s.getDownloadClient)
	clients.PUT("/:id", s.updateDownloadClient)
	clients.DELETE("/:id", s.deleteDownloadClient)
	clients.POST("/:id/test", s.testDownloadClient)

	protected.GET("/queue", s.getQueue)
	protected.POST("/queue/:id/pause", s.pauseDownload)
	protected.POST("/queue/:id/resume", s.resumeDownload)
	protected.POST("/queue/:id/fastforward", s.fastForwardDownload)
	protected.DELETE("/queue/:id", s.removeFromQueue)
}

func (s *Server) setupMediaRoutes(api, protected *echo.Group) {
	_ = api // reserved for future shared media routes

	historyHandlers := history.NewHandlers(s.historyService)
	historyHandlers.RegisterRoutes(protected.Group("/history"))
	protected.GET("/history/indexer", s.getIndexerHistory)

	searchHandlers := search.NewHandlers(s.searchRouter, s.qualityService)
	searchHandlers.RegisterRoutes(protected.Group("/search"))

	grabHandlers := grab.NewHandlers(s.grabService)
	grabHandlers.RegisterRoutes(protected.Group("/search"))

	defaultsHandlers := defaults.NewHandlers(s.defaultsService)
	defaultsHandlers.RegisterRoutes(protected.Group("/defaults"))

	preferencesHandlers := preferences.NewHandlers(s.preferencesService)
	preferencesHandlers.RegisterRoutes(protected.Group("/preferences"))

	calendarHandlers := calendar.NewHandlers(s.calendarService)
	calendarHandlers.RegisterRoutes(protected.Group("/calendar"))

	missingHandlers := missing.NewHandlers(s.missingService)
	missingHandlers.RegisterRoutes(protected.Group("/missing"))
}

func (s *Server) setupAutomationRoutes(protected, settings *echo.Group) {
	autosearchHandlers := autosearch.NewHandlers(s.autosearchService)
	autosearchHandlers.SetScheduledSearcher(s.scheduledSearcher)
	autosearchHandlers.RegisterRoutes(protected.Group("/autosearch"))

	autosearchSettings := autosearch.NewSettingsHandler(sqlc.New(s.startupDB), &s.cfg.AutoSearch)
	if s.scheduler != nil {
		autosearchSettings.SetScheduler(s.scheduler, s.scheduledSearcher, tasks.UpdateAutoSearchTask)
	}
	settings.GET("/autosearch", autosearchSettings.GetSettings)
	settings.PUT("/autosearch", autosearchSettings.UpdateSettings)

	rssSyncHandlers := rsssync.NewHandlers(s.rssSyncService)
	rssSyncHandlers.RegisterRoutes(protected.Group("/rsssync"))

	s.rssSyncSettingsHandler = rsssync.NewSettingsHandler(sqlc.New(s.startupDB), &s.cfg.RssSync)
	if s.scheduler != nil {
		s.rssSyncSettingsHandler.SetScheduler(s.scheduler, s.rssSyncService, tasks.UpdateRssSyncTask)
	}
	settings.GET("/rsssync", s.rssSyncSettingsHandler.GetSettings)
	settings.PUT("/rsssync", s.rssSyncSettingsHandler.UpdateSettings)

	importHandlers := importer.NewHandlers(s.importService, s.startupDB)
	importHandlers.RegisterRoutes(protected.Group("/import"))

	arrImportHandlers := arrimport.NewHandlers(s.arrImportService)
	arrImportHandlers.RegisterRoutes(protected.Group("/arrimport"))

	s.importSettingsHandlers = importer.NewSettingsHandlers(s.startupDB, s.importService)
	s.importSettingsHandlers.RegisterSettingsRoutes(settings)
}

func (s *Server) setupNotificationRoutes(api, protected *echo.Group) {
	notificationHandlers := notification.NewHandlers(s.notificationService)
	notificationHandlers.RegisterRoutes(protected.Group("/notifications"))

	s.plexHandlers.RegisterRoutes(protected.Group("/notifications/plex"))

	notificationsShared := api.Group("/notifications")
	notificationsShared.Use(s.portalAuthMiddleware.AnyAuth())
	notificationHandlers.RegisterSharedRoutes(notificationsShared)
}

func (s *Server) setupSchedulerRoutes(protected *echo.Group) {
	if s.scheduler == nil {
		return
	}
	schedulerHandler := handlers.NewSchedulerHandler(s.scheduler)
	schedulerGroup := protected.Group("/scheduler")
	schedulerGroup.GET("/tasks", schedulerHandler.ListTasks)
	schedulerGroup.GET("/tasks/:id", schedulerHandler.GetTask)
	schedulerGroup.POST("/tasks/:id/run", schedulerHandler.RunTask)
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

	// Portal library browse
	libraryHandlers := portallibrary.NewHandlers(s.movieService, s.tvService, s.portalLibraryChecker, s.portalUsersService)
	libraryGroup := requestsGroup.Group("/library")
	libraryGroup.Use(s.portalAuthMiddleware.AnyAuth())
	libraryHandlers.RegisterRoutes(libraryGroup)

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

	s.setupPortalAdminRoutes(api)
}

func (s *Server) setupPortalAdminRoutes(api *echo.Group) {
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
	s.portalRequestSearcher.SetUserGetter(&portalUserQualityProfileAdapter{usersSvc: s.portalUsersService})
	s.portalAutoApproveService.SetRequestSearcher(s.portalRequestSearcher)
	adminRequestHandlers := admin.NewRequestsHandlers(
		s.portalRequestsService,
		&portalRequestSearcherAdapter{searcher: s.portalRequestSearcher},
	)
	adminRequestHandlers.SetLibraryChecker(s.adminRequestLibraryChecker)
	adminRequestHandlers.RegisterRoutes(adminGroup, s.portalAuthMiddleware)

	// Admin settings
	s.adminSettingsHandlers = admin.NewSettingsHandlers(s.portalQuotaService, sqlc.New(s.startupDB))
	s.adminSettingsHandlers.RegisterRoutes(adminGroup.Group("/settings"), s.portalAuthMiddleware)
}
