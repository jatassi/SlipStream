package quality

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for quality profile operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new quality profile handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the quality profile routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/qualities", h.ListQualities)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

// List returns all quality profiles.
// GET /api/v1/qualityprofiles
func (h *Handlers) List(c echo.Context) error {
	profiles, err := h.service.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, profiles)
}

// Get returns a single quality profile.
// GET /api/v1/qualityprofiles/:id
func (h *Handlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	profile, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, profile)
}

// Create creates a new quality profile.
// POST /api/v1/qualityprofiles
func (h *Handlers) Create(c echo.Context) error {
	var input CreateProfileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	profile, err := h.service.Create(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, ErrInvalidProfile) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, profile)
}

// Update updates an existing quality profile.
// PUT /api/v1/qualityprofiles/:id
func (h *Handlers) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input UpdateProfileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	profile, err := h.service.Update(c.Request().Context(), id, input)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrInvalidProfile) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, profile)
}

// Delete deletes a quality profile.
// DELETE /api/v1/qualityprofiles/:id
func (h *Handlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrProfileInUse) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// ListQualities returns the predefined quality definitions.
// GET /api/v1/qualityprofiles/qualities
func (h *Handlers) ListQualities(c echo.Context) error {
	return c.JSON(http.StatusOK, h.service.GetQualities())
}
