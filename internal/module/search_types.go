package module

// SearchOptions configures a metadata search request.
type SearchOptions struct {
	Year  int // Filter by year (0 = no filter)
	Limit int // Max results (0 = provider default)
}

// SearchResult represents a single result from a metadata search.
type SearchResult struct {
	ExternalID  string
	Title       string
	Year        int
	Overview    string
	PosterURL   string
	BackdropURL string
	ExternalIDs map[string]string
	Extra       any
}

// MediaMetadata represents full metadata for an entity fetched by external ID.
type MediaMetadata struct {
	ExternalID  string
	Title       string
	Year        int
	Overview    string
	PosterURL   string
	BackdropURL string
	ExternalIDs map[string]string
	Extra       any
}

// ExtendedMetadata represents additional metadata (credits, ratings, etc.)
type ExtendedMetadata struct {
	Credits       any
	Ratings       any
	ContentRating string
	TrailerURL    string
	Extra         any
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

// SearchableItem represents an item that can be searched for by the search pipeline.
// Deviation from spec §5.3: Uses ID-based quality accessors instead of full objects
// to avoid circular dependency (WantedCollector importing quality service).
type SearchableItem interface {
	GetModuleType() string
	GetMediaType() string
	GetEntityID() int64
	GetTitle() string
	GetExternalIDs() map[string]string
	GetQualityProfileID() int64
	GetCurrentQualityID() *int64
	GetSearchParams() SearchParams
}

// SearchParams holds additional search parameters for a SearchableItem.
type SearchParams struct {
	Extra map[string]any
}

// SearchCriteria contains Newznab search parameters.
type SearchCriteria struct {
	Query      string
	Categories []int
	IDs        map[string]string
}
