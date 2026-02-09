package autosearch

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for automatic search operations.
type Handlers struct {
	service           *Service
	scheduledSearcher *ScheduledSearcher
}

// NewHandlers creates new autosearch handlers.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// SetScheduledSearcher sets the scheduled searcher for bulk operations.
func (h *Handlers) SetScheduledSearcher(ss *ScheduledSearcher) {
	h.scheduledSearcher = ss
}

// RegisterRoutes registers the autosearch routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.POST("/movie/:id", h.SearchMovie)
	g.POST("/movie/:id/slot/:slotId", h.SearchMovieSlot)
	g.POST("/episode/:id", h.SearchEpisode)
	g.POST("/episode/:id/slot/:slotId", h.SearchEpisodeSlot)
	g.POST("/season/:seriesId/:seasonNumber", h.SearchSeason)
	g.POST("/series/:id", h.SearchSeries)
	g.GET("/status/:mediaType/:id", h.GetStatus)

	// Retry endpoints (reset failed → missing/upgradable)
	g.POST("/retry/movie/:id", h.RetryMovie)
	g.POST("/retry/episode/:id", h.RetryEpisode)

	// Bulk search endpoints
	g.POST("/missing/all", h.SearchAllMissing)
	g.POST("/missing/movies", h.SearchAllMissingMovies)
	g.POST("/missing/series", h.SearchAllMissingSeries)

	// Upgradable bulk search endpoints
	g.POST("/upgradable/all", h.SearchAllUpgradable)
	g.POST("/upgradable/movies", h.SearchAllUpgradableMovies)
	g.POST("/upgradable/series", h.SearchAllUpgradableSeries)
}

// SearchMovie triggers automatic search for a movie.
// POST /api/v1/autosearch/movie/:id
func (h *Handlers) SearchMovie(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	result, err := h.service.SearchMovie(c.Request().Context(), id, SearchSourceManual)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		}
		if err == ErrAlreadyInQueue {
			return echo.NewHTTPError(http.StatusConflict, "movie already in download queue")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// SearchMovieSlot triggers automatic search for a specific slot of a movie.
// POST /api/v1/autosearch/movie/:id/slot/:slotId
func (h *Handlers) SearchMovieSlot(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	slotIDStr := c.Param("slotId")
	slotID, err := strconv.ParseInt(slotIDStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	result, err := h.service.SearchMovieSlot(c.Request().Context(), id, slotID, SearchSourceManual)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		}
		if err == ErrAlreadyInQueue {
			return echo.NewHTTPError(http.StatusConflict, "movie already in download queue")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// SearchEpisode triggers automatic search for an episode.
// POST /api/v1/autosearch/episode/:id
func (h *Handlers) SearchEpisode(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	result, err := h.service.SearchEpisode(c.Request().Context(), id, SearchSourceManual)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "episode not found")
		}
		if err == ErrAlreadyInQueue {
			return echo.NewHTTPError(http.StatusConflict, "episode already in download queue")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// SearchEpisodeSlot triggers automatic search for a specific slot of an episode.
// POST /api/v1/autosearch/episode/:id/slot/:slotId
func (h *Handlers) SearchEpisodeSlot(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	slotIDStr := c.Param("slotId")
	slotID, err := strconv.ParseInt(slotIDStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid slot id")
	}

	result, err := h.service.SearchEpisodeSlot(c.Request().Context(), id, slotID, SearchSourceManual)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "episode not found")
		}
		if err == ErrAlreadyInQueue {
			return echo.NewHTTPError(http.StatusConflict, "episode already in download queue")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// SearchSeason triggers automatic search for all missing episodes in a season.
// POST /api/v1/autosearch/season/:seriesId/:seasonNumber
func (h *Handlers) SearchSeason(c echo.Context) error {
	seriesIDStr := c.Param("seriesId")
	seriesID, err := strconv.ParseInt(seriesIDStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid series id")
	}

	seasonStr := c.Param("seasonNumber")
	seasonNumber, err := strconv.Atoi(seasonStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid season number")
	}

	result, err := h.service.SearchSeason(c.Request().Context(), seriesID, seasonNumber, SearchSourceManual)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "series not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// SearchSeries triggers automatic search for all missing episodes in a series.
// POST /api/v1/autosearch/series/:id
func (h *Handlers) SearchSeries(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid series id")
	}

	result, err := h.service.SearchSeries(c.Request().Context(), id, SearchSourceManual)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "series not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// GetStatus returns the current search status for an item.
// GET /api/v1/autosearch/status/:mediaType/:id
func (h *Handlers) GetStatus(c echo.Context) error {
	mediaTypeStr := c.Param("mediaType")
	mediaType := MediaType(mediaTypeStr)

	// Validate media type
	switch mediaType {
	case MediaTypeMovie, MediaTypeEpisode, MediaTypeSeason, MediaTypeSeries:
		// Valid
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid media type")
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	status := SearchStatus{
		MediaType: mediaType,
		MediaID:   id,
		Searching: h.service.IsSearching(mediaType, id),
		InQueue:   false, // TODO: Check download queue
	}

	return c.JSON(http.StatusOK, status)
}

// SearchAllMissing triggers automatic search for all missing items (movies and series).
// POST /api/v1/autosearch/missing/all
func (h *Handlers) SearchAllMissing(c echo.Context) error {
	if h.scheduledSearcher == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "scheduled searcher not available")
	}

	if h.scheduledSearcher.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "search task already running")
	}

	// Run in background with a new context (request context will be canceled after response)
	go func() {
		_ = h.scheduledSearcher.Run(context.Background())
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Search started for all missing items",
	})
}

