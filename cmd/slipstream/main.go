package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
	"github.com/slipstream/slipstream/internal/startup"
	"github.com/slipstream/slipstream/internal/websocket"
	"github.com/slipstream/slipstream/web"
)

// bootstrapLog writes early diagnostic messages to a file before the main logger is initialized.
// This helps diagnose startup failures on Windows where GUI apps have no console output.
func bootstrapLog(msg string) {
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

	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, "bootstrap.log")

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] %s\n", timestamp, msg)
}

func main() {
	// Handle --complete-update before anything else (used by self-updater on all platforms)
	if len(os.Args) >= 4 && os.Args[1] == "--complete-update" {
		port := 0
		if p, err := strconv.Atoi(os.Args[3]); err == nil {
			port = p
		}
		completeUpdate(os.Args[2], port)
		return
	}

	// Lock the main goroutine to the main OS thread.
	// This is required for macOS where UI elements (NSWindow, NSApplication)
	// must be created and manipulated on the main thread.
	runtime.LockOSThread()

	bootstrapLog("=== SlipStream starting ===")
	bootstrapLog(fmt.Sprintf("OS: %s, Arch: %s", runtime.GOOS, runtime.GOARCH))
	bootstrapLog(fmt.Sprintf("Executable: %s", os.Args[0]))
	bootstrapLog(fmt.Sprintf("Working directory: %s", func() string { wd, _ := os.Getwd(); return wd }()))

	configPath := flag.String("config", "", "Path to config file")
	noTray := flag.Bool("no-tray", false, "Run without system tray (console mode)")
	flag.Parse()

	bootstrapLog(fmt.Sprintf("Flags parsed: config=%q, no-tray=%v", *configPath, *noTray))
	bootstrapLog("Loading configuration...")

	cfg, err := config.Load(*configPath)
	if err != nil {
		bootstrapLog(fmt.Sprintf("FATAL: failed to load config: %v", err))
		panic("failed to load config: " + err.Error())
	}
	bootstrapLog(fmt.Sprintf("Config loaded: port=%d, db=%s, logPath=%s", cfg.Server.Port, cfg.Database.Path, cfg.Logging.Path))

	bootstrapLog("Initializing logger...")
	log := logger.New(logger.Config{
		Level:           cfg.Logging.Level,
		Format:          cfg.Logging.Format,
		Path:            cfg.Logging.Path,
		MaxSizeMB:       cfg.Logging.MaxSizeMB,
		MaxBackups:      cfg.Logging.MaxBackups,
		MaxAgeDays:      cfg.Logging.MaxAgeDays,
		Compress:        cfg.Logging.Compress,
		EnableStreaming: true,
		BufferSize:      1000,
	})
	defer log.Close()
	bootstrapLog("Logger initialized")

	log.Info().
		Str("version", config.Version).
		Str("logLevel", cfg.Logging.Level).
		Msg("starting SlipStream")

	isFirstRun := platform.IsFirstRun(cfg.Database.Path)
	bootstrapLog(fmt.Sprintf("First run: %v", isFirstRun))

	devDBPath := cfg.Database.Path[:len(cfg.Database.Path)-3] + "_dev.db"

	bootstrapLog("Initializing database manager...")
	dbManager, err := database.NewManager(cfg.Database.Path, devDBPath, log.Logger)
	if err != nil {
		bootstrapLog(fmt.Sprintf("FATAL: failed to initialize database manager: %v", err))
		log.Fatal().Err(err).Msg("failed to initialize database manager")
	}
	defer dbManager.Close()
	bootstrapLog("Database manager initialized")

	bootstrapLog("Running database migrations...")
	log.Info().Msg("running database migrations")
	if err := dbManager.Migrate(); err != nil {
		bootstrapLog(fmt.Sprintf("FATAL: failed to run migrations: %v", err))
		log.Fatal().Err(err).Msg("failed to run migrations")
	}
	bootstrapLog("Database migrations complete")

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
	if setting, err := queries.GetSetting(context.Background(), "external_access_enabled"); err == nil && setting.Value == "true" {
		cfg.Server.Host = "0.0.0.0"
		log.Info().Msg("external access enabled, binding to all interfaces")
	}

	bootstrapLog(fmt.Sprintf("Finding available port starting from %d...", cfg.Server.Port))
	configuredPort := cfg.Server.Port
	actualPort, err := config.FindAvailablePort(cfg.Server.Port, 10)
	if err != nil {
		bootstrapLog(fmt.Sprintf("FATAL: failed to find available port: %v", err))
		log.Fatal().Err(err).Int("configuredPort", cfg.Server.Port).Msg("failed to find available port")
	}
	if actualPort != configuredPort {
		log.Warn().
			Int("configuredPort", configuredPort).
			Int("actualPort", actualPort).
			Msg("configured port in use, using alternative port")
		cfg.Server.Port = actualPort
	}
	bootstrapLog(fmt.Sprintf("Using port %d", cfg.Server.Port))

	hub := websocket.NewHub()
	go hub.Run()

	// Enable log streaming via WebSocket now that hub is available
	log.SetBroadcastHub(hub)

	restartChan := make(chan bool, 1)
	quitChan := make(chan struct{}, 1)
	var shouldRestart bool

	bootstrapLog("Creating API server...")
	server := api.NewServer(dbManager, hub, cfg, log.Logger, restartChan)
	server.SetConfiguredPort(configuredPort)
	server.SetLogsProvider(log)

	if err := server.EnsureDefaults(context.Background()); err != nil {
		log.Warn().Err(err).Msg("failed to ensure defaults")
	}
	bootstrapLog("API server created")

	bootstrapLog("Initializing network services...")
	retryCfg := startup.DefaultRetryConfig()
	err = startup.WithRetry(
		context.Background(),
		"network services initialization",
		retryCfg,
		func() error {
			return server.InitializeNetworkServices(context.Background())
		},
		log.Logger,
	)
	if err != nil {
		bootstrapLog(fmt.Sprintf("Warning: network services initialization failed: %v", err))
		log.Warn().Err(err).Msg("failed to initialize network services, some features may be unavailable until network is restored")
	} else {
		bootstrapLog("Network services initialized")
	}

	server.Echo().GET("/ws", hub.HandleWebSocket)

	if distFS, err := web.DistFS(); err == nil {
		registerFrontendHandler(server.Echo(), distFS)
		bootstrapLog("Frontend handler registered")
	} else {
		bootstrapLog(fmt.Sprintf("Warning: failed to get frontend dist FS: %v", err))
	}

	bootstrapLog("Starting HTTP server...")
	go func() {
		addr := cfg.Server.Address()
		bootstrapLog(fmt.Sprintf("HTTP server listening on %s", addr))
		log.Info().Str("address", addr).Msg("HTTP server listening")
		if err := server.Start(addr); err != nil {
			bootstrapLog(fmt.Sprintf("HTTP server stopped: %v", err))
			log.Info().Msg("server stopped")
		}
	}()

	serverURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	bootstrapLog(fmt.Sprintf("Server URL: %s", serverURL))

	bootstrapLog("Creating platform app...")
	app := platform.NewApp(platform.AppConfig{
		ServerURL: serverURL,
		DataPath:  cfg.Database.Path,
		Port:      cfg.Server.Port,
		NoTray:    *noTray || runtime.GOOS == "linux",
		OnQuit: func() {
			close(quitChan)
		},
	})
	bootstrapLog(fmt.Sprintf("Platform app created (NoTray=%v)", *noTray || runtime.GOOS == "linux"))

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
			bootstrapLog("Received shutdown signal")
			log.Info().Msg("received shutdown signal")
			app.Stop()
		case spawnNew := <-restartChan:
			bootstrapLog("Restart requested")
			log.Info().Msg("restarting server...")
			shouldRestart = spawnNew // Only spawn new process if not an update
			app.Stop()
		case <-quitChan:
			bootstrapLog("Quit channel closed")
		}
	}()

	bootstrapLog("Calling app.Run() - entering main loop...")
	if err := app.Run(); err != nil {
		bootstrapLog(fmt.Sprintf("Platform app error: %v", err))
		log.Error().Err(err).Msg("platform app error")
	}
	bootstrapLog("app.Run() returned, shutting down...")

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

