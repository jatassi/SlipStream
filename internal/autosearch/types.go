// Package autosearch provides automatic release searching and grabbing functionality.
package autosearch

import (
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

// Type aliases for zero-breakage migration to decisioning package.
type MediaType = decisioning.MediaType
type SearchableItem = decisioning.SearchableItem

const (
	MediaTypeMovie   = decisioning.MediaTypeMovie
	MediaTypeEpisode = decisioning.MediaTypeEpisode
	MediaTypeSeason  = decisioning.MediaTypeSeason
	MediaTypeSeries  = decisioning.MediaTypeSeries
)

// SearchSource indicates what triggered the search.
type SearchSource string

const (
	SearchSourceManual    SearchSource = "manual"    // User clicked button
	SearchSourceScheduled SearchSource = "scheduled" // Background task
	SearchSourceAdd       SearchSource = "add"       // Adding to library
	SearchSourceRequest   SearchSource = "request"   // External request approved
)

// SearchRequest contains parameters for an automatic search operation.
type SearchRequest struct {
	MediaType    MediaType    `json:"mediaType"`              // movie, episode, season, series
	MediaID      int64        `json:"mediaId"`                // movie_id, episode_id, or series_id
	SeasonNumber *int         `json:"seasonNumber,omitempty"` // For season searches
	ClientID     *int64       `json:"clientId,omitempty"`     // Optional preferred download client
	Source       SearchSource `json:"source"`                 // What triggered the search
}

// SearchResult contains the outcome of a search operation.
type SearchResult struct {
	Found      bool               `json:"found"`                // Whether a suitable release was found
	Downloaded bool               `json:"downloaded"`           // Whether the release was sent to download client
	Release    *types.TorrentInfo `json:"release,omitempty"`    // The selected release, if found
	Error      string             `json:"error,omitempty"`      // Error message if failed
	Upgraded   bool               `json:"upgraded"`             // Was this a quality upgrade?
	ClientName string             `json:"clientName,omitempty"` // Name of download client used
	DownloadID string             `json:"downloadId,omitempty"` // Download client's ID for the download
}

// BatchSearchResult contains results from searching multiple items.
type BatchSearchResult struct {
	TotalSearched int             `json:"totalSearched"`
	Found         int             `json:"found"`
	Downloaded    int             `json:"downloaded"`
	Failed        int             `json:"failed"`
	Results       []*SearchResult `json:"results,omitempty"`
}

// RetryResult contains the outcome of a manual retry operation.
type RetryResult struct {
	NewStatus string `json:"newStatus"` // The status the item was reset to (missing or upgradable)
	Message   string `json:"message"`
}

// SearchStatus represents the current status of a search operation.
type SearchStatus struct {
	MediaType  MediaType `json:"mediaType"`
	MediaID    int64     `json:"mediaId"`
	Searching  bool      `json:"searching"`  // Currently being searched
	InQueue    bool      `json:"inQueue"`    // Already has pending download
	LastSearch string    `json:"lastSearch"` // ISO timestamp of last search
}
