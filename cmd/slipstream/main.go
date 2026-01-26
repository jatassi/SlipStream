package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/slipstream/slipstream/internal/api"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/database/sqlc"
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
		Path:   cfg.Logging.Path,
	})
	defer log.Close()

	log.Info().
		Str("version", "0.0.1-dev").
		Str("logLevel", cfg.Logging.Level).
		Msg("starting SlipStream")

	// Derive dev database path from production path
	devDBPath := cfg.Database.Path[:len(cfg.Database.Path)-3] + "_dev.db"

	// Initialize database manager
	dbManager, err := database.NewManager(cfg.Database.Path, devDBPath, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database manager")
	}
	defer dbManager.Close()

	// Run migrations on production database
	log.Info().Msg("running database migrations")
	if err := dbManager.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Load settings from database and override config
	queries := sqlc.New(dbManager.Conn())
	if setting, err := queries.GetSetting(context.Background(), "server_port"); err == nil {
		if port, err := strconv.Atoi(setting.Value); err == nil {
			cfg.Server.Port = port
			log.Info().Int("port", port).Msg("loaded server port from database")
		}
	}
	if setting, err := queries.GetSetting(context.Background(), "log_level"); err == nil && setting.Value != "" {
		cfg.Logging.Level = setting.Value
		log.Info().Str("level", setting.Value).Msg("loaded log level from database")
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create restart channel
	restartChan := make(chan struct{}, 1)

	// Initialize API server
	server := api.NewServer(dbManager, hub, cfg, log.Logger, restartChan)

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

	// Wait for interrupt signal or restart request
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var shouldRestart bool
	select {
	case <-quit:
		log.Info().Msg("shutting down server...")
	case <-restartChan:
		log.Info().Msg("restarting server...")
		shouldRestart = true
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server shutdown error")
	}

	if shouldRestart {
		if err := spawnNewProcess(); err != nil {
			log.Error().Err(err).Msg("failed to spawn new process")
		}
	}

	log.Info().Msg("server stopped")
}

func spawnNewProcess() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Start()
}
