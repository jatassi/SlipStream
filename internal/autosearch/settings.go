package autosearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/scheduler"
)

const settingsKey = "autosearch_settings"

// Settings represents user-configurable autosearch settings.
type Settings struct {
	Enabled          bool `json:"enabled"`
	IntervalHours    int  `json:"intervalHours"`
	BackoffThreshold int  `json:"backoffThreshold"`
}

// ScheduleUpdater is a function that updates the autosearch task schedule.
type ScheduleUpdater func(sched *scheduler.Scheduler, searcher *ScheduledSearcher, cfg *config.AutoSearchConfig) error

// SettingsHandler provides HTTP handlers for autosearch settings.
type SettingsHandler struct {
	queries         *sqlc.Queries
	config          *config.AutoSearchConfig
	scheduler       *scheduler.Scheduler
	searcher        *ScheduledSearcher
	scheduleUpdater ScheduleUpdater
}

// NewSettingsHandler creates a new settings handler.
func NewSettingsHandler(queries *sqlc.Queries, cfg *config.AutoSearchConfig) *SettingsHandler {
	return &SettingsHandler{
		queries: queries,
		config:  cfg,
	}
}

// SetScheduler sets the scheduler and searcher for dynamic task updates.
func (h *SettingsHandler) SetScheduler(sched *scheduler.Scheduler, searcher *ScheduledSearcher, updater ScheduleUpdater) {
	h.scheduler = sched
	h.searcher = searcher
	h.scheduleUpdater = updater
}

// GetSettings returns current autosearch settings.
// GET /api/v1/settings/autosearch
func (h *SettingsHandler) GetSettings(c echo.Context) error {
	ctx := c.Request().Context()

	settings, err := h.loadSettings(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates autosearch settings.
// PUT /api/v1/settings/autosearch
func (h *SettingsHandler) UpdateSettings(c echo.Context) error {
	ctx := c.Request().Context()

	var input Settings
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate interval range
	if input.IntervalHours < 1 || input.IntervalHours > 24 {
		return echo.NewHTTPError(http.StatusBadRequest, "intervalHours must be between 1 and 24")
	}

	// Validate backoff threshold
	if input.BackoffThreshold < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "backoffThreshold must be at least 1")
	}

	// Save to database
	if err := h.saveSettings(ctx, &input); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Update in-memory config
	h.config.Enabled = input.Enabled
	h.config.IntervalHours = input.IntervalHours
	h.config.BackoffThreshold = input.BackoffThreshold

	// Update the scheduler task dynamically
	if h.scheduler != nil && h.scheduleUpdater != nil {
		if err := h.scheduleUpdater(h.scheduler, h.searcher, h.config); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to update schedule: "+err.Error())
		}
	}

	return c.JSON(http.StatusOK, input)
}

// loadSettings loads settings from database, falling back to config defaults.
func (h *SettingsHandler) loadSettings(ctx context.Context) (*Settings, error) {
	row, err := h.queries.GetSetting(ctx, settingsKey)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		// No saved settings, return config defaults
		return &Settings{
			Enabled:          h.config.Enabled,
			IntervalHours:    h.config.IntervalHours,
			BackoffThreshold: h.config.BackoffThreshold,
		}, nil
	}

	var settings Settings
	if err := json.Unmarshal([]byte(row.Value), &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return &settings, nil
}

// saveSettings saves settings to database.
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

// LoadSettingsIntoConfig loads saved settings from database into config at startup.
func LoadSettingsIntoConfig(ctx context.Context, queries *sqlc.Queries, cfg *config.AutoSearchConfig) error {
	row, err := queries.GetSetting(ctx, settingsKey)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// No saved settings, use defaults
		return nil
	}

	var settings Settings
	if err := json.Unmarshal([]byte(row.Value), &settings); err != nil {
		return fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	// Apply to config
	cfg.Enabled = settings.Enabled
	cfg.IntervalHours = settings.IntervalHours
	cfg.BackoffThreshold = settings.BackoffThreshold

	return nil
}
