// Package plan is the holder's pure core: given every torrent it holds, the
// Node's limits and the announce policy, Decide says what to start, pause,
// check, fetch and remove now, why each other torrent waits, and when to
// decide again.
package plan

import (
	"cmp"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
)

// Hold is why a torrent that could run doesn't. The Controller maps it to
// Paused or Queued.
type Hold uint8

const (
	HoldNone       Hold = iota
	HoldWanted          // the Admin paused it
	HoldWindow          // the announce policy or the start spread; Until set
	HoldDiskFull        // a file error out of space paused it
	HoldSlot            // no free download slot
	HoldCheckSlot       // no free check slot
	HoldDiskGuard       // the Controller holds new downloads
	HoldDraining        // the Role is draining and the torrent needs a fetch
	HoldController      // out of step and the torrent needs a fetch
)

// Phase is where a torrent is in the engine. The holder moves it as it runs
// commands and reads events.
type Phase uint8

const (
	Fetching Phase = iota + 1 // no local data: Torrent(id) is due or in flight
	Added                     // in the engine, paused and not checking
	Checking                  // Recheck sent, the check's pause not yet seen
	Running                   // Resume sent
	Pausing                   // Pause sent, Paused not yet seen
	Parting                   // Remove sent
)

// Discard is whether the intent discards a torrent, and its download copy.
type Discard uint8

const (
	Kept Discard = iota
	KeepFiles
	DeleteFiles
)

type Torrent struct {
	Hash       bencode.InfoHash
	Order      int64
	Paused     bool // wanted paused
	Private    bool
	Discard    Discard
	Phase      Phase
	Complete   bool
	NeedsCheck bool // added without resume data, a mismatch, or an Admin's Recheck
	DiskFull   bool
	// NotBefore is the start spread, drawn once when the holder re-adds it.
	NotBefore time.Time
	// The stall clock runs from the later of the last payload and the last
	// Resume, since a queued torrent gets no payload.
	LastPayload, Resumed time.Time
}

type Node struct {
	ActiveDownloads int // 0: no limit
	ActiveChecks    int
	StalledAfter    time.Duration
	Live            bool // in step with the Controller
	Draining        bool
	HoldDownloads   bool // the disk guard
	DiskRecovered   bool // a volume sample since a disk-full pause shows more free
}

// Policy is the announce policy: when each event would be accepted.
type Policy interface {
	StartAt(h bencode.InfoHash, private bool, now time.Time) time.Time
	StopAt(h bencode.InfoHash, now time.Time) time.Time
}

type Command interface{ command() }

type (
	Fetch   struct{ Hash bencode.InfoHash }
	Resume  struct{ Hash bencode.InfoHash }
	Pause   struct{ Hash bencode.InfoHash }
	Recheck struct{ Hash bencode.InfoHash }
	Remove  struct {
		Hash        bencode.InfoHash
		DeleteFiles bool
	}
	// Parted confirms a discard the Node doesn't hold.
	Parted struct{ Hash bencode.InfoHash }
)

func (Fetch) command()   {}
func (Resume) command()  {}
func (Pause) command()   {}
func (Recheck) command() {}
func (Remove) command()  {}
func (Parted) command()  {}

// State is what Decide concluded about a torrent, for its report.
type State struct {
	Hold    Hold
	Until   time.Time
	Stalled bool
}

type Plan struct {
	Commands []Command
	// States holds only torrents whose State isn't zero.
	States map[bencode.InfoHash]State
	// Wake is when to decide again; zero: only on the next input.
	Wake time.Time
}

// Decide is pure: the same torrents, node, policy and now give the same
// plan. Every start goes through the policy, and a running download is
// never paused to make room.
func Decide(ts []Torrent, n Node, p Policy, now time.Time) Plan {
	d := decision{n: n, p: p, now: now, plan: Plan{States: map[bencode.InfoHash]State{}}}
	for _, t := range ts {
		switch {
		case t.Phase == Checking:
			d.checks++
		case d.holdsSlot(t):
			d.running++
		}
	}
	for _, t := range slices.SortedStableFunc(slices.Values(ts), func(a, b Torrent) int { return cmp.Compare(a.Order, b.Order) }) {
		if t.Discard == Kept {
			d.keep(t)
		} else {
			d.discard(t)
		}
	}
	return d.plan
}

