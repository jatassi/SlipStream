package admin

import (
	"context"
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
		Enabled:             true,
		DefaultMovieQuota:   *quotaDefaults.MoviesLimit,
		DefaultSeasonQuota:  *quotaDefaults.SeasonsLimit,
		DefaultEpisodeQuota: *quotaDefaults.EpisodesLimit,
		SearchRateLimit:     60,
	}

	settings.Enabled = h.readSettingBool(ctx, SettingPortalEnabled, true)
	settings.DefaultRootFolderID = h.readSettingInt64Ptr(ctx, SettingDefaultRootFolderID)
	settings.AdminNotifyNew = h.readSettingBool(ctx, SettingAdminNotifyNewRequest, false)
	if v := h.readSettingInt64Ptr(ctx, SettingSearchRateLimit); v != nil {
		settings.SearchRateLimit = *v
	}

	return c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandlers) readSettingBool(ctx context.Context, key string, defaultVal bool) bool {
	setting, err := h.queries.GetSetting(ctx, key)
	if err != nil {
		return defaultVal
	}
	if defaultVal {
		return setting.Value != "0" && setting.Value != "false"
	}
	return setting.Value == "1" || setting.Value == "true"
}

func (h *SettingsHandlers) readSettingInt64Ptr(ctx context.Context, key string) *int64 {
	setting, err := h.queries.GetSetting(ctx, key)
	if err != nil || setting.Value == "" {
		return nil
	}
	v, err := strconv.ParseInt(setting.Value, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

// Update updates global request settings
// PUT /api/v1/admin/requests/settings
func (h *SettingsHandlers) Update(c echo.Context) error {
	ctx := c.Request().Context()

	var req UpdateSettingsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.applySettingsUpdate(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return h.Get(c)
}

func (h *SettingsHandlers) applySettingsUpdate(ctx context.Context, req *UpdateSettingsRequest) error {
	if req.Enabled != nil {
		if err := h.setBoolSetting(ctx, SettingPortalEnabled, *req.Enabled); err != nil {
			return err
		}
	}

	if err := h.updateQuotas(ctx, req); err != nil {
		return err
	}

	if err := h.updateRootFolderSetting(ctx, req.DefaultRootFolderID); err != nil {
		return err
	}

	if req.AdminNotifyNew != nil {
		if err := h.setBoolSetting(ctx, SettingAdminNotifyNewRequest, *req.AdminNotifyNew); err != nil {
			return err
		}
	}

	if req.SearchRateLimit != nil {
		if err := h.setStringSetting(ctx, SettingSearchRateLimit, strconv.FormatInt(*req.SearchRateLimit, 10)); err != nil {
			return err
		}
	}

	return nil
}

func (h *SettingsHandlers) updateRootFolderSetting(ctx context.Context, rootFolderID *int64) error {
	if rootFolderID == nil {
		return nil
	}
	value := ""
	if *rootFolderID > 0 {
		value = strconv.FormatInt(*rootFolderID, 10)
	}
	return h.setStringSetting(ctx, SettingDefaultRootFolderID, value)
}

func (h *SettingsHandlers) updateQuotas(ctx context.Context, req *UpdateSettingsRequest) error {
	if req.DefaultMovieQuota == nil && req.DefaultSeasonQuota == nil && req.DefaultEpisodeQuota == nil {
		return nil
	}
	return h.quotaService.SetGlobalDefaults(ctx, quota.QuotaLimits{
		MoviesLimit:   req.DefaultMovieQuota,
		SeasonsLimit:  req.DefaultSeasonQuota,
		EpisodesLimit: req.DefaultEpisodeQuota,
	})
}

func (h *SettingsHandlers) setBoolSetting(ctx context.Context, key string, value bool) error {
	v := "0"
	if value {
		v = "1"
	}
	_, err := h.queries.SetSetting(ctx, sqlc.SetSettingParams{Key: key, Value: v})
	return err
}

func (h *SettingsHandlers) setStringSetting(ctx context.Context, key, value string) error {
	_, err := h.queries.SetSetting(ctx, sqlc.SetSettingParams{Key: key, Value: value})
	return err
}
