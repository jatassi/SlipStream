package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// --- Handler implementations ---

func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getStatus(c echo.Context) error {
	ctx := c.Request().Context()

	movieCount, _ := s.movieService.Count(ctx)
	seriesCount, _ := s.tvService.Count(ctx)

	adminExists, _ := s.portalUsersService.AdminExists(ctx)

	// Check if portal is enabled (defaults to true if not set)
	portalEnabled := true
	queries := sqlc.New(s.startupDB)
	if setting, err := queries.GetSetting(ctx, "requests_portal_enabled"); err == nil {
		portalEnabled = setting.Value != "0" && setting.Value != "false"
	}

	response := map[string]interface{}{
		"version":            config.Version,
		"startTime":          time.Now().Format(time.RFC3339),
		"movieCount":         movieCount,
		"seriesCount":        seriesCount,
		"developerMode":      s.dbManager.IsDevMode(),
		"portalEnabled":      portalEnabled,
		"requiresSetup":      !adminExists,
		"requiresAuth":       true,
		"actualPort":         s.cfg.Server.Port,
		"mediainfoAvailable": s.mediainfoService.IsAvailable(),
		"tmdb": map[string]interface{}{
			"disableSearchOrdering": s.cfg.Metadata.TMDB.DisableSearchOrdering,
		},
	}
	if s.configuredPort > 0 && s.configuredPort != s.cfg.Server.Port {
		response["configuredPort"] = s.configuredPort
	}
	return c.JSON(http.StatusOK, response)
}

// UpdateTMDBSearchOrdering toggles search ordering for TMDB.
// POST /api/v1/metadata/tmdb/search-ordering
func (s *Server) updateTMDBSearchOrdering(c echo.Context) error {
	if !s.dbManager.IsDevMode() {
		return echo.NewHTTPError(http.StatusForbidden, "debug features require developer mode")
	}

	var request struct {
		DisableSearchOrdering bool `json:"disableSearchOrdering"`
	}

	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Update configuration
	s.cfg.Metadata.TMDB.DisableSearchOrdering = request.DisableSearchOrdering

	s.logger.Info().
		Bool("disableSearchOrdering", request.DisableSearchOrdering).
		Msg("TMDB search ordering setting updated")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"disableSearchOrdering": s.cfg.Metadata.TMDB.DisableSearchOrdering,
	})
}

func (s *Server) getSettings(c echo.Context) error {
	ctx := c.Request().Context()
	queries := sqlc.New(s.dbManager.Conn())

	return c.JSON(http.StatusOK, map[string]interface{}{
		"serverPort":            s.getServerPort(ctx, queries),
		"logLevel":              s.getLogLevel(ctx, queries),
		"apiKey":                s.getAPIKey(ctx, queries),
		"logPath":               s.getLogPath(),
		"logMaxSizeMB":          s.getLogMaxSizeMB(ctx, queries),
		"logMaxBackups":         s.getLogMaxBackups(ctx, queries),
		"logMaxAgeDays":         s.getLogMaxAgeDays(ctx, queries),
		"logCompress":           s.getLogCompress(ctx, queries),
		"externalAccessEnabled": s.getExternalAccessEnabled(ctx, queries),
	})
}

func (s *Server) getServerPort(ctx context.Context, queries *sqlc.Queries) int {
	serverPort := s.cfg.Server.Port
	if setting, err := queries.GetSetting(ctx, "server_port"); err == nil {
		if port, err := strconv.Atoi(setting.Value); err == nil {
			serverPort = port
		}
	}
	return serverPort
}

func (s *Server) getLogLevel(ctx context.Context, queries *sqlc.Queries) string {
	logLevel := s.cfg.Logging.Level
	if setting, err := queries.GetSetting(ctx, "log_level"); err == nil {
		logLevel = setting.Value
	}
	return logLevel
}

