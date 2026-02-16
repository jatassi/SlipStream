package rsssync

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const settingsKey = "rsssync_settings"

// Settings represents user-configurable RSS sync settings.
type Settings struct {
	Enabled     bool `json:"enabled"`
	IntervalMin int  `json:"intervalMin"`
}

// ScheduleUpdater is a function that updates the RSS sync task schedule.
type ScheduleUpdater func(sched *scheduler.Scheduler, service *Service, cfg *config.RssSyncConfig) error

// SettingsHandler provides HTTP handlers for RSS sync settings.
type SettingsHandler struct {
	queries         *sqlc.Queries
	config          *config.RssSyncConfig
	scheduler       *scheduler.Scheduler
	service         *Service
	scheduleUpdater ScheduleUpdater
}

// NewSettingsHandler creates a new settings handler.
func NewSettingsHandler(queries *sqlc.Queries, cfg *config.RssSyncConfig) *SettingsHandler {
	return &SettingsHandler{
		queries: queries,
		config:  cfg,
	}
}

// SetDB updates the database connection (for dev mode switching).
func (h *SettingsHandler) SetDB(queries *sqlc.Queries) {
	h.queries = queries
}

// SetScheduler sets the scheduler and service for dynamic task updates.
func (h *SettingsHandler) SetScheduler(sched *scheduler.Scheduler, service *Service, updater ScheduleUpdater) {
	h.scheduler = sched
	h.service = service
	h.scheduleUpdater = updater
}

// GetSettings returns current RSS sync settings.
// GET /api/v1/settings/rsssync
func (h *SettingsHandler) GetSettings(c echo.Context) error {
	ctx := c.Request().Context()

	settings := h.loadSettings(ctx)
	return c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates RSS sync settings.
// PUT /api/v1/settings/rsssync
func (h *SettingsHandler) UpdateSettings(c echo.Context) error {
	ctx := c.Request().Context()

	var input Settings
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if input.IntervalMin < 10 || input.IntervalMin > 120 {
		return echo.NewHTTPError(http.StatusBadRequest, "intervalMin must be between 10 and 120")
	}

	if err := h.saveSettings(ctx, &input); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	h.config.Enabled = input.Enabled
	h.config.IntervalMin = input.IntervalMin

	if h.scheduler != nil && h.scheduleUpdater != nil {
		if err := h.scheduleUpdater(h.scheduler, h.service, h.config); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update schedule: "+err.Error())
		}
	}

	return c.JSON(http.StatusOK, input)
}

func (h *SettingsHandler) loadSettings(ctx context.Context) *Settings {
	row, err := h.queries.GetSetting(ctx, settingsKey)
	if err != nil {
		return &Settings{
			Enabled:     h.config.Enabled,
			IntervalMin: h.config.IntervalMin,
		}
	}

	var settings Settings
	if err := json.Unmarshal([]byte(row.Value), &settings); err != nil {
		return &Settings{
			Enabled:     h.config.Enabled,
			IntervalMin: h.config.IntervalMin,
		}
	}

	return &settings
}

func (h *SettingsHandler) saveSettings(ctx context.Context, settings *Settings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	_, err = h.queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   settingsKey,
		Value: string(data),
	})
	return err
}

// LoadSettingsIntoConfig loads saved RSS sync settings from database into config at startup.
func LoadSettingsIntoConfig(ctx context.Context, queries *sqlc.Queries, cfg *config.RssSyncConfig) error {
	row, err := queries.GetSetting(ctx, settingsKey)
	if err == nil {
		var settings Settings
		if err = json.Unmarshal([]byte(row.Value), &settings); err == nil {
			cfg.Enabled = settings.Enabled
			cfg.IntervalMin = settings.IntervalMin
		}
	}
	return nil
}
