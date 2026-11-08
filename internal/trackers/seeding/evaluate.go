package seeding

import "time"

// Hash is everything Evaluate reads for one info hash.
type Hash struct {
	Now     time.Time
	Privacy Privacy
	// Goal is the Tracker's own, else the default.
	Goal Goal
	// Margin is added to every rule's time.
	Margin      time.Duration
	Reconciling bool
	// Stored is the window as last written, nil for none.
	Stored *Window
	// Copies are every live built-in Torrent of the hash.
	Copies []Copy
	// Last is the hash's latest removed built-in Torrent, nil for none.
	Last *Removed
}

// Privacy is whom a hash owes: Public, Private or Untracked, decided once
// by the caller from the Tracker's privacy and the torrent's flag.
type Privacy interface{ privacy() }

type (
	Public struct{}
	// Private owes its Tracker. Admin is the rule the Admin set, which
	// wins over every snapshot. Opening is what a window snapshots: the
	// strictest floor among the hash's Grabs, else the Tracker's rule from
	// its Releases. Either is nil for none.
	Private struct{ Admin, Opening Rule }
	// Untracked is a private torrent whose Tracker isn't known. Its window
	// owes with no rule until the Tracker is found.
	Untracked struct{}
)

func (Public) privacy()    {}
func (Private) privacy()   {}
func (Untracked) privacy() {}

// Totals are a Torrent's counters. A hash's totals are each field's
// maximum across its live copies.
type Totals struct {
	Uploaded, Downloaded int64
	// Size is the torrent's, so a Torrent that came complete from another
	// client reads its ratio against it.
	Size int64
	// Announced is announced_seconds: the seeding time the Tracker credits.
	// Every rule reads it, and so does a private torrent's goal.
	Announced time.Duration
	// Seeding is libtorrent's seeding time, which a public torrent's goal
	// reads, since a public torrent may have no tracker that answers.
	Seeding time.Duration
}

// Copy is one live built-in Torrent of the hash.
type Copy struct {
	Totals  Totals
	AddedAt time.Time
	// Complete is every selected piece present.
	Complete bool
	// Partial is a selection smaller than the torrent. It never announces
	// as a seeder, so the Tracker credits it no seeding time.
	Partial bool
	Seen    Seen
	// Unregistered is S28's sticky unregistered fault.
	Unregistered bool
	// Paused is wanted paused, by the Admin or a goal.
	Paused   bool
	Removing bool
	Draining bool
	Grabs    Grabs
	Ran      Ran
}

type Grabs struct {
	Downloading, Importing bool
	// Unimported is a Grab on it that ended without importing.
	Unimported bool
}

// Ran is what the Copy's goal run did, NotRun for none.
type Ran string

const (
	NotRun    Ran = ""
	RanPause  Ran = "pause"
	RanRemove Ran = "remove"
)

type Removed struct {
	At           time.Time
	Totals       Totals
	Unregistered bool
	// Continues is a removal whose Grab carries on in an external client,
	// which seeds it from then on.
	Continues bool
}

type Verdict struct {
	View View
	// Write is View.Window differing from Hash.Stored.
	Write bool
	// Plans holds one Plan per Copy, in Copies' order.
	Plans []Plan
}

// View is a hash as every read shows it.
type View struct {
	// Window is the window as it is now, nil while the hash owes nothing.
	Window *Window
	// Rule is the rule in force, nil for none.
	Rule Rule
	Goal Goal
	// Ratio's rule mark is the rule's ratio. Time's value is the time the
	// goal reads, and its rule mark includes the margin.
	Ratio Lane[Ratio]
	Time  Lane[time.Duration]
	// Owes is set while the window owes under a known rule.
	Owes *Owes
	// Risk is Safe unless the window owes.
	Risk    Risk
	GoalMet bool
}

// Lane is one lane of the meter. Rule and Goal are nil where they don't
// count it.
type Lane[T any] struct {
	Value      T
	Rule, Goal *T
}

// Plan is what a Copy's goal does now: Act, Keep or Wait.
type Plan interface{ plan() }

type (
	Act struct{ Do Action }
	// Keep removes the Torrent and keeps its download copy. On a Copy with
	// a goal pause, it replaces that run.
	Keep struct{ Why Downgrade }
	Wait struct{ Why Why }
)

func (Act) plan()  {}
func (Keep) plan() {}
func (Wait) plan() {}

// Downgrade is why a goal keeps the download copy, as a goal run stores it.
type Downgrade string

const (
	// KeptNotImported is remove-delete kept, because a Grab on the Torrent
	// didn't import.
	KeptNotImported Downgrade = "not-imported"
	// KeptDraining is a pause kept, because the Torrent's Node is draining
	// Downloading.
	KeptDraining Downgrade = "draining"
)

// Why is why a goal doesn't act on a Copy now.
type Why string

