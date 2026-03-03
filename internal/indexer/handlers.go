package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/status"
)

const redactedSentinel = "********"

// Handlers provides HTTP handlers for indexer operations.
type Handlers struct {
	service       *Service
	statusService *status.Service
}

// NewHandlers creates new indexer handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// SetStatusService sets the status service for health tracking.
func (h *Handlers) SetStatusService(statusService *status.Service) {
	h.statusService = statusService
}

// RegisterRoutes registers the indexer routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/definitions", h.ListDefinitions)
	g.GET("/definitions/search", h.SearchDefinitions)
	g.GET("/definitions/:id", h.GetDefinition)
	g.GET("/definitions/:id/schema", h.GetDefinitionSchema)
	g.POST("/definitions/update", h.UpdateDefinitions)
	g.GET("/status", h.GetAllStatuses)
	g.POST("/test", h.TestConfig)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.POST("/:id/test", h.Test)
	g.GET("/:id/status", h.GetStatus)
}

// indexerSensitiveFields returns the set of settings field names that are passwords for the given definition.
func (h *Handlers) indexerSensitiveFields(defID string) map[string]bool {
	schema, err := h.service.GetDefinitionSchema(defID)
	if err != nil {
		return nil
	}
	fields := make(map[string]bool)
	for _, s := range schema {
		if s.Type == "password" {
			fields[s.Name] = true
		}
	}
	return fields
}

// applySettingsRedaction replaces non-empty password field values in a parsed settings map with the sentinel.
func applySettingsRedaction(settings map[string]interface{}, sensitive map[string]bool) {
	for key, val := range settings {
		if !sensitive[key] {
			continue
		}
		if str, ok := val.(string); ok && str != "" {
			settings[key] = redactedSentinel
		}
	}
}

// settingsHasSentinel reports whether any sensitive field in the map holds the sentinel.
func settingsHasSentinel(m map[string]interface{}, sensitive map[string]bool) bool {
	for key, val := range m {
		if sensitive[key] {
			if str, ok := val.(string); ok && str == redactedSentinel {
				return true
			}
		}
	}
	return false
}

// replaceSettingsSentinels overwrites sentinel values in dst with the corresponding values from src.
func replaceSettingsSentinels(dst, src map[string]interface{}, sensitive map[string]bool) {
	for key, val := range dst {
		if !sensitive[key] {
			continue
		}
		if str, ok := val.(string); ok && str == redactedSentinel {
			if existing, ok := src[key]; ok {
				dst[key] = existing
			}
		}
	}
}

