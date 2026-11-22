// Package entry is what an owner of a fact needs to record a History
// entry in the Write that makes it true: a typed catalog row, the Recorder
// its Config holds, and the provider-neutral Message its row renders.
package entry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Kind is a catalog row's id. It is stored on entries and in each
// connection's event set, so it never changes once shipped.
type Kind string

// Group is a kind's History filter group and where its toggle sits in a
// connection's event list. History has exactly seven.
type Group string

const (
	Grabs     Group = "grabs"
	Imports   Group = "imports"
	Library   Group = "library"
	Subtitles Group = "subtitles"
	Health    Group = "health"
	Backups   Group = "backups"
	Moves     Group = "moves"
)

var groupLabels = map[Group]string{Grabs: "Grabs", Imports: "Imports", Library: "Library", Subtitles: "Subtitles", Health: "Health", Backups: "Backups", Moves: "Moves"}

// Groups is History's groups in filter order.
func Groups() []Group { return []Group{Grabs, Imports, Library, Subtitles, Health, Backups, Moves} }

// Label is g's name on History's filter.
func (g Group) Label() string { return groupLabels[g] }

// Spec is a row's declaration, the same whatever its facts type.
type Spec struct {
	Kind Kind
	// Label is the event's name on its toggle: "Grab failed".
	Label string
	Group Group
	// OnByDefault: a new connection has it on (the recommended events).
	OnByDefault bool
	// BellByDefault: the bell has it on until the Admin turns it off.
	BellByDefault bool
	// Fixed: no toggle, and never recorded (Test).
	Fixed bool
	// Tally is the noun this kind adds to History's 7-day sentence: "14
	// Grabs". Zero leaves it out. Kinds with one noun add up.
	Tally works.Noun
}

// About is what History keeps beside an entry's facts, so it filters and
// links without decoding them. It is read once, at capture.
type About struct {
	// Type is zero for Health.
	Type works.TypeID
	// Works is every Work the entry is about.
	Works  []works.WorkID
	Grab   storage.UUID
	Import storage.UUID
	// Quiet: History lists it, and connections and the bell never get it.
	Quiet bool
}

// Facts is a kind's stored facts. A kind without About doesn't compile.
type Facts interface{ About() About }

// Row is one catalog row whose facts are F. Record and Render take the same
// F, so what is stored is what is rendered. F is stored as JSON for as long
// as an entry is kept, so its fields are only ever added.
type Row[F Facts] struct {
	Spec
	// Render is pure: the words at send time from the facts recorded then.
	Render func(f F, c Context) Message
}

// Entry is a row with its facts type erased, for the catalog.
type Entry interface {
	Declared() Spec
	// RenderStored decodes a stored subject as the row's facts and renders
	// it.
	RenderStored(subject []byte, c Context) (Message, error)
}

func (r Row[F]) Declared() Spec { return r.Spec }

func (r Row[F]) RenderStored(subject []byte, c Context) (Message, error) {
	var f F
	if err := json.Unmarshal(subject, &f); err != nil {
		return Message{}, fmt.Errorf("decode the facts of a %s entry: %w", r.Kind, err)
	}
	return r.Render(f, c), nil
}

// Record stores f as an entry of r's kind in tx, the Write that makes it
// true. It does row work only and fails only on a storage error. A zero
// Recorder, or a kind rec's owner didn't register, panics: a wiring bug
// that every owner's tests hit through entrytest.
func (r Row[F]) Record(ctx context.Context, tx *storage.Tx[storage.Library], rec Recorder, f F) error {
	if rec.put == nil {
		panic(fmt.Sprintf("entry: %s recorded through a zero Recorder", r.Kind))
	}
	if !rec.rows[r.Kind] {
		panic(fmt.Sprintf("entry: %s records %s, which it didn't register", rec.owner, r.Kind))
	}
	subject, err := json.Marshal(f)
	if err != nil {
		panic(fmt.Sprintf("entry: the facts of %s don't encode: %v", r.Kind, err))
	}
	return rec.put(ctx, tx, r.Kind, subject, f.About())
}

// Put stores one entry's subject and what it is about in tx.
type Put func(ctx context.Context, tx *storage.Tx[storage.Library], k Kind, subject []byte, a About) error

// Recorder is what an owner holds in its Config to record entries. It has
// no exported method: the only way to record is Row.Record. Only Sink makes
// one.
type Recorder struct {
	owner string
	rows  map[Kind]bool
	put   Put
}

// Sink is a Recorder for owner's rows, handing each entry to put.
func Sink(owner string, rows []Entry, put Put) Recorder {
	kinds := make(map[Kind]bool, len(rows))
	for _, e := range rows {
		kinds[e.Declared().Kind] = true
	}
	return Recorder{owner: owner, rows: kinds, put: put}
}
