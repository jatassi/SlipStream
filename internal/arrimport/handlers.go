package arrimport

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for arr import operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates a new arr import handlers instance.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers arr import routes on an Echo group.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("/detect-db", h.DetectDB)
	g.POST("/connect", h.Connect)
	g.GET("/source/rootfolders", h.GetSourceRootFolders)
	g.GET("/source/qualityprofiles", h.GetSourceQualityProfiles)
	g.POST("/preview", h.Preview)
	g.POST("/execute", h.Execute)
	g.GET("/config/preview", h.GetConfigPreview)
	g.POST("/config/import", h.ExecuteConfigImport)
	g.DELETE("/session", h.Disconnect)
}

// DetectDB returns candidate database paths for the given source type.
func (h *Handlers) DetectDB(c echo.Context) error {
	sourceType := SourceType(c.QueryParam("sourceType"))
	if sourceType != SourceTypeRadarr && sourceType != SourceTypeSonarr {
		return echo.NewHTTPError(http.StatusBadRequest, "sourceType must be 'radarr' or 'sonarr'")
	}

	return c.JSON(http.StatusOK, detectDBPaths(sourceType))
}

// Connect establishes a connection to the source application.
func (h *Handlers) Connect(c echo.Context) error {
	var cfg ConnectionConfig
	if err := c.Bind(&cfg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.service.Connect(c.Request().Context(), cfg); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// GetSourceRootFolders returns root folders from the connected source.
func (h *Handlers) GetSourceRootFolders(c echo.Context) error {
	folders, err := h.service.GetSourceRootFolders(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, folders)
}

// GetSourceQualityProfiles returns quality profiles from the connected source.
func (h *Handlers) GetSourceQualityProfiles(c echo.Context) error {
	profiles, err := h.service.GetSourceQualityProfiles(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, profiles)
}

// Preview generates a preview of the import without making changes.
func (h *Handlers) Preview(c echo.Context) error {
	var mappings ImportMappings
	if err := c.Bind(&mappings); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	preview, err := h.service.Preview(c.Request().Context(), mappings)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, preview)
}

// Execute starts the import process asynchronously.
func (h *Handlers) Execute(c echo.Context) error {
	var mappings ImportMappings
	if err := c.Bind(&mappings); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.service.Execute(c.Request().Context(), mappings); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusAccepted)
}

// GetConfigPreview returns a preview of importable config entities from the source.
func (h *Handlers) GetConfigPreview(c echo.Context) error {
	preview, err := h.service.GetConfigPreview(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, preview)
}

// ExecuteConfigImport imports selected config entities from the source.
func (h *Handlers) ExecuteConfigImport(c echo.Context) error {
	var selections ConfigImportSelections
	if err := c.Bind(&selections); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	report, err := h.service.ExecuteConfigImport(c.Request().Context(), &selections)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, report)
}

// Disconnect closes the connection to the source and clears session state.
func (h *Handlers) Disconnect(c echo.Context) error {
	if err := h.service.Disconnect(c.Request().Context()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
