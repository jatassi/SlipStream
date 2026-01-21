package prowlarr

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for Prowlarr operations.
type Handlers struct {
	service     *Service
	modeManager *ModeManager
}

// NewHandlers creates new Prowlarr handlers.
func NewHandlers(service *Service, modeManager *ModeManager) *Handlers {
	return &Handlers{
		service:     service,
		modeManager: modeManager,
	}
}

// RegisterRoutes registers the Prowlarr routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	// Prowlarr configuration endpoints
	prowlarr := g.Group("/prowlarr")
	prowlarr.GET("", h.GetConfig)
	prowlarr.PUT("", h.UpdateConfig)
	prowlarr.POST("/test", h.TestConnection)
	prowlarr.GET("/indexers", h.GetIndexers)
	prowlarr.GET("/status", h.GetStatus)
	prowlarr.POST("/refresh", h.RefreshData)
	prowlarr.GET("/capabilities", h.GetCapabilities)

	// Per-indexer settings endpoints
	prowlarr.GET("/indexers/settings", h.GetAllIndexerSettings)
	prowlarr.GET("/indexers/:id/settings", h.GetIndexerSettings)
	prowlarr.PUT("/indexers/:id/settings", h.UpdateIndexerSettings)
	prowlarr.DELETE("/indexers/:id/settings", h.DeleteIndexerSettings)
	prowlarr.POST("/indexers/:id/reset-stats", h.ResetIndexerStats)

	// Mode management endpoints
	g.GET("/mode", h.GetMode)
	g.PUT("/mode", h.SetMode)
}

// GetConfig returns the Prowlarr configuration.
// GET /api/v1/indexers/prowlarr
func (h *Handlers) GetConfig(c echo.Context) error {
	config, err := h.service.GetConfig(c.Request().Context())
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			// Return empty/default config
			return c.JSON(http.StatusOK, &Config{
				MovieCategories: DefaultMovieCategories(),
				TVCategories:    DefaultTVCategories(),
				Timeout:         90,
				SkipSSLVerify:   true,
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, config)
}

// UpdateConfig updates the Prowlarr configuration.
// PUT /api/v1/indexers/prowlarr
func (h *Handlers) UpdateConfig(c echo.Context) error {
	var input ConfigInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate required fields when enabling
	if input.Enabled {
		if input.URL == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "URL is required when enabling Prowlarr")
		}
		if input.APIKey == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "API key is required when enabling Prowlarr")
		}
	}

	// Apply defaults
	if input.Timeout <= 0 {
		input.Timeout = 90
	}
	if len(input.MovieCategories) == 0 {
		input.MovieCategories = DefaultMovieCategories()
	}
	if len(input.TVCategories) == 0 {
		input.TVCategories = DefaultTVCategories()
	}

	config, err := h.service.UpdateConfig(c.Request().Context(), input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, config)
}

// TestConnectionRequest represents a test connection request.
type TestConnectionRequest struct {
	URL           string `json:"url"`
	APIKey        string `json:"apiKey"`
	Timeout       int    `json:"timeout"`
	SkipSSLVerify bool   `json:"skipSslVerify"`
}

// TestConnection tests the Prowlarr connection.
// POST /api/v1/indexers/prowlarr/test
func (h *Handlers) TestConnection(c echo.Context) error {
	var input TestConnectionRequest
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if input.URL == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "URL is required")
	}
	if input.APIKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "API key is required")
	}

	if input.Timeout <= 0 {
		input.Timeout = 90
	}

	err := h.service.TestConnection(c.Request().Context(), input.URL, input.APIKey, input.Timeout, input.SkipSSLVerify)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// GetIndexers returns the list of indexers from Prowlarr.
// GET /api/v1/indexers/prowlarr/indexers
func (h *Handlers) GetIndexers(c echo.Context) error {
	indexers, err := h.service.GetIndexers(c.Request().Context())
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return c.JSON(http.StatusOK, []Indexer{})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, indexers)
}

// GetStatus returns the Prowlarr connection status.
// GET /api/v1/indexers/prowlarr/status
func (h *Handlers) GetStatus(c echo.Context) error {
	status, err := h.service.GetConnectionStatus(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, status)
}

