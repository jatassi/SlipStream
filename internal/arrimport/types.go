package arrimport

import "time"

// SourceType identifies the source application type.
type SourceType string

const (
	SourceTypeRadarr SourceType = "radarr"
	SourceTypeSonarr SourceType = "sonarr"
)

// ConnectionConfig holds the connection parameters for a source.
type ConnectionConfig struct {
	SourceType SourceType `json:"sourceType"`
	DBPath     string     `json:"dbPath,omitempty"`
	URL        string     `json:"url,omitempty"`
	APIKey     string     `json:"apiKey,omitempty"`
}

// SourceRootFolder represents a root folder from the source application.
type SourceRootFolder struct {
	ID   int64  `json:"id"`
	Path string `json:"path"`
}

// SourceQualityProfile represents a quality profile from the source application.
type SourceQualityProfile struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	InUse bool   `json:"inUse"`
}

// SourceMovie represents a movie from the source (Radarr).
type SourceMovie struct {
	ID               int64            `json:"id"`
	Title            string           `json:"title"`
	SortTitle        string           `json:"sortTitle"`
	Year             int              `json:"year"`
	TmdbID           int              `json:"tmdbId"`
	ImdbID           string           `json:"imdbId"`
	Overview         string           `json:"overview"`
	Runtime          int              `json:"runtime"`
	Path             string           `json:"path"`
	RootFolderPath   string           `json:"rootFolderPath"`
	QualityProfileID int64            `json:"qualityProfileId"`
	Monitored        bool             `json:"monitored"`
	Status           string           `json:"status"`
	InCinemas        time.Time        `json:"inCinemas"`
	PhysicalRelease  time.Time        `json:"physicalRelease"`
	DigitalRelease   time.Time        `json:"digitalRelease"`
	Studio           string           `json:"studio"`
	Certification    string           `json:"certification"`
	Added            time.Time        `json:"added"`
	HasFile          bool             `json:"hasFile"`
	PosterURL        string           `json:"posterUrl,omitempty"`
	File             *SourceMovieFile `json:"movieFile,omitempty"`
}

// SourceMovieFile represents a movie file from the source.
type SourceMovieFile struct {
	ID               int64     `json:"id"`
	Path             string    `json:"path"`
	Size             int64     `json:"size"`
	QualityID        int       `json:"qualityId"`
	QualityName      string    `json:"qualityName"`
	VideoCodec       string    `json:"videoCodec"`
	AudioCodec       string    `json:"audioCodec"`
	Resolution       string    `json:"resolution"`
	AudioChannels    string    `json:"audioChannels"`
	DynamicRange     string    `json:"dynamicRange"`
	OriginalFilePath string    `json:"originalFilePath"`
	DateAdded        time.Time `json:"dateAdded"`
}

// SourceSeries represents a TV series from the source (Sonarr).
type SourceSeries struct {
	ID               int64          `json:"id"`
	Title            string         `json:"title"`
	SortTitle        string         `json:"sortTitle"`
	Year             int            `json:"year"`
	TvdbID           int            `json:"tvdbId"`
	TmdbID           int            `json:"tmdbId"`
	ImdbID           string         `json:"imdbId"`
	Overview         string         `json:"overview"`
	Runtime          int            `json:"runtime"`
	Path             string         `json:"path"`
	RootFolderPath   string         `json:"rootFolderPath"`
	QualityProfileID int64          `json:"qualityProfileId"`
	Monitored        bool           `json:"monitored"`
	SeasonFolder     bool           `json:"seasonFolder"`
	Status           string         `json:"status"`
	Network          string         `json:"network"`
	SeriesType       string         `json:"seriesType"`
	Certification    string         `json:"certification"`
	Added            time.Time      `json:"added"`
	PosterURL        string         `json:"posterUrl,omitempty"`
	Seasons          []SourceSeason `json:"seasons"`
}

// SourceSeason represents a season from the source.
type SourceSeason struct {
	SeasonNumber int  `json:"seasonNumber"`
	Monitored    bool `json:"monitored"`
}

