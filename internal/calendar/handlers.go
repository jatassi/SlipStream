package calendar

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for calendar operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new calendar handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the calendar routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.GetEvents)
}

// GetEventsRequest represents the query parameters for getting calendar events.
type GetEventsRequest struct {
	Start string `query:"start"` // YYYY-MM-DD
	End   string `query:"end"`   // YYYY-MM-DD
}

// GetEvents returns calendar events for a date range.
// GET /api/v1/calendar?start=2024-01-01&end=2024-01-31
func (h *Handlers) GetEvents(c echo.Context) error {
	var req GetEventsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request parameters")
	}

	// Validate and parse start date
	if req.Start == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "start date is required")
	}
	start, err := time.Parse("2006-01-02", req.Start)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start date format, expected YYYY-MM-DD")
	}

	// Validate and parse end date
	if req.End == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "end date is required")
	}
	end, err := time.Parse("2006-01-02", req.End)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end date format, expected YYYY-MM-DD")
	}

	// Ensure end is after start
	if end.Before(start) {
		return echo.NewHTTPError(http.StatusBadRequest, "end date must be after start date")
	}

	// Get events from service
	events, err := h.service.GetEvents(c.Request().Context(), start, end)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return empty array instead of null
	if events == nil {
		events = []CalendarEvent{}
	}

	return c.JSON(http.StatusOK, events)
}
