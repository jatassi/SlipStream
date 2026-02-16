//nolint:revive // Package name 'api' is intentionally generic for the HTTP API layer
package api

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/logger"
)

// LogsProvider provides access to log data.
type LogsProvider interface {
	GetRecentLogs() []logger.LogEntry
	GetLogFilePath() string
}

// LogsHandlers handles log-related HTTP endpoints.
type LogsHandlers struct {
	provider LogsProvider
}

// NewLogsHandlers creates a new logs handlers instance.
func NewLogsHandlers(provider LogsProvider) *LogsHandlers {
	return &LogsHandlers{provider: provider}
}

// RegisterRoutes registers log routes on the given group.
func (h *LogsHandlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.GetRecentLogs)
	g.GET("/download", h.DownloadLogFile)
}

// GetRecentLogs returns recent log entries from the ring buffer.
func (h *LogsHandlers) GetRecentLogs(c echo.Context) error {
	logs := h.provider.GetRecentLogs()
	if logs == nil {
		logs = []logger.LogEntry{}
	}
	return c.JSON(http.StatusOK, logs)
}

// DownloadLogFile serves the current log file for download.
func (h *LogsHandlers) DownloadLogFile(c echo.Context) error {
	logPath := h.provider.GetLogFilePath()
	if logPath == "" {
		return echo.NewHTTPError(http.StatusNotFound, "no log file configured")
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return echo.NewHTTPError(http.StatusNotFound, "log file not found")
	}

	return c.Attachment(logPath, "slipstream.log")
}
