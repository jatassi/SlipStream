package prowlarr

import (
	"context"

	"github.com/slipstream/slipstream/internal/indexer/types"
)

// SearchAdapter wraps the Prowlarr service to implement search interfaces
// used by the search router.
type SearchAdapter struct {
	service *Service
}

// NewSearchAdapter creates a new Prowlarr search adapter.
func NewSearchAdapter(service *Service) *SearchAdapter {
	return &SearchAdapter{
		service: service,
	}
}

// Search executes a search through Prowlarr and returns TorrentInfo results.
// This implements the ProwlarrSearcher interface used by the search router.
func (a *SearchAdapter) Search(ctx context.Context, criteria *types.SearchCriteria) ([]types.TorrentInfo, error) {
	return a.service.Search(ctx, criteria)
}
