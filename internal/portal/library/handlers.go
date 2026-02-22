package library

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
)

type MovieService interface {
	List(ctx context.Context, opts movies.ListMoviesOptions) ([]*movies.Movie, error)
}

type TVService interface {
	ListSeries(ctx context.Context, opts tv.ListSeriesOptions) ([]*tv.Series, error)
}

type MovieResult struct {
	ID           int                          `json:"id"`
	TmdbID       int                          `json:"tmdbId"`
	Title        string                       `json:"title"`
	Year         int                          `json:"year,omitempty"`
	Overview     string                       `json:"overview,omitempty"`
	PosterURL    string                       `json:"posterUrl,omitempty"`
	Availability *requests.AvailabilityResult `json:"availability,omitempty"`
}

type SeriesResult struct {
	ID             int                          `json:"id"`
	TmdbID         int                          `json:"tmdbId"`
	TvdbID         int                          `json:"tvdbId,omitempty"`
	Title          string                       `json:"title"`
	Year           int                          `json:"year,omitempty"`
	Overview       string                       `json:"overview,omitempty"`
	Network        string                       `json:"network,omitempty"`
	NetworkLogoURL string                       `json:"networkLogoUrl,omitempty"`
	PosterURL      string                       `json:"posterUrl,omitempty"`
	Availability   *requests.AvailabilityResult `json:"availability,omitempty"`
}

type Handlers struct {
	movieService   MovieService
	tvService      TVService
	libraryChecker *requests.LibraryChecker
	usersService   *users.Service
}

func NewHandlers(
	movieService MovieService,
	tvService TVService,
	libraryChecker *requests.LibraryChecker,
	usersService *users.Service,
) *Handlers {
	return &Handlers{
		movieService:   movieService,
		tvService:      tvService,
		libraryChecker: libraryChecker,
		usersService:   usersService,
	}
}

func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.GET("/movies", h.ListMovies)
	g.GET("/series", h.ListSeries)
}

// ListMovies returns all library movies enriched with availability data.
// GET /api/v1/requests/library/movies
func (h *Handlers) ListMovies(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()

	allMovies, err := h.movieService.List(ctx, movies.ListMoviesOptions{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list movies")
	}

	profileID := h.getUserQualityProfileID(ctx, claims.UserID)

	results := make([]MovieResult, 0, len(allMovies))
	for _, m := range allMovies {
		if m.Status != "available" && m.Status != "upgradable" {
			continue
		}

		result := MovieResult{
			ID:        m.TmdbID,
			TmdbID:    m.TmdbID,
			Title:     m.Title,
			Year:      m.Year,
			Overview:  m.Overview,
			PosterURL: fmt.Sprintf("/api/v1/metadata/artwork/movie/%d/poster", m.TmdbID),
		}

		if m.TmdbID > 0 {
			availability, err := h.libraryChecker.CheckMovieAvailability(ctx, int64(m.TmdbID), profileID, claims.UserID)
			if err == nil {
				result.Availability = availability
			}
		}

		results = append(results, result)
	}

	return c.JSON(http.StatusOK, results)
}

// ListSeries returns all library series enriched with availability data.
// GET /api/v1/requests/library/series
func (h *Handlers) ListSeries(c echo.Context) error {
	claims := portalmw.GetPortalUser(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()

	allSeries, err := h.tvService.ListSeries(ctx, tv.ListSeriesOptions{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list series")
	}

	profileID := h.getUserQualityProfileID(ctx, claims.UserID)

	results := make([]SeriesResult, 0, len(allSeries))
	for _, s := range allSeries {
		result := SeriesResult{
			ID:             s.TmdbID,
			TmdbID:         s.TmdbID,
			TvdbID:         s.TvdbID,
			Title:          s.Title,
			Year:           s.Year,
			Overview:       s.Overview,
			Network:        s.Network,
			NetworkLogoURL: s.NetworkLogoURL,
			PosterURL:      fmt.Sprintf("/api/v1/metadata/artwork/series/%d/poster", s.TmdbID),
		}

		if s.TvdbID > 0 || s.TmdbID > 0 {
			availability, err := h.libraryChecker.CheckSeriesAvailability(ctx, int64(s.TvdbID), int64(s.TmdbID), profileID)
			if err == nil {
				result.Availability = availability
			}
		}

		if !seriesHasFiles(result.Availability) {
			continue
		}

		results = append(results, result)
	}

	return c.JSON(http.StatusOK, results)
}

func seriesHasFiles(availability *requests.AvailabilityResult) bool {
	if availability == nil {
		return false
	}
	for _, sa := range availability.SeasonAvailability {
		if sa.HasAnyFiles {
			return true
		}
	}
	return false
}

func (h *Handlers) getUserQualityProfileID(ctx context.Context, userID int64) *int64 {
	user, err := h.usersService.Get(ctx, userID)
	if err != nil || user == nil {
		return nil
	}
	return user.QualityProfileID
}
