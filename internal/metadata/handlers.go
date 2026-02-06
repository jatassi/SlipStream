package metadata

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Handlers provides HTTP handlers for metadata operations.
type Handlers struct {
	service  *Service
	artwork  *ArtworkDownloader
}

// NewHandlers creates new metadata handlers.
func NewHandlers(service *Service, artwork *ArtworkDownloader) *Handlers {
	return &Handlers{
		service: service,
		artwork: artwork,
	}
}

// RegisterRoutes registers the metadata routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	// Movie metadata
	g.GET("/movie/search", h.SearchMovies)
	g.GET("/movie/:id", h.GetMovie)
	g.GET("/movie/:id/extended", h.GetExtendedMovie)
	g.POST("/movie/:id/artwork", h.DownloadMovieArtwork)

	// Series metadata
	g.GET("/series/search", h.SearchSeries)
	g.GET("/series/tmdb/:id", h.GetSeriesByTMDB)
	g.GET("/series/tvdb/:id", h.GetSeriesByTVDB)
	g.GET("/series/:id", h.GetSeries)
	g.GET("/series/:id/extended", h.GetExtendedSeries)
	g.POST("/series/:id/artwork", h.DownloadSeriesArtwork)

	// Note: Artwork serving route is registered separately via RegisterArtworkRoutes
	// to allow public access (images loaded via <img> tags don't include auth headers)

	// Cache management
	g.DELETE("/cache", h.ClearCache)

	// Provider status
	g.GET("/status", h.GetStatus)
}

// RegisterArtworkRoutes registers artwork routes separately (public, no auth required).
// This is needed because images loaded via <img> tags don't include Authorization headers.
func (h *Handlers) RegisterArtworkRoutes(g *echo.Group) {
	g.GET("/artwork/:type/:id/:artworkType", h.GetArtwork)
}