const (
	BeingRemoved      Why = "removing"
	ResumedAfterPause Why = "resumed"
	AlreadyRan        Why = "ran"
	SeedsForever      Why = "forever"
	NoTracker         Why = "no-tracker"
	NoRule            Why = "no-rule"
	// PartialOwesTime is a partial copy owing a rule only time meets.
	PartialOwesTime Why = "partial"
	StillOwes       Why = "owes"
	NotMet          Why = "not-met"
	Incomplete      Why = "incomplete"
	GrabDownloading Why = "grab-downloading"
	GrabImporting   Why = "grab-importing"
	Reconciling     Why = "reconciling"
)

// Endless is a wait with no end of its own: the Torrent seeds until the
// Admin removes, moves or writes it off.
func (w Why) Endless() bool {
	switch w {
	case ResumedAfterPause, SeedsForever, NoTracker, NoRule, PartialOwesTime:
		return true
	case BeingRemoved, AlreadyRan, StillOwes, NotMet, Incomplete, GrabDownloading, GrabImporting, Reconciling:
	}
	return false
}

func Evaluate(h Hash) Verdict {
	t := totalsOf(h)
	w := latch(h, t)
	owing := w != nil && w.State == Owing
	rule := inForce(h.Privacy, w)
	seeded := t.Announced
	if _, public := h.Privacy.(Public); public {
		seeded = t.Seeding
	}
	met := len(h.Copies) > 0 && reached(h.Goal, t, seeded)
	v := View{Window: w, Rule: rule, Goal: h.Goal, Risk: Safe, GoalMet: met && !owing}
	v.Ratio.Value, v.Time.Value = RatioOf(t), seeded
	if rule != nil {
		ratio, seed := rule.Parts()
		if ratio > 0 {
			v.Ratio.Rule = &ratio
		}
		if seed > 0 {
			v.Time.Rule = new(seed + h.Margin)
		}
	}
	if f, ok := h.Goal.(Finite); ok {
		v.Ratio.Goal, v.Time.Goal = &f.Ratio, &f.Seed
	}
	if owing {
		v.Risk = riskOf(h.Copies)
		if rule != nil {
			o, _ := owes(rule, t, h.Margin)
			v.Owes = &o
		}
	}
	plans := make([]Plan, len(h.Copies))
	for i, c := range h.Copies {
		plans[i] = planOf(h, c, rule, owing, met)
	}
	return Verdict{View: v, Write: !sameWindow(h.Stored, w), Plans: plans}
}

func totalsOf(h Hash) Totals {
	if len(h.Copies) == 0 && h.Last != nil {
		return h.Last.Totals
	}
	var t Totals
	for _, c := range h.Copies {
		t.Uploaded, t.Downloaded, t.Size = max(t.Uploaded, c.Totals.Uploaded), max(t.Downloaded, c.Totals.Downloaded), max(t.Size, c.Totals.Size)
		t.Announced, t.Seeding = max(t.Announced, c.Totals.Announced), max(t.Seeding, c.Totals.Seeding)
	}
	return t
}

// planOf is c's goal. A goal acts only on a complete copy of a hash that
// owes nothing, with nothing blocking it and no goal run on the copy.
func planOf(h Hash, c Copy, rule Rule, owing, met bool) Plan {
	then, why := intent(h.Goal, c)
	if why == "" {
		why = blocked(h, c, rule, owing, met || c.Ran == RanPause)
	}
	switch {
	case why != "":
		return Wait{Why: why}
	case then == Pause && c.Draining:
		return Keep{Why: KeptDraining}
	case then == RemoveDelete && c.Grabs.Unimported:
		return Keep{Why: KeptNotImported}
	}
	return Act{Do: then}
}

// intent is the action c's goal asks for. A goal pause on a draining Node
// asks again, so the drain can finish.
func intent(g Goal, c Copy) (Action, Why) {
	switch {
	case c.Removing:
		return "", BeingRemoved
	case c.Ran == RanPause && !c.Paused:
		return "", ResumedAfterPause
	case c.Ran == RanPause && c.Draining:
		return Pause, ""
	case c.Ran != NotRun:
		return "", AlreadyRan
	}
	if f, ok := g.(Finite); ok {
		return f.Then, ""
	}
	return "", SeedsForever
}

func blocked(h Hash, c Copy, rule Rule, owing, met bool) Why {
	switch {
	case owing:
		return owingWhy(h.Privacy, c, rule)
	case !met:
		return NotMet
	case !c.Complete:
		return Incomplete
	case c.Grabs.Downloading:
		return GrabDownloading
	case c.Grabs.Importing:
		return GrabImporting
	case h.Reconciling:
		return Reconciling
	}
	return ""
}

func owingWhy(p Privacy, c Copy, rule Rule) Why {
	if _, untracked := p.(Untracked); untracked {
		return NoTracker
	}
	switch {
	case rule == nil:
		return NoRule
	case c.Partial && needsTime(rule):
		return PartialOwesTime
	}
	return StillOwes
}
