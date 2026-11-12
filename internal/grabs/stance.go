package grabs

import (
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/targets"
)

// stance is how one judged Release stands at one cell for a Manual pick.
type stance struct {
	kind stanceKind
	// lifts is the soft reasons a grab there gets past, its own included,
	// each with its Target: what the row says and what the pick names.
	lifts []Reason
	// edition is the cell's Edition when it doesn't take the Release's
	// cut, "" otherwise: the Edition override.
	edition targets.Edition
	// why is the reasons it can't be grabbed there, for standsRefused.
	why []Reason
}

type stanceKind uint8

const (
	// standsRefused: a hard reason or not_an_upgrade.
	standsRefused stanceKind = iota
	// standsAuto: grabbable at the cell with no refusal of its own, what
	// picks keeps.
	standsAuto
	// standsLifted: every reason at the cell and of its own is liftable,
	// or cut_not_wanted under the Edition override.
	standsLifted
	// standsBlocked: it would stand auto or lifted but for the Blocklist.
	standsBlocked
)

// grabbable reports a stance a pick may take: auto or lifted.
func (s stance) grabbable() bool { return s.kind == standsAuto || s.kind == standsLifted }

// stanceAt is how j stands at cell, whose verdict is v.
func stanceAt(j judged, cell files.Cell, v verdict) stance {
	var s stance
	blocked := false
	for _, r := range j.refusals {
		switch {
		case r.Code == targets.Blocklisted:
			blocked = true
		case slices.Contains(Liftable, r.Code):
			s.lifts = append(s.lifts, r)
		default:
			s.why = append(s.why, r)
		}
	}
	if !v.wanted {
		s.edition = cell.Edition
	}
	for _, r := range v.v.Reasons() {
		at := Reason{Reason: r, Target: v.target}
		switch {
		case r.Code == targets.SizeNotChecked:
		case slices.Contains(Liftable, r.Code):
			s.lifts = append(s.lifts, at)
		default:
			s.why = append(s.why, at)
		}
	}
	switch {
	case len(s.why) > 0:
		s.kind = standsRefused
	case blocked:
		s.kind = standsBlocked
	case len(s.lifts) == 0 && s.edition == "":
		s.kind = standsAuto
	default:
		s.kind = standsLifted
	}
	return s
}

// stances is how j, judged at every cell of c, stands at each, by
// position in c. A Release with a hard refusal of its own has no verdict
// and stands refused everywhere.
func stances(c []files.Cell, j judged) []stance {
	out := make([]stance, len(c))
	if len(j.verdicts) == 0 {
		for i := range out {
			out[i] = stance{why: j.refusals}
		}
		return out
	}
	for _, v := range j.verdicts {
		out[v.cell] = stanceAt(j, c[v.cell], v)
	}
	return out
}

// rankAt is js ranked as the chain ranks Releases whose first cell is i
// of c: each one's first set to i, then rank. Other choices and a gap's
// choices use it, so a cell's choices are in the chain's own order.
func rankAt(c cells, i int, js []judged) []judged {
	out := slices.Clone(js)
	for k := range out {
		out[k].first = i
	}
	rank(c, out)
	return out
}
