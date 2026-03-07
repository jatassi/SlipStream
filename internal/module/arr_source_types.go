package module

import "time"

// ArrSourceRootFolder represents a root folder from an external *arr app.
type ArrSourceRootFolder struct {
	ID   int64
	Path string
}

// ArrSourceQualityProfile represents a quality profile from an external *arr app.
type ArrSourceQualityProfile struct {
	ID    int64
	Name  string
	InUse bool
}

// ArrSourceMovie represents a movie from an external *arr app (Radarr).
type ArrSourceMovie struct {
	ID               int64
	Title            string
	SortTitle        string
	Year             int
	TmdbID           int
	ImdbID           string
	Overview         string
	Runtime          int
	Path             string
	RootFolderPath   string
	QualityProfileID int64
	Monitored        bool
	Status           string
	InCinemas        time.Time
	PhysicalRelease  time.Time
	DigitalRelease   time.Time
	Studio           string
	Certification    string
	Added            time.Time
	HasFile          bool
	PosterURL        string
	File             *ArrSourceMovieFile
}

// ArrSourceMovieFile represents a movie file from an external *arr app.
type ArrSourceMovieFile struct {
	ID               int64
	Path             string
	Size             int64
	QualityID        int
	QualityName      string
	VideoCodec       string
	AudioCodec       string
	Resolution       string
	AudioChannels    string
	DynamicRange     string
	OriginalFilePath string
	DateAdded        time.Time
}

// ArrSourceSeries represents a TV series from an external *arr app (Sonarr).
type ArrSourceSeries struct {
	ID               int64
	Title            string
	SortTitle        string
	Year             int
	TvdbID           int
	TmdbID           int
	ImdbID           string
	Overview         string
	Runtime          int
	Path             string
	RootFolderPath   string
	QualityProfileID int64
	Monitored        bool
	SeasonFolder     bool
	Status           string
	Network          string
	SeriesType       string
	Certification    string
	Added            time.Time
	PosterURL        string
	Seasons          []ArrSourceSeason
}

// ArrSourceSeason represents a season from an external *arr app.
type ArrSourceSeason struct {
	SeasonNumber int
	Monitored    bool
}

// ArrSourceEpisode represents an episode from an external *arr app.
type ArrSourceEpisode struct {
	ID            int64
	SeriesID      int64
	SeasonNumber  int
	EpisodeNumber int
	Title         string
	Overview      string
	AirDateUtc    string
	Monitored     bool
	EpisodeFileID int64
	HasFile       bool
}

// ArrSourceEpisodeFile represents an episode file from an external *arr app.
type ArrSourceEpisodeFile struct {
	ID               int64
	SeriesID         int64
	SeasonNumber     int
	RelativePath     string
	Size             int64
	QualityID        int
	QualityName      string
	VideoCodec       string
	AudioCodec       string
	Resolution       string
	AudioChannels    string
	DynamicRange     string
	OriginalFilePath string
	DateAdded        time.Time
}
