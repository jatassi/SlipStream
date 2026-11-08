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
	// Fetch returns the whole Work as the source has it now, read as
	// options say: the Work's fetch options as its library half stored
	// them, nil for the type's default. Its error wraps works.ErrNotFound,
	// ErrRefused, ErrUnavailable, ErrInvalid, ErrNotOnSource or
	// ErrOrderUnavailable, inside a *works.SourceError when the source said
	// more.
	Fetch(ctx context.Context, ask Ask, key works.Key, options []byte) (works.Fetched, error)
	// Sources are the image hosts the artwork references in its answers
	// name.
	Sources() []artwork.Source
	// Changes reads page, from 1, of the keys the source changed from since
	// to until. A source that lists no changes from that far back, or keeps
	// no changes list, answers ErrNoChanges.
	Changes(ctx context.Context, ask Ask, since, until time.Time, page int) (ChangesPage, error)
}

// ChangesPage is one page of a source's changes list. A key may be one the
// type's library half knows a Work by through another provider; its
// Refreshing.KeysOf maps those.
type ChangesPage struct {
	Keys []works.Key
	// Pages is how many pages the list runs to.
	Pages int
}

// Refreshing is what Metadata refresh asks of a Content Type's library
// half: which of ws can still gain Items, so the growing tier refreshes
// them, and which Works changed keys name that match no Work's own key. A
// type whose Works never gain Items answers none; one whose keys come from
// one provider maps none.
type Refreshing interface {
	Growing(ctx context.Context, r storage.Reader[storage.Library], ws []works.WorkID) (map[works.WorkID]bool, error)
	// KeysOf maps changed keys another provider names a record by, such as
	// a TMDB id, to the keys of the type's Works in the library.
	KeysOf(ctx context.Context, r storage.Reader[storage.Library], keys []works.Key) ([]works.Key, error)
}
