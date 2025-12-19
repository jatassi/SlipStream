package tvdb

// LoginRequest is the request body for TVDB authentication.
type LoginRequest struct {
	APIKey string `json:"apikey"`
}

// LoginResponse is the response from TVDB authentication.
type LoginResponse struct {
	Status string `json:"status"`
	Data   struct {
		Token string `json:"token"`
	} `json:"data"`
}

// SearchResponse is the response from TVDB search.
type SearchResponse struct {
	Status string         `json:"status"`
	Data   []SearchResult `json:"data"`
}

// SearchResult is a search result from TVDB.
type SearchResult struct {
	ObjectID       string   `json:"objectID"`
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	Type           string   `json:"type"` // "series", "movie", etc.
	Year           string   `json:"year"`
	Overview       string   `json:"overview"`
	ImageURL       string   `json:"image_url"`
	PrimaryType    string   `json:"primary_type"`
	Status         string   `json:"status"`
	FirstAirTime   string   `json:"first_air_time"`
	Network        string   `json:"network"`
	TvdbID         string   `json:"tvdb_id"`
	RemoteIDs      []RemoteID `json:"remote_ids"`
	Overviews      map[string]string `json:"overviews"`
	Translations   map[string]string `json:"translations"`
}

// RemoteID represents an external ID.
type RemoteID struct {
	ID         string `json:"id"`
	Type       int    `json:"type"`
	SourceName string `json:"sourceName"`
}

// SeriesResponse is the response for a single series.
type SeriesResponse struct {
	Status string       `json:"status"`
	Data   SeriesDetail `json:"data"`
}

// SeriesDetail contains detailed series information.
type SeriesDetail struct {
	ID                 int            `json:"id"`
	Name               string         `json:"name"`
	Slug               string         `json:"slug"`
	Image              string         `json:"image"`
	FirstAired         string         `json:"firstAired"`
	LastAired          string         `json:"lastAired"`
	NextAired          string         `json:"nextAired"`
	Score              float64        `json:"score"`
	Status             SeriesStatus   `json:"status"`
	OriginalCountry    string         `json:"originalCountry"`
	OriginalLanguage   string         `json:"originalLanguage"`
	DefaultSeasonType  int            `json:"defaultSeasonType"`
	IsOrderRandomized  bool           `json:"isOrderRandomized"`
	LastUpdated        string         `json:"lastUpdated"`
	AverageRuntime     int            `json:"averageRuntime"`
	Overview           string         `json:"overview"`
	Year               string         `json:"year"`
	Artworks           []Artwork      `json:"artworks"`
	Genres             []Genre        `json:"genres"`
	RemoteIDs          []SeriesRemoteID `json:"remoteIds"`
	Aliases            []Alias        `json:"aliases"`
}

// SeriesStatus represents the status of a series.
type SeriesStatus struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	RecordType  string `json:"recordType"`
	KeepUpdated bool   `json:"keepUpdated"`
}

// Genre represents a genre.
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Artwork represents artwork for a series.
type Artwork struct {
	ID         int    `json:"id"`
	Image      string `json:"image"`
	Thumbnail  string `json:"thumbnail"`
	Language   string `json:"language"`
	Type       int    `json:"type"`
	Score      float64 `json:"score"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

// SeriesRemoteID represents an external ID for a series.
type SeriesRemoteID struct {
	ID         string `json:"id"`
	Type       int    `json:"type"`
	SourceName string `json:"sourceName"`
}

// Alias represents an alias for a series.
type Alias struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

// EpisodesResponse is the response for series episodes.
type EpisodesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Series   SeriesDetail `json:"series"`
		Episodes []Episode    `json:"episodes"`
	} `json:"data"`
	Links Links `json:"links"`
}

// Episode represents a TV episode.
type Episode struct {
	ID                 int     `json:"id"`
	SeriesID           int     `json:"seriesId"`
	Name               string  `json:"name"`
	Aired              string  `json:"aired"`
	Runtime            int     `json:"runtime"`
	Overview           string  `json:"overview"`
	Image              string  `json:"image"`
	ImageType          int     `json:"imageType"`
	ProductionCode     string  `json:"productionCode"`
	SeasonNumber       int     `json:"seasonNumber"`
	Number             int     `json:"number"`
	AbsoluteNumber     int     `json:"absoluteNumber"`
	IsMovie            int     `json:"isMovie"`
	LastUpdated        string  `json:"lastUpdated"`
	FinaleType         string  `json:"finaleType"`
	Year               string  `json:"year"`
}

// Links contains pagination links.
type Links struct {
	Prev  string `json:"prev"`
	Self  string `json:"self"`
	Next  string `json:"next"`
	Total int    `json:"total_items"`
}

// ErrorResponse is an error from the TVDB API.
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NormalizedMovieResult is the normalized movie result (TVDB doesn't support movies).
type NormalizedMovieResult struct {
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

// NormalizedSeriesResult is the normalized series result returned by the client.
type NormalizedSeriesResult struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Overview    string   `json:"overview"`
	PosterURL   string   `json:"posterUrl,omitempty"`
	BackdropURL string   `json:"backdropUrl,omitempty"`
	ImdbID      string   `json:"imdbId,omitempty"`
	TvdbID      int      `json:"tvdbId,omitempty"`
	TmdbID      int      `json:"tmdbId,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Status      string   `json:"status,omitempty"`
	Runtime     int      `json:"runtime,omitempty"`
}
