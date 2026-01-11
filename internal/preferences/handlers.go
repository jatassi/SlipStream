package preferences

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
	g.GET("/addflow", h.GetAddFlowPreferences)
	g.PUT("/addflow", h.SetAddFlowPreferences)
}

// GetAddFlowPreferences returns the add-flow preferences
// GET /api/v1/preferences/addflow
func (h *Handlers) GetAddFlowPreferences(c echo.Context) error {
	prefs, err := h.service.GetAddFlowPreferences(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, prefs)
}

// SetAddFlowPreferences updates the add-flow preferences
// PUT /api/v1/preferences/addflow
func (h *Handlers) SetAddFlowPreferences(c echo.Context) error {
	var prefs AddFlowPreferences
	if err := c.Bind(&prefs); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.service.SetAddFlowPreferences(c.Request().Context(), prefs); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	updated, err := h.service.GetAddFlowPreferences(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}
