// Package types contains shared type definitions for indexer packages.
package types

import (
	"encoding/json"
	"time"
)

// Protocol represents the download protocol.
type Protocol string

const (
	ProtocolTorrent Protocol = "torrent"
	ProtocolUsenet  Protocol = "usenet"
)

// Privacy represents indexer privacy level.
type Privacy string

const (
	PrivacyPublic      Privacy = "public"
	PrivacySemiPrivate Privacy = "semi-private"
	PrivacyPrivate     Privacy = "private"
)

// IndexerType represents the type of indexer API.
type IndexerType string

const (
	IndexerTypeTorznab IndexerType = "torznab"
	IndexerTypeNewznab IndexerType = "newznab"
)

// IndexerDefinition represents a configured indexer.
type IndexerDefinition struct {
	ID             int64           `json:"id"`
	Name           string          `json:"name"`
	DefinitionID   string          `json:"definitionId"`           // Cardigann definition ID
	Categories     []int           `json:"categories"`
	Protocol       Protocol        `json:"protocol"`
	Privacy        Privacy         `json:"privacy"`
	SupportsMovies bool            `json:"supportsMovies"`
	SupportsTV     bool            `json:"supportsTV"`
	SupportsSearch bool            `json:"supportsSearch"`
	SupportsRSS    bool            `json:"supportsRss"`
	Priority          int             `json:"priority"`
	Enabled           bool            `json:"enabled"`
	AutoSearchEnabled bool            `json:"autoSearchEnabled"`
	Settings          json.RawMessage `json:"settings,omitempty"`
	CreatedAt         time.Time       `json:"createdAt,omitempty"`
	UpdatedAt         time.Time       `json:"updatedAt,omitempty"`
}

// SearchCriteria defines search parameters.
type SearchCriteria struct {
	Query      string `json:"query,omitempty"`
	Type       string `json:"type"` // search, tvsearch, movie
	Categories []int  `json:"categories,omitempty"`

	// Movie-specific
	ImdbID string `json:"imdbId,omitempty"`
	TmdbID int    `json:"tmdbId,omitempty"`
	Year   int    `json:"year,omitempty"`

	// TV-specific
	TvdbID  int `json:"tvdbId,omitempty"`
	Season  int `json:"season,omitempty"`
	Episode int `json:"episode,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ReleaseInfo represents a search result from an indexer.
type ReleaseInfo struct {
	GUID        string    `json:"guid"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	DownloadURL string    `json:"downloadUrl"`
	InfoURL     string    `json:"infoUrl,omitempty"`
	Size        int64     `json:"size"`
	PublishDate time.Time `json:"publishDate"`
	Categories  []int     `json:"categories"`

	// Indexer info
	IndexerID   int64    `json:"indexerId"`
	IndexerName string   `json:"indexer"`
	Protocol    Protocol `json:"protocol"`

	// External IDs
	ImdbID int `json:"imdbId,omitempty"`
	TmdbID int `json:"tmdbId,omitempty"`
	TvdbID int `json:"tvdbId,omitempty"`

	// Parsed quality info (from title)
	Quality    string `json:"quality,omitempty"`    // "720p", "1080p", "2160p"
	Source     string `json:"source,omitempty"`     // "BluRay", "WEB-DL", "HDTV"
	Resolution int    `json:"resolution,omitempty"` // 720, 1080, 2160
}

// ScoreBreakdown provides detailed scoring information for a release.
type ScoreBreakdown struct {
	QualityScore float64 `json:"qualityScore"`
	QualityID    int     `json:"qualityId,omitempty"`
	QualityName  string  `json:"qualityName,omitempty"`
	HealthScore  float64 `json:"healthScore"`
	IndexerScore float64 `json:"indexerScore"`
	MatchScore   float64 `json:"matchScore"`
	AgeScore     float64 `json:"ageScore"`
}

// TorrentInfo extends ReleaseInfo with torrent-specific fields.
type TorrentInfo struct {
	ReleaseInfo
	Seeders              int     `json:"seeders"`
	Leechers             int     `json:"leechers"`
	InfoHash             string  `json:"infoHash,omitempty"`
	MagnetURL            string  `json:"magnetUrl,omitempty"`
	MinimumRatio         float64 `json:"minimumRatio,omitempty"`
	MinimumSeedTime      int64   `json:"minimumSeedTime,omitempty"` // seconds
	DownloadVolumeFactor float64 `json:"downloadVolumeFactor"`      // 0 = freeleech
	UploadVolumeFactor   float64 `json:"uploadVolumeFactor"`        // 2 = double upload

	// Scoring fields (populated by scored search endpoints)
	Score           float64         `json:"score,omitempty"`
	NormalizedScore int             `json:"normalizedScore,omitempty"` // 0-100 for UI display
	ScoreBreakdown  *ScoreBreakdown `json:"scoreBreakdown,omitempty"`
}

// UsenetInfo extends ReleaseInfo with usenet-specific fields.
type UsenetInfo struct {
	ReleaseInfo
	Grabs     int    `json:"grabs,omitempty"`
	UsenetAge int    `json:"usenetAge,omitempty"` // days
	Poster    string `json:"poster,omitempty"`
	Group     string `json:"group,omitempty"`
}

// Capabilities describes what an indexer supports.
type Capabilities struct {
	SupportsMovies      bool              `json:"supportsMovies"`
	SupportsTV          bool              `json:"supportsTV"`
	SupportsSearch      bool              `json:"supportsSearch"`
	SupportsRSS         bool              `json:"supportsRss"`
	SearchParams        []string          `json:"searchParams"`
	TvSearchParams      []string          `json:"tvSearchParams"`
	MovieSearchParams   []string          `json:"movieSearchParams"`
	Categories          []CategoryMapping `json:"categories"`
	MaxResultsPerSearch int               `json:"maxResultsPerSearch"`
}

// CategoryMapping maps indexer categories to standard Newznab categories.
type CategoryMapping struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// IndexerStatus represents the health status of an indexer.
type IndexerStatus struct {
	IndexerID         int64      `json:"indexerId"`
	InitialFailure    *time.Time `json:"initialFailure,omitempty"`
	MostRecentFailure *time.Time `json:"mostRecentFailure,omitempty"`
	EscalationLevel   int        `json:"escalationLevel"`
	DisabledTill      *time.Time `json:"disabledTill,omitempty"`
	LastRssSync       *time.Time `json:"lastRssSync,omitempty"`
}
