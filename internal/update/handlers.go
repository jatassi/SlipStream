package update

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Handlers struct {
	service *Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.GetStatus)
	g.POST("/check", h.CheckForUpdate)
	g.POST("/install", h.Install)
	g.POST("/cancel", h.Cancel)
	g.GET("/settings", h.GetSettings)
	g.PUT("/settings", h.UpdateSettings)
}

// GetStatus returns the current update status.
// GET /api/v1/update
func (h *Handlers) GetStatus(c echo.Context) error {
	status := h.service.GetStatus()
	return c.JSON(http.StatusOK, status)
}

// CheckForUpdate triggers a check for new versions.
// POST /api/v1/update/check
func (h *Handlers) CheckForUpdate(c echo.Context) error {
	ctx := c.Request().Context()
	release, err := h.service.CheckForUpdate(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	status := h.service.GetStatus()
	response := map[string]interface{}{
		"status":        status,
		"updateAvailable": release != nil,
	}
	if release != nil {
		response["release"] = release
	}

	return c.JSON(http.StatusOK, response)
}

// Install starts the download and installation process.
// POST /api/v1/update/install
func (h *Handlers) Install(c echo.Context) error {
	ctx := c.Request().Context()

	go func() {
		if err := h.service.DownloadAndInstall(ctx); err != nil {
			h.service.logger.Error().Err(err).Msg("Update installation failed")
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Update started",
	})
}

// Cancel stops an in-progress update.
// POST /api/v1/update/cancel
func (h *Handlers) Cancel(c echo.Context) error {
	if err := h.service.Cancel(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Update cancelled",
	})
}

// GetSettings returns the update settings.
// GET /api/v1/update/settings
func (h *Handlers) GetSettings(c echo.Context) error {
	ctx := c.Request().Context()
	settings, err := h.service.GetSettings(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates the update settings.
// PUT /api/v1/update/settings
func (h *Handlers) UpdateSettings(c echo.Context) error {
	ctx := c.Request().Context()

	var settings Settings
	if err := c.Bind(&settings); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if err := h.service.UpdateSettings(ctx, &settings); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, settings)
}
