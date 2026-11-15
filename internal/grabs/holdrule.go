package grabs

import (
	"slices"
	"time"
)

// holdRule is hooks.hold for a decision whose trigger's Holds apply, over
// in, whose rows also widens. A pick of as[g] led by cell w waits when
// every cell it would hold waits for it, until the earliest of those
// waits ends. A cell waits while its Target has a Hold, the Release isn't
// final there, and its publish time plus the Hold is after now. One cell
// that doesn't wait grabs it: a Release meant for several Targets goes
// once any of them stops waiting (#87 Q14).
func holdRule(in planIn, also func(planned) []int, now time.Time) func(w, g int, p Pick, taken []bool) (time.Time, bool) {
	return func(w, g int, p Pick, taken []bool) (time.Time, bool) {
		cells, _ := in.fills(g, w, func(k int) bool { return !taken[k] && in.fill[k] })
		for _, k := range also(planned{pick: p}) {
			if !taken[k] && !slices.Contains(cells, k) {
				cells = append(cells, k)
			}
		}
		var until time.Time
		for _, k := range cells {
			t, v := in.item[k].Target, in.js[g].verdictOf(k).v
			end := p.Release.Published.Add(t.Hold)
			if r, ranked := v.Rank(); t.Hold == 0 || ranked && t.Final(r, v.Score()) || !end.After(now) {
				return time.Time{}, false
			}
			if until.IsZero() || end.Before(until) {
				until = end
			}
		}
		return until, true
	}
}
