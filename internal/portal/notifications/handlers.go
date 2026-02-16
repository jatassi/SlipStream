package notifications

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/notification"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotOwner             = errors.New("not the owner of this notification")
)

type CreateNotificationRequest struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Settings    json.RawMessage `json:"settings"`
	OnAvailable bool            `json:"onAvailable"`
	OnApproved  bool            `json:"onApproved"`
	OnDenied    bool            `json:"onDenied"`
	Enabled     bool            `json:"enabled"`
}

type Handlers struct {
	service *Service
}

func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AnyAuth())

	protected.GET("", h.List)
	protected.POST("", h.Create)
	protected.GET("/schema", h.GetSchemas)
	protected.GET("/:id", h.Get)
	protected.PUT("/:id", h.Update)
	protected.DELETE("/:id", h.Delete)
	protected.POST("/:id/test", h.Test)
}

// List returns all notifications for the authenticated user
// GET /api/v1/requests/notifications
func (h *Handlers) List(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	notifications, err := h.service.ListUserNotifications(c.Request().Context(), claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, notifications)
}

// Get returns a single notification by ID
// GET /api/v1/requests/notifications/:id
func (h *Handlers) Get(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	notif, err := h.service.GetUserNotification(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "notification not found")
	}

	if notif.UserID != claims.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "not the owner of this notification")
	}

	return c.JSON(http.StatusOK, notif)
}

// Create creates a new notification for the authenticated user
// POST /api/v1/requests/notifications
func (h *Handlers) Create(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req CreateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.Type == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}

	notif, err := h.service.CreateUserNotification(c.Request().Context(), claims.UserID, CreateNotificationInput(req))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, notif)
}

// Update updates an existing notification
// PUT /api/v1/requests/notifications/:id
func (h *Handlers) Update(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	existing, err := h.service.GetUserNotification(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "notification not found")
	}

	if existing.UserID != claims.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "not the owner of this notification")
	}

	var req CreateNotificationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	notif, err := h.service.UpdateUserNotification(c.Request().Context(), id, CreateNotificationInput(req))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, notif)
}

// Delete deletes a notification
// DELETE /api/v1/requests/notifications/:id
func (h *Handlers) Delete(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	existing, err := h.service.GetUserNotification(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "notification not found")
	}

	if existing.UserID != claims.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "not the owner of this notification")
	}

	if err := h.service.DeleteUserNotification(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// Test tests an existing notification configuration
// POST /api/v1/requests/notifications/:id/test
func (h *Handlers) Test(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	existing, err := h.service.GetUserNotification(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "notification not found")
	}

	if existing.UserID != claims.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "not the owner of this notification")
	}

	if err := h.service.TestUserNotification(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// GetSchemas returns the available notification types and their settings schemas
// GET /api/v1/requests/notifications/schema
func (h *Handlers) GetSchemas(c echo.Context) error {
	schemas := notification.GetAllSchemas()
	return c.JSON(http.StatusOK, schemas)
}
