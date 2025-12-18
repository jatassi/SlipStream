package metadata

// Provider defines the interface for metadata providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// SearchMovies searches for movies.
	SearchMovies(query string) ([]MovieResult, error)

	// GetMovie gets movie details by ID.
	GetMovie(id int) (*MovieResult, error)

	// SearchSeries searches for TV series.
	SearchSeries(query string) ([]SeriesResult, error)

	// GetSeries gets series details by ID.
	GetSeries(id int) (*SeriesResult, error)
}

// MovieResult represents a movie from a metadata provider.
type MovieResult struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Overview    string   `json:"overview"`
	PosterURL   string   `json:"posterUrl,omitempty"`
	BackdropURL string   `json:"backdropUrl,omitempty"`
	ImdbID      string   `json:"imdbId,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Runtime     int      `json:"runtime,omitempty"`
}

// SeriesResult represents a TV series from a metadata provider.
type SeriesResult struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Overview    string   `json:"overview"`
	PosterURL   string   `json:"posterUrl,omitempty"`
	BackdropURL string   `json:"backdropUrl,omitempty"`
	ImdbID      string   `json:"imdbId,omitempty"`
	TvdbID      int      `json:"tvdbId,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Status      string   `json:"status,omitempty"`
}
