package metadata

import (
	"context"

	"github.com/slipstream/slipstream/internal/metadata/omdb"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
	"github.com/slipstream/slipstream/internal/metadata/tvdb"
)

// NetworkLogoStore provides cached network name → logo URL lookups.
type NetworkLogoStore interface {
	UpsertNetworkLogo(ctx context.Context, name, logoURL string) error
	GetNetworkLogoURL(ctx context.Context, name string) (string, error)
	GetAllNetworkLogos(ctx context.Context) (map[string]string, error)
}

// TMDBClient defines the interface for TMDB API operations.
type TMDBClient interface {
	Name() string
	IsConfigured() bool
	Test(ctx context.Context) error
	SearchMovies(ctx context.Context, query string, year int) ([]tmdb.NormalizedMovieResult, error)
	GetMovie(ctx context.Context, id int) (*tmdb.NormalizedMovieResult, error)
	GetMovieReleaseDates(ctx context.Context, id int) (digital, physical, theatrical string, err error)
	SearchSeries(ctx context.Context, query string) ([]tmdb.NormalizedSeriesResult, error)
	GetSeries(ctx context.Context, id int) (*tmdb.NormalizedSeriesResult, error)
	GetAllSeasons(ctx context.Context, seriesID int) ([]tmdb.NormalizedSeasonResult, error)
	GetImageURL(path string, size string) string
	GetMovieCredits(ctx context.Context, id int) (*tmdb.NormalizedCredits, error)
	GetSeriesCredits(ctx context.Context, id int) (*tmdb.NormalizedCredits, error)
	GetMovieContentRating(ctx context.Context, id int) (string, error)
	GetSeriesContentRating(ctx context.Context, id int) (string, error)
	GetMovieStudio(ctx context.Context, id int) (string, error)
	GetMovieLogoURL(ctx context.Context, id int) (string, error)
	GetSeriesLogoURL(ctx context.Context, id int) (string, error)
}

// TVDBClient defines the interface for TVDB API operations.
type TVDBClient interface {
	Name() string
	IsConfigured() bool
	Test(ctx context.Context) error
	SearchSeries(ctx context.Context, query string) ([]tvdb.NormalizedSeriesResult, error)
	GetSeries(ctx context.Context, id int) (*tvdb.NormalizedSeriesResult, error)
	GetSeriesEpisodes(ctx context.Context, id int) ([]tvdb.NormalizedSeasonResult, error)
}

// OMDBClient defines the interface for OMDb API operations.
type OMDBClient interface {
	Name() string
	IsConfigured() bool
	Test(ctx context.Context) error
	GetByIMDbID(ctx context.Context, imdbID string) (*omdb.NormalizedRatings, error)
	GetSeasonEpisodes(ctx context.Context, imdbID string, season int) (map[int]float64, error)
}