// SearchMovies searches for movies by query.
// GET /api/v1/metadata/movie/search?query=...&year=...
func (h *Handlers) SearchMovies(c echo.Context) error {
	query := c.QueryParam("query")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter is required")
	}

	// Parse optional year parameter
	var year int
	if yearStr := c.QueryParam("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	results, err := h.service.SearchMovies(c.Request().Context(), query, year)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// GetMovie gets detailed movie info by TMDB ID.
// GET /api/v1/metadata/movie/:id
func (h *Handlers) GetMovie(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.GetMovie(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		if errors.Is(err, ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// GetExtendedMovie gets extended movie info including credits, ratings, and content rating.
// GET /api/v1/metadata/movie/:id/extended
func (h *Handlers) GetExtendedMovie(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.GetExtendedMovie(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		if errors.Is(err, ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DownloadMovieArtwork downloads artwork for a movie.
// POST /api/v1/metadata/movie/:id/artwork
func (h *Handlers) DownloadMovieArtwork(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	// Get movie details first
	movie, err := h.service.GetMovie(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		if errors.Is(err, ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "movie not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Download artwork
	if err := h.artwork.DownloadMovieArtwork(c.Request().Context(), movie); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return paths if available
	response := map[string]string{
		"poster":   h.artwork.GetArtworkPath(MediaTypeMovie, id, ArtworkTypePoster),
		"backdrop": h.artwork.GetArtworkPath(MediaTypeMovie, id, ArtworkTypeBackdrop),
	}

	return c.JSON(http.StatusOK, response)
}

// SearchSeries searches for TV series by query.
// GET /api/v1/metadata/series/search?query=...
func (h *Handlers) SearchSeries(c echo.Context) error {
	query := c.QueryParam("query")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter is required")
	}

	results, err := h.service.SearchSeries(c.Request().Context(), query)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, results)
}

// GetSeries gets series details by ID.
// Tries TMDB first, then falls back to TVDB if the ID might be a TVDB ID.
// GET /api/v1/metadata/series/:id
func (h *Handlers) GetSeries(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	ctx := c.Request().Context()

	// Try TMDB first
	result, err := h.service.GetSeriesByTMDB(ctx, id)
	if err == nil {
		return c.JSON(http.StatusOK, result)
	}

	// If TMDB failed (likely 404), try TVDB as the ID might be a TVDB ID
	// This happens when search results come from TVDB
	if h.service.IsTVDBConfigured() {
		tvdbResult, tvdbErr := h.service.GetSeriesByTVDB(ctx, id)
		if tvdbErr == nil {
			return c.JSON(http.StatusOK, tvdbResult)
		}
	}

	// Both failed, return appropriate error
	if errors.Is(err, ErrNoProvidersConfigured) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
	}
	if errors.Is(err, ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "series not found")
	}
	return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}

// GetExtendedSeries gets extended series info including credits, ratings, seasons, and content rating.
// GET /api/v1/metadata/series/:id/extended
func (h *Handlers) GetExtendedSeries(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.GetExtendedSeries(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		if errors.Is(err, ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "series not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// GetSeriesByTMDB gets series details by TMDB ID.
// GET /api/v1/metadata/series/tmdb/:id
func (h *Handlers) GetSeriesByTMDB(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.GetSeriesByTMDB(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		if errors.Is(err, ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "series not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// GetSeriesByTVDB gets series details by TVDB ID.
// GET /api/v1/metadata/series/tvdb/:id
func (h *Handlers) GetSeriesByTVDB(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	result, err := h.service.GetSeriesByTVDB(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		if errors.Is(err, ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "series not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DownloadSeriesArtwork downloads artwork for a series.
// POST /api/v1/metadata/series/:id/artwork
func (h *Handlers) DownloadSeriesArtwork(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	// Check if this is a TVDB or TMDB ID based on query param, default to TMDB
	source := c.QueryParam("source")

	var series *SeriesResult
	switch source {
	case "tvdb":
		series, err = h.service.GetSeriesByTVDB(c.Request().Context(), id)
	default:
		series, err = h.service.GetSeriesByTMDB(c.Request().Context(), id)
	}

	if err != nil {
		if errors.Is(err, ErrNoProvidersConfigured) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "no metadata providers configured")
		}
		if errors.Is(err, ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "series not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Download artwork
	if err := h.artwork.DownloadSeriesArtwork(c.Request().Context(), series); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Return paths if available
	response := map[string]string{
		"poster":   h.artwork.GetArtworkPath(MediaTypeSeries, id, ArtworkTypePoster),
		"backdrop": h.artwork.GetArtworkPath(MediaTypeSeries, id, ArtworkTypeBackdrop),
	}

	return c.JSON(http.StatusOK, response)
}

// ClearCache clears the metadata cache.
// DELETE /api/v1/metadata/cache
func (h *Handlers) ClearCache(c echo.Context) error {
	h.service.ClearCache()
	return c.NoContent(http.StatusNoContent)
}

// ProviderStatus represents the status of a metadata provider.
type ProviderStatus struct {
	Name       string `json:"name"`
	Configured bool   `json:"configured"`
}

// StatusResponse represents the metadata service status.
type StatusResponse struct {
	Movie  []ProviderStatus `json:"movie"`
	Series []ProviderStatus `json:"series"`
}

// GetStatus returns the status of configured metadata providers.
// GET /api/v1/metadata/status
func (h *Handlers) GetStatus(c echo.Context) error {
	response := StatusResponse{
		Movie: []ProviderStatus{
			{Name: "tmdb", Configured: h.service.HasMovieProvider()},
		},
		Series: []ProviderStatus{
			{Name: "tmdb", Configured: h.service.IsTMDBConfigured()},
			{Name: "tvdb", Configured: h.service.IsTVDBConfigured()},
		},
	}

	return c.JSON(http.StatusOK, response)
}

// GetArtwork serves artwork images from local storage.
// GET /api/v1/metadata/artwork/:type/:id/:artworkType
// :type is "movie" or "series"
// :artworkType is "poster" or "backdrop"
func (h *Handlers) GetArtwork(c echo.Context) error {
	mediaTypeStr := c.Param("type")
	idStr := c.Param("id")
	artworkTypeStr := c.Param("artworkType")

	// Validate media type
	var mediaType MediaType
	switch mediaTypeStr {
	case "movie":
		mediaType = MediaTypeMovie
	case "series":
		mediaType = MediaTypeSeries
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid media type, must be 'movie' or 'series'")
	}

	// Parse ID
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	// Validate artwork type
	var artworkType ArtworkType
	switch artworkTypeStr {
	case "poster":
		artworkType = ArtworkTypePoster
	case "backdrop":
		artworkType = ArtworkTypeBackdrop
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid artwork type, must be 'poster' or 'backdrop'")
	}

	// Get artwork path
	path := h.artwork.GetArtworkPath(mediaType, id, artworkType)
	if path == "" {
		return echo.NewHTTPError(http.StatusNotFound, "artwork not found")
	}

	// Set immutable cache headers — frontend uses ?v= query params for cache busting
	c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	return c.File(path)
}
