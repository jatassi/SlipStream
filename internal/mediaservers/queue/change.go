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
	// FromRoot is the root From is in when it isn't Root: a Move's carry.
	// Zero otherwise.
	FromRoot roots.ID
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
	// SourceGone is when a Move removed the files at From, zero before.
	SourceGone time.Time
	Result     link.Result
	Lost       []link.Loss
	Problem    *link.Problem
}

// Carry is whether c must keep its item's watch data and collections: a
// moved change, or a replaced one under a new name.
func (c Change) Carry() bool { return c.From != (storing.Rel{}) }

// Move is whether c is a Move's carry, from another root.
func (c Change) Move() bool { return !c.FromRoot.IsZero() }

// At is a path in a root. Two roots may hold one Rel, so the queue keys
// every path by both.
type At struct {
	Root roots.ID
	Rel  storing.Rel
}

// At is where c's bytes are: Path in Root.
func (c Change) At() At { return At{c.Root, c.Path} }

// Origin is where a carry's item is on the server: From, in FromRoot for
// a Move and in Root for a rename.
func (c Change) Origin() At {
	if c.Move() {
		return At{c.FromRoot, c.From}
	}
	return At{c.Root, c.From}
}

// movedTo is c with its bytes now at a, keeping its origin.
func (c Change) movedTo(a At) Change {
	o := c.Origin()
	c.Root, c.Path, c.FromRoot = a.Root, a.Rel, roots.ID{}
	if o.Root != a.Root {
		c.FromRoot = o.Root
	}
	return c
}

// Foldable is whether a new change merges into c: c hasn't changed anything
// on the server yet. The partial unique indexes on mediaservers_change use
// the same predicate (TestFoldablePredicate).
func (c Change) Foldable() bool {
	return (c.State == Prepared || c.State == Due) && !c.Step.Started()
}

// Paths are the paths c touches: its path, and its origin when it is a
// carry. A library change touches its root's zero path.
func (c Change) Paths() []At {
	if c.Carry() {
		return []At{c.At(), c.Origin()}
	}
	return []At{c.At()}
}

// Pairing is whether c is half of an upgrade still waiting for the other.
func (c Change) Pairing() bool {
	return (c.Kind == link.ChangeRemoved && !c.ReplacedBy.IsZero()) || (c.Kind == link.ChangeAdded && c.Upgrade)
}
