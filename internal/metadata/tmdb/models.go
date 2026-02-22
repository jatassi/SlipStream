package tmdb

// SearchMoviesResponse is the response from TMDB movie search.
type SearchMoviesResponse struct {
	Page         int           `json:"page"`
	Results      []MovieResult `json:"results"`
	TotalPages   int           `json:"total_pages"`
	TotalResults int           `json:"total_results"`
}

// MovieResult is a movie from TMDB search results.
type MovieResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date"`
	PosterPath    *string `json:"poster_path"`
	BackdropPath  *string `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Popularity    float64 `json:"popularity"`
	Adult         bool    `json:"adult"`
	GenreIDs      []int   `json:"genre_ids"`
}

// MovieDetails is the detailed movie info from TMDB.
type MovieDetails struct {
	ID                  int                 `json:"id"`
	Title               string              `json:"title"`
	OriginalTitle       string              `json:"original_title"`
	Overview            string              `json:"overview"`
	ReleaseDate         string              `json:"release_date"`
	PosterPath          *string             `json:"poster_path"`
	BackdropPath        *string             `json:"backdrop_path"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
	Popularity          float64             `json:"popularity"`
	Adult               bool                `json:"adult"`
	Runtime             int                 `json:"runtime"`
	Budget              int64               `json:"budget"`
	Revenue             int64               `json:"revenue"`
	Status              string              `json:"status"`
	Tagline             string              `json:"tagline"`
	ImdbID              string              `json:"imdb_id"`
	OriginalLanguage    string              `json:"original_language"`
	Genres              []Genre             `json:"genres"`
	ExternalIDs         *ExternalIDs        `json:"external_ids,omitempty"`
	ProductionCompanies []ProductionCompany `json:"production_companies,omitempty"`
}

// SearchTVResponse is the response from TMDB TV search.
type SearchTVResponse struct {
	Page         int        `json:"page"`
	Results      []TVResult `json:"results"`
	TotalPages   int        `json:"total_pages"`
	TotalResults int        `json:"total_results"`
}

// TVResult is a TV series from TMDB search results.
type TVResult struct {
	ID               int      `json:"id"`
	Name             string   `json:"name"`
	OriginalName     string   `json:"original_name"`
	Overview         string   `json:"overview"`
	FirstAirDate     string   `json:"first_air_date"`
	PosterPath       *string  `json:"poster_path"`
	BackdropPath     *string  `json:"backdrop_path"`
	VoteAverage      float64  `json:"vote_average"`
	VoteCount        int      `json:"vote_count"`
	Popularity       float64  `json:"popularity"`
	GenreIDs         []int    `json:"genre_ids"`
	OriginCountry    []string `json:"origin_country"`
	OriginalLanguage string   `json:"original_language"`
}

// TVDetails is the detailed TV series info from TMDB.
type TVDetails struct {
	ID               int          `json:"id"`
	Name             string       `json:"name"`
	OriginalName     string       `json:"original_name"`
	Overview         string       `json:"overview"`
	FirstAirDate     string       `json:"first_air_date"`
	LastAirDate      string       `json:"last_air_date"`
	PosterPath       *string      `json:"poster_path"`
	BackdropPath     *string      `json:"backdrop_path"`
	VoteAverage      float64      `json:"vote_average"`
	VoteCount        int          `json:"vote_count"`
	Popularity       float64      `json:"popularity"`
	Status           string       `json:"status"`
	Type             string       `json:"type"`
	Tagline          string       `json:"tagline"`
	OriginalLanguage string       `json:"original_language"`
	Genres           []Genre      `json:"genres"`
	Networks         []Network    `json:"networks"`
	NumberOfSeasons  int          `json:"number_of_seasons"`
	NumberOfEpisodes int          `json:"number_of_episodes"`
	EpisodeRunTime   []int        `json:"episode_run_time"`
	Seasons          []Season     `json:"seasons"`
	ExternalIDs      *ExternalIDs `json:"external_ids,omitempty"`
	CreatedBy        []TVCreator  `json:"created_by,omitempty"`
}

