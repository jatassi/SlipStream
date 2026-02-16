package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

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
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] [NewServer] %s\n", timestamp, msg)
}

// InitializeNetworkServices initializes services that require network connectivity.
// This should be called with retry logic to handle network unavailability at startup.
func (s *Server) InitializeNetworkServices(ctx context.Context) error {
	return s.indexerService.InitializeDefinitions(ctx)
}
