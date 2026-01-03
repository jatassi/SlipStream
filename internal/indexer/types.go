package indexer

import (
	"github.com/slipstream/slipstream/internal/indexer/types"
)

// Re-export types from the types package for convenience.
// This allows external packages to use indexer.IndexerDefinition instead of types.IndexerDefinition.

type (
	Protocol          = types.Protocol
	Privacy           = types.Privacy
	IndexerDefinition = types.IndexerDefinition
	SearchCriteria    = types.SearchCriteria
	ReleaseInfo       = types.ReleaseInfo
	TorrentInfo       = types.TorrentInfo
	UsenetInfo        = types.UsenetInfo
	Capabilities      = types.Capabilities
	CategoryMapping   = types.CategoryMapping
	IndexerStatus     = types.IndexerStatus
)

// Re-export constants.
const (
	ProtocolTorrent    = types.ProtocolTorrent
	ProtocolUsenet     = types.ProtocolUsenet
	PrivacyPublic      = types.PrivacyPublic
	PrivacySemiPrivate = types.PrivacySemiPrivate
	PrivacyPrivate     = types.PrivacyPrivate
)

// IndexerHistoryEvent represents a logged event for an indexer.
type IndexerHistoryEvent struct {
	ID           int64  `json:"id"`
	IndexerID    int64  `json:"indexerId"`
	EventType    string `json:"eventType"` // query, rss, grab, auth
	Successful   bool   `json:"successful"`
	Query        string `json:"query,omitempty"`
	Categories   []int  `json:"categories,omitempty"`
	ResultsCount int    `json:"resultsCount,omitempty"`
	ElapsedMs    int    `json:"elapsedMs,omitempty"`
	Data         string `json:"data,omitempty"`
}

// SearchResult contains aggregated search results.
type SearchResult struct {
	Releases      []ReleaseInfo        `json:"releases"`
	TotalResults  int                  `json:"totalResults"`
	IndexersUsed  int                  `json:"indexersUsed"`
	IndexerErrors []SearchIndexerError `json:"indexerErrors,omitempty"`
}

// SearchIndexerError represents an error from an indexer during search (for JSON serialization).
type SearchIndexerError struct {
	IndexerID   int64  `json:"indexerId"`
	IndexerName string `json:"indexerName"`
	Error       string `json:"error"`
}
