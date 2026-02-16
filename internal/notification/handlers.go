package notification

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for notification management
type Handlers struct {
	service *Service
}

// NewHandlers creates a new Handlers instance
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers admin-only notification routes on the provided group
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.POST("/:id/test", h.Test)
}

// RegisterSharedRoutes registers notification routes accessible to both admin and portal users
func (h *Handlers) RegisterSharedRoutes(g *echo.Group) {
	g.GET("/schema", h.GetSchemas)
	g.POST("/test", h.TestNew)
}

// List returns all configured notifications
// GET /api/v1/notifications
func (h *Handlers) List(c echo.Context) error {
	ctx := c.Request().Context()

	notifications, err := h.service.List(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, notifications)
}

// Get returns a single notification by ID
// GET /api/v1/notifications/:id
func (h *Handlers) Get(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	notification, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "notification not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, notification)
}

// Create creates a new notification
// POST /api/v1/notifications
func (h *Handlers) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var input CreateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	if input.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "type is required"})
	}

	notification, err := h.service.Create(ctx, &input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, notification)
}

// Update updates an existing notification
// PUT /api/v1/notifications/:id
func (h *Handlers) Update(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var input UpdateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	notification, err := h.service.Update(ctx, id, &input)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "notification not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, notification)
}

// Delete deletes a notification
// DELETE /api/v1/notifications/:id
func (h *Handlers) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := h.service.Delete(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

// Test tests an existing notification configuration
// POST /api/v1/notifications/:id/test
func (h *Handlers) Test(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	result, err := h.service.Test(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "notification not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// TestNew tests a notification configuration without saving
// POST /api/v1/notifications/test
func (h *Handlers) TestNew(c echo.Context) error {
	ctx := c.Request().Context()

	var input CreateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if input.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "type is required"})
	}

	result, err := h.service.TestConfig(ctx, &input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// GetSchemas returns the available notification types and their settings schemas
// GET /api/v1/notifications/schema
func (h *Handlers) GetSchemas(c echo.Context) error {
	schemas := GetAllSchemas()
	return c.JSON(http.StatusOK, schemas)
}
