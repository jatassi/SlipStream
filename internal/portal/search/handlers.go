package search

import (
	"context"
	"net/http"
	"strconv"
	"time"

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

type EnrichedEpisodeResult struct {
	metadata.EpisodeResult
	HasFile   bool `json:"hasFile"`
	Monitored bool `json:"monitored"`
	Aired     bool `json:"aired"`
}

type EnrichedSeasonResult struct {
	metadata.SeasonResult
	Episodes                  []EnrichedEpisodeResult `json:"episodes,omitempty"`
	InLibrary                 bool                    `json:"inLibrary"`
	Available                 bool                    `json:"available"`
	Monitored                 bool                    `json:"monitored"`
	AiredEpisodesWithFiles    int                     `json:"airedEpisodesWithFiles"`
	TotalAiredEpisodes        int                     `json:"totalAiredEpisodes"`
	EpisodeCount              int                     `json:"episodeCount"`
	ExistingRequestID         *int64                  `json:"existingRequestId,omitempty"`
	ExistingRequestUserID     *int64                  `json:"existingRequestUserId,omitempty"`
	ExistingRequestStatus     *string                 `json:"existingRequestStatus,omitempty"`
	ExistingRequestIsWatching *bool                   `json:"existingRequestIsWatching,omitempty"`
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
	enriched := h.enrichMovieResults(c.Request().Context(), results, profileID, claims.UserID)
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

func (h *Handlers) enrichMovieResults(ctx context.Context, results []metadata.MovieResult, profileID *int64, currentUserID int64) []MovieSearchResult {
	enriched := make([]MovieSearchResult, len(results))
	for i := range results {
		enriched[i] = MovieSearchResult{MovieResult: results[i]}
		if results[i].ID > 0 {
			availability, err := h.libraryChecker.CheckMovieAvailability(ctx, int64(results[i].ID), profileID, currentUserID)
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

// GetSeriesSeasons returns the seasons for a series enriched with library availability data
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

	ctx := c.Request().Context()

	var availMap map[int]requests.SeasonAvailability
	var coveredMap map[int]requests.CoveredSeason
	if tvdbID > 0 {
		availMap, _ = h.libraryChecker.GetSeasonAvailabilityMap(ctx, int64(tvdbID))
		coveredMap, _ = h.libraryChecker.GetCoveredSeasons(ctx, int64(tvdbID), claims.UserID)
	}

	enriched := h.enrichSeasons(ctx, seasons, int64(tvdbID), availMap, coveredMap)
	return c.JSON(http.StatusOK, enriched)
}

func (h *Handlers) enrichSeasons(ctx context.Context, seasons []metadata.SeasonResult, tvdbID int64, availMap map[int]requests.SeasonAvailability, coveredMap map[int]requests.CoveredSeason) []EnrichedSeasonResult {
	enriched := make([]EnrichedSeasonResult, len(seasons))
	for i, season := range seasons {
		esr := EnrichedSeasonResult{
			SeasonResult: season,
			EpisodeCount: len(season.Episodes),
		}

		if sa, ok := availMap[season.SeasonNumber]; ok {
			esr.InLibrary = sa.HasAnyFiles
			esr.Available = sa.Available
			esr.Monitored = sa.Monitored
			esr.AiredEpisodesWithFiles = sa.AiredEpisodesWithFiles
			esr.TotalAiredEpisodes = sa.TotalAiredEpisodes
		}

		if cs, ok := coveredMap[season.SeasonNumber]; ok {
			esr.ExistingRequestID = &cs.RequestID
			esr.ExistingRequestUserID = &cs.UserID
			esr.ExistingRequestStatus = &cs.Status
			esr.ExistingRequestIsWatching = &cs.IsWatching
		}

		esr.Episodes = h.enrichEpisodes(ctx, &seasons[i], tvdbID, availMap)
		enriched[i] = esr
	}
	return enriched
}

func (h *Handlers) enrichEpisodes(ctx context.Context, season *metadata.SeasonResult, tvdbID int64, availMap map[int]requests.SeasonAvailability) []EnrichedEpisodeResult {
	episodes := make([]EnrichedEpisodeResult, len(season.Episodes))

	var epAvailMap map[int]requests.EpisodeAvailability
	if _, inLibrary := availMap[season.SeasonNumber]; inLibrary && tvdbID > 0 {
		epRows, err := h.libraryChecker.GetEpisodeAvailabilityForSeason(ctx, tvdbID, season.SeasonNumber)
		if err == nil {
			epAvailMap = make(map[int]requests.EpisodeAvailability, len(epRows))
			for _, ea := range epRows {
				epAvailMap[ea.EpisodeNumber] = ea
			}
		}
	}

	for i, ep := range season.Episodes {
		enriched := EnrichedEpisodeResult{
			EpisodeResult: ep,
		}

		if ea, ok := epAvailMap[ep.EpisodeNumber]; ok {
			enriched.HasFile = ea.HasFile
			enriched.Monitored = ea.Monitored
			enriched.Aired = ea.Aired
		} else if ep.AirDate != "" {
			enriched.Aired = isAired(ep.AirDate)
		}

		episodes[i] = enriched
	}

	return episodes
}

func isAired(airDate string) bool {
	if len(airDate) < 10 {
		return false
	}
	t, err := time.Parse("2006-01-02", airDate[:10])
	if err != nil {
		return false
	}
	return !t.After(time.Now().Truncate(24 * time.Hour))
}
