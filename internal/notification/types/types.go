// Package types contains shared type definitions for notification packages.
package types

import (
	"context"
	"time"
)

// NotifierType identifies a notification provider
type NotifierType string

const (
	NotifierDiscord      NotifierType = "discord"
	NotifierTelegram     NotifierType = "telegram"
	NotifierWebhook      NotifierType = "webhook"
	NotifierEmail        NotifierType = "email"
	NotifierSlack        NotifierType = "slack"
	NotifierPushover     NotifierType = "pushover"
	NotifierGotify       NotifierType = "gotify"
	NotifierNtfy         NotifierType = "ntfy"
	NotifierApprise      NotifierType = "apprise"
	NotifierPushbullet   NotifierType = "pushbullet"
	NotifierJoin         NotifierType = "join"
	NotifierProwl        NotifierType = "prowl"
	NotifierSimplepush   NotifierType = "simplepush"
	NotifierSignal       NotifierType = "signal"
	NotifierCustomScript NotifierType = "custom_script"
)

// Notifier is the interface all notification providers must implement
type Notifier interface {
	Type() NotifierType
	Name() string
	Test(ctx context.Context) error

	OnGrab(ctx context.Context, event GrabEvent) error
	OnDownload(ctx context.Context, event DownloadEvent) error
	OnUpgrade(ctx context.Context, event UpgradeEvent) error
	OnMovieAdded(ctx context.Context, event MovieAddedEvent) error
	OnMovieDeleted(ctx context.Context, event MovieDeletedEvent) error
	OnSeriesAdded(ctx context.Context, event SeriesAddedEvent) error
	OnSeriesDeleted(ctx context.Context, event SeriesDeletedEvent) error
	OnHealthIssue(ctx context.Context, event HealthEvent) error
	OnHealthRestored(ctx context.Context, event HealthEvent) error
	OnApplicationUpdate(ctx context.Context, event AppUpdateEvent) error
}

// EventType identifies the type of notification event
type EventType string

const (
	EventGrab           EventType = "grab"
	EventDownload       EventType = "download"
	EventUpgrade        EventType = "upgrade"
	EventMovieAdded     EventType = "movie_added"
	EventMovieDeleted   EventType = "movie_deleted"
	EventSeriesAdded    EventType = "series_added"
	EventSeriesDeleted  EventType = "series_deleted"
	EventHealthIssue    EventType = "health_issue"
	EventHealthRestored EventType = "health_restored"
	EventAppUpdate      EventType = "app_update"
)

// MediaInfo contains common media metadata for events
type MediaInfo struct {
	ID         int64    `json:"id"`
	Title      string   `json:"title"`
	Year       int      `json:"year,omitempty"`
	TMDbID     int64    `json:"tmdbId,omitempty"`
	IMDbID     string   `json:"imdbId,omitempty"`
	TraktID    int64    `json:"traktId,omitempty"`
	Overview   string   `json:"overview,omitempty"`
	PosterURL  string   `json:"posterUrl,omitempty"`
	FanartURL  string   `json:"fanartUrl,omitempty"`
	TrailerURL string   `json:"trailerUrl,omitempty"`
	WebsiteURL string   `json:"websiteUrl,omitempty"`
	Genres     []string `json:"genres,omitempty"`
	Tags       []int64  `json:"tags,omitempty"`
	Rating     float64  `json:"rating,omitempty"`
}

// SeriesInfo contains series-specific metadata
type SeriesInfo struct {
	MediaInfo
	TVDbID  int64 `json:"tvdbId,omitempty"`
	TraktID int64 `json:"traktId,omitempty"`
}

