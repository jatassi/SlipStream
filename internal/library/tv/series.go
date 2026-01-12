package tv

import "time"

// Series represents a TV series in the library.
type Series struct {
	ID               int64     `json:"id"`
	Title            string    `json:"title"`
	SortTitle        string    `json:"sortTitle"`
	Year             int       `json:"year,omitempty"`
	TvdbID           int       `json:"tvdbId,omitempty"`
	TmdbID           int       `json:"tmdbId,omitempty"`
	ImdbID           string    `json:"imdbId,omitempty"`
	Overview         string    `json:"overview,omitempty"`
	Runtime          int       `json:"runtime,omitempty"`
	Path             string    `json:"path,omitempty"`
	RootFolderID     int64     `json:"rootFolderId,omitempty"`
	QualityProfileID int64     `json:"qualityProfileId,omitempty"`
	Monitored        bool      `json:"monitored"`
	SeasonFolder     bool      `json:"seasonFolder"`
	Status           string    `json:"status"` // "continuing", "ended", "upcoming"
	Network          string    `json:"network,omitempty"`
	AddedAt          time.Time `json:"addedAt"`
	UpdatedAt        time.Time `json:"updatedAt,omitempty"`
	EpisodeCount     int       `json:"episodeCount"`
	EpisodeFileCount int       `json:"episodeFileCount"`
	SizeOnDisk       int64     `json:"sizeOnDisk,omitempty"`
	Seasons            []Season `json:"seasons,omitempty"`
	Released           bool     `json:"released"`           // True if all seasons are released
	AvailabilityStatus string   `json:"availabilityStatus"` // Badge text: "Available", "Airing", "Seasons 1-2 Available", etc.
	FormatType         string   `json:"formatType,omitempty"` // "standard", "daily", "anime" - empty means auto-detect
}

// Season represents a season of a TV series.
type Season struct {
	ID               int64  `json:"id"`
	SeriesID         int64  `json:"seriesId"`
	SeasonNumber     int    `json:"seasonNumber"`
	Monitored        bool   `json:"monitored"`
	Overview         string `json:"overview,omitempty"`
	PosterURL        string `json:"posterUrl,omitempty"`
	EpisodeCount     int    `json:"episodeCount"`
	EpisodeFileCount int    `json:"episodeFileCount"`
	SizeOnDisk       int64  `json:"sizeOnDisk,omitempty"`
	Released         bool   `json:"released"` // True if all episodes in season have aired
}

// Episode represents an episode of a TV series.
type Episode struct {
	ID            int64        `json:"id"`
	SeriesID      int64        `json:"seriesId"`
	SeasonNumber  int          `json:"seasonNumber"`
	EpisodeNumber int          `json:"episodeNumber"`
	Title         string       `json:"title"`
	Overview      string       `json:"overview,omitempty"`
	AirDate       *time.Time   `json:"airDate,omitempty"`
	Monitored     bool         `json:"monitored"`
	HasFile       bool         `json:"hasFile"`
	EpisodeFile   *EpisodeFile `json:"episodeFile,omitempty"`
	Released      bool         `json:"released"` // True if air date is in the past
}

