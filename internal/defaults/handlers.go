package defaults

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for defaults operations
type Handlers struct {
	service *Service
}

// NewHandlers creates new defaults handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers defaults routes
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.GetAll)
	g.GET("/:entityType", h.GetByEntityType)
	g.GET("/:entityType/:mediaType", h.Get)
	g.POST("/:entityType/:mediaType", h.Set)
	g.DELETE("/:entityType/:mediaType", h.Clear)
}

// GetAll returns all default settings
// GET /api/v1/defaults
func (h *Handlers) GetAll(c echo.Context) error {
	defaults, err := h.service.GetAllDefaults(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, defaults)
}

// GetByEntityType returns all defaults for an entity type
// GET /api/v1/defaults/:entityType
func (h *Handlers) GetByEntityType(c echo.Context) error {
	entityType := EntityType(c.Param("entityType"))

	// Validate entity type
	switch entityType {
	case EntityTypeRootFolder, EntityTypeQualityProfile, EntityTypeDownloadClient, EntityTypeIndexer:
		// Valid
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid entity type")
	}

	defaults, err := h.service.GetDefaultsForEntityType(c.Request().Context(), entityType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, defaults)
}

// Get returns a specific default setting
// GET /api/v1/defaults/:entityType/:mediaType
func (h *Handlers) Get(c echo.Context) error {
	entityType := EntityType(c.Param("entityType"))
	mediaType := MediaType(c.Param("mediaType"))

	// Validate inputs
	if err := h.validateInputs(entityType, mediaType); err != nil {
		return err
	}

	defaultEntry, err := h.service.GetDefault(c.Request().Context(), entityType, mediaType)
	if err != nil {
		if errors.Is(err, ErrNoDefault) {
			return c.JSON(http.StatusOK, map[string]interface{}{"exists": false, "defaultEntry": nil})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"exists":       true,
		"defaultEntry": defaultEntry,
	})
}

// SetDefaultRequest represents the request body for setting a default
type SetDefaultRequest struct {
	EntityID int64 `json:"entityId" validate:"required"`
}

// Set sets a default setting
// POST /api/v1/defaults/:entityType/:mediaType
func (h *Handlers) Set(c echo.Context) error {
	entityType := EntityType(c.Param("entityType"))
	mediaType := MediaType(c.Param("mediaType"))

	// Validate inputs
	if err := h.validateInputs(entityType, mediaType); err != nil {
		return err
	}

	var req SetDefaultRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.SetDefault(c.Request().Context(), entityType, mediaType, req.EntityID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "default set successfully"})
}

// Clear clears a default setting
// DELETE /api/v1/defaults/:entityType/:mediaType
func (h *Handlers) Clear(c echo.Context) error {
	entityType := EntityType(c.Param("entityType"))
	mediaType := MediaType(c.Param("mediaType"))

	// Validate inputs
	if err := h.validateInputs(entityType, mediaType); err != nil {
		return err
	}

	if err := h.service.ClearDefault(c.Request().Context(), entityType, mediaType); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "default cleared successfully"})
}

// validateInputs validates entity type and media type parameters
func (h *Handlers) validateInputs(entityType EntityType, mediaType MediaType) *echo.HTTPError {
	// Validate entity type
	switch entityType {
	case EntityTypeRootFolder, EntityTypeQualityProfile, EntityTypeDownloadClient, EntityTypeIndexer:
		// Valid
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid entity type")
	}

	// Validate media type
	switch mediaType {
	case MediaTypeMovie, MediaTypeTV:
		// Valid
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid media type")
	}

	return nil
}