// SearchAllMissingMovies triggers automatic search for all missing movies.
// POST /api/v1/autosearch/missing/movies
func (h *Handlers) SearchAllMissingMovies(c echo.Context) error {
	if h.scheduledSearcher == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "scheduled searcher not available")
	}

	if h.scheduledSearcher.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "search task already running")
	}

	// Run in background with a new context (request context will be canceled after response)
	go func() {
		_ = h.scheduledSearcher.RunMoviesOnly(context.Background())
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Search started for all missing movies",
	})
}

// SearchAllMissingSeries triggers automatic search for all missing series episodes.
// POST /api/v1/autosearch/missing/series
func (h *Handlers) SearchAllMissingSeries(c echo.Context) error {
	if h.scheduledSearcher == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "scheduled searcher not available")
	}

	if h.scheduledSearcher.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "search task already running")
	}

	// Run in background with a new context (request context will be canceled after response)
	go func() {
		_ = h.scheduledSearcher.RunSeriesOnly(context.Background())
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Search started for all missing series",
	})
}

// SearchAllUpgradable triggers automatic search for all upgradable items.
// POST /api/v1/autosearch/upgradable/all
func (h *Handlers) SearchAllUpgradable(c echo.Context) error {
	if h.scheduledSearcher == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "scheduled searcher not available")
	}

	if h.scheduledSearcher.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "search task already running")
	}

	go func() {
		_ = h.scheduledSearcher.Run(context.Background())
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Search started for all upgradable items",
	})
}

// SearchAllUpgradableMovies triggers automatic search for all upgradable movies.
// POST /api/v1/autosearch/upgradable/movies
func (h *Handlers) SearchAllUpgradableMovies(c echo.Context) error {
	if h.scheduledSearcher == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "scheduled searcher not available")
	}

	if h.scheduledSearcher.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "search task already running")
	}

	go func() {
		_ = h.scheduledSearcher.RunUpgradeMoviesOnly(context.Background())
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Search started for all upgradable movies",
	})
}

// SearchAllUpgradableSeries triggers automatic search for all upgradable series episodes.
// POST /api/v1/autosearch/upgradable/series
func (h *Handlers) SearchAllUpgradableSeries(c echo.Context) error {
	if h.scheduledSearcher == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "scheduled searcher not available")
	}

	if h.scheduledSearcher.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "search task already running")
	}

	go func() {
		_ = h.scheduledSearcher.RunUpgradeSeriesOnly(context.Background())
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Search started for all upgradable series",
	})
}

// RetryMovie resets a failed movie back to missing/upgradable and optionally triggers a search.
// POST /api/v1/autosearch/retry/movie/:id
func (h *Handlers) RetryMovie(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid movie id")
	}

	result, err := h.service.RetryMovie(c.Request().Context(), id)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// RetryEpisode resets a failed episode back to missing/upgradable and optionally triggers a search.
// POST /api/v1/autosearch/retry/episode/:id
func (h *Handlers) RetryEpisode(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid episode id")
	}

	result, err := h.service.RetryEpisode(c.Request().Context(), id)
	if err != nil {
		if err == ErrItemNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "episode not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
