package rsssync

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for RSS sync operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new RSS sync handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the RSS sync routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.POST("/trigger", h.Trigger)
	g.GET("/status", h.GetStatus)
}

// Trigger manually triggers an RSS sync.
// POST /api/v1/rsssync/trigger
func (h *Handlers) Trigger(c echo.Context) error {
	if h.service.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "RSS sync already running")
	}

	go func() {
		_ = h.service.Run(context.Background())
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "RSS sync started",
	})
}

// GetStatus returns the last RSS sync status.
// GET /api/v1/rsssync/status
func (h *Handlers) GetStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, h.service.LastStatus())
}
