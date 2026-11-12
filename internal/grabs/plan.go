package grabs

import (
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
)

// planIn is one Manual run's plan inputs. Cells are counted by their
// position in item, every cell of the Work in Want order, so a pack's row
// fills cells of every Item it covers.
type planIn struct {
	item []files.Cell
	// walk is the cells the plan walks, in Want order: the missing,
	// failed, upgradable and unreleased cells of the run's Item in scope
	// that no unfinished Grab holds.
	walk []int
	// fill is whether a row may fill each cell: every missing, failed,
	// upgradable or unreleased cell of the Work, since a Grab fills every
	// Target it can (#87 Q15).
	fill []bool
	// before is the cells taken before the walk starts: those a row of an
	// earlier run fills, and those a pin of another run holds.
	before []bool
	// as is the run's Releases, deduped, with their own refusals worked
	// out now, js each judged at every cell of item and st how each
	// stands at each.
	as []assessed
	js []judged
	st [][]stance
	// pins is the Release, by its index in as, pinned at a cell.
	pins map[int]int
	take taker
}

// planned is one row of a plan: a Grab of as[group], or a gap when group
// is -1. cells is the cells it fills, first the one that leads it.
type planned struct {
	cells  []int
	group  int
	pick   Pick
	stance stance
	pinned bool
	// waits is why the Indexers won't take the best Release that stands
	// at a gap's cell, nil when none stands auto there.
	waits *indexers.NotTakenError
}

// plan walks in.walk. At each cell no earlier row fills, a live pin takes
// it. Otherwise it calls decide over the cells left and choose over its
// picks, as finish and take do, and keeps the pick when its first cell is
// this one. A cell nothing takes is a gap. On a monitored, released Item
// with no pin at its first cell, the first call's cells are openCells
// narrowed to the Item, so the first row is automatic search's take. A
// pin whose Release doesn't stand at its cell, or whose Indexers won't
// take it, is dropped.
func plan(in planIn) ([]planned, map[int]int, error) {
	taken := make([]bool, len(in.item))
	copy(taken, in.before)
	live := map[int]int{}
	var out []planned
	for _, w := range in.walk {
		if taken[w] {
			continue
		}
		if g, ok := in.pins[w]; ok {
			p, ok, err := in.offer(g, w)
			if err != nil {
				return nil, nil, err
			}
			if ok {
				r := in.row(w, g, p, taken)
				r.pinned = true
				live[w] = g
				out = append(out, r)
				continue
			}
		}
		p, g, ok, err := in.next(taken)
		if err != nil {
			return nil, nil, err
		}
		if ok && g.lead == w {
			out = append(out, in.row(w, g.group, p, taken))
			continue
		}
		taken[w] = true
		gap := planned{cells: []int{w}, group: -1}
		gap.waits, err = in.waitsAt(w)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, gap)
	}
	return out, live, nil
}

// led is the Release choose picked, by its index in as, and the cell
// leading what it fills.
type led struct{ group, lead int }

// next is what decide and choose pick over the walk's cells no row
// fills yet, and the cell it leads at. It narrows in.js to those cells
// rather than judging again, which gives the same verdicts.
func (in planIn) next(taken []bool) (Pick, led, bool, error) {
	var ks []int
	var left cells
	for _, w := range in.walk {
		if !taken[w] {
			ks, left.open = append(ks, w), append(left.open, in.item[w])
		}
	}
	js := make([]judged, len(in.js))
	for g, j := range in.js {
		js[g] = j.only(ks)
	}
	p, _, ok, err := choose(decided(left, js).picks(left), nil, in.take)
	if !ok || err != nil {
		return p, led{}, ok, err
	}
	g := groupOf(in.as, p.at)
	return p, led{group: g, lead: ks[js[g].first]}, true, nil
}

