package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/api"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/logger"
	"github.com/slipstream/slipstream/internal/platform"
	"github.com/slipstream/slipstream/internal/websocket"
	"github.com/slipstream/slipstream/web"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	noTray := flag.Bool("no-tray", false, "Run without system tray (console mode)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.New(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Path:   cfg.Logging.Path,
	})
	defer log.Close()

	log.Info().
		Str("version", config.Version).
		Str("logLevel", cfg.Logging.Level).
		Msg("starting SlipStream")

	isFirstRun := platform.IsFirstRun(cfg.Database.Path)

	devDBPath := cfg.Database.Path[:len(cfg.Database.Path)-3] + "_dev.db"

	dbManager, err := database.NewManager(cfg.Database.Path, devDBPath, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database manager")
	}
	defer dbManager.Close()

	log.Info().Msg("running database migrations")
	if err := dbManager.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

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

	hub := websocket.NewHub()
	go hub.Run()

	restartChan := make(chan struct{}, 1)
	quitChan := make(chan struct{}, 1)
	var shouldRestart bool

	server := api.NewServer(dbManager, hub, cfg, log.Logger, restartChan)

	if err := server.EnsureDefaults(context.Background()); err != nil {
		log.Warn().Err(err).Msg("failed to ensure defaults")
	}

	server.Echo().GET("/ws", hub.HandleWebSocket)

	if distFS, err := web.DistFS(); err == nil {
		registerFrontendHandler(server.Echo(), distFS)
	}

	go func() {
		addr := cfg.Server.Address()
		log.Info().Str("address", addr).Msg("HTTP server listening")
		if err := server.Start(addr); err != nil {
			log.Info().Msg("server stopped")
		}
	}()

	serverURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)

	app := platform.NewApp(platform.AppConfig{
		ServerURL: serverURL,
		DataPath:  cfg.Database.Path,
		Port:      cfg.Server.Port,
		NoTray:    *noTray || runtime.GOOS == "linux",
		OnQuit: func() {
			close(quitChan)
		},
	})

	if isFirstRun {
		go func() {
			time.Sleep(500 * time.Millisecond)
			if err := app.OpenBrowser(serverURL); err != nil {
				log.Warn().Err(err).Msg("failed to open browser on first run")
			}
		}()
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-sigChan:
			log.Info().Msg("received shutdown signal")
			app.Stop()
		case <-restartChan:
			log.Info().Msg("restarting server...")
			shouldRestart = true
			app.Stop()
		case <-quitChan:
		}
	}()

	if err := app.Run(); err != nil {
		log.Error().Err(err).Msg("platform app error")
	}

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

func registerFrontendHandler(e *echo.Echo, distFS fs.FS) {
	fileServer := http.FileServer(http.FS(distFS))

	e.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path

		if strings.HasPrefix(path, "/api/") || path == "/ws" {
			return echo.ErrNotFound
		}

		if path != "/" {
			cleanPath := strings.TrimPrefix(path, "/")
			if file, err := distFS.Open(cleanPath); err == nil {
				file.Close()
				fileServer.ServeHTTP(c.Response(), c.Request())
				return nil
			}
		}

		indexFile, err := distFS.Open("index.html")
		if err != nil {
			return echo.ErrNotFound
		}
		defer indexFile.Close()

		if _, err := indexFile.Stat(); err != nil {
			return echo.ErrNotFound
		}

		return c.Stream(http.StatusOK, "text/html; charset=utf-8", indexFile)
	})
}