// SourceEpisode represents an episode from the source (Sonarr).
type SourceEpisode struct {
	ID            int64  `json:"id"`
	SeriesID      int64  `json:"seriesId"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	Overview      string `json:"overview"`
	AirDateUtc    string `json:"airDateUtc"`
	Monitored     bool   `json:"monitored"`
	EpisodeFileID int64  `json:"episodeFileId"`
	HasFile       bool   `json:"hasFile"`
}

// SourceEpisodeFile represents an episode file from the source.
type SourceEpisodeFile struct {
	ID               int64     `json:"id"`
	SeriesID         int64     `json:"seriesId"`
	SeasonNumber     int       `json:"seasonNumber"`
	RelativePath     string    `json:"relativePath"`
	Size             int64     `json:"size"`
	QualityID        int       `json:"qualityId"`
	QualityName      string    `json:"qualityName"`
	VideoCodec       string    `json:"videoCodec"`
	AudioCodec       string    `json:"audioCodec"`
	Resolution       string    `json:"resolution"`
	AudioChannels    string    `json:"audioChannels"`
	DynamicRange     string    `json:"dynamicRange"`
	OriginalFilePath string    `json:"originalFilePath"`
	DateAdded        time.Time `json:"dateAdded"`
}

// ImportMappings holds the user's mapping choices for the import.
type ImportMappings struct {
	RootFolderMapping     map[string]int64 `json:"rootFolderMapping"`
	QualityProfileMapping map[int64]int64  `json:"qualityProfileMapping"`
	SelectedMovieTmdbIDs  []int            `json:"selectedMovieTmdbIds,omitempty"`
	SelectedSeriesTvdbIDs []int            `json:"selectedSeriesTvdbIds,omitempty"`
}

// ImportPreview contains the preview results before executing an import.
type ImportPreview struct {
	Movies  []MoviePreview  `json:"movies"`
	Series  []SeriesPreview `json:"series"`
	Summary ImportSummary   `json:"summary"`
}

// MoviePreview represents a movie in the import preview.
type MoviePreview struct {
	Title            string `json:"title"`
	Year             int    `json:"year"`
	TmdbID           int    `json:"tmdbId"`
	HasFile          bool   `json:"hasFile"`
	Quality          string `json:"quality"`
	Monitored        bool   `json:"monitored"`
	QualityProfileID int64  `json:"qualityProfileId"`
	PosterURL        string `json:"posterUrl,omitempty"`
	Status           string `json:"status"` // "new", "duplicate", "skip"
	SkipReason       string `json:"skipReason,omitempty"`
}

// SeriesPreview represents a series in the import preview.
type SeriesPreview struct {
	Title            string `json:"title"`
	Year             int    `json:"year"`
	TvdbID           int    `json:"tvdbId"`
	TmdbID           int    `json:"tmdbId"`
	EpisodeCount     int    `json:"episodeCount"`
	FileCount        int    `json:"fileCount"`
	Monitored        bool   `json:"monitored"`
	QualityProfileID int64  `json:"qualityProfileId"`
	PosterURL        string `json:"posterUrl,omitempty"`
	Status           string `json:"status"` // "new", "duplicate", "skip"
	SkipReason       string `json:"skipReason,omitempty"`
}

// ImportSummary contains aggregate counts for the import preview.
type ImportSummary struct {
	TotalMovies     int `json:"totalMovies"`
	TotalSeries     int `json:"totalSeries"`
	TotalEpisodes   int `json:"totalEpisodes"`
	TotalFiles      int `json:"totalFiles"`
	NewMovies       int `json:"newMovies"`
	NewSeries       int `json:"newSeries"`
	DuplicateMovies int `json:"duplicateMovies"`
	DuplicateSeries int `json:"duplicateSeries"`
	SkippedMovies   int `json:"skippedMovies"`
	SkippedSeries   int `json:"skippedSeries"`
}

// ImportReport contains the results after executing an import.
type ImportReport struct {
	MoviesCreated int      `json:"moviesCreated"`
	MoviesSkipped int      `json:"moviesSkipped"`
	MoviesErrored int      `json:"moviesErrored"`
	SeriesCreated int      `json:"seriesCreated"`
	SeriesSkipped int      `json:"seriesSkipped"`
	SeriesErrored int      `json:"seriesErrored"`
	TotalFiles    int      `json:"totalFiles"`
	FilesImported int      `json:"filesImported"`
	Errors        []string `json:"errors"`
}
