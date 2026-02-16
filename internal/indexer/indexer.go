package indexer

import (
	"context"
)

// Indexer defines the interface for search indexers.
type Indexer interface {
	// Identity
	Name() string
	Definition() *IndexerDefinition

	// Configuration
	GetSettings() map[string]string

	// Operations
	Test(ctx context.Context) error
	Search(ctx context.Context, criteria *SearchCriteria) ([]ReleaseInfo, error)
	Download(ctx context.Context, url string) ([]byte, error)

	// Capabilities
	Capabilities() *Capabilities
	SupportsSearch() bool
	SupportsRSS() bool
}

// TorrentIndexer extends Indexer with torrent-specific methods.
type TorrentIndexer interface {
	Indexer
	SearchTorrents(ctx context.Context, criteria *SearchCriteria) ([]TorrentInfo, error)
}

// UsenetIndexer extends Indexer with usenet-specific methods.
type UsenetIndexer interface {
	Indexer
	SearchUsenet(ctx context.Context, criteria *SearchCriteria) ([]UsenetInfo, error)
}
