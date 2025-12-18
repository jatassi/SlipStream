package rootfolder

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for root folder operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new root folder handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the root folder routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.DELETE("/:id", h.Delete)
}

// List returns all root folders.
// GET /api/v1/rootfolders
func (h *Handlers) List(c echo.Context) error {
	mediaType := c.QueryParam("mediaType")

	var folders []*RootFolder
	var err error

	if mediaType != "" {
		folders, err = h.service.ListByType(c.Request().Context(), mediaType)
	} else {
		folders, err = h.service.List(c.Request().Context())
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, folders)
}

// Get returns a single root folder.
// GET /api/v1/rootfolders/:id
func (h *Handlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	folder, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrRootFolderNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, folder)
}

// Create creates a new root folder.
// POST /api/v1/rootfolders
func (h *Handlers) Create(c echo.Context) error {
	var input CreateRootFolderInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	folder, err := h.service.Create(c.Request().Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrPathNotFound):
			return echo.NewHTTPError(http.StatusBadRequest, "path does not exist")
		case errors.Is(err, ErrPathNotDirectory):
			return echo.NewHTTPError(http.StatusBadRequest, "path is not a directory")
		case errors.Is(err, ErrPathAlreadyExists):
			return echo.NewHTTPError(http.StatusConflict, "path already exists as root folder")
		case errors.Is(err, ErrInvalidMediaType):
			return echo.NewHTTPError(http.StatusBadRequest, "media type must be 'movie' or 'tv'")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(http.StatusCreated, folder)
}

// Delete deletes a root folder.
// DELETE /api/v1/rootfolders/:id
func (h *Handlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, ErrRootFolderNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
