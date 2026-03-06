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
