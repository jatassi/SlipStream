// Package pace is every rule of the search queue that can be stated
// without storage: lanes, how a trigger merges into a group's queue row,
// and when a run may start. It is pure. The Backlog gaps and spread are
// pacing's.
package pace

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/searches"
)

// Lane is a queue lane, stored as its rank: a lower rank is served first.
// The zero Lane is not queued. S40 (Requests and approval) adds rank 20.
type Lane int64

const (
	// Now is the Admin's presses on one Work, an add, monitoring on, a
	// Profile or Editions change of one Work, and search again.
	Now Lane = 10
	// Bulk is Search all and the Library's bulk Search.
	Bulk Lane = 30
	// Backlog is Backlog search, release dates, refreshes, bulk changes
	// and failures at Backlog pace.
	Backlog Lane = 40
)

// Lanes is every Lane in the order dispatch serves them. A test pins the
// lane CHECKs to it.
var Lanes = []Lane{Now, Bulk, Backlog}

// String is l's wire word. The wire never shows a rank.
func (l Lane) String() string {
	switch l {
	case Now:
		return "now"
	case Bulk:
		return "bulk"
	case Backlog:
		return "backlog"
	}
	return ""
}

// Press reports whether l's lines are presses: one slot above Searches at
// once and the search limit's last slots.
func (l Lane) Press() bool { return l == Now }

// Entry is one trigger landing on a group.
type Entry struct {
	// Lane zero queues nothing: the Entry only resets the gap.
	Lane    Lane
	Trigger searches.Trigger
	// Holds is whether a Target's Hold applies to the grabs of a run of
	// Trigger.
	Holds bool
	// Items nil is the whole group.
	Items []string
	// Gone opens gone cells.
	Gone     bool
	Order    Order
	ResetGap bool
	At       time.Time
}

// Outranks reports whether a takes a row's trigger and gone cells from b:
// a's lane ranks first, or the lanes tie and a has no Hold while b has one.
// So the Admin's press is never put on a Hold or loses its gone cells to an
// add, monitoring on, a Profile change or a failure.
func (a Entry) Outranks(b Entry) bool {
	return a.Lane < b.Lane || a.Lane == b.Lane && !a.Holds && b.Holds
}

// Order is a queued row's place inside its lane: missing and failed before
// upgradable, Items inside the full-pace window first, newest release
// first, then queued time.
type Order struct {
	UpgradeOnly bool
	Fresh       bool
	Released    time.Time
}

// Row is one group's queue row as Merge sees it.
type Row struct {
	Exists bool
	// Lane zero is not queued. Trigger, Holds, Items, Gone, Order and
	// Queued are the queued entry's.
	Lane    Lane
	Trigger searches.Trigger
	Holds   bool
	// Items nil is the whole group.
	Items  []string
	Gone   bool
	Order  Order
	Queued time.Time
	// Due is when Backlog search next queues the row, Gap its spacing and
	// Searched its last search, zero when never searched.
	Due      time.Time
	Gap      time.Duration
	Searched time.Time
	// Searching is the open run, nil when none is open.
	Searching *Run
	Changed   time.Time
}

// Run is a group's open run as Merge sees it.
type Run struct {
	Lane  Lane
	Holds bool
	// Whole is a run of the whole group, else Items is its scope.
	Whole bool
	Items []string
}

// Again reports whether the row is queued behind its own open run: the
// spec's search again.
func (r Row) Again() bool { return r.Lane != 0 && r.Searching != nil }

// State is what a press reads back for a group.
type State uint8

const (
	// Queued is newly queued, or queued again behind its own run.
	Queued State = iota + 1
	// Merged is already queued or searching with this scope.
	Merged
)

