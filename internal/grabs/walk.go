package grabs

import (
	"slices"

	"github.com/jatassi/slipstream/internal/indexers"
)

// hooks are what Manual search's plan and an automatic decision walk
// apart. waits is why the Indexers won't take what stands at a gap's
// cell, nil for no reason. also is the cells beyond a row's own its Grab
// holds: for an automatic decision, the other Items' cells where its
// Release ranks first.
type hooks struct {
	waits func(w int) (*indexers.NotTakenError, error)
	also  func(r planned) []int
}

// walk walks in.walk in order, the one walk Manual search's plan and
// every automatic decision share. At each cell no earlier row fills, a
// live pin takes it. Otherwise it calls decide over the walk's cells left
// and choose over the picks first grabbable at this cell, and the pick
// choose takes is a row over the cells it fills, which h.also widens. A
// cell nothing takes is a gap. A pin whose Release doesn't stand at its
// cell, or whose Indexers won't take it, is dropped. It answers the rows
// and each live pin's Release, by cell.
func walk(in planIn, h hooks) ([]planned, map[int]int, error) {
	taken := make([]bool, len(in.item))
	copy(taken, in.before)
	live := map[int]int{}
	var out []planned
	for _, w := range in.walk {
		if taken[w] {
			continue
		}
		r, ok, err := in.pinned(w, taken)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			live[w] = r.group
			out = append(out, r)
			continue
		}
		l, ok, err := in.next(taken)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			out = append(out, in.widened(w, l, taken, h))
			continue
		}
		taken[w] = true
		gap := planned{cells: []int{w}, group: -1}
		if h.waits != nil {
			if gap.waits, err = h.waits(w); err != nil {
				return nil, nil, err
			}
		}
		out = append(out, gap)
	}
	return out, live, nil
}

// pinned is the row of the live pin at cell w, false when w has none or
// its pin is dropped.
func (in planIn) pinned(w int, taken []bool) (planned, bool, error) {
	g, ok := in.pins[w]
	if !ok {
		return planned{}, false, nil
	}
	p, ok, err := in.offer(g, w)
	if !ok || err != nil {
		return planned{}, false, err
	}
	r := in.row(w, g, p, taken)
	r.pinned = true
	return r, true, nil
}

// widened is the row of l, led by cell w, widened by h.also to the cells no
// row fills yet. It marks them taken.
func (in planIn) widened(w int, l led, taken []bool, h hooks) planned {
	r := in.row(w, l.group, l.pick, taken)
	r.ticket, r.fills = l.ticket, l.fills
	if h.also == nil {
		return r
	}
	for _, k := range h.also(r) {
		if !taken[k] {
			taken[k] = true
			r.cells = append(r.cells, k)
		}
	}
	return r
}

// led is what next picked: the pick, the Ticket take readied it with,
// its Release's index in as, and the Items it was chosen to fill when
// the fewer-Releases step put it first, 0 otherwise.
type led struct {
	pick   Pick
	ticket indexers.Ticket
	group  int
	fills  int
}

// next is what decide and choose pick among the Releases first
// grabbable at the first of the walk's cells no row fills yet, judged
// over those cells. It narrows in.js to them rather than judging again,
// which gives the same verdicts. Only a pick it keeps is offered to
// in.take, so a take that reserves a grab reserves none it drops.
func (in planIn) next(taken []bool) (led, bool, error) {
	var ks []int
	left := cells{wanted: in.wanted, chosen: in.chosen}
	for _, w := range in.walk {
		if !taken[w] {
			ks, left.open = append(ks, w), append(left.open, in.item[w])
		}
	}
	var js []judged
	for _, j := range in.js {
		o := j.only(ks)
		if o.first != 0 {
			continue
		}
		o.beyond = in.beyond(j, ks, taken)
		js = append(js, o)
	}
	d := decided(left, js)
	p, t, ok, err := choose(d.picks(left), in.gone, in.take)
	if !ok || err != nil {
		return led{}, ok, err
	}
	return led{pick: p, ticket: t, group: groupOf(in.as, p.at), fills: fewerFills(left, d.Ranked, p)}, true, nil
}

// fewerFills is the Items with no file p's Release fills when the
// fewer-Releases step ranked it above the next Release of ranked an
// automatic search may take, 0 when another step did, there is none, or
// it fills fewer than two.
func fewerFills(c cells, ranked []judged, p Pick) int {
	i := slices.IndexFunc(ranked, func(j judged) bool {
		return j.at == p.at || slices.ContainsFunc(j.copies, func(o copied) bool { return o.at == p.at })
	})
	k := slices.IndexFunc(ranked[i+1:], func(j judged) bool { return len(j.refusals) == 0 })
	if k < 0 {
		return 0
	}
	if l, _, _ := differ(c, ranked[i], ranked[i+1+k]); l.step != StepFewerReleases {
		return 0
	}
	if n := c.missing(ranked[i]); n > 1 {
		return n
	}
	return 0
}
