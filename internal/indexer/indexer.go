package indexer

// Indexer defines the interface for search indexers.
type Indexer interface {
	// Name returns the indexer name.
	Name() string

	// Test verifies the indexer connection.
	Test() error

	// Search searches for releases.
	Search(query SearchQuery) ([]Release, error)

	// Capabilities returns the indexer capabilities.
	Capabilities() Capabilities
}

// SearchQuery defines search parameters.
type SearchQuery struct {
	Query      string   `json:"query,omitempty"`
	TmdbID     int      `json:"tmdbId,omitempty"`
	TvdbID     int      `json:"tvdbId,omitempty"`
	ImdbID     string   `json:"imdbId,omitempty"`
	Season     int      `json:"season,omitempty"`
	Episode    int      `json:"episode,omitempty"`
	Categories []int    `json:"categories,omitempty"`
}

// Release represents a search result from an indexer.
type Release struct {
	Title       string  `json:"title"`
	DownloadURL string  `json:"downloadUrl"`
	InfoURL     string  `json:"infoUrl,omitempty"`
	Size        int64   `json:"size"`
	Seeders     int     `json:"seeders,omitempty"`
	Leechers    int     `json:"leechers,omitempty"`
	Indexer     string  `json:"indexer"`
	Protocol    string  `json:"protocol"` // torrent, usenet
	PublishDate string  `json:"publishDate,omitempty"`
}

// Capabilities describes what an indexer supports.
type Capabilities struct {
	SupportsMovies bool `json:"supportsMovies"`
	SupportsTV     bool `json:"supportsTV"`
	SupportsSearch bool `json:"supportsSearch"`
}
