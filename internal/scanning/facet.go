// Package scanning decides what a library scan does, pure: no IO, no
// clock, no writes. It compares a root's walk with its records, groups
// what it found by Work folder, matches each group to a Work, and reads a
// found file's quality. internal/imports runs it and writes the result.
package scanning

import (
	"context"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Facet is what a library scan asks a Content Type. It joins
// contenttype.Library, so every type answers it. None is the documented
// empty answer: a type that embeds it beside its Names matches only by
// recorded folder and places no file.
type Facet interface {
	// Names reads a found file's name, for its quality and its cut.
	release.Names
	// Hint reads a Work folder's name, or a loose file's without its
	// extension.
	Hint(name string) Hint
	// Claims answers, for each of ws, the year and provider ids it is
	// known by, read from the type's own rows: the core holds no year and
	// reads no provider id.
	Claims(ctx context.Context, r storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[works.WorkID]Claim, error)
	// YearOf is a search candidate's year, zero when it lists none.
	YearOf(c works.Candidate) int
	// Place reads work's Items and numbers once and answers a pure Placer
	// for the files of its folder. Aired numbers for an Absolute-order
	// series live in the type's rows, not in works.Tree, so Place takes a
	// reader.
	Place(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (Placer, error)
}

// Placer places one file by its path inside its Work folder, such as
// "Season 2/Show - S02E05.mkv".
type Placer func(rel string) Placement

// Hint is what a name says.
type Hint struct {
	Title string // as written, before folding
	Year  int    // zero when the name has none
	IDs   []ID
	// Key is the Work key at the type's own source that an id names, so
	// the scan can fetch it without a search. Empty when no id names one.
	Key works.Key
}

// ID is a provider-qualified id.
type ID struct {
	Provider Provider
	Value    string // digits, or "tt" and digits
}

type Provider string

const (
	TMDB Provider = "tmdb"
	IMDb Provider = "imdb"
	TVDB Provider = "tvdb"
)

// Of is the value of h's first id at p, empty when it carries none.
func (h Hint) Of(p Provider) string {
	for _, id := range h.IDs {
		if id.Provider == p {
			return id.Value
		}
	}
	return ""
}

// Claim is what a library Work is known by beyond its titles.
type Claim struct {
	Year int
	IDs  []ID
}

// Placement is the Items a file holds, or why it has none. Only Placed,
// PlacedPart and Unplaced make one, so it never holds both.
type Placement struct {
	items  []works.ItemID
	part   int
	reason Reason
}

// Placed is a file holding items, at least one, in tree order.
func Placed(items ...works.ItemID) Placement { return Placement{items: items} }

// PlacedPart is a file holding part n of items, such as CD2: a type that
// keeps its Items in one file answers it for a name that reads as one of
// several.
func PlacedPart(n int, items ...works.ItemID) Placement { return Placement{items: items, part: n} }

func Unplaced(r Reason) Placement { return Placement{reason: r} }

func (p Placement) Items() ([]works.ItemID, bool) { return p.items, len(p.items) > 0 }

func (p Placement) Reason() Reason { return p.reason }

// Part is the part of its Items p reads as, zero for the whole.
func (p Placement) Part() int { return p.part }

// None is the empty facet: no hint, no claims, no year, and every file
// unplaced with NoRule.
type None struct{}

func (None) Hint(string) Hint { return Hint{} }

func (None) Claims(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.WorkID]Claim, error) {
	return nil, nil
}

func (None) YearOf(works.Candidate) int { return 0 }

func (None) Place(context.Context, storage.Reader[storage.Library], works.WorkID) (Placer, error) {
	return func(string) Placement { return Unplaced(Reason{Code: NoRule}) }, nil
}
