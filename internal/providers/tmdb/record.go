package tmdb

// Movie is TMDB's whole record of one movie, every appended block decoded.
// ReleaseDate is the primary release date, YYYY-MM-DD, or empty.
type Movie struct {
	ID                  int64             `json:"id"`
	IMDbID              string            `json:"imdb_id"`
	Title               string            `json:"title"`
	OriginalTitle       string            `json:"original_title"`
	OriginalLanguage    string            `json:"original_language"`
	Overview            string            `json:"overview"`
	Tagline             string            `json:"tagline"`
	ReleaseDate         string            `json:"release_date"`
	Runtime             int64             `json:"runtime"`
	Status              string            `json:"status"`
	Genres              []Genre           `json:"genres"`
	ProductionCompanies []Company         `json:"production_companies"`
	BelongsToCollection *Collection       `json:"belongs_to_collection"`
	VoteAverage         float64           `json:"vote_average"`
	VoteCount           int64             `json:"vote_count"`
	PosterPath          string            `json:"poster_path"`
	BackdropPath        string            `json:"backdrop_path"`
	ExternalIDs         ExternalIDs       `json:"external_ids"`
	ReleaseDates        ReleaseDates      `json:"release_dates"`
	AlternativeTitles   AlternativeTitles `json:"alternative_titles"`
	Credits             Credits           `json:"credits"`
	Videos              Videos            `json:"videos"`
	Images              Images            `json:"images"`
	Translations        Translations      `json:"translations"`
}

type Genre struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Company struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type Collection struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
}

type ExternalIDs struct {
	IMDbID string `json:"imdb_id"`
}

type ReleaseDates struct {
	Results []CountryReleases `json:"results"`
}

type CountryReleases struct {
	Country  string    `json:"iso_3166_1"`
	Releases []Release `json:"release_dates"`
}

// Release is one dated release in one country. Date is RFC 3339.
type Release struct {
	Certification string      `json:"certification"`
	Language      string      `json:"iso_639_1"`
	Note          string      `json:"note"`
	Date          string      `json:"release_date"`
	Type          ReleaseType `json:"type"`
}

// ReleaseType is TMDB's kind of release.
type ReleaseType int

const (
	Premiere ReleaseType = iota + 1
	TheatricalLimited
	Theatrical
	Digital
	Physical
	TV
)

type AlternativeTitles struct {
	Titles []AlternativeTitle `json:"titles"`
}

// AlternativeTitle is a title the movie goes by in Country. Type is
// TMDB's free-text kind, such as "working title".
type AlternativeTitle struct {
	Country string `json:"iso_3166_1"`
	Title   string `json:"title"`
	Type    string `json:"type"`
}

type Credits struct {
	Cast []Cast `json:"cast"`
	Crew []Crew `json:"crew"`
}

// Cast is one role; Order is its billing, 0 first.
type Cast struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	Order       int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

type Crew struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Department  string `json:"department"`
	Job         string `json:"job"`
	ProfilePath string `json:"profile_path"`
}

type Videos struct {
	Results []Video `json:"results"`
}

// Video is one clip TMDB links, such as a YouTube trailer: Site and Key
// name it there.
type Video struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Site        string `json:"site"`
	Type        string `json:"type"`
	Official    bool   `json:"official"`
	Language    string `json:"iso_639_1"`
	Country     string `json:"iso_3166_1"`
	PublishedAt string `json:"published_at"`
}

type Images struct {
	Backdrops []Image `json:"backdrops"`
	Logos     []Image `json:"logos"`
	Posters   []Image `json:"posters"`
}

// Image is one image file. Language is empty for an image with no text.
type Image struct {
	FilePath    string  `json:"file_path"`
	Language    string  `json:"iso_639_1"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int64   `json:"vote_count"`
}

type Translations struct {
	Translations []Translation `json:"translations"`
}

// Translation is the movie's text in one language, such as en and US.
type Translation struct {
	Language string          `json:"iso_639_1"`
	Country  string          `json:"iso_3166_1"`
	Data     TranslationData `json:"data"`
}

type TranslationData struct {
	Title    string `json:"title"`
	Overview string `json:"overview"`
	Tagline  string `json:"tagline"`
}