type decision struct {
	n               Node
	p               Policy
	now             time.Time
	plan            Plan
	running, checks int
}

func (d *decision) discard(t Torrent) {
	switch t.Phase {
	case Fetching:
		d.do(Parted{Hash: t.Hash})
	case Added, Checking:
		d.do(Remove{Hash: t.Hash, DeleteFiles: t.Discard == DeleteFiles})
	case Running:
		if d.stopped(t) {
			d.do(Remove{Hash: t.Hash, DeleteFiles: t.Discard == DeleteFiles})
		}
	case Pausing, Parting:
	}
}

func (d *decision) keep(t Torrent) {
	switch {
	case t.Phase == Fetching:
		d.fetch(t)
	case t.Phase == Running && (t.Paused || t.NeedsCheck):
		if d.stopped(t) {
			d.do(Pause{Hash: t.Hash})
		}
		d.holdIfPaused(t)
	case t.Phase == Running:
		if d.stalled(t) {
			d.state(t, State{Stalled: true})
		} else if !t.Complete {
			d.wake(t.stallClock().Add(d.n.StalledAfter))
		}
	case t.Phase != Added:
		d.holdIfPaused(t)
	case t.NeedsCheck:
		if d.checks >= d.n.ActiveChecks {
			d.state(t, State{Hold: HoldCheckSlot})
			return
		}
		d.checks++
		d.do(Recheck{Hash: t.Hash})
	case t.Paused:
		d.holdIfPaused(t)
	case t.DiskFull && (!d.n.DiskRecovered || d.n.HoldDownloads):
		d.state(t, State{Hold: HoldDiskFull})
	case t.Complete:
		d.start(t)
	case d.n.HoldDownloads:
		d.state(t, State{Hold: HoldDiskGuard})
	case d.n.ActiveDownloads != 0 && d.running >= d.n.ActiveDownloads:
		d.state(t, State{Hold: HoldSlot})
	default:
		d.running++
		d.start(t)
	}
}

func (d *decision) fetch(t Torrent) {
	switch {
	case !d.n.Live:
		d.state(t, State{Hold: HoldController})
	case d.n.Draining:
		d.state(t, State{Hold: HoldDraining})
	default:
		d.do(Fetch{Hash: t.Hash})
		d.holdIfPaused(t)
	}
}

func (d *decision) start(t Torrent) {
	at := d.p.StartAt(t.Hash, t.Private, d.now)
	if t.NotBefore.After(at) {
		at = t.NotBefore
	}
	if at.After(d.now) {
		d.state(t, State{Hold: HoldWindow, Until: at})
		d.wake(at)
		return
	}
	d.do(Resume{Hash: t.Hash})
}

// stopped reports whether a stopped may go out now, and wakes when it may.
func (d *decision) stopped(t Torrent) bool {
	at := d.p.StopAt(t.Hash, d.now)
	if at.After(d.now) {
		d.wake(at)
		return false
	}
	return true
}

func (d *decision) holdIfPaused(t Torrent) {
	if t.Paused {
		d.state(t, State{Hold: HoldWanted})
	}
}

// holdsSlot is a download that keeps running: a stalled one, and one about
// to pause, free their slot.
func (d *decision) holdsSlot(t Torrent) bool {
	return t.Discard == Kept && t.Phase == Running && !t.Paused && !t.NeedsCheck && !t.Complete && !d.stalled(t)
}

func (d *decision) stalled(t Torrent) bool {
	return !t.Complete && !d.now.Before(t.stallClock().Add(d.n.StalledAfter))
}

func (t Torrent) stallClock() time.Time {
	if t.LastPayload.After(t.Resumed) {
		return t.LastPayload
	}
	return t.Resumed
}

func (d *decision) do(c Command) { d.plan.Commands = append(d.plan.Commands, c) }

func (d *decision) state(t Torrent, s State) { d.plan.States[t.Hash] = s }

func (d *decision) wake(at time.Time) {
	if d.plan.Wake.IsZero() || at.Before(d.plan.Wake) {
		d.plan.Wake = at
	}
}
