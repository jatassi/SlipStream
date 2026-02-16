package search

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/metadata"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
)

type MetadataService interface {
	SearchMovies(ctx context.Context, query string, year int) ([]metadata.MovieResult, error)
	SearchSeries(ctx context.Context, query string) ([]metadata.SeriesResult, error)
	GetSeriesSeasons(ctx context.Context, tmdbID, tvdbID int) ([]metadata.SeasonResult, error)
}

type MovieSearchResult struct {
	metadata.MovieResult
	Availability *requests.AvailabilityResult `json:"availability,omitempty"`
}

type SeriesSearchResult struct {
	metadata.SeriesResult
	Availability *requests.AvailabilityResult `json:"availability,omitempty"`
}

type Handlers struct {
	metadataService MetadataService
	libraryChecker  *requests.LibraryChecker
	usersService    *users.Service
}

func NewHandlers(
	metadataService MetadataService,
	libraryChecker *requests.LibraryChecker,
	usersService *users.Service,
) *Handlers {
	return &Handlers{
		metadataService: metadataService,
		libraryChecker:  libraryChecker,
		usersService:    usersService,
	}
}

func (h *Handlers) RegisterRoutes(g *echo.Group, authMiddleware *portalmw.AuthMiddleware) {
	protected := g.Group("")
	protected.Use(authMiddleware.AnyAuth())

	protected.GET("/movie", h.SearchMovies)
	protected.GET("/series", h.SearchSeries)
	protected.GET("/series/seasons", h.GetSeriesSeasons)
}

// SearchMovies searches for movies and enriches with availability
// GET /api/v1/requests/search/movie?query=...&year=...
func (h *Handlers) SearchMovies(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	query := c.QueryParam("query")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter is required")
	}

	year := parseYearParam(c.QueryParam("year"))

	results, err := h.metadataService.SearchMovies(c.Request().Context(), query, year)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	profileID := h.getUserQualityProfileID(c.Request().Context(), claims.UserID)
	enriched := h.enrichMovieResults(c.Request().Context(), results, profileID)
	return c.JSON(http.StatusOK, enriched)
}

func parseYearParam(yearStr string) int {
	if yearStr == "" {
		return 0
	}
	y, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0
	}
	return y
}

func (h *Handlers) getUserQualityProfileID(ctx context.Context, userID int64) *int64 {
	user, err := h.usersService.Get(ctx, userID)
	if err != nil || user == nil {
		return nil
	}
	return user.QualityProfileID
}

func (h *Handlers) enrichMovieResults(ctx context.Context, results []metadata.MovieResult, profileID *int64) []MovieSearchResult {
	enriched := make([]MovieSearchResult, len(results))
	for i := range results {
		enriched[i] = MovieSearchResult{MovieResult: results[i]}
		if results[i].ID > 0 {
			availability, err := h.libraryChecker.CheckMovieAvailability(ctx, int64(results[i].ID), profileID)
			if err == nil {
				enriched[i].Availability = availability
			}
		}
	}
	return enriched
}

// SearchSeries searches for series and enriches with availability
// GET /api/v1/requests/search/series?query=...
func (h *Handlers) SearchSeries(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	query := c.QueryParam("query")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter is required")
	}

	results, err := h.metadataService.SearchSeries(c.Request().Context(), query)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var userQualityProfileID *int64
	user, err := h.usersService.Get(c.Request().Context(), claims.UserID)
	if err == nil && user != nil {
		userQualityProfileID = user.QualityProfileID
	}

	enrichedResults := make([]SeriesSearchResult, len(results))
	for i := range results {
		result := &results[i]
		enrichedResults[i] = SeriesSearchResult{
			SeriesResult: *result,
		}

		if result.TvdbID > 0 {
			availability, err := h.libraryChecker.CheckSeriesAvailability(
				c.Request().Context(),
				int64(result.TvdbID),
				userQualityProfileID,
			)
			if err == nil {
				enrichedResults[i].Availability = availability
			}
		}
	}

	return c.JSON(http.StatusOK, enrichedResults)
}

// GetSeriesSeasons returns the seasons for a series
// GET /api/v1/requests/search/series/seasons?tmdbId=...&tvdbId=...
func (h *Handlers) GetSeriesSeasons(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var tmdbID, tvdbID int
	if tmdbIDStr := c.QueryParam("tmdbId"); tmdbIDStr != "" {
		if id, err := strconv.Atoi(tmdbIDStr); err == nil {
			tmdbID = id
		}
	}
	if tvdbIDStr := c.QueryParam("tvdbId"); tvdbIDStr != "" {
		if id, err := strconv.Atoi(tvdbIDStr); err == nil {
			tvdbID = id
		}
	}

	if tmdbID == 0 && tvdbID == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "either tmdbId or tvdbId is required")
	}

	seasons, err := h.metadataService.GetSeriesSeasons(c.Request().Context(), tmdbID, tvdbID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, seasons)
}