// redactIndexerSettings replaces password field values in the indexer's Settings JSON
// with the sentinel so callers know a value exists without exposing it.
func (h *Handlers) redactIndexerSettings(indexer *IndexerDefinition) {
	if indexer == nil || len(indexer.Settings) == 0 {
		return
	}
	sensitive := h.indexerSensitiveFields(indexer.DefinitionID)
	if len(sensitive) == 0 {
		return
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(indexer.Settings, &settings); err != nil {
		return
	}
	applySettingsRedaction(settings, sensitive)
	if redacted, err := json.Marshal(settings); err == nil {
		indexer.Settings = redacted
	}
}

// mergeRedactedSettings replaces any sentinel values in incoming settings
// with the real values from the existing DB record, so sentinels are never persisted.
func (h *Handlers) mergeRedactedSettings(ctx context.Context, indexerID int64, incoming json.RawMessage, defID string) json.RawMessage {
	if len(incoming) == 0 {
		return incoming
	}
	sensitive := h.indexerSensitiveFields(defID)
	if len(sensitive) == 0 {
		return incoming
	}
	var incomingMap map[string]interface{}
	if err := json.Unmarshal(incoming, &incomingMap); err != nil {
		return incoming
	}
	if !settingsHasSentinel(incomingMap, sensitive) {
		return incoming
	}
	existing, err := h.service.Get(ctx, indexerID)
	if err != nil {
		return incoming
	}
	var existingMap map[string]interface{}
	if err := json.Unmarshal(existing.Settings, &existingMap); err != nil {
		return incoming
	}
	replaceSettingsSentinels(incomingMap, existingMap, sensitive)
	if merged, err := json.Marshal(incomingMap); err == nil {
		return merged
	}
	return incoming
}

// resolveUpdateDefID determines the definition ID for an update request.
// It prefers the value from the input; if absent, it fetches the existing record.
func (h *Handlers) resolveUpdateDefID(ctx context.Context, id int64, input *UpdateIndexerInput) string {
	if input.DefinitionID != nil {
		return *input.DefinitionID
	}
	existing, err := h.service.Get(ctx, id)
	if err != nil {
		return ""
	}
	return existing.DefinitionID
}

// List returns all indexers.
// GET /api/v1/indexers
func (h *Handlers) List(c echo.Context) error {
	indexers, err := h.service.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	for _, idx := range indexers {
		h.redactIndexerSettings(idx)
	}
	return c.JSON(http.StatusOK, indexers)
}

// Get returns a single indexer.
// GET /api/v1/indexers/:id
func (h *Handlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	indexer, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrIndexerNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	h.redactIndexerSettings(indexer)
	return c.JSON(http.StatusOK, indexer)
}

// Create creates a new indexer.
// POST /api/v1/indexers
func (h *Handlers) Create(c echo.Context) error {
	var input CreateIndexerInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Debug: Log received settings
	h.service.logger.Debug().
		Str("name", input.Name).
		Str("definitionId", input.DefinitionID).
		RawJSON("settings", input.Settings).
		Msg("Create indexer request received")

	indexer, err := h.service.Create(c.Request().Context(), &input)
	if err != nil {
		if errors.Is(err, ErrInvalidIndexer) || errors.Is(err, ErrDefinitionNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	h.redactIndexerSettings(indexer)
	return c.JSON(http.StatusCreated, indexer)
}

// Update updates an existing indexer.
// PUT /api/v1/indexers/:id
func (h *Handlers) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input UpdateIndexerInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if len(input.Settings) > 0 {
		defID := h.resolveUpdateDefID(c.Request().Context(), id, &input)
		if defID != "" {
			input.Settings = h.mergeRedactedSettings(c.Request().Context(), id, input.Settings, defID)
		}
	}

	indexer, err := h.service.Update(c.Request().Context(), id, &input)
	if err != nil {
		if errors.Is(err, ErrIndexerNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrInvalidIndexer) || errors.Is(err, ErrDefinitionNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	h.redactIndexerSettings(indexer)
	return c.JSON(http.StatusOK, indexer)
}

// Delete deletes an indexer.
// DELETE /api/v1/indexers/:id
func (h *Handlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, ErrIndexerNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// Test tests an indexer connection by ID.
// POST /api/v1/indexers/:id/test
func (h *Handlers) Test(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.Test(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrIndexerNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// TestConfig tests an indexer configuration without saving.
// POST /api/v1/indexers/test
func (h *Handlers) TestConfig(c echo.Context) error {
	var input TestConfigInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate required fields
	if input.DefinitionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "definitionId is required")
	}

	result, err := h.service.TestConfig(c.Request().Context(), input)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// GetStatus returns the status of an indexer.
// GET /api/v1/indexers/:id/status
func (h *Handlers) GetStatus(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	// Verify the indexer exists and get its name
	indexer, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrIndexerNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// If status service is not configured, return default status
	if h.statusService == nil {
		return c.JSON(http.StatusOK, IndexerStatus{
			IndexerID:       id,
			EscalationLevel: 0,
		})
	}

	// Get health info from status service
	health, err := h.statusService.GetHealth(c.Request().Context(), id, indexer.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, health)
}

// GetAllStatuses returns the status of all indexers.
// GET /api/v1/indexers/status
func (h *Handlers) GetAllStatuses(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all indexers
	indexers, err := h.service.List(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// If status service is not configured, return default statuses
	if h.statusService == nil {
		statuses := make([]*status.IndexerHealth, 0, len(indexers))
		for _, idx := range indexers {
			statuses = append(statuses, &status.IndexerHealth{
				IndexerID:   idx.ID,
				IndexerName: idx.Name,
				Status:      status.HealthStatusHealthy,
				Message:     "Operating normally",
			})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"indexers": statuses,
		})
	}

	// Get health for each indexer
	statuses := make([]*status.IndexerHealth, 0, len(indexers))
	for _, idx := range indexers {
		health, err := h.statusService.GetHealth(ctx, idx.ID, idx.Name)
		if err != nil {
			// Log error but continue with other indexers
			continue
		}
		statuses = append(statuses, health)
	}

	// Get overall stats
	stats, err := h.statusService.GetStats(ctx, len(indexers))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"indexers": statuses,
		"stats":    stats,
	})
}

// ListDefinitions returns all available Cardigann definitions.
// GET /api/v1/indexers/definitions
func (h *Handlers) ListDefinitions(c echo.Context) error {
	definitions, err := h.service.ListDefinitions()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Lazy load: If no definitions cached, trigger an update
	if len(definitions) == 0 {
		if err := h.service.UpdateDefinitions(c.Request().Context()); err == nil {
			// Retry listing after successful update
			definitions, err = h.service.ListDefinitions()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
		}
		// If update fails, silently return empty list - user can manually trigger update
	}

	// Transform to response format (maps Type -> privacy for frontend)
	type resp struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Privacy     string `json:"privacy"`
		Language    string `json:"language"`
		Protocol    string `json:"protocol"`
	}
	out := make([]resp, len(definitions))
	for i, def := range definitions {
		out[i] = resp{
			ID:          def.ID,
			Name:        def.Name,
			Description: def.Description,
			Privacy:     def.Type,
			Language:    def.Language,
			Protocol:    def.Protocol,
		}
	}

	return c.JSON(http.StatusOK, out)
}

// SearchDefinitions searches for definitions matching query and filters.
// GET /api/v1/indexers/definitions/search?q=...&protocol=...&privacy=...&language=...
func (h *Handlers) SearchDefinitions(c echo.Context) error {
	query := c.QueryParam("q")
	filters := cardigann.DefinitionFilters{
		Protocol: c.QueryParam("protocol"),
		Privacy:  c.QueryParam("privacy"),
		Language: c.QueryParam("language"),
	}

	definitions, err := h.service.SearchDefinitions(query, filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, definitions)
}

// GetDefinition returns a single Cardigann definition.
// GET /api/v1/indexers/definitions/:id
func (h *Handlers) GetDefinition(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "definition id is required")
	}

	definition, err := h.service.GetDefinition(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, definition)
}

// GetDefinitionSchema returns the settings schema for a definition.
// GET /api/v1/indexers/definitions/:id/schema
func (h *Handlers) GetDefinitionSchema(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "definition id is required")
	}

	schema, err := h.service.GetDefinitionSchema(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, schema)
}

// UpdateDefinitions triggers a refresh of the definition cache.
// POST /api/v1/indexers/definitions/update
func (h *Handlers) UpdateDefinitions(c echo.Context) error {
	if err := h.service.UpdateDefinitions(c.Request().Context()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Definitions updated successfully",
	})
}
