package history

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for history operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates a new history handlers instance.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers history routes on an Echo group.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.DELETE("", h.Clear)
}

// List returns paginated history entries.
// GET /api/v1/history
func (h *Handlers) List(c echo.Context) error {
	page := 1
	if p := c.QueryParam("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	pageSize := 50
	if ps := c.QueryParam("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	opts := ListOptions{
		EventType: c.QueryParam("eventType"),
		MediaType: c.QueryParam("mediaType"),
		Page:      page,
		PageSize:  pageSize,
	}

	result, err := h.service.List(c.Request().Context(), opts)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// Clear deletes all history entries.
// DELETE /api/v1/history
func (h *Handlers) Clear(c echo.Context) error {
	if err := h.service.DeleteAll(c.Request().Context()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
