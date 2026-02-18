package tv

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for TV operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates new TV handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// RegisterRoutes registers the TV routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.ListSeries)
	g.POST("", h.CreateSeries)
	g.GET("/:id", h.GetSeries)
	g.PUT("/:id", h.UpdateSeries)
	g.DELETE("/:id", h.DeleteSeries)
	g.PUT("/:id/monitor", h.BulkMonitor)
	g.GET("/:id/monitor/stats", h.GetMonitoringStats)
	g.GET("/:id/seasons", h.ListSeasons)
	g.PUT("/:id/seasons/:seasonNumber", h.UpdateSeason)
	g.GET("/:id/episodes", h.ListEpisodes)
	g.PUT("/:id/episodes/monitor", h.BulkMonitorEpisodes)
	g.GET("/:id/episodes/:episodeId", h.GetEpisode)
	g.PUT("/:id/episodes/:episodeId", h.UpdateEpisode)
	g.POST("/:id/episodes/:episodeId/files", h.AddEpisodeFile)
	g.DELETE("/:id/episodes/:episodeId/files/:fileId", h.RemoveEpisodeFile)
}

// ListSeries returns all series with optional filtering.
// GET /api/v1/series
func (h *Handlers) ListSeries(c echo.Context) error {
	opts := ListSeriesOptions{
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

	series, err := h.service.ListSeries(c.Request().Context(), opts)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, series)
}

// GetSeries returns a single series.
// GET /api/v1/series/:id
func (h *Handlers) GetSeries(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	series, err := h.service.GetSeries(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, series)
}

// CreateSeries creates a new series.
// POST /api/v1/series
func (h *Handlers) CreateSeries(c echo.Context) error {
	var input CreateSeriesInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	series, err := h.service.CreateSeries(c.Request().Context(), &input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSeries):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrDuplicateTvdbID):
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(http.StatusCreated, series)
}

// UpdateSeries updates an existing series.
// PUT /api/v1/series/:id
func (h *Handlers) UpdateSeries(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input UpdateSeriesInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	series, err := h.service.UpdateSeries(c.Request().Context(), id, &input)
	if err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, series)
}

// DeleteSeries deletes a series.
// DELETE /api/v1/series/:id
func (h *Handlers) DeleteSeries(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	deleteFiles := c.QueryParam("deleteFiles") == "true"

	if err := h.service.DeleteSeries(c.Request().Context(), id, deleteFiles); err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// BulkMonitor applies a monitoring preset to a series.
// PUT /api/v1/series/:id/monitor
func (h *Handlers) BulkMonitor(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input BulkMonitorInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.BulkMonitor(c.Request().Context(), id, input); err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GetMonitoringStats returns monitoring statistics for a series.
// GET /api/v1/series/:id/monitor/stats
func (h *Handlers) GetMonitoringStats(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	stats, err := h.service.GetMonitoringStats(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, stats)
}

// BulkMonitorEpisodes updates the monitored status of multiple episodes.
// PUT /api/v1/series/:id/episodes/monitor
func (h *Handlers) BulkMonitorEpisodes(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var input BulkEpisodeMonitorInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.BulkMonitorEpisodes(c.Request().Context(), id, input); err != nil {
		if errors.Is(err, ErrSeriesNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ListSeasons returns all seasons for a series.
// GET /api/v1/series/:id/seasons
func (h *Handlers) ListSeasons(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	seasons, err := h.service.ListSeasons(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, seasons)
}

// UpdateSeason updates a season's monitored status.
// PUT /api/v1/series/:id/seasons/:seasonNumber
func (h *Handlers) UpdateSeason(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	seasonNumber, err := strconv.Atoi(c.Param("seasonNumber"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid season number")
	}

	var input struct {
		Monitored bool `json:"monitored"`
	}
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	season, err := h.service.UpdateSeasonMonitored(c.Request().Context(), id, seasonNumber, input.Monitored)
	if err != nil {
		if errors.Is(err, ErrSeasonNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, season)
}

// ListEpisodes returns episodes for a series.
// GET /api/v1/series/:id/episodes
func (h *Handlers) ListEpisodes(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var seasonNumber *int
	if s := c.QueryParam("seasonNumber"); s != "" {
		sn, err := strconv.Atoi(s)
		if err == nil {
			seasonNumber = &sn
		}
	}

	episodes, err := h.service.ListEpisodes(c.Request().Context(), id, seasonNumber)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, episodes)
}

// GetEpisode returns a single episode.
// GET /api/v1/series/:id/episodes/:episodeId
func (h *Handlers) GetEpisode(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	episode, err := h.service.GetEpisode(c.Request().Context(), episodeID)
	if err != nil {
		if errors.Is(err, ErrEpisodeNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, episode)
}

// UpdateEpisode updates an episode.
// PUT /api/v1/series/:id/episodes/:episodeId
func (h *Handlers) UpdateEpisode(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	var input UpdateEpisodeInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	episode, err := h.service.UpdateEpisode(c.Request().Context(), episodeID, input)
	if err != nil {
		if errors.Is(err, ErrEpisodeNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, episode)
}

// AddEpisodeFile adds a file to an episode.
// POST /api/v1/series/:id/episodes/:episodeId/files
func (h *Handlers) AddEpisodeFile(c echo.Context) error {
	episodeID, err := strconv.ParseInt(c.Param("episodeId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	var input CreateEpisodeFileInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	file, err := h.service.AddEpisodeFile(c.Request().Context(), episodeID, &input)
	if err != nil {
		if errors.Is(err, ErrEpisodeNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "episode not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, file)
}

// RemoveEpisodeFile removes a file from an episode.
// DELETE /api/v1/series/:id/episodes/:episodeId/files/:fileId
func (h *Handlers) RemoveEpisodeFile(c echo.Context) error {
	fileID, err := strconv.ParseInt(c.Param("fileId"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	if err := h.service.RemoveEpisodeFile(c.Request().Context(), fileID); err != nil {
		if errors.Is(err, ErrEpisodeFileNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
