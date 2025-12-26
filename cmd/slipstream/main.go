package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/slipstream/slipstream/internal/api"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/logger"
	"github.com/slipstream/slipstream/internal/websocket"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger
	log := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	log.Info().
		Str("version", "0.0.1-dev").
		Msg("starting SlipStream")

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer db.Close()

	// Run migrations
	log.Info().Msg("running database migrations")
	if err := db.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize API server
	server := api.NewServer(db.Conn(), hub, cfg, log.Logger)

	// Ensure default data exists (like quality profiles)
	if err := server.EnsureDefaults(context.Background()); err != nil {
		log.Warn().Err(err).Msg("failed to ensure defaults")
	}

	// Register WebSocket endpoint
	server.Echo().GET("/ws", hub.HandleWebSocket)

	// Start server in goroutine
	go func() {
		addr := cfg.Server.Address()
		log.Info().Str("address", addr).Msg("HTTP server listening")
		if err := server.Start(addr); err != nil {
			log.Info().Msg("server stopped")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server shutdown error")
	}

	log.Info().Msg("server stopped")
}