// RefreshData refreshes cached Prowlarr data (capabilities and indexers).
// POST /api/v1/indexers/prowlarr/refresh
func (h *Handlers) RefreshData(c echo.Context) error {
	ctx := c.Request().Context()

	// Clear search cache
	h.service.ClearSearchCache()

	// Refresh capabilities
	_, capErr := h.service.RefreshCapabilities(ctx)

	// Refresh indexers
	indexers, idxErr := h.service.RefreshIndexers(ctx)

	if capErr != nil && idxErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to refresh data")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"indexers": indexers,
		"refreshed": true,
	})
}

// GetCapabilities returns the Prowlarr capabilities.
// GET /api/v1/indexers/prowlarr/capabilities
func (h *Handlers) GetCapabilities(c echo.Context) error {
	caps, err := h.service.GetCapabilities(c.Request().Context())
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return c.JSON(http.StatusOK, nil)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, caps)
}

// GetMode returns the current indexer mode.
// GET /api/v1/indexers/mode
func (h *Handlers) GetMode(c echo.Context) error {
	modeInfo, err := h.modeManager.GetModeInfo(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, modeInfo)
}

// SetModeRequest represents a set mode request.
type SetModeRequest struct {
	Mode string `json:"mode"` // "slipstream" or "prowlarr"
}

// SetMode sets the indexer mode.
// PUT /api/v1/indexers/mode
func (h *Handlers) SetMode(c echo.Context) error {
	var input SetModeRequest
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var mode IndexerMode
	switch input.Mode {
	case "slipstream", "":
		mode = ModeSlipStream
	case "prowlarr":
		mode = ModeProwlarr
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mode: must be 'slipstream' or 'prowlarr'")
	}

	if err := h.modeManager.SetMode(c.Request().Context(), mode); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return updated mode info
	modeInfo, err := h.modeManager.GetModeInfo(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, modeInfo)
}

// GetAllIndexerSettings returns all per-indexer settings.
// GET /api/v1/indexers/prowlarr/indexers/settings
func (h *Handlers) GetAllIndexerSettings(c echo.Context) error {
	// Return indexers with their settings combined
	indexersWithSettings, err := h.service.GetIndexersWithSettings(c.Request().Context())
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return c.JSON(http.StatusOK, []IndexerWithSettings{})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, indexersWithSettings)
}

// GetIndexerSettings returns settings for a specific indexer.
// GET /api/v1/indexers/prowlarr/indexers/:id/settings
func (h *Handlers) GetIndexerSettings(c echo.Context) error {
	indexerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid indexer ID")
	}

	settings, err := h.service.GetIndexerSettings(c.Request().Context(), indexerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if settings == nil {
		// Return default settings if none exist
		return c.JSON(http.StatusOK, &IndexerSettings{
			ProwlarrIndexerID: indexerID,
			Priority:          25,
			ContentType:       ContentTypeBoth,
		})
	}

	return c.JSON(http.StatusOK, settings)
}

// UpdateIndexerSettings creates or updates settings for an indexer.
// PUT /api/v1/indexers/prowlarr/indexers/:id/settings
func (h *Handlers) UpdateIndexerSettings(c echo.Context) error {
	indexerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid indexer ID")
	}

	var input IndexerSettingsInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate content type
	switch input.ContentType {
	case ContentTypeMovies, ContentTypeSeries, ContentTypeBoth, "":
		// Valid
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid content type: must be 'movies', 'series', or 'both'")
	}

	settings, err := h.service.UpdateIndexerSettings(c.Request().Context(), indexerID, input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, settings)
}

// DeleteIndexerSettings removes settings for an indexer.
// DELETE /api/v1/indexers/prowlarr/indexers/:id/settings
func (h *Handlers) DeleteIndexerSettings(c echo.Context) error {
	indexerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid indexer ID")
	}

	if err := h.service.DeleteIndexerSettings(c.Request().Context(), indexerID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// ResetIndexerStats resets the success/failure counts for an indexer.
// POST /api/v1/indexers/prowlarr/indexers/:id/reset-stats
func (h *Handlers) ResetIndexerStats(c echo.Context) error {
	indexerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid indexer ID")
	}

	if err := h.service.ResetIndexerStats(c.Request().Context(), indexerID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return updated settings
	settings, err := h.service.GetIndexerSettings(c.Request().Context(), indexerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if settings == nil {
		return c.JSON(http.StatusOK, &IndexerSettings{
			ProwlarrIndexerID: indexerID,
			Priority:          25,
			ContentType:       ContentTypeBoth,
		})
	}

	return c.JSON(http.StatusOK, settings)
}
