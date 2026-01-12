package filesystem

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MediaParser is a function that parses a filename and returns parsed info
type MediaParser func(filename string) *ParsedInfo

// Handlers provides HTTP handlers for filesystem operations
type Handlers struct {
	service        *Service
	storageService *StorageService
	mediaParser    MediaParser
}

// NewHandlers creates a new filesystem handlers instance
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// NewHandlersWithStorage creates a new filesystem handlers instance with storage support
func NewHandlersWithStorage(service *Service, storageService *StorageService) *Handlers {
	return &Handlers{
		service:        service,
		storageService: storageService,
	}
}

// SetMediaParser sets the media parser for scanning files
func (h *Handlers) SetMediaParser(parser MediaParser) {
	h.mediaParser = parser
}

// RegisterRoutes registers the filesystem routes
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("/browse", h.Browse)
	g.GET("/browse/import", h.BrowseForImport)
	g.GET("/storage", h.GetStorage)
	g.POST("/scan", h.ScanForMedia)
}

// Browse handles GET /api/v1/filesystem/browse?path=
// Returns list of directories at the given path
func (h *Handlers) Browse(c echo.Context) error {
	path := c.QueryParam("path")

	result, err := h.service.BrowseDirectory(path)
	if err != nil {
		return h.mapError(err)
	}

	return c.JSON(http.StatusOK, result)
}

// BrowseForImport handles GET /api/v1/filesystem/browse/import?path=
// Returns list of directories and video files at the given path
func (h *Handlers) BrowseForImport(c echo.Context) error {
	path := c.QueryParam("path")

	result, err := h.service.BrowseForImport(path)
	if err != nil {
		return h.mapError(err)
	}

	return c.JSON(http.StatusOK, result)
}

// GetStorage handles GET /api/v1/filesystem/storage
// Returns aggregated storage information with associated root folders
func (h *Handlers) GetStorage(c echo.Context) error {
	if h.storageService == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Storage service not available")
	}

	storage, err := h.storageService.GetStorageInfo(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, storage)
}

// ScanForMedia handles POST /api/v1/filesystem/scan
// Recursively scans a directory for video files and returns parsed metadata
func (h *Handlers) ScanForMedia(c echo.Context) error {
	type request struct {
		Path string `json:"path"`
	}

	var req request
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Path == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Path is required")
	}

	result, err := h.service.ScanForMedia(req.Path, h.mediaParser)
	if err != nil {
		return h.mapError(err)
	}

	return c.JSON(http.StatusOK, result)
}

// mapError maps service errors to HTTP errors
func (h *Handlers) mapError(err error) error {
	switch {
	case errors.Is(err, ErrPathNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "Path does not exist")
	case errors.Is(err, ErrNotDirectory):
		return echo.NewHTTPError(http.StatusBadRequest, "Path is not a directory")
	case errors.Is(err, ErrAccessDenied):
		return echo.NewHTTPError(http.StatusForbidden, "Access denied to this path")
	case errors.Is(err, ErrInvalidPath):
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid path format")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
}