func completeUpdate(targetPath string, port int) {
	bootstrapLog("=== Update completion starting ===")
	bootstrapLog(fmt.Sprintf("Target path: %s", targetPath))
	bootstrapLog(fmt.Sprintf("Port to wait for: %d", port))

	currentExe, err := os.Executable()
	if err != nil {
		bootstrapLog(fmt.Sprintf("Failed to get current executable: %v", err))
		os.Exit(1)
	}
	currentExe, _ = filepath.EvalSymlinks(currentExe)
	bootstrapLog(fmt.Sprintf("Current executable: %s", currentExe))

	// Wait for the old process to exit by polling the port
	bootstrapLog("Waiting for old process to release port...")
	if !waitForPortFree(port, 60*time.Second) {
		bootstrapLog("Warning: Timed out waiting for port to be free, proceeding anyway")
	} else {
		bootstrapLog("Port is free, old process has exited")
	}

	// Determine if we're updating an app bundle (macOS) or a single file (Windows/Linux)
	isAppBundle := strings.HasSuffix(targetPath, ".app")

	if isAppBundle {
		// macOS: Copy the entire app bundle
		bootstrapLog("Copying new app bundle to target location...")

		// Find the .app bundle containing the current executable
		currentAppBundle := currentExe
		for !strings.HasSuffix(currentAppBundle, ".app") && currentAppBundle != "/" {
			currentAppBundle = filepath.Dir(currentAppBundle)
		}
		if !strings.HasSuffix(currentAppBundle, ".app") {
			bootstrapLog(fmt.Sprintf("Failed to find app bundle for: %s", currentExe))
			os.Exit(1)
		}

		// Remove old app bundle and copy new one
		if err := os.RemoveAll(targetPath); err != nil {
			bootstrapLog(fmt.Sprintf("Failed to remove old app bundle: %v", err))
			os.Exit(1)
		}

		cmd := exec.Command("cp", "-R", currentAppBundle, targetPath)
		if err := cmd.Run(); err != nil {
			bootstrapLog(fmt.Sprintf("Failed to copy app bundle: %v", err))
			os.Exit(1)
		}
		bootstrapLog("App bundle copied successfully")

		// Launch the updated application
		newExePath := filepath.Join(targetPath, "Contents", "MacOS", "slipstream")
		bootstrapLog(fmt.Sprintf("Launching updated application: %s", newExePath))
		cmd = exec.Command(newExePath)
		cmd.Dir = filepath.Dir(targetPath)
		if err := cmd.Start(); err != nil {
			bootstrapLog(fmt.Sprintf("Failed to launch updated application: %v", err))
			os.Exit(1)
		}
		bootstrapLog(fmt.Sprintf("Updated application launched (PID: %d)", cmd.Process.Pid))
	} else {
		// Windows/Linux: Copy single executable
		bootstrapLog("Copying new executable to target location...")

		// On Windows, try to rename the old file first (will fail if still locked)
		if runtime.GOOS == "windows" {
			oldExePath := targetPath + ".old"
			if err := os.Rename(targetPath, oldExePath); err == nil {
				bootstrapLog("Old executable renamed successfully")
				os.Remove(oldExePath)
			}
		} else {
			// On Linux, just remove the old file
			os.Remove(targetPath)
		}

		if err := copyFile(currentExe, targetPath); err != nil {
			bootstrapLog(fmt.Sprintf("Failed to copy executable: %v", err))
			os.Exit(1)
		}
		bootstrapLog("Executable copied successfully")

		// Launch the updated application
		bootstrapLog("Launching updated application...")
		cmd := exec.Command(targetPath)
		cmd.Dir = filepath.Dir(targetPath)
		if err := cmd.Start(); err != nil {
			bootstrapLog(fmt.Sprintf("Failed to launch updated application: %v", err))
			os.Exit(1)
		}
		bootstrapLog(fmt.Sprintf("Updated application launched (PID: %d)", cmd.Process.Pid))
	}

	// Clean up temp files
	scheduleCleanup(currentExe)

	bootstrapLog("Update complete, exiting updater")
	os.Exit(0)
}

func waitForPortFree(port int, timeout time.Duration) bool {
	if port <= 0 {
		return true
	}
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err != nil {
			// Connection refused = port is free
			return true
		}
		conn.Close()
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func scheduleCleanup(currentExe string) {
	tempDir := filepath.Dir(currentExe)

	// Only clean up if we're in a temp/update directory
	if !strings.Contains(tempDir, "slipstream-update") {
		return
	}

	bootstrapLog(fmt.Sprintf("Scheduling cleanup of temp directory: %s", tempDir))

	switch runtime.GOOS {
	case "windows":
		// On Windows, use cmd /c with a delay to delete after this process exits
		cleanupCmd := exec.Command("cmd", "/c", "timeout", "/t", "5", "/nobreak", ">nul", "&&", "rd", "/s", "/q", tempDir)
		cleanupCmd.Start()
	case "darwin", "linux":
		// On Unix, we can delete the directory in a background process
		cleanupCmd := exec.Command("sh", "-c", fmt.Sprintf("sleep 5 && rm -rf '%s'", tempDir))
		cleanupCmd.Start()
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	// Preserve executable permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	return os.Chmod(dst, sourceInfo.Mode())
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
