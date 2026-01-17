package search

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
)

// QualityService provides quality profile operations for search handlers.
type QualityService interface {
	Get(ctx context.Context, id int64) (*quality.Profile, error)
}

// Handlers provides HTTP handlers for search operations.
type Handlers struct {
	service        SearchService
	qualityService QualityService
}

// NewHandlers creates new search handlers.
func NewHandlers(service SearchService, qualityService QualityService) *Handlers {
	return &Handlers{
		service:        service,
		qualityService: qualityService,
	}
}

// RegisterRoutes registers the search routes.
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("", h.Search)
	g.GET("/movie", h.SearchMovie)
	g.GET("/tv", h.SearchTV)
	g.GET("/torrents", h.SearchTorrents)
}

// SearchRequest represents a search request.
// All torrent search endpoints require qualityProfileId for scoring.
type SearchRequest struct {
	Query            string `query:"query"`
	Type             string `query:"type"`       // search, tvsearch, movie
	Categories       string `query:"categories"` // comma-separated category IDs
	ImdbID           string `query:"imdbId"`
	TmdbID           int    `query:"tmdbId"`
	TvdbID           int    `query:"tvdbId"`
	Season           int    `query:"season"`
	Episode          int    `query:"episode"`
	Year             int    `query:"year"`
	Limit            int    `query:"limit"`
	Offset           int    `query:"offset"`
	QualityProfileID int64  `query:"qualityProfileId"`
}

// Search handles general search requests.
// GET /api/v1/search?query=...&type=...&categories=...
// Note: This endpoint returns basic ReleaseInfo without torrent-specific fields or scoring.
// For torrent searches with scoring, use /movie, /tv, or /torrents endpoints.
func (h *Handlers) Search(c echo.Context) error {
	var req SearchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request parameters",
		})
	}

	criteria := h.toCriteria(req)

	result, err := h.service.Search(c.Request().Context(), criteria)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// SearchMovie handles movie-specific search requests with desirability scoring.
// GET /api/v1/search/movie?qualityProfileId=...&query=...&tmdbId=...&imdbId=...&year=...
// Returns TorrentSearchResult with scores populated, sorted by score descending.
func (h *Handlers) SearchMovie(c echo.Context) error {
	var req SearchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request parameters",
		})
	}

	// Quality profile is required for scoring
	if req.QualityProfileID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "qualityProfileId is required",
		})
	}

	// Fetch the quality profile
	profile, err := h.qualityService.Get(c.Request().Context(), req.QualityProfileID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Quality profile not found",
		})
	}

	criteria := h.toCriteria(req)
	criteria.Type = "movie"

	params := ScoredSearchParams{
		QualityProfile: profile,
		SearchYear:     req.Year,
	}

	result, err := h.service.SearchTorrents(c.Request().Context(), criteria, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// SearchTV handles TV-specific search requests with desirability scoring.
// GET /api/v1/search/tv?qualityProfileId=...&query=...&tvdbId=...&season=...&episode=...
// Returns TorrentSearchResult with scores populated, sorted by score descending.
func (h *Handlers) SearchTV(c echo.Context) error {
	var req SearchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request parameters",
		})
	}

	// Quality profile is required for scoring
	if req.QualityProfileID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "qualityProfileId is required",
		})
	}

	// Fetch the quality profile
	profile, err := h.qualityService.Get(c.Request().Context(), req.QualityProfileID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Quality profile not found",
		})
	}

	criteria := h.toCriteria(req)
	criteria.Type = "tvsearch"

	params := ScoredSearchParams{
		QualityProfile: profile,
		SearchSeason:   req.Season,
		SearchEpisode:  req.Episode,
	}

	result, err := h.service.SearchTorrents(c.Request().Context(), criteria, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// SearchTorrents handles torrent-specific search requests with desirability scoring.
// GET /api/v1/search/torrents?qualityProfileId=...&query=...&type=...
// Returns TorrentSearchResult with scores populated, sorted by score descending.
func (h *Handlers) SearchTorrents(c echo.Context) error {
	var req SearchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request parameters",
		})
	}

	// Quality profile is required for scoring
	if req.QualityProfileID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "qualityProfileId is required",
		})
	}

	// Fetch the quality profile
	profile, err := h.qualityService.Get(c.Request().Context(), req.QualityProfileID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Quality profile not found",
		})
	}

	criteria := h.toCriteria(req)

	params := ScoredSearchParams{
		QualityProfile: profile,
		SearchYear:     req.Year,
		SearchSeason:   req.Season,
		SearchEpisode:  req.Episode,
	}

	result, err := h.service.SearchTorrents(c.Request().Context(), criteria, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// toCriteria converts a SearchRequest to SearchCriteria.
func (h *Handlers) toCriteria(req SearchRequest) types.SearchCriteria {
	criteria := types.SearchCriteria{
		Query:   req.Query,
		Type:    req.Type,
		ImdbID:  req.ImdbID,
		TmdbID:  req.TmdbID,
		TvdbID:  req.TvdbID,
		Season:  req.Season,
		Episode: req.Episode,
		Year:    req.Year,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}

	// Default search type
	if criteria.Type == "" {
		criteria.Type = "search"
	}

	// Parse categories
	if req.Categories != "" {
		parts := strings.Split(req.Categories, ",")
		categories := make([]int, 0, len(parts))
		for _, part := range parts {
			if cat, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				categories = append(categories, cat)
			}
		}
		criteria.Categories = categories
	}

	// Default limit
	if criteria.Limit == 0 {
		criteria.Limit = 100
	}

	return criteria
}
