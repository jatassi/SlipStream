package files

import (
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// refill picks each cell's file: among the Item's files of the cut the
// cell's Edition wants, any file under any cut, that meet its Target, the
// best by the Target's order of tier, revision and score; on a tie the
// incumbent keeps the cell, then the oldest file. Each fill holds its
// file's score at the Target. It reads w.fills only to find incumbents,
// so refilling its own answer changes nothing. Size never decides a fill:
// a file past a size range still meets the Target.
func refill(w world) map[CellKey]Fill {
	of := map[works.ItemID][]File{}
	for _, f := range w.files {
		of[f.ItemID] = append(of[f.ItemID], f)
	}
	out := map[CellKey]Fill{}
	for _, it := range w.tree.Items {
		for _, e := range targets.Wanted(w.goals.Editions, w.seen[it.ID]) {
			for _, t := range w.goals.Targets {
				key := CellKey{Item: it.ID, Edition: e, Target: t.ID}
				if best, ok := fillOf(w.scoring, t, key, of[it.ID], w.fills[key].File); ok {
					out[key] = Fill{File: best.ID, Score: t.Score(w.scoring.of(best))}
				}
			}
		}
	}
	return out
}

// fillOf is the file of files, oldest first, that fills key's cell of t
// when incumbent fills it now, false when none meets t.
func fillOf(s *scoring, t targets.Target, key CellKey, files []File, incumbent ID) (File, bool) {
	var best File
	found := false
	for _, f := range files {
		if !key.Edition.Takes(f.Cut) || !targets.Judge(t, targets.Scored{Facts: s.facts(f)}, nil).Meets() {
			continue
		}
		if !found {
			best, found = f, true
			continue
		}
		if c := s.compare(t, f, best); c > 0 || c == 0 && f.ID == incumbent {
			best = f
		}
	}
	return best, found
}