func (s *Server) getAPIKey(ctx context.Context, queries *sqlc.Queries) string {
	apiKey := ""
	if setting, err := queries.GetSetting(ctx, "api_key"); err == nil {
		apiKey = setting.Value
	}
	return apiKey
}

func (s *Server) getLogPath() string {
	logPath := s.cfg.Logging.Path
	if logPath != "" {
		if absPath, err := filepath.Abs(logPath); err == nil {
			logPath = absPath
		}
	}
	return logPath
}

func (s *Server) getLogMaxSizeMB(ctx context.Context, queries *sqlc.Queries) int {
	logMaxSizeMB := s.cfg.Logging.MaxSizeMB
	if logMaxSizeMB <= 0 {
		logMaxSizeMB = 10
	}
	if setting, err := queries.GetSetting(ctx, "log_max_size_mb"); err == nil {
		if v, err := strconv.Atoi(setting.Value); err == nil {
			logMaxSizeMB = v
		}
	}
	return logMaxSizeMB
}

func (s *Server) getLogMaxBackups(ctx context.Context, queries *sqlc.Queries) int {
	logMaxBackups := s.cfg.Logging.MaxBackups
	if logMaxBackups <= 0 {
		logMaxBackups = 5
	}
	if setting, err := queries.GetSetting(ctx, "log_max_backups"); err == nil {
		if v, err := strconv.Atoi(setting.Value); err == nil {
			logMaxBackups = v
		}
	}
	return logMaxBackups
}

func (s *Server) getLogMaxAgeDays(ctx context.Context, queries *sqlc.Queries) int {
	logMaxAgeDays := s.cfg.Logging.MaxAgeDays
	if logMaxAgeDays <= 0 {
		logMaxAgeDays = 30
	}
	if setting, err := queries.GetSetting(ctx, "log_max_age_days"); err == nil {
		if v, err := strconv.Atoi(setting.Value); err == nil {
			logMaxAgeDays = v
		}
	}
	return logMaxAgeDays
}

func (s *Server) getLogCompress(ctx context.Context, queries *sqlc.Queries) bool {
	logCompress := s.cfg.Logging.Compress
	if setting, err := queries.GetSetting(ctx, "log_compress"); err == nil {
		logCompress = setting.Value == queryTrue
	}
	return logCompress
}

func (s *Server) getExternalAccessEnabled(ctx context.Context, queries *sqlc.Queries) bool {
	externalAccessEnabled := false
	if setting, err := queries.GetSetting(ctx, "external_access_enabled"); err == nil {
		externalAccessEnabled = setting.Value == queryTrue
	}
	return externalAccessEnabled
}