// EpisodeFile represents an episode file on disk.
type EpisodeFile struct {
	ID         int64     `json:"id"`
	EpisodeID  int64     `json:"episodeId"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Quality    string    `json:"quality,omitempty"`
	VideoCodec string    `json:"videoCodec,omitempty"`
	AudioCodec string    `json:"audioCodec,omitempty"`
	Resolution string    `json:"resolution,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CreateSeriesInput contains fields for creating a series.
type CreateSeriesInput struct {
	Title            string        `json:"title"`
	Year             int           `json:"year,omitempty"`
	TvdbID           int           `json:"tvdbId,omitempty"`
	TmdbID           int           `json:"tmdbId,omitempty"`
	ImdbID           string        `json:"imdbId,omitempty"`
	Overview         string        `json:"overview,omitempty"`
	Runtime          int           `json:"runtime,omitempty"`
	Path             string        `json:"path,omitempty"`
	RootFolderID     int64         `json:"rootFolderId"`
	QualityProfileID int64         `json:"qualityProfileId"`
	Monitored        bool          `json:"monitored"`
	SeasonFolder     bool          `json:"seasonFolder"`
	Network          string        `json:"network,omitempty"`
	FormatType       string        `json:"formatType,omitempty"` // "standard", "daily", "anime" - empty for auto-detect
	Seasons          []SeasonInput `json:"seasons,omitempty"`
}

// SeasonInput is used when creating seasons.
type SeasonInput struct {
	SeasonNumber int            `json:"seasonNumber"`
	Monitored    bool           `json:"monitored"`
	Episodes     []EpisodeInput `json:"episodes,omitempty"`
}

// EpisodeInput is used when creating episodes.
type EpisodeInput struct {
	EpisodeNumber int        `json:"episodeNumber"`
	Title         string     `json:"title"`
	Overview      string     `json:"overview,omitempty"`
	AirDate       *time.Time `json:"airDate,omitempty"`
	Monitored     bool       `json:"monitored"`
}

// UpdateSeriesInput contains fields for updating a series.
type UpdateSeriesInput struct {
	Title            *string `json:"title,omitempty"`
	Year             *int    `json:"year,omitempty"`
	TvdbID           *int    `json:"tvdbId,omitempty"`
	TmdbID           *int    `json:"tmdbId,omitempty"`
	ImdbID           *string `json:"imdbId,omitempty"`
	Overview         *string `json:"overview,omitempty"`
	Runtime          *int    `json:"runtime,omitempty"`
	Path             *string `json:"path,omitempty"`
	RootFolderID     *int64  `json:"rootFolderId,omitempty"`
	QualityProfileID *int64  `json:"qualityProfileId,omitempty"`
	Monitored        *bool   `json:"monitored,omitempty"`
	SeasonFolder     *bool   `json:"seasonFolder,omitempty"`
	Status           *string `json:"status,omitempty"`
	FormatType       *string `json:"formatType,omitempty"` // "standard", "daily", "anime", or null for auto-detect
}

// UpdateEpisodeInput contains fields for updating an episode.
type UpdateEpisodeInput struct {
	Title     *string    `json:"title,omitempty"`
	Overview  *string    `json:"overview,omitempty"`
	AirDate   *time.Time `json:"airDate,omitempty"`
	Monitored *bool      `json:"monitored,omitempty"`
}

// ListSeriesOptions contains options for listing series.
type ListSeriesOptions struct {
	Search       string `json:"search,omitempty"`
	Monitored    *bool  `json:"monitored,omitempty"`
	RootFolderID *int64 `json:"rootFolderId,omitempty"`
	Page         int    `json:"page,omitempty"`
	PageSize     int    `json:"pageSize,omitempty"`
}

// CreateEpisodeFileInput contains fields for creating an episode file.
type CreateEpisodeFileInput struct {
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Quality    string `json:"quality,omitempty"`
	QualityID  *int64 `json:"qualityId,omitempty"`
	VideoCodec string `json:"videoCodec,omitempty"`
	AudioCodec string `json:"audioCodec,omitempty"`
	Resolution string `json:"resolution,omitempty"`
}

// MonitorType represents the type of bulk monitoring operation.
type MonitorType string

const (
	MonitorTypeAll         MonitorType = "all"
	MonitorTypeNone        MonitorType = "none"
	MonitorTypeFuture      MonitorType = "future"
	MonitorTypeFirstSeason MonitorType = "first_season"
	MonitorTypeLatest      MonitorType = "latest_season"
)

// BulkMonitorInput contains fields for bulk monitoring operations.
type BulkMonitorInput struct {
	MonitorType     MonitorType `json:"monitorType"`
	IncludeSpecials bool        `json:"includeSpecials"`
}

// BulkEpisodeMonitorInput contains fields for bulk episode monitoring.
type BulkEpisodeMonitorInput struct {
	EpisodeIDs []int64 `json:"episodeIds"`
	Monitored  bool    `json:"monitored"`
}

// MonitoringStats contains monitoring statistics for a series.
type MonitoringStats struct {
	TotalSeasons      int64 `json:"totalSeasons"`
	MonitoredSeasons  int64 `json:"monitoredSeasons"`
	TotalEpisodes     int64 `json:"totalEpisodes"`
	MonitoredEpisodes int64 `json:"monitoredEpisodes"`
}

// generateSortTitle creates a sort-friendly title by removing leading articles.
func generateSortTitle(title string) string {
	prefixes := []string{"The ", "A ", "An "}
	for _, prefix := range prefixes {
		if len(title) > len(prefix) && title[:len(prefix)] == prefix {
			return title[len(prefix):]
		}
	}
	return title
}
