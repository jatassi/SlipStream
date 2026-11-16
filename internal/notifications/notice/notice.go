// Package notice is what an owner of a fact needs to record a notification
// event in the Write that makes it true: a typed catalog row, the Recorder
// its Config holds, and the provider-neutral Message its row renders.
package notice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/internal/storage"
)

// Kind is a catalog row's id. It is stored on events and in each
// connection's event set, so it never changes once shipped.
type Kind string

// Group is where a row's toggle sits in the event list.
type Group string

const (
	Grabs   Group = "grabs"
	Imports Group = "imports"
	Library Group = "library"
	Health  Group = "health"
	System  Group = "system"
)

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
}

// Row is one catalog row whose facts are F. Record and Render take the same
// F, so what is stored is what is rendered. F is stored as JSON for as long
// as an event is kept, so its fields are only ever added.
type Row[F any] struct {
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
		return Message{}, fmt.Errorf("decode the facts of a %s event: %w", r.Kind, err)
	}
	return r.Render(f, c), nil
}

// Record stores f as an event of r's kind in tx, the Write that makes it
// true. It does row work only and fails only on a storage error. A zero
// Recorder, or a kind rec's owner didn't register, panics: a wiring bug
// that every owner's tests hit through noticetest.
func (r Row[F]) Record(ctx context.Context, tx *storage.Tx[storage.Library], rec Recorder, f F) error {
	if rec.put == nil {
		panic(fmt.Sprintf("notice: %s recorded through a zero Recorder", r.Kind))
	}
	if !rec.rows[r.Kind] {
		panic(fmt.Sprintf("notice: %s records %s, which it didn't register", rec.owner, r.Kind))
	}
	subject, err := json.Marshal(f)
	if err != nil {
		panic(fmt.Sprintf("notice: the facts of %s don't encode: %v", r.Kind, err))
	}
	return rec.put(ctx, tx, r.Kind, subject)
}

// Put stores one event's subject in tx.
type Put func(ctx context.Context, tx *storage.Tx[storage.Library], k Kind, subject []byte) error

// Recorder is what an owner holds in its Config to record events. It has
// no exported method: the only way to record is Row.Record. Only Sink makes
// one.
type Recorder struct {
	owner string
	rows  map[Kind]bool
	put   Put
}

// Sink is a Recorder for owner's rows, handing each event to put.
func Sink(owner string, rows []Entry, put Put) Recorder {
	kinds := make(map[Kind]bool, len(rows))
	for _, e := range rows {
		kinds[e.Declared().Kind] = true
	}
	return Recorder{owner: owner, rows: kinds, put: put}
}
