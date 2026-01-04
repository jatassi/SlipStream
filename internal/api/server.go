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

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/filesystem"
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
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/progress"
	"github.com/slipstream/slipstream/internal/watcher"
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
	libraryManagerService *librarymanager.Service
	watcherService        *watcher.Service
	progressManager       *progress.Manager
	downloaderService     *downloader.Service
	indexerService        *indexer.Service
	searchService         *search.Service
	statusService         *status.Service
	rateLimiter           *ratelimit.Limiter
	grabService           *grab.Service
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

	// Initialize services
	s.scannerService = scanner.NewService(logger)
	s.movieService = movies.NewService(db, hub, logger)
	s.tvService = tv.NewService(db, hub, logger)
	s.qualityService = quality.NewService(db, logger)
	s.rootFolderService = rootfolder.NewService(db, logger)

	// Initialize metadata service and artwork downloader
	s.metadataService = metadata.NewService(cfg.Metadata, logger)
	s.artworkDownloader = metadata.NewArtworkDownloader(metadata.DefaultArtworkConfig(), logger)
	s.artworkDownloader.SetBroadcaster(hub)

	// Initialize filesystem service
	s.filesystemService = filesystem.NewService(logger)

	// Initialize downloader service
	s.downloaderService = downloader.NewService(db, logger)
	s.downloaderService.SetDeveloperMode(cfg.DeveloperMode)

	// Initialize Cardigann manager for indexer definitions
	// Note: Definitions are fetched lazily when the user opens the Add Indexer dialog
	cardigannManager, err := cardigann.NewManager(cardigann.DefaultManagerConfig(), logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize Cardigann manager")
	}

	// Initialize indexer service
	s.indexerService = indexer.NewService(db, cardigannManager, logger)

	// Initialize indexer status service
	s.statusService = status.NewService(db, logger)

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

	// Initialize watcher service for real-time file monitoring
	watcherSvc, err := watcher.NewService(s.rootFolderService, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize watcher service")
	} else {
		s.watcherService = watcherSvc
		// Set file processor to use library manager
		s.watcherService.SetFileProcessor(func(ctx context.Context, filePath string) error {
			return s.libraryManagerService.ScanSingleFile(ctx, filePath)
		})
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

	// Filesystem routes (for folder browsing)
	filesystemHandlers := filesystem.NewHandlers(s.filesystemService)
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
	api.GET("/history", s.getHistory)
	api.GET("/history/indexer", s.getIndexerHistory)

	// Search routes
	searchHandlers := search.NewHandlers(s.searchService)
	searchHandlers.RegisterRoutes(api.Group("/search"))

	// Grab routes (under /search for grabbing search results)
	grabHandlers := grab.NewHandlers(s.grabService)
	grabHandlers.RegisterRoutes(api.Group("/search"))
}

// Start begins listening for HTTP requests.
func (s *Server) Start(address string) error {
	s.logger.Info().Str("address", address).Msg("starting HTTP server")

	// Start the file watcher service
	if s.watcherService != nil {
		if err := s.watcherService.Start(context.Background()); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to start watcher service")
		}
	}

	return s.echo.Start(address)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")

	// Stop the file watcher service
	if s.watcherService != nil {
		if err := s.watcherService.Stop(); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to stop watcher service")
		}
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

	return c.NoContent(http.StatusNoContent)
}

// History handler (placeholder)
func (s *Server) getHistory(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
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
