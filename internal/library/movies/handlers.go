package movies

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for movie operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new movie handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the movie routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	g.GET("/:id/files", h.ListFiles)
	g.POST("/:id/files", h.AddFile)
	g.DELETE("/:id/files/:fileId", h.RemoveFile)
}

// List returns all movies with optional filtering.
// GET /api/v1/movies
func (h *Handlers) List(c echo.Context) error {
	opts := ListMoviesOptions{
		Search: c.QueryParam("search"),
	}

	if monitored := c.QueryParam("monitored"); monitored == "true" {
		m := true
		opts.Monitored = &m
	}
	if rootFolder := c.QueryParam("rootFolderId"); rootFolder != "" {
		id, _ := strconv.ParseInt(rootFolder, 10, 64)
		opts.RootFolderID = &id
	}

	movies, err := h.service.List(c.Request().Context(), opts)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, movies)
}

// Get returns a single movie.
// GET /api/v1/movies/:id
func (h *Handlers) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	movie, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, movie)
}

// Create creates a new movie.
// POST /api/v1/movies
func (h *Handlers) Create(c echo.Context) error {
	var input CreateMovieInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	movie, err := h.service.Create(c.Request().Context(), &input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMovie):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrDuplicateTmdbID):
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(http.StatusCreated, movie)
}

// Update updates an existing movie.
// PUT /api/v1/movies/:id
func (h *Handlers) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input UpdateMovieInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	movie, err := h.service.Update(c.Request().Context(), id, &input)
	if err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, movie)
}

// Delete deletes a movie.
// DELETE /api/v1/movies/:id
func (h *Handlers) Delete(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	deleteFiles := c.QueryParam("deleteFiles") == "true"

	if err := h.service.Delete(c.Request().Context(), id, deleteFiles); err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// ListFiles returns all files for a movie.
// GET /api/v1/movies/:id/files
func (h *Handlers) ListFiles(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	files, err := h.service.GetFiles(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, files)
}

// AddFile adds a file to a movie.
// POST /api/v1/movies/:id/files
func (h *Handlers) AddFile(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input CreateMovieFileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	file, err := h.service.AddFile(c.Request().Context(), id, &input)
	if err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, file)
}

// RemoveFile removes a file from a movie.
// DELETE /api/v1/movies/:id/files/:fileId
func (h *Handlers) RemoveFile(c echo.Context) error {
	fileID, err := strconv.ParseInt(c.Param("fileId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	if err := h.service.RemoveFile(c.Request().Context(), fileID); err != nil {
		if errors.Is(err, ErrMovieFileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
