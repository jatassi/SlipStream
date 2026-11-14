// Package queue is the pure model of a Media Server change: one row per
// server and path, folded on write. Fold merges a new change into the open
// row it meets, by the spec's merge table, so a dead server's queue grows by
// paths and not by events.
package queue

import (
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storing"
)

// State is where a change is in its life.
type State string

const (
	// Prepared holds a snapshot taken before a rename's or an upgrade's
	// bytes moved. It never sends: the move's own change makes it due.
	Prepared State = "prepared"
	Due      State = "due"
	Sending  State = "sending"
	Sent     State = "sent"
)

// Change is one mediaservers_change row.
type Change struct {
	ID     link.ChangeID
	Server link.ServerID
	Kind   link.ChangeKind
	Root   roots.ID
	// Path is zero for a library change. From is set only on a carry.
	Path, From storing.Rel
	// File is zero for a library change.
	File files.ID
	// ReplacedBy is, on a removed, the file an upgrade put in its place.
	ReplacedBy files.ID
	// Upgrade is, on an added, that an upgrade's removed will follow.
	Upgrade  bool
	State    State
	Step     link.Step
	StepAt   time.Time
	Snapshot *link.Snapshot
	Attempts int
	NextAt   time.Time
	Created  time.Time
	Sent     time.Time
	Result   link.Result
	Lost     []link.Loss
	Problem  *link.Problem
}

// Carry is whether c must keep its item's watch data and collections: a
// moved change, or a replaced one under a new name.
func (c Change) Carry() bool { return c.From != (storing.Rel{}) }

// Foldable is whether a new change merges into c: c hasn't changed anything
// on the server yet. The partial unique indexes on mediaservers_change use
// the same predicate (TestFoldablePredicate).
func (c Change) Foldable() bool {
	return (c.State == Prepared || c.State == Due) && !c.Step.Started()
}

// Paths are the root-relative paths c touches: its path, and its origin
// when it is a carry. A library change touches the zero path.
func (c Change) Paths() []storing.Rel {
	if c.Carry() {
		return []storing.Rel{c.Path, c.From}
	}
	return []storing.Rel{c.Path}
}

// Pairing is whether c is half of an upgrade still waiting for the other.
func (c Change) Pairing() bool {
	return (c.Kind == link.ChangeRemoved && !c.ReplacedBy.IsZero()) || (c.Kind == link.ChangeAdded && c.Upgrade)
}