// offer is the offer of as[g], or of a copy, that choose takes for cell
// w, false when g doesn't stand at w or no Indexer of it takes it now.
func (in planIn) offer(g, w int) (Pick, bool, error) {
	if in.st[g][w].kind == standsRefused {
		return Pick{}, false, nil
	}
	j := in.js[g]
	p := Pick{Release: j.Release, Edition: in.item[w].Edition, at: j.at, copies: j.copies}
	p, _, ok, err := choose([]Pick{p}, nil, in.take)
	return p, ok, err
}

// row is the row of as[g], offered as p, led by cell w: what it fills
// of the cells it may fill that no row fills yet. It marks them taken.
func (in planIn) row(w, g int, p Pick, taken []bool) planned {
	r := planned{group: g, pick: p}
	r.cells, r.stance = in.fills(g, w, func(k int) bool { return !taken[k] && in.fill[k] })
	for _, k := range r.cells {
		taken[k] = true
	}
	return r
}

// fills is the cells as[g] fills led by cell w, of those free: w first,
// then each other free cell of w's Edition where it stands auto, or as it
// stands at w when that isn't auto. Its stance is its kind and Edition at
// w with every lift across those cells. A row, a pick and an offer all
// read it, so a Grab fills and gets past what the sheet showed.
func (in planIn) fills(g, w int, free func(k int) bool) ([]int, stance) {
	lead := in.st[g][w]
	out, s := []int{w}, stance{kind: lead.kind, edition: lead.edition}
	for k, cell := range in.item {
		st := in.st[g][k]
		if k != w && (!free(k) || cell.Edition != in.item[w].Edition || st.kind != standsAuto && (st.kind != lead.kind || lead.kind == standsAuto)) {
			continue
		}
		if k != w {
			out = append(out, k)
		}
		for _, l := range st.lifts {
			if !slices.Contains(s.lifts, l) {
				s.lifts = append(s.lifts, l)
			}
		}
	}
	return out, s
}

// waitsAt is the refusal of the Indexer of the best Release standing
// auto at cell w, which choose passed over, nil when none stands auto.
func (in planIn) waitsAt(w int) (*indexers.NotTakenError, error) {
	var auto []judged
	for g, st := range in.st {
		if st[w].kind == standsAuto {
			auto = append(auto, in.js[g])
		}
	}
	if len(auto) == 0 {
		return nil, nil
	}
	best := rankAt(cells{open: in.item}, w, auto)[0]
	_, err := in.take(best.Release)
	if e, ok := errors.AsType[*indexers.NotTakenError](err); ok {
		return e, nil
	}
	return nil, err
}

// choicesAt is every Release but as[except] that could fill cell w,
// ranked at w: those standing auto first, then lifted, then on the
// Blocklist. A Release none of whose Indexers take it now is left out.
func (in planIn) choicesAt(w, except int) ([]int, error) {
	var groups [3][]judged
	at := map[int]int{}
	for g, st := range in.st {
		k := st[w].kind
		if g == except || k == standsRefused {
			continue
		}
		if _, ok, err := in.offer(g, w); err != nil || !ok {
			if err != nil {
				return nil, err
			}
			continue
		}
		at[in.js[g].at] = g
		groups[k-standsAuto] = append(groups[k-standsAuto], in.js[g])
	}
	var out []int
	for _, js := range groups {
		for _, j := range rankAt(cells{open: in.item}, w, js) {
			out = append(out, at[j.at])
		}
	}
	return out, nil
}

// groupOf is the index in as of the Release listed at position, or of
// the Release it is a copy of, -1 for none.
func groupOf(as []assessed, position int) int {
	return slices.IndexFunc(as, func(a assessed) bool {
		return a.at == position || slices.ContainsFunc(a.copies, func(c copied) bool { return c.at == position })
	})
}

// verdictOf is j's verdict at cell k, one of the cells it was judged at.
func (j judged) verdictOf(k int) verdict {
	return j.verdicts[slices.IndexFunc(j.verdicts, func(v verdict) bool { return v.cell == k })]
}
