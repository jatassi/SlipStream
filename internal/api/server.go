package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/filesystem"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/websocket"
)

// Server handles HTTP requests for the SlipStream API.
type Server struct {
	echo   *echo.Echo
	db     *sql.DB
	hub    *websocket.Hub
	logger zerolog.Logger

	// Services
	movieService       *movies.Service
	tvService          *tv.Service
	qualityService     *quality.Service
	rootFolderService  *rootfolder.Service
	metadataService    *metadata.Service
	artworkDownloader  *metadata.ArtworkDownloader
	filesystemService  *filesystem.Service
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
	}

	// Initialize services
	s.movieService = movies.NewService(db, hub, logger)
	s.tvService = tv.NewService(db, hub, logger)
	s.qualityService = quality.NewService(db, logger)
	s.rootFolderService = rootfolder.NewService(db, logger)

	// Initialize metadata service and artwork downloader
	s.metadataService = metadata.NewService(cfg.Metadata, logger)
	s.artworkDownloader = metadata.NewArtworkDownloader(metadata.DefaultArtworkConfig(), logger)

	// Initialize filesystem service
	s.filesystemService = filesystem.NewService(logger)

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

	// Quality profiles routes
	qualityHandlers := quality.NewHandlers(s.qualityService)
	qualityHandlers.RegisterRoutes(api.Group("/qualityprofiles"))

	// Root folders routes
	rootFolderHandlers := rootfolder.NewHandlers(s.rootFolderService)
	rootFolderHandlers.RegisterRoutes(api.Group("/rootfolders"))

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
	indexers := api.Group("/indexers")
	indexers.GET("", s.listIndexers)
	indexers.POST("", s.addIndexer)
	indexers.GET("/:id", s.getIndexer)
	indexers.PUT("/:id", s.updateIndexer)
	indexers.DELETE("/:id", s.deleteIndexer)
	indexers.POST("/:id/test", s.testIndexer)

	// Download clients routes
	clients := api.Group("/downloadclients")
	clients.GET("", s.listDownloadClients)
	clients.POST("", s.addDownloadClient)
	clients.GET("/:id", s.getDownloadClient)
	clients.PUT("/:id", s.updateDownloadClient)
	clients.DELETE("/:id", s.deleteDownloadClient)
	clients.POST("/:id/test", s.testDownloadClient)

	// Queue/Downloads routes
	api.GET("/queue", s.getQueue)
	api.DELETE("/queue/:id", s.removeFromQueue)

	// History routes
	api.GET("/history", s.getHistory)

	// Search routes
	api.GET("/search", s.search)
}

// Start begins listening for HTTP requests.
func (s *Server) Start(address string) error {
	s.logger.Info().Str("address", address).Msg("starting HTTP server")
	return s.echo.Start(address)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")
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
		"version":     "0.0.1-dev",
		"startTime":   time.Now().Format(time.RFC3339),
		"movieCount":  movieCount,
		"seriesCount": seriesCount,
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

// Indexer handlers (placeholders)
func (s *Server) listIndexers(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) addIndexer(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) getIndexer(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) updateIndexer(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) deleteIndexer(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) testIndexer(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// Download client handlers (placeholders)
func (s *Server) listDownloadClients(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) addDownloadClient(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) getDownloadClient(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) updateDownloadClient(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) deleteDownloadClient(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) testDownloadClient(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// Queue handlers (placeholders)
func (s *Server) getQueue(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

func (s *Server) removeFromQueue(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// History handler (placeholder)
func (s *Server) getHistory(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}

// Search handler (placeholder)
func (s *Server) search(c echo.Context) error {
	return c.JSON(http.StatusOK, []interface{}{})
}