// Merge is r with e landed on it, the State a press reads, and whether the
// open run is promoted to e's lane, trigger and gone cells. It is
// idempotent: Merge of its own answer with e answers the same Row.
//
//   - A new row gets Gap interval and Due e.At plus spread.
//   - ResetGap sets Gap to interval and, once searched, Due to the last
//     search plus it.
//   - An Entry with no Lane queues nothing and reads Merged.
//   - A run open whose scope covers e's Items: nothing is queued, State
//     Merged, and Promote when e outranks the run.
//   - A run open and e adds Items: the row is queued with the Items the
//     run doesn't cover (the whole group when e asks it), by the rule
//     below, and Promote as above.
//   - Queuing: Lane, Trigger, Holds and Gone are the outranking entry's,
//     the queued one keeping them on a full tie. Items are the union, nil
//     absorbing. Queued is the earlier time, Order e's. State Merged when
//     the row was queued with a scope that covers e's, else Queued.
func Merge(r Row, e Entry, interval, spread time.Duration) (Row, State, bool) {
	r.Items, r.Changed = clone(r.Items), e.At
	if !r.Exists {
		r.Exists, r.Gap, r.Due = true, interval, e.At.Add(spread)
	}
	if e.ResetGap {
		r.Gap = interval
		if !r.Searched.IsZero() {
			r.Due = r.Searched.Add(interval)
		}
	}
	if e.Lane == 0 {
		return r, Merged, false
	}
	items, covered, promote := beside(r.Searching, e)
	if promote {
		r.Searching = &Run{Lane: e.Lane, Holds: e.Holds, Whole: r.Searching.Whole, Items: clone(r.Searching.Items)}
	}
	if covered {
		return r, Merged, promote
	}
	if r.Lane == 0 {
		r.Lane, r.Trigger, r.Holds, r.Gone = e.Lane, e.Trigger, e.Holds, e.Gone
		r.Items, r.Queued, r.Order = sorted(items), e.At, e.Order
		return r, Queued, promote
	}
	state := Queued
	if r.Items == nil || items != nil && covers(r.Items, items) {
		state = Merged
	}
	if e.Outranks(Entry{Lane: r.Lane, Holds: r.Holds}) {
		r.Lane, r.Trigger, r.Holds, r.Gone = e.Lane, e.Trigger, e.Holds, e.Gone
	}
	if r.Items != nil {
		r.Items = union(r.Items, items)
	}
	if e.At.Before(r.Queued) {
		r.Queued = e.At
	}
	r.Order = e.Order
	return r, state, promote
}

// beside is e's Items an open run doesn't cover, whether it covers them
// all, and whether e outranks it. With no run open it is e's Items.
func beside(run *Run, e Entry) (items []string, covered, promote bool) {
	if run == nil {
		return e.Items, false, false
	}
	promote = e.Outranks(Entry{Lane: run.Lane, Holds: run.Holds})
	switch {
	case run.Whole || e.Items != nil && covers(run.Items, e.Items):
		return nil, true, promote
	case e.Items == nil:
		return nil, false, promote
	}
	return slices.DeleteFunc(clone(e.Items), func(i string) bool { return slices.Contains(run.Items, i) }), false, promote
}

func clone(items []string) []string {
	if items == nil {
		return nil
	}
	return slices.Clone(items)
}

func covers(have, want []string) bool {
	for _, i := range want {
		if !slices.Contains(have, i) {
			return false
		}
	}
	return true
}

// union is a and b's Items, sorted, nil when b is the whole group.
func union(a, b []string) []string {
	if b == nil {
		return nil
	}
	return sorted(append(clone(a), b...))
}

func sorted(items []string) []string {
	if items == nil {
		return nil
	}
	items = slices.Clone(items)
	slices.Sort(items)
	return slices.Compact(items)
}

// Room is one Indexer as the start rule sees it.
type Room struct {
	// AtOnce is the Indexer's Searches at once.
	AtOnce int
	// Waiting is the planned, undispatched lines of open runs on it.
	Waiting int
}

// Starts reports whether a run that asks this Indexer may start: fewer
// waiting lines than Searches at once, plus one for a press. A run starts
// only when every Indexer it would ask Starts, so a fast Indexer runs at
// most one line ahead of the slowest and open runs stay bounded.
func (r Room) Starts(press bool) bool {
	n := r.AtOnce
	if press {
		n++
	}
	return r.Waiting < n
}