func (s *Server) updateSettings(c echo.Context) error {
	ctx := c.Request().Context()
	queries := sqlc.New(s.dbManager.Conn())

	var input struct {
		ServerPort            *int    `json:"serverPort"`
		LogLevel              *string `json:"logLevel"`
		LogMaxSizeMB          *int    `json:"logMaxSizeMB"`
		LogMaxBackups         *int    `json:"logMaxBackups"`
		LogMaxAgeDays         *int    `json:"logMaxAgeDays"`
		LogCompress           *bool   `json:"logCompress"`
		ExternalAccessEnabled *bool   `json:"externalAccessEnabled"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if err := s.updateServerPort(ctx, queries, input.ServerPort); err != nil {
		return err
	}
	if err := s.updateLogLevel(ctx, queries, input.LogLevel); err != nil {
		return err
	}
	if err := s.updateLogMaxSizeMB(ctx, queries, input.LogMaxSizeMB); err != nil {
		return err
	}
	if err := s.updateLogMaxBackups(ctx, queries, input.LogMaxBackups); err != nil {
		return err
	}
	if err := s.updateLogMaxAgeDays(ctx, queries, input.LogMaxAgeDays); err != nil {
		return err
	}
	if err := s.updateLogCompress(ctx, queries, input.LogCompress); err != nil {
		return err
	}
	if err := s.updateExternalAccessEnabled(ctx, queries, input.ExternalAccessEnabled); err != nil {
		return err
	}

	return s.getSettings(c)
}

func (s *Server) updateServerPort(ctx context.Context, queries *sqlc.Queries, serverPort *int) error {
	if serverPort == nil {
		return nil
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "server_port",
		Value: strconv.Itoa(*serverPort),
	})
	return err
}

func (s *Server) updateLogLevel(ctx context.Context, queries *sqlc.Queries, logLevel *string) error {
	if logLevel == nil {
		return nil
	}
	validLevels := map[string]bool{"trace": true, "debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[*logLevel] {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid log level")
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "log_level",
		Value: *logLevel,
	})
	return err
}

func (s *Server) updateLogMaxSizeMB(ctx context.Context, queries *sqlc.Queries, logMaxSizeMB *int) error {
	if logMaxSizeMB == nil {
		return nil
	}
	if *logMaxSizeMB < 1 || *logMaxSizeMB > 100 {
		return echo.NewHTTPError(http.StatusBadRequest, "log max size must be between 1 and 100 MB")
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "log_max_size_mb",
		Value: strconv.Itoa(*logMaxSizeMB),
	})
	return err
}

func (s *Server) updateLogMaxBackups(ctx context.Context, queries *sqlc.Queries, logMaxBackups *int) error {
	if logMaxBackups == nil {
		return nil
	}
	if *logMaxBackups < 1 || *logMaxBackups > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "log max backups must be between 1 and 20")
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "log_max_backups",
		Value: strconv.Itoa(*logMaxBackups),
	})
	return err
}

func (s *Server) updateLogMaxAgeDays(ctx context.Context, queries *sqlc.Queries, logMaxAgeDays *int) error {
	if logMaxAgeDays == nil {
		return nil
	}
	if *logMaxAgeDays < 1 || *logMaxAgeDays > 365 {
		return echo.NewHTTPError(http.StatusBadRequest, "log max age must be between 1 and 365 days")
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "log_max_age_days",
		Value: strconv.Itoa(*logMaxAgeDays),
	})
	return err
}

func (s *Server) updateLogCompress(ctx context.Context, queries *sqlc.Queries, logCompress *bool) error {
	if logCompress == nil {
		return nil
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "log_compress",
		Value: strconv.FormatBool(*logCompress),
	})
	return err
}

func (s *Server) updateExternalAccessEnabled(ctx context.Context, queries *sqlc.Queries, externalAccessEnabled *bool) error {
	if externalAccessEnabled == nil {
		return nil
	}
	_, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "external_access_enabled",
		Value: strconv.FormatBool(*externalAccessEnabled),
	})
	return err
}

func (s *Server) regenerateAPIKey(c echo.Context) error {
	ctx := c.Request().Context()
	queries := sqlc.New(s.dbManager.Conn())

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate API key"})
	}
	apiKey := hex.EncodeToString(bytes)

	if _, err := queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "api_key",
		Value: apiKey,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"apiKey": apiKey})
}

func (s *Server) restart(c echo.Context) error {
	s.logger.Info().Msg("Restart requested via API")

	select {
	case s.restartChan <- true: // true = spawn new process after shutdown
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Restart initiated",
		})
	default:
		return echo.NewHTTPError(http.StatusConflict, "Restart already in progress")
	}
}

func (s *Server) checkFirewall(c echo.Context) error {
	ctx := c.Request().Context()

	// Get the configured server port
	port := s.cfg.Server.Port
	queries := sqlc.New(s.dbManager.Conn())
	if setting, err := queries.GetSetting(ctx, "server_port"); err == nil {
		if p, err := strconv.Atoi(setting.Value); err == nil {
			port = p
		}
	}

	firewallStatus, err := s.firewallChecker.CheckPort(ctx, port)
	if err != nil {
		s.logger.Warn().Err(err).Int("port", port).Msg("Failed to check firewall status")
	}

	return c.JSON(http.StatusOK, firewallStatus)
}
