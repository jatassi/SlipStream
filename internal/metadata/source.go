package metadata

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
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
	// Sources are the image hosts the artwork references in its answers
	// name.
	Sources() []artwork.Source
	// Changes reads page, from 1, of the keys the source changed from since
	// to until. A source that lists no changes from that far back, or keeps
	// no changes list, answers ErrNoChanges.
	Changes(ctx context.Context, ask Ask, since, until time.Time, page int) (ChangesPage, error)
}

// ChangesPage is one page of a source's changes list.
type ChangesPage struct {
	Keys []works.Key
	// Pages is how many pages the list runs to.
	Pages int
}

// Refreshing is what Metadata refresh asks of a Content Type's library
// half: which of ws can still gain Items, so the growing tier refreshes
// them. A type whose Works never gain Items answers none.
type Refreshing interface {
	Growing(ctx context.Context, r storage.Reader[storage.Library], ws []works.WorkID) (map[works.WorkID]bool, error)
}
