package seeding

import "time"

// State is a window's place in its lifecycle:
//
//	owing -> met          the rule plus the margin is reached (final)
//	owing -> released     a copy is unregistered, or the last copy left for an external client (final)
//	owing -> missed       the last live copy was removed while owing
//	missed -> owing       a live copy came back (a regrab), keeping the miss in the history
type State string

const (
	Owing    State = "owing"
	Met      State = "met"
	Missed   State = "missed"
	Released State = "released"
)

// Window is a private info hash's hit and run window as trackers_window
// stores it. It opens at the first downloaded byte.
type Window struct {
	State State
	// Rule is the snapshot taken when it opened, nil while none is known.
	// A nil snapshot takes the first rule known, and a known one never
	// changes. An Admin's rule wins over it while set.
	Rule     Rule
	OpenedAt time.Time
	MetAt    time.Time
	// ClosedAt is the latest miss or release. A regrab keeps it, so the
	// miss stays in the history.
	ClosedAt    time.Time
	RegrabbedAt time.Time
	// Owed is what the latest miss still owed, zero before one.
	Owed Owes
	// QuietSince is when it first read at risk, zero while it doesn't. A
	// window whose Nodes are out of step keeps it.
	QuietSince time.Time
}

func sameWindow(a, b *Window) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.State == b.State && a.Rule == b.Rule && a.Owed == b.Owed && a.OpenedAt.Equal(b.OpenedAt) && a.MetAt.Equal(b.MetAt) &&
		a.ClosedAt.Equal(b.ClosedAt) && a.RegrabbedAt.Equal(b.RegrabbedAt) && a.QuietSince.Equal(b.QuietSince)
}

// latch is the window as the hash shows it now. A public hash never opens
// one, and a window opens only on a live copy, so a hash removed before
// any window was written leaves none.
func latch(h Hash, t Totals) *Window {
	p, private := h.Privacy.(Private)
	if _, untracked := h.Privacy.(Untracked); !private && !untracked {
		return h.Stored
	}
	var w Window
	switch {
	case h.Stored != nil:
		w = *h.Stored
	case len(h.Copies) > 0 && t.Downloaded > 0:
		w = Window{State: Owing, OpenedAt: h.Now}
	default:
		return nil
	}
	switch w.State {
	case Met, Released:
		return &w
	case Missed:
		if len(h.Copies) == 0 {
			return &w
		}
		w.State, w.RegrabbedAt = Owing, firstAdded(h.Copies)
	case Owing:
	}
	if w.Rule == nil && private {
		w.Rule = p.Opening
	}
	w = settle(h, w, t)
	return &w
}

// settle moves an owing window on. A removal and a regrab that both fell
// between two runs read as the miss they were.
func settle(h Hash, w Window, t Totals) Window {
	rule := inForce(h.Privacy, &w)
	owed, met := owedOf(rule, t, h.Margin)
	last := h.Last
	inRun := last != nil && last.At.After(lastOpen(w))
	switch {
	case met:
		w.State, w.MetAt = Met, h.Now
	case released(h.Copies, last, inRun):
		w.State, w.ClosedAt = Released, h.Now
	case len(h.Copies) == 0:
		w.State, w.ClosedAt, w.Owed = Missed, last.At, owed
	case inRun && firstAdded(h.Copies).After(last.At):
		w.ClosedAt, w.RegrabbedAt = last.At, firstAdded(h.Copies)
		w.Owed, _ = owedOf(rule, last.Totals, h.Margin)
	}
	if w.State != Owing {
		w.QuietSince = time.Time{}
		return w
	}
	switch riskOf(h.Copies) {
	case AtRisk:
		if w.QuietSince.IsZero() {
			w.QuietSince = h.Now
		}
	case Safe:
		w.QuietSince = time.Time{}
	case RiskUnknown:
	}
	return w
}

func released(copies []Copy, last *Removed, inRun bool) bool {
	for _, c := range copies {
		if c.Unregistered {
			return true
		}
	}
	return inRun && last.Unregistered || len(copies) == 0 && last.Continues
}

// inForce is the rule a hash owes now: the Admin's, else the window's
// snapshot, else what a window opening now would snapshot.
func inForce(p Privacy, w *Window) Rule {
	private, ok := p.(Private)
	switch {
	case !ok:
		return nil
	case private.Admin != nil:
		return private.Admin
	case w != nil:
		return w.Rule
	}
	return private.Opening
}

// owedOf is what t still lacks of rule, zero with no rule, which is never met.
func owedOf(rule Rule, t Totals, margin time.Duration) (Owes, bool) {
	if rule == nil {
		return Owes{}, false
	}
	return owes(rule, t, margin)
}

func lastOpen(w Window) time.Time {
	if w.RegrabbedAt.After(w.OpenedAt) {
		return w.RegrabbedAt
	}
	return w.OpenedAt
}

func firstAdded(copies []Copy) time.Time {
	first := copies[0].AddedAt
	for _, c := range copies[1:] {
		if c.AddedAt.Before(first) {
			first = c.AddedAt
		}
	}
	return first
}
