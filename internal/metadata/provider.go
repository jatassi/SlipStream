package metadata

import "context"

// Provider defines the interface for metadata providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// IsConfigured returns true if the provider has required configuration.
	IsConfigured() bool

	// SearchMovies searches for movies.
	SearchMovies(ctx context.Context, query string) ([]MovieResult, error)

	// GetMovie gets movie details by ID.
	GetMovie(ctx context.Context, id int) (*MovieResult, error)

	// SearchSeries searches for TV series.
	SearchSeries(ctx context.Context, query string) ([]SeriesResult, error)

	// GetSeries gets series details by ID.
	GetSeries(ctx context.Context, id int) (*SeriesResult, error)
}

// MovieResult represents a movie from a metadata provider.
type MovieResult struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Overview    string   `json:"overview"`
	PosterURL   string   `json:"posterUrl,omitempty"`
	BackdropURL string   `json:"backdropUrl,omitempty"`
	LogoURL     string   `json:"logoUrl,omitempty"`
	ImdbID      string   `json:"imdbId,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Runtime     int      `json:"runtime,omitempty"`
	Studio      string   `json:"studio,omitempty"`
}

// SeriesResult represents a TV series from a metadata provider.
type SeriesResult struct {
	ID             int      `json:"id"`
	Title          string   `json:"title"`
	Year           int      `json:"year"`
	Overview       string   `json:"overview"`
	PosterURL      string   `json:"posterUrl,omitempty"`
	BackdropURL    string   `json:"backdropUrl,omitempty"`
	LogoURL        string   `json:"logoUrl,omitempty"`
	ImdbID         string   `json:"imdbId,omitempty"`
	TvdbID         int      `json:"tvdbId,omitempty"`
	TmdbID         int      `json:"tmdbId,omitempty"`
	Genres         []string `json:"genres,omitempty"`
	Status         string   `json:"status,omitempty"`
	Runtime        int      `json:"runtime,omitempty"`
	Network        string   `json:"network,omitempty"`
	NetworkLogoURL string   `json:"networkLogoUrl,omitempty"`
}

// SeasonResult represents a TV season with episodes from a metadata provider.
type SeasonResult struct {
	SeasonNumber int             `json:"seasonNumber"`
	Name         string          `json:"name"`
	Overview     string          `json:"overview,omitempty"`
	PosterURL    string          `json:"posterUrl,omitempty"`
	AirDate      string          `json:"airDate,omitempty"`
	Episodes     []EpisodeResult `json:"episodes,omitempty"`
}

// EpisodeResult represents a TV episode from a metadata provider.
type EpisodeResult struct {
	EpisodeNumber int    `json:"episodeNumber"`
	SeasonNumber  int    `json:"seasonNumber"`
	Title         string `json:"title"`
	Overview      string `json:"overview,omitempty"`
	AirDate       string `json:"airDate,omitempty"`
	Runtime       int    `json:"runtime,omitempty"`
}

// Person represents a person (actor, director, writer, etc.) from metadata.
type Person struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role,omitempty"`
	PhotoURL string `json:"photoUrl,omitempty"`
}

// Credits represents cast and crew information.
type Credits struct {
	Directors []Person `json:"directors,omitempty"`
	Writers   []Person `json:"writers,omitempty"`
	Creators  []Person `json:"creators,omitempty"`
	Cast      []Person `json:"cast"`
}

// ExternalRatings represents ratings from external sources like IMDB and Rotten Tomatoes.
type ExternalRatings struct {
	ImdbRating     float64 `json:"imdbRating,omitempty"`
	ImdbVotes      int     `json:"imdbVotes,omitempty"`
	RottenTomatoes int     `json:"rottenTomatoes,omitempty"`
	RottenAudience int     `json:"rottenAudience,omitempty"`
	Metacritic     int     `json:"metacritic,omitempty"`
	Awards         string  `json:"awards,omitempty"`
}

// ExtendedMovieResult extends MovieResult with credits, ratings, and content rating.
type ExtendedMovieResult struct {
	MovieResult
	Credits       *Credits         `json:"credits,omitempty"`
	ContentRating string           `json:"contentRating,omitempty"`
	Studio        string           `json:"studio,omitempty"`
	Ratings       *ExternalRatings `json:"ratings,omitempty"`
}

// ExtendedSeriesResult extends SeriesResult with credits, ratings, seasons, and content rating.
type ExtendedSeriesResult struct {
	SeriesResult
	Credits       *Credits         `json:"credits,omitempty"`
	ContentRating string           `json:"contentRating,omitempty"`
	Ratings       *ExternalRatings `json:"ratings,omitempty"`
	Seasons       []SeasonResult   `json:"seasons,omitempty"`
}