// EpisodeInfo contains episode-specific metadata
type EpisodeInfo struct {
	SeriesID      int64  `json:"seriesId"`
	SeriesTitle   string `json:"seriesTitle"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	EpisodeTitle  string `json:"episodeTitle,omitempty"`
	AirDate       string `json:"airDate,omitempty"`
}

// ReleaseInfo contains release/quality information
type ReleaseInfo struct {
	ReleaseName        string         `json:"releaseName"`
	Quality            string         `json:"quality"`
	QualityVersion     int            `json:"qualityVersion,omitempty"`
	Size               int64          `json:"size,omitempty"`
	Indexer            string         `json:"indexer,omitempty"`
	ReleaseGroup       string         `json:"releaseGroup,omitempty"`
	SceneName          string         `json:"sceneName,omitempty"`
	IndexerFlags       []string       `json:"indexerFlags,omitempty"`
	CustomFormats      []CustomFormat `json:"customFormats,omitempty"`
	CustomFormatScore  int            `json:"customFormatScore,omitempty"`
	Languages          []string       `json:"languages,omitempty"`
}

// CustomFormat represents a custom format match
type CustomFormat struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// DownloadClientInfo contains download client details
type DownloadClientInfo struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	DownloadID string `json:"downloadId,omitempty"`
}

// MediaFileInfo contains detailed media file information
type MediaFileInfo struct {
	VideoCodec      string   `json:"videoCodec,omitempty"`
	VideoBitrate    int64    `json:"videoBitrate,omitempty"`
	VideoResolution string   `json:"videoResolution,omitempty"`
	VideoDynamicRange string `json:"videoDynamicRange,omitempty"`
	AudioCodec      string   `json:"audioCodec,omitempty"`
	AudioBitrate    int64    `json:"audioBitrate,omitempty"`
	AudioChannels   string   `json:"audioChannels,omitempty"`
	AudioLanguages  []string `json:"audioLanguages,omitempty"`
	Subtitles       []string `json:"subtitles,omitempty"`
	Runtime         int      `json:"runtime,omitempty"`
	ScanType        string   `json:"scanType,omitempty"`
}

// GrabEvent is triggered when a release is grabbed from an indexer
type GrabEvent struct {
	Movie          *MediaInfo         `json:"movie,omitempty"`
	Episode        *EpisodeInfo       `json:"episode,omitempty"`
	Release        ReleaseInfo        `json:"release"`
	DownloadClient DownloadClientInfo `json:"downloadClient"`
	DownloadID     string             `json:"downloadId,omitempty"`
	GrabbedAt      time.Time          `json:"grabbedAt"`
}

// DownloadEvent is triggered when a file is imported
type DownloadEvent struct {
	Movie             *MediaInfo      `json:"movie,omitempty"`
	Episode           *EpisodeInfo    `json:"episode,omitempty"`
	Quality           string          `json:"quality"`
	SourcePath        string          `json:"sourcePath"`
	DestinationPath   string          `json:"destinationPath"`
	ReleaseName       string          `json:"releaseName,omitempty"`
	ReleaseGroup      string          `json:"releaseGroup,omitempty"`
	SceneName         string          `json:"sceneName,omitempty"`
	DownloadID        string          `json:"downloadId,omitempty"`
	DownloadClient    string          `json:"downloadClient,omitempty"`
	CustomFormats     []CustomFormat  `json:"customFormats,omitempty"`
	CustomFormatScore int             `json:"customFormatScore,omitempty"`
	Languages         []string        `json:"languages,omitempty"`
	MediaInfo         *MediaFileInfo  `json:"mediaInfo,omitempty"`
	ImportedAt        time.Time       `json:"importedAt"`
}

// UpgradeEvent is triggered when a file is upgraded
type UpgradeEvent struct {
	Movie             *MediaInfo      `json:"movie,omitempty"`
	Episode           *EpisodeInfo    `json:"episode,omitempty"`
	OldQuality        string          `json:"oldQuality"`
	NewQuality        string          `json:"newQuality"`
	OldPath           string          `json:"oldPath"`
	NewPath           string          `json:"newPath"`
	ReleaseName       string          `json:"releaseName,omitempty"`
	ReleaseGroup      string          `json:"releaseGroup,omitempty"`
	CustomFormats     []CustomFormat  `json:"customFormats,omitempty"`
	CustomFormatScore int             `json:"customFormatScore,omitempty"`
	Languages         []string        `json:"languages,omitempty"`
	MediaInfo         *MediaFileInfo  `json:"mediaInfo,omitempty"`
	UpgradedAt        time.Time       `json:"upgradedAt"`
}

// MovieAddedEvent is triggered when a movie is added to the library
type MovieAddedEvent struct {
	Movie   MediaInfo `json:"movie"`
	AddedAt time.Time `json:"addedAt"`
}

// MovieDeletedEvent is triggered when a movie is removed from the library
type MovieDeletedEvent struct {
	Movie        MediaInfo `json:"movie"`
	DeletedFiles bool      `json:"deletedFiles"`
	DeletedAt    time.Time `json:"deletedAt"`
}

// SeriesAddedEvent is triggered when a series is added to the library
type SeriesAddedEvent struct {
	Series  SeriesInfo `json:"series"`
	AddedAt time.Time  `json:"addedAt"`
}

// SeriesDeletedEvent is triggered when a series is removed from the library
type SeriesDeletedEvent struct {
	Series       SeriesInfo `json:"series"`
	DeletedFiles bool       `json:"deletedFiles"`
	DeletedAt    time.Time  `json:"deletedAt"`
}

// HealthEvent is triggered when a health check issue occurs or is resolved
type HealthEvent struct {
	Source    string    `json:"source"`
	Type      string    `json:"type"` // "error" or "warning"
	Message   string    `json:"message"`
	WikiURL   string    `json:"wikiUrl,omitempty"`
	OccuredAt time.Time `json:"occuredAt"`
}

// AppUpdateEvent is triggered when the application is updated
type AppUpdateEvent struct {
	PreviousVersion string    `json:"previousVersion"`
	NewVersion      string    `json:"newVersion"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
