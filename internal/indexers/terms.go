package indexers

import (
	"context"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Facet is how a Content Type's Works are matched and its Items searched.
// Categories are the type's Newznab categories, such as Movies'
// Family(2000). A type that can't be searched answers no categories and
// the zero Expect and Terms, whose Covers is nil.
type Facet interface {
	Expect(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (Expect, error)
	SearchTerms(ctx context.Context, r storage.Reader[storage.Library], item works.ItemID) (Terms, error)
	Categories() []newznab.ID
}

// Expect is what a Work expects of a name that names it.
type Expect struct {
	// Claim is what title matching compares a name to: the Work's ids, its
	// label then its other titles, its year and its type's slack.
	Claim release.Claim
	// Original is the Work's original language.
	Original release.Fact[release.Language]
	// Covers answers the Items a Name covers, none when it names none of
	// the Work's. Every searchable type sets it, and a type with one Item
	// per Work sets CoversOnly.
	Covers func(release.Name) []works.ItemID
}

// Terms are Expect and how to search one Item. Claim.IDs are the Torznab
// id parameters, by name.
type Terms struct {
	Expect
	// Mode is the Torznab t= value the type searches with.
	Mode string
	// Query is the free-text search when the Indexer takes none of the ids.
	Query string
	// Season and Episode are Torznab's season and ep in the type's current
	// numbering, nil and empty when the type searches neither. Season 0 is
	// sent.
	Season  *int
	Episode string
}

// CoversOnly is every name covering item, for a type with one Item per
// Work.
func CoversOnly(item works.ItemID) func(release.Name) []works.ItemID {
	return func(release.Name) []works.ItemID { return []works.ItemID{item} }
}
