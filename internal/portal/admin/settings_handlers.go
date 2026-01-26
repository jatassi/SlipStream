package admin

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/quota"
)

const (
	SettingPortalEnabled         = "requests_portal_enabled"
	SettingDefaultRootFolderID   = "requests_default_root_folder_id"
	SettingAdminNotifyNewRequest = "requests_admin_notify_new"
	SettingSearchRateLimit       = "requests_search_rate_limit"
)

type RequestSettings struct {
	Enabled             bool   `json:"enabled"`
	DefaultMovieQuota   int64  `json:"defaultMovieQuota"`
	DefaultSeasonQuota  int64  `json:"defaultSeasonQuota"`
	DefaultEpisodeQuota int64  `json:"defaultEpisodeQuota"`
	DefaultRootFolderID *int64 `json:"defaultRootFolderId"`
	AdminNotifyNew      bool   `json:"adminNotifyNew"`
	SearchRateLimit     int64  `json:"searchRateLimit"`
}

type UpdateSettingsRequest struct {
	Enabled             *bool  `json:"enabled"`
	DefaultMovieQuota   *int64 `json:"defaultMovieQuota"`
	DefaultSeasonQuota  *int64 `json:"defaultSeasonQuota"`
	DefaultEpisodeQuota *int64 `json:"defaultEpisodeQuota"`
	DefaultRootFolderID *int64 `json:"defaultRootFolderId"`
	AdminNotifyNew      *bool  `json:"adminNotifyNew"`
	SearchRateLimit     *int64 `json:"searchRateLimit"`
}

type SettingsHandlers struct {
	quotaService *quota.Service
	queries      *sqlc.Queries
}

func NewSettingsHandlers(quotaService *quota.Service, queries *sqlc.Queries) *SettingsHandlers {
	return &SettingsHandlers{
		quotaService: quotaService,
		queries:      queries,
	}
}

// SetDB updates the database queries used by the handlers.
func (h *SettingsHandlers) SetDB(queries *sqlc.Queries) {
	h.queries = queries
}

func (h *SettingsHandlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AdminAuth())

	protected.GET("", h.Get)
	protected.PUT("", h.Update)
}

// Get returns global request settings
// GET /api/v1/admin/requests/settings
func (h *SettingsHandlers) Get(c echo.Context) error {
	ctx := c.Request().Context()

	quotaDefaults, err := h.quotaService.GetGlobalDefaults(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	settings := RequestSettings{
		Enabled:             true, // Default to enabled
		DefaultMovieQuota:   *quotaDefaults.MoviesLimit,
		DefaultSeasonQuota:  *quotaDefaults.SeasonsLimit,
		DefaultEpisodeQuota: *quotaDefaults.EpisodesLimit,
		SearchRateLimit:     60, // Default 60 requests per minute
	}

	if setting, err := h.queries.GetSetting(ctx, SettingPortalEnabled); err == nil {
		settings.Enabled = setting.Value != "0" && setting.Value != "false"
	}

	if setting, err := h.queries.GetSetting(ctx, SettingDefaultRootFolderID); err == nil && setting.Value != "" {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			settings.DefaultRootFolderID = &v
		}
	}

	if setting, err := h.queries.GetSetting(ctx, SettingAdminNotifyNewRequest); err == nil {
		settings.AdminNotifyNew = setting.Value == "1" || setting.Value == "true"
	}

	if setting, err := h.queries.GetSetting(ctx, SettingSearchRateLimit); err == nil && setting.Value != "" {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			settings.SearchRateLimit = v
		}
	}

	return c.JSON(http.StatusOK, settings)
}

// Update updates global request settings
// PUT /api/v1/admin/requests/settings
func (h *SettingsHandlers) Update(c echo.Context) error {
	ctx := c.Request().Context()

	var req UpdateSettingsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Enabled != nil {
		value := "1"
		if !*req.Enabled {
			value = "0"
		}
		if _, err := h.queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   SettingPortalEnabled,
			Value: value,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if req.DefaultMovieQuota != nil || req.DefaultSeasonQuota != nil || req.DefaultEpisodeQuota != nil {
		if err := h.quotaService.SetGlobalDefaults(ctx, quota.QuotaLimits{
			MoviesLimit:   req.DefaultMovieQuota,
			SeasonsLimit:  req.DefaultSeasonQuota,
			EpisodesLimit: req.DefaultEpisodeQuota,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if req.DefaultRootFolderID != nil {
		value := ""
		if *req.DefaultRootFolderID > 0 {
			value = strconv.FormatInt(*req.DefaultRootFolderID, 10)
		}
		if _, err := h.queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   SettingDefaultRootFolderID,
			Value: value,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if req.AdminNotifyNew != nil {
		value := "0"
		if *req.AdminNotifyNew {
			value = "1"
		}
		if _, err := h.queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   SettingAdminNotifyNewRequest,
			Value: value,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if req.SearchRateLimit != nil {
		if _, err := h.queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   SettingSearchRateLimit,
			Value: strconv.FormatInt(*req.SearchRateLimit, 10),
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return h.Get(c)
}
