package targets

import (
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
)

// Editions is an Edition list, a Profile's or a Work's own: any cut, or
// named cuts in order. The zero value is any cut. NamedCuts is the only
// way to name cuts, so a list can't name none, name one twice, or mix
// any cut with named ones.
type Editions struct{ cuts []release.CutKey }

var (
	ErrNoCuts   = errors.New("an Edition list of named cuts names at least one")
	ErrCutTwice = errors.New("an Edition list names a cut twice")
)

// AnyCut is the list that wants one file per Target, of whatever cut.
func AnyCut() Editions { return Editions{} }

// NamedCuts is the list of cuts, in order. Whether each is a cut
// SlipStream knows is the save's to check.
func NamedCuts(cuts ...release.CutKey) (Editions, error) {
	if len(cuts) == 0 {
		return Editions{}, ErrNoCuts
	}
	for i, c := range cuts {
		if slices.Contains(cuts[:i], c) {
			return Editions{}, ErrCutTwice
		}
	}
	return Editions{cuts: slices.Clone(cuts)}, nil
}

// Any reports whether e wants any cut.
func (e Editions) Any() bool { return len(e.cuts) == 0 }

// Cuts is e's cuts in list order, none under any cut.
func (e Editions) Cuts() []release.CutKey { return slices.Clone(e.cuts) }

func (e Editions) Equal(o Editions) bool { return slices.Equal(e.cuts, o.cuts) }

// Filling is the Edition a file of cut c fills under e: any under any
// cut, else its own cut, whose cells exist only where e wants it.
func (e Editions) Filling(c release.CutKey) Edition {
	if e.Any() {
		return AnyEdition
	}
	return EditionOf(c)
}

// Edition is a wanted Edition: a cut, or AnyEdition under any cut. It
// keys cells, fills and Grabs; a file's own cut stays a release.CutKey.
type Edition string

const AnyEdition Edition = "any"

func EditionOf(c release.CutKey) Edition { return Edition(c) }

// Takes reports whether a file or release of cut c can fill e.
func (e Edition) Takes(c release.CutKey) bool { return e == AnyEdition || e == EditionOf(c) }

// Wanted applies the Edition rule to one Item whose seen cuts are seen:
// any cut wants AnyEdition; a list naming Standard wants Standard and
// each other cut it names that is seen; a list without Standard wants
// every cut it names. Editions come in list order.
func Wanted(list Editions, seen []release.CutKey) []Edition {
	if list.Any() {
		return []Edition{AnyEdition}
	}
	outright := !slices.Contains(list.cuts, release.Standard)
	var out []Edition
	for _, c := range list.cuts {
		if outright || c == release.Standard || slices.Contains(seen, c) {
			out = append(out, EditionOf(c))
		}
	}
	return out
}

// editionsOf reads a stored list of kind l naming cuts.
func editionsOf(l db.List, cuts []release.CutKey) Editions {
	if l == db.AnyList {
		return AnyCut()
	}
	return Editions{cuts: cuts}
}

// listOf is how e is stored: its kind, and its cuts' rows.
func listOf(e Editions) db.List {
	if e.Any() {
		return db.AnyList
	}
	return db.CutsList
}
