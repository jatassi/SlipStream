package files

import (
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// refill picks each cell's file: among the Item's files of the cut the
// cell's Edition wants, any file under any cut, that meet its Target, the
// best rank; on a tie the incumbent keeps the cell, then the oldest file.
// It reads w.fills only to find incumbents, so refilling its own answer
// changes nothing. Size never decides a fill: a file past a size range
// still meets the Target.
func refill(w world) map[CellKey]ID {
	of := map[works.ItemID][]File{}
	for _, f := range w.files {
		of[f.ItemID] = append(of[f.ItemID], f)
	}
	out := map[CellKey]ID{}
	for _, it := range w.tree.Items {
		for _, e := range targets.Wanted(w.goals.Editions, w.seen[it.ID]) {
			for _, t := range w.goals.Targets {
				key := CellKey{Item: it.ID, Edition: e, Target: t.ID}
				if best := fillOf(t, key, of[it.ID], w.fills[key]); !best.IsZero() {
					out[key] = best
				}
			}
		}
	}
	return out
}

// fillOf is the file of files, oldest first, that fills key's cell of t
// when incumbent fills it now: the zero ID when none meets t.
func fillOf(t targets.Target, key CellKey, files []File, incumbent ID) ID {
	var best ID
	var rank targets.Rank
	for _, f := range files {
		if !key.Edition.Takes(f.Cut) {
			continue
		}
		v := targets.Judge(t, f.Facts(), nil)
		r, _ := v.Rank()
		if !v.Meets() || !best.IsZero() && (r > rank || r == rank && f.ID != incumbent) {
			continue
		}
		best, rank = f.ID, r
	}
	return best
}
