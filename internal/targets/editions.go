package targets

import (
	"slices"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
)

// Editions is an Edition list, a Profile's or a Work's own: any cut, or
// named cuts in order. The zero value is any cut, the only list until
// cuts can be chosen; named lists add a constructor and keep this shape.
type Editions struct{ cuts []release.CutKey }

// AnyCut is the list that wants one file per Target, of whatever cut.
func AnyCut() Editions { return Editions{} }

// Any reports whether e wants any cut.
func (e Editions) Any() bool { return len(e.cuts) == 0 }

func (e Editions) Equal(o Editions) bool { return slices.Equal(e.cuts, o.cuts) }

// editionsOf reads a Work's own list from its kind.
func editionsOf(db.List) Editions { return AnyCut() }

// listOf is the kind e is stored as.
func listOf(Editions) db.List { return db.AnyList }
