package metadata

import (
	"context"

	"github.com/jatassi/slipstream/internal/works"
)

// Source is a Content Type's provider half. It runs on any Metadata holder
// with only what ProviderDeps gives it: no library and no key at rest.
type Source interface {
	// Search returns at most 25 candidates, best first.
	Search(ctx context.Context, ask Ask, query string) ([]works.Candidate, error)
	// Fetch returns the whole Work as the source has it now. Its error wraps
	// works.ErrNotFound, ErrRefused, ErrUnavailable or ErrInvalid, inside a
	// *works.SourceError when the source said more.
	Fetch(ctx context.Context, ask Ask, key works.Key) (works.Fetched, error)
}