// Genre represents a genre from TMDB.
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Network represents a TV network from TMDB.
type Network struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

// Season represents a TV season from TMDB.
type Season struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	AirDate      string  `json:"air_date"`
	EpisodeCount int     `json:"episode_count"`
	PosterPath   *string `json:"poster_path"`
	SeasonNumber int     `json:"season_number"`
}

// ExternalIDs contains external IDs from TMDB.
type ExternalIDs struct {
	ImdbID      string `json:"imdb_id"`
	TvdbID      int    `json:"tvdb_id"`
	WikidataID  string `json:"wikidata_id"`
	FacebookID  string `json:"facebook_id"`
	InstagramID string `json:"instagram_id"`
	TwitterID   string `json:"twitter_id"`
}

// ErrorResponse is an error from the TMDB API.
type ErrorResponse struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Success       bool   `json:"success"`
}

// NormalizedMovieResult is the normalized movie result returned by the client.
type NormalizedMovieResult struct {
	ID                  int      `json:"id"`
	Title               string   `json:"title"`
	Year                int      `json:"year"`
	Overview            string   `json:"overview"`
	PosterURL           string   `json:"posterUrl,omitempty"`
	BackdropURL         string   `json:"backdropUrl,omitempty"`
	ImdbID              string   `json:"imdbId,omitempty"`
	Genres              []string `json:"genres,omitempty"`
	Runtime             int      `json:"runtime,omitempty"`
	ReleaseDate         string   `json:"releaseDate,omitempty"`
	DigitalReleaseDate  string   `json:"digitalReleaseDate,omitempty"`
	PhysicalReleaseDate string   `json:"physicalReleaseDate,omitempty"`
	Studio              string   `json:"studio,omitempty"`
	StudioLogoURL       string   `json:"studioLogoUrl,omitempty"`
}

// NormalizedSeriesResult is the normalized series result returned by the client.
type NormalizedSeriesResult struct {
	ID             int      `json:"id"`
	Title          string   `json:"title"`
	Year           int      `json:"year"`
	Overview       string   `json:"overview"`
	PosterURL      string   `json:"posterUrl,omitempty"`
	BackdropURL    string   `json:"backdropUrl,omitempty"`
	ImdbID         string   `json:"imdbId,omitempty"`
	TvdbID         int      `json:"tvdbId,omitempty"`
	TmdbID         int      `json:"tmdbId,omitempty"`
	Genres         []string `json:"genres,omitempty"`
	Status         string   `json:"status,omitempty"`
	Runtime        int      `json:"runtime,omitempty"`
	Network        string   `json:"network,omitempty"`
	NetworkLogoURL string   `json:"networkLogoUrl,omitempty"`
}

// SeasonDetails is the detailed season info from TMDB /tv/{id}/season/{number} endpoint.
type SeasonDetails struct {
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	Overview     string           `json:"overview"`
	AirDate      string           `json:"air_date"`
	PosterPath   *string          `json:"poster_path"`
	SeasonNumber int              `json:"season_number"`
	Episodes     []EpisodeDetails `json:"episodes"`
}

// EpisodeDetails is the episode info from TMDB season details.
type EpisodeDetails struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	AirDate       string  `json:"air_date"`
	EpisodeNumber int     `json:"episode_number"`
	SeasonNumber  int     `json:"season_number"`
	StillPath     *string `json:"still_path"`
	Runtime       int     `json:"runtime"`
}

// NormalizedSeasonResult is the normalized season result with episodes.
type NormalizedSeasonResult struct {
	SeasonNumber int                       `json:"seasonNumber"`
	Name         string                    `json:"name"`
	Overview     string                    `json:"overview"`
	PosterURL    string                    `json:"posterUrl,omitempty"`
	AirDate      string                    `json:"airDate,omitempty"`
	Episodes     []NormalizedEpisodeResult `json:"episodes"`
}

