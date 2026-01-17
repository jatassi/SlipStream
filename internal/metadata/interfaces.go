package metadata

import (
	"context"

	"github.com/slipstream/slipstream/internal/metadata/tmdb"
	"github.com/slipstream/slipstream/internal/metadata/tvdb"
)

// TMDBClient defines the interface for TMDB API operations.
type TMDBClient interface {
	Name() string
	IsConfigured() bool
	Test(ctx context.Context) error
	SearchMovies(ctx context.Context, query string, year int) ([]tmdb.NormalizedMovieResult, error)
	GetMovie(ctx context.Context, id int) (*tmdb.NormalizedMovieResult, error)
	GetMovieReleaseDates(ctx context.Context, id int) (digital, physical string, err error)
	SearchSeries(ctx context.Context, query string) ([]tmdb.NormalizedSeriesResult, error)
	GetSeries(ctx context.Context, id int) (*tmdb.NormalizedSeriesResult, error)
	GetAllSeasons(ctx context.Context, seriesID int) ([]tmdb.NormalizedSeasonResult, error)
	GetImageURL(path string, size string) string
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
