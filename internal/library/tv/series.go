package tv

import "time"

// StatusCounts holds episode status counts for a series or season.
type StatusCounts struct {
	Unreleased  int `json:"unreleased"`
	Missing     int `json:"missing"`
	Downloading int `json:"downloading"`
	Failed      int `json:"failed"`
	Upgradable  int `json:"upgradable"`
	Available   int `json:"available"`
	Total       int `json:"total"`
}

// Series represents a TV series in the library.
type Series struct {
	ID               int64        `json:"id"`
	Title            string       `json:"title"`
	SortTitle        string       `json:"sortTitle"`
	Year             int          `json:"year,omitempty"`
	TvdbID           int          `json:"tvdbId,omitempty"`
	TmdbID           int          `json:"tmdbId,omitempty"`
	ImdbID           string       `json:"imdbId,omitempty"`
	Overview         string       `json:"overview,omitempty"`
	Runtime          int          `json:"runtime,omitempty"`
	Path             string       `json:"path,omitempty"`
	RootFolderID     int64        `json:"rootFolderId,omitempty"`
	QualityProfileID int64        `json:"qualityProfileId,omitempty"`
	Monitored        bool         `json:"monitored"`
	SeasonFolder     bool         `json:"seasonFolder"`
	ProductionStatus string       `json:"productionStatus"`
	Network          string       `json:"network,omitempty"`
	NetworkLogoURL   string       `json:"networkLogoUrl,omitempty"`
	AddedAt          time.Time    `json:"addedAt"`
	UpdatedAt        time.Time    `json:"updatedAt,omitempty"`
	SizeOnDisk       int64        `json:"sizeOnDisk,omitempty"`
	Seasons          []Season     `json:"seasons,omitempty"`
	StatusCounts     StatusCounts `json:"statusCounts"`
	FormatType       string       `json:"formatType,omitempty"`
	FirstAired       *time.Time   `json:"firstAired,omitempty"`
	LastAired        *time.Time   `json:"lastAired,omitempty"`
	NextAiring       *time.Time   `json:"nextAiring,omitempty"`

	AddedBy         *int64 `json:"addedBy,omitempty"`
	AddedByUsername string `json:"addedByUsername,omitempty"`
}

// Season represents a season of a TV series.
type Season struct {
	ID           int64        `json:"id"`
	SeriesID     int64        `json:"seriesId"`
	SeasonNumber int          `json:"seasonNumber"`
	Monitored    bool         `json:"monitored"`
	Overview     string       `json:"overview,omitempty"`
	PosterURL    string       `json:"posterUrl,omitempty"`
	SizeOnDisk   int64        `json:"sizeOnDisk,omitempty"`
	StatusCounts StatusCounts `json:"statusCounts"`
}

// Episode represents an episode of a TV series.
type Episode struct {
	ID               int64        `json:"id"`
	SeriesID         int64        `json:"seriesId"`
	SeasonNumber     int          `json:"seasonNumber"`
	EpisodeNumber    int          `json:"episodeNumber"`
	Title            string       `json:"title"`
	Overview         string       `json:"overview,omitempty"`
	AirDate          *time.Time   `json:"airDate,omitempty"`
	Monitored        bool         `json:"monitored"`
	Status           string       `json:"status"`
	StatusMessage    *string      `json:"statusMessage"`
	ActiveDownloadID *string      `json:"activeDownloadId"`
	EpisodeFile      *EpisodeFile `json:"episodeFile,omitempty"`
}

// EpisodeFile represents an episode file on disk.
type EpisodeFile struct {
	ID            int64     `json:"id"`
	EpisodeID     int64     `json:"episodeId"`
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	Quality       string    `json:"quality,omitempty"`
	VideoCodec    string    `json:"videoCodec,omitempty"`
	AudioCodec    string    `json:"audioCodec,omitempty"`
	AudioChannels string    `json:"audioChannels,omitempty"`
	DynamicRange  string    `json:"dynamicRange,omitempty"`
	Resolution    string    `json:"resolution,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	SlotID        *int64    `json:"slotId,omitempty"`
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
	NetworkLogoURL   string        `json:"networkLogoUrl,omitempty"`
	FormatType       string        `json:"formatType,omitempty"`
	ProductionStatus string        `json:"productionStatus,omitempty"`
	Seasons          []SeasonInput `json:"seasons,omitempty"`

	AddedBy *int64 `json:"-"`
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
	ProductionStatus *string `json:"productionStatus,omitempty"`
	FormatType       *string `json:"formatType,omitempty"`
	Network          *string `json:"network,omitempty"`
	NetworkLogoURL   *string `json:"networkLogoUrl,omitempty"`
}

// UpdateEpisodeInput contains fields for updating an episode.
type UpdateEpisodeInput struct {
	Title     *string    `json:"title,omitempty"`
	Overview  *string    `json:"overview,omitempty"`
	AirDate   *time.Time `json:"airDate,omitempty"`
	Monitored *bool      `json:"monitored,omitempty"`
}

// BulkSeriesMonitorInput contains fields for bulk monitor/unmonitor of multiple series.
type BulkSeriesMonitorInput struct {
	IDs       []int64 `json:"ids"`
	Monitored bool    `json:"monitored"`
}

// ListSeriesOptions contains options for listing series.
type ListSeriesOptions struct {
	Search       string `json:"search,omitempty"`
	Monitored    *bool  `json:"monitored,omitempty"`
	RootFolderID *int64 `json:"rootFolderId,omitempty"`
}

// CreateEpisodeFileInput contains fields for creating an episode file.
type CreateEpisodeFileInput struct {
	Path             string `json:"path"`
	Size             int64  `json:"size"`
	Quality          string `json:"quality,omitempty"`
	QualityID        *int64 `json:"qualityId,omitempty"`
	VideoCodec       string `json:"videoCodec,omitempty"`
	AudioCodec       string `json:"audioCodec,omitempty"`
	AudioChannels    string `json:"audioChannels,omitempty"`
	DynamicRange     string `json:"dynamicRange,omitempty"`
	Resolution       string `json:"resolution,omitempty"`
	OriginalPath     string `json:"originalPath,omitempty"`     // Source path before import
	OriginalFilename string `json:"originalFilename,omitempty"` // Original filename before rename
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
