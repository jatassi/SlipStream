package module

import "time"

// SearchOptions contains query parameters for metadata search.
type SearchOptions struct {
	Year int
	IDs  map[string]string
}

// SearchResult is a single search result from a metadata provider.
type SearchResult struct {
	Title      string
	Year       int
	ExternalID string
	Overview   string
	Images     []string
}

// MediaMetadata is the full metadata for an entity.
type MediaMetadata struct {
	Title       string
	ExternalID  string
	Year        int
	Overview    string
	Runtime     int
	Genres      []string
	Status      string
	ReleaseDate *time.Time
}

// ExtendedMetadata contains additional metadata (credits, recommendations, etc.).
type ExtendedMetadata struct {
	Credits         []Credit
	Recommendations []SearchResult
}

// Credit represents a person credit (cast or crew).
type Credit struct {
	Name      string
	Role      string
	Character string
}

// RefreshResult is the diff result from a metadata refresh.
type RefreshResult struct {
	Added   int
	Updated int
	Removed int
}

// Release represents a release found by an indexer.
type Release struct {
	Title    string
	Size     int64
	Indexer  string
	InfoURL  string
	Age      int
	Category int
}

// SearchableItem is an interface for items that can be searched for.
type SearchableItem interface {
	GetEntityType() EntityType
	GetEntityID() int64
	GetTitle() string
}

// SearchCriteria contains Newznab search parameters.
type SearchCriteria struct {
	Query      string
	Categories []int
	IDs        map[string]string
}
