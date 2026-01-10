package missing

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for missing media operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new missing handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the missing routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("/movies", h.GetMovies)
	g.GET("/series", h.GetSeries)
	g.GET("/counts", h.GetCounts)
}

// GetMovies returns all missing movies.
// GET /api/v1/missing/movies
func (h *Handlers) GetMovies(c echo.Context) error {
	movies, err := h.service.GetMissingMovies(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, movies)
}

// GetSeries returns all series with missing episodes.
// GET /api/v1/missing/series
func (h *Handlers) GetSeries(c echo.Context) error {
	series, err := h.service.GetMissingSeries(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, series)
}

// GetCounts returns the counts of missing movies and episodes.
// GET /api/v1/missing/counts
func (h *Handlers) GetCounts(c echo.Context) error {
	counts, err := h.service.GetMissingCounts(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, counts)
}
