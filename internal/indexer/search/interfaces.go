package search

import (
	"context"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// SearchService defines the interface for search operations used by handlers.
type SearchService interface {
	Search(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error)
	SearchTorrents(ctx context.Context, criteria types.SearchCriteria, params ScoredSearchParams) (*TorrentSearchResult, error)
	SearchMovies(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error)
	SearchTV(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error)
}
