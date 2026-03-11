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

// ReleaseForFilter provides a read-only view of a release for module filtering.
type ReleaseForFilter struct {
	Title            string
	Year             int
	Season           int
	EndSeason        int
	Episode          int
	EndEpisode       int
	IsTV             bool
	IsSeasonPack     bool
	IsCompleteSeries bool
	Quality          string
	Source           string
	Languages        []string
	Size             int64
	Categories       []int
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

// SearchParams holds module-specific search parameters on a SearchableItem.
type SearchParams struct {
	Extra map[string]any // Module-specific fields (e.g., seriesId, seasonNumber for TV)
}

// SearchCriteria defines parameters for an indexer search, constructed by a module.
type SearchCriteria struct {
	ModuleType Type   // Which module produced this criteria
	Query      string // Title to search for
	SearchType string // Newznab search type: "movie", "tvsearch", "audio", "book"
	Categories []int  // Newznab category IDs

	// External IDs (module sets whichever are relevant)
	ExternalIDs map[string]string // "imdbId" -> "tt1234567", "tmdbId" -> "550", etc.

	// Structured search parameters (TV season/episode, year, etc.)
	Year    int
	Season  int
	Episode int
}
