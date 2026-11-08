package indexers

import (
	"context"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Facet is how a Content Type's Items are searched. Categories are the
// type's Newznab categories, such as Movies' Family(2000). A type that
// can't be searched answers no categories and the zero Terms, whose
// Covers is nil.
type Facet interface {
	SearchTerms(ctx context.Context, r storage.Reader[storage.Library], item works.ItemID) (Terms, error)
	Categories() []newznab.ID
}

type Terms struct {
	// Mode is the Torznab t= value the type searches with.
	Mode string
	// IDs are Torznab id parameters, by name.
	IDs map[string]string
	// Query is the free-text search when the Indexer takes none of IDs.
	Query string
	// Season and Episode are Torznab's season and ep in the type's current
	// numbering, nil and empty when the type searches neither. Season 0 is
	// sent.
	Season  *int
	Episode string
	// Covers judges a release title against the searched Item. Every type
	// whose Terms can be searched sets it, and a type with one Item per
	// Work sets CoversOnly.
	Covers func(title string) Covering
}

// CoverKind is what a release title covers against the searched Item.
type CoverKind uint8

const (
	// CoversItem is the searched Item alone.
	CoversItem CoverKind = iota + 1
	// CoversOther is another Item of the Work, or no Item the title names.
	CoversOther
	// CoversMore is more than one Item, such as a season pack.
	CoversMore
)

// Covering is a title's verdict and what the title names, such as S01E02,
// empty when it names nothing.
type Covering struct {
	Kind  CoverKind
	Names string
}

// CoversOnly is every title covering the searched Item, for a type with
// one Item per Work.
func CoversOnly(string) Covering { return Covering{Kind: CoversItem} }