// NormalizedEpisodeResult is the normalized episode result.
type NormalizedEpisodeResult struct {
	EpisodeNumber int    `json:"episodeNumber"`
	SeasonNumber  int    `json:"seasonNumber"`
	Title         string `json:"title"`
	Overview      string `json:"overview,omitempty"`
	AirDate       string `json:"airDate,omitempty"`
	Runtime       int    `json:"runtime,omitempty"`
}

// ReleaseDatesResponse is the response from TMDB /movie/{id}/release_dates endpoint.
type ReleaseDatesResponse struct {
	ID      int                    `json:"id"`
	Results []ReleaseDatesByRegion `json:"results"`
}

// ReleaseDatesByRegion contains release dates for a specific country/region.
type ReleaseDatesByRegion struct {
	Iso31661     string        `json:"iso_3166_1"`
	ReleaseDates []ReleaseDate `json:"release_dates"`
}

// ReleaseDate represents a single release date entry.
// Type: 1=Premiere, 2=Theatrical (limited), 3=Theatrical, 4=Digital, 5=Physical, 6=TV
type ReleaseDate struct {
	Certification string `json:"certification"`
	Descriptors   []any  `json:"descriptors"`
	Iso6391       string `json:"iso_639_1"`
	Note          string `json:"note"`
	ReleaseDate   string `json:"release_date"`
	Type          int    `json:"type"`
}

// ReleaseDateType constants for TMDB release date types.
const (
	ReleaseDateTypePremiere          = 1
	ReleaseDateTypeTheatricalLimited = 2
	ReleaseDateTypeTheatrical        = 3
	ReleaseDateTypeDigital           = 4
	ReleaseDateTypePhysical          = 5
	ReleaseDateTypeTV                = 6
)

// CreditsResponse is the response from TMDB credits endpoint.
type CreditsResponse struct {
	ID   int          `json:"id"`
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

// CastMember represents a cast member from TMDB credits.
type CastMember struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Character   string  `json:"character"`
	Order       int     `json:"order"`
	ProfilePath *string `json:"profile_path"`
}

// CrewMember represents a crew member from TMDB credits.
type CrewMember struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Job         string  `json:"job"`
	Department  string  `json:"department"`
	ProfilePath *string `json:"profile_path"`
}

// NormalizedCredits is the normalized credits output.
type NormalizedCredits struct {
	Directors []NormalizedPerson `json:"directors,omitempty"`
	Writers   []NormalizedPerson `json:"writers,omitempty"`
	Creators  []NormalizedPerson `json:"creators,omitempty"`
	Cast      []NormalizedPerson `json:"cast"`
}

// NormalizedPerson represents a person with optional role and photo.
type NormalizedPerson struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role,omitempty"`
	PhotoURL string `json:"photoUrl,omitempty"`
}

// ContentRatingsResponse is the response from TMDB TV content ratings endpoint.
type ContentRatingsResponse struct {
	Results []ContentRating `json:"results"`
}

// ContentRating represents a content rating from TMDB.
type ContentRating struct {
	Iso31661 string `json:"iso_3166_1"`
	Rating   string `json:"rating"`
}

// TVCreator represents a series creator from TMDB TV details.
type TVCreator struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	ProfilePath *string `json:"profile_path"`
}

// ProductionCompany represents a production company from TMDB.
type ProductionCompany struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	LogoPath *string `json:"logo_path"`
}

// VideosResponse is the response from TMDB /movie/{id}/videos or /tv/{id}/videos.
type VideosResponse struct {
	Results []Video `json:"results"`
}

// Video represents a video (trailer, teaser, etc.) from TMDB.
type Video struct {
	Key      string `json:"key"`
	Site     string `json:"site"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Official bool   `json:"official"`
}

// ImagesResponse is the response from TMDB /movie/{id}/images or /tv/{id}/images.
type ImagesResponse struct {
	Logos []ImageResult `json:"logos"`
}

// ImageResult represents a single image from TMDB images endpoint.
type ImageResult struct {
	FilePath    string  `json:"file_path"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	VoteAverage float64 `json:"vote_average"`
	Iso6391     string  `json:"iso_639_1"`
}
