package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/module"
)

const redactedSentinel = "********"

// Handlers provides HTTP handlers for notification management
type Handlers struct {
	service  *Service
	registry *module.Registry
}

// NewHandlers creates a new Handlers instance
func NewHandlers(service *Service, registry *module.Registry) *Handlers {
	return &Handlers{service: service, registry: registry}
}

// RegisterRoutes registers admin-only notification routes on the provided group
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/events", h.GetEventCatalog)
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

// sensitiveFieldSet returns a set of field names that are password-type for the given notifier type.
func sensitiveFieldSet(notifType NotifierType) map[string]bool {
	schema, ok := SchemaRegistry[notifType]
	if !ok {
		return nil
	}
	fields := make(map[string]bool)
	for i := range schema.Fields {
		if schema.Fields[i].Type == FieldTypePassword {
			fields[schema.Fields[i].Name] = true
		}
	}
	return fields
}

// redactSettings replaces password field values in a settings map with the redaction sentinel.
func redactSettings(settings map[string]interface{}, sensitiveFields map[string]bool) {
	for key, val := range settings {
		if sensitiveFields[key] {
			if str, ok := val.(string); ok && str != "" {
				settings[key] = redactedSentinel
			}
		}
	}
}

// redactNotificationSettings replaces password field values in the config's Settings JSON
// with the sentinel so callers know a value exists without seeing it.
func redactNotificationSettings(config *Config) {
	if config == nil || len(config.Settings) == 0 {
		return
	}
	sensitiveFields := sensitiveFieldSet(config.Type)
	if len(sensitiveFields) == 0 {
		return
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(config.Settings, &settings); err != nil {
		return
	}
	redactSettings(settings, sensitiveFields)
	if redacted, err := json.Marshal(settings); err == nil {
		config.Settings = redacted
	}
}

// hasSentinelValues returns true if any sensitive field in the map equals the redaction sentinel.
func hasSentinelValues(m map[string]interface{}, sensitiveFields map[string]bool) bool {
	for key, val := range m {
		if sensitiveFields[key] {
			if str, ok := val.(string); ok && str == redactedSentinel {
				return true
			}
		}
	}
	return false
}

// replaceSentinels replaces sentinel values in incomingMap with real values from existingMap.
func replaceSentinels(incomingMap, existingMap map[string]interface{}, sensitiveFields map[string]bool) {
	for key, val := range incomingMap {
		if !sensitiveFields[key] {
			continue
		}
		if str, ok := val.(string); ok && str == redactedSentinel {
			if existingVal, exists := existingMap[key]; exists {
				incomingMap[key] = existingVal
			}
		}
	}
}

// mergeRedactedNotificationSettings replaces any sentinel values in incoming settings
// with the real values from the existing DB record, so sentinels are never persisted.
func (h *Handlers) mergeRedactedNotificationSettings(ctx context.Context, id int64, notifType NotifierType, incoming json.RawMessage) json.RawMessage {
	if len(incoming) == 0 {
		return incoming
	}
	sensitiveFields := sensitiveFieldSet(notifType)
	if len(sensitiveFields) == 0 {
		return incoming
	}
	var incomingMap map[string]interface{}
	if err := json.Unmarshal(incoming, &incomingMap); err != nil {
		return incoming
	}
	if !hasSentinelValues(incomingMap, sensitiveFields) {
		return incoming
	}
	existing, err := h.service.Get(ctx, id)
	if err != nil {
		return incoming
	}
	var existingMap map[string]interface{}
	if err := json.Unmarshal(existing.Settings, &existingMap); err != nil {
		return incoming
	}
	replaceSentinels(incomingMap, existingMap, sensitiveFields)
	if merged, err := json.Marshal(incomingMap); err == nil {
		return merged
	}
	return incoming
}

// notifTypeForUpdate resolves the notifier type for an update operation.
// If the update specifies a new type, that is used; otherwise the existing record's type is returned.
func (h *Handlers) notifTypeForUpdate(ctx context.Context, id int64, input *UpdateInput) NotifierType {
	if input.Type != nil {
		return *input.Type
	}
	existing, err := h.service.Get(ctx, id)
	if err != nil {
		return ""
	}
	return existing.Type
}

// List returns all configured notifications
// GET /api/v1/notifications
func (h *Handlers) List(c echo.Context) error {
	ctx := c.Request().Context()

	notifications, err := h.service.List(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	for i := range notifications {
		redactNotificationSettings(&notifications[i])
	}

	return c.JSON(http.StatusOK, notifications)
}

// Get returns a single notification by ID
// GET /api/v1/notifications/:id
func (h *Handlers) Get(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	notification, err := h.service.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "notification not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	redactNotificationSettings(notification)
	return c.JSON(http.StatusOK, notification)
}

// Create creates a new notification
// POST /api/v1/notifications
func (h *Handlers) Create(c echo.Context) error {
	ctx := c.Request().Context()

	var input CreateInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if input.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	if input.Type == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}

	notification, err := h.service.Create(ctx, &input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	redactNotificationSettings(notification)
	return c.JSON(http.StatusCreated, notification)
}

// Update updates an existing notification
// PUT /api/v1/notifications/:id
func (h *Handlers) Update(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input UpdateInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if input.Settings != nil {
		notifType := h.notifTypeForUpdate(ctx, id, &input)
		if notifType != "" {
			merged := h.mergeRedactedNotificationSettings(ctx, id, notifType, *input.Settings)
			input.Settings = &merged
		}
	}

	notification, err := h.service.Update(ctx, id, &input)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "notification not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	redactNotificationSettings(notification)
	return c.JSON(http.StatusOK, notification)
}

// Delete deletes a notification
// DELETE /api/v1/notifications/:id
func (h *Handlers) Delete(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.service.Delete(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// Test tests an existing notification configuration
// POST /api/v1/notifications/:id/test
func (h *Handlers) Test(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.Test(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "notification not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// TestNew tests a notification configuration without saving
// POST /api/v1/notifications/test
func (h *Handlers) TestNew(c echo.Context) error {
	ctx := c.Request().Context()

	var input CreateInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if input.Type == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}

	result, err := h.service.TestConfig(ctx, &input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// GetEventCatalog returns the available notification events grouped by source
// GET /api/v1/notifications/events
func (h *Handlers) GetEventCatalog(c echo.Context) error {
	groups := h.registry.CollectNotificationEvents()
	return c.JSON(http.StatusOK, groups)
}

// GetSchemas returns the available notification types and their settings schemas
// GET /api/v1/notifications/schema
func (h *Handlers) GetSchemas(c echo.Context) error {
	schemas := GetAllSchemas()
	return c.JSON(http.StatusOK, schemas)
}
