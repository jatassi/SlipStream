package files

import (
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// refill picks each cell's file: among the live files holding the cell's
// Item, of the cut its Edition wants, any file under any cut, that meet its
// Target, the best by the Target's order of tier, revision and score; on a
// tie the incumbent keeps the cell, then the oldest file. A fresh file
// forcing the cell takes it whatever it ranks. A forced fill keeps its file
// unless the best file is one arriving now, from another Import. Each fill
// holds its file's score at the Target. It reads w.fills only to find
// incumbents, so refilling its own answer changes nothing. Size never
// decides a fill: a file past a size range still meets the Target.
func refill(w world) map[CellKey]Fill {
	of := map[works.ItemID][]File{}
	for _, f := range w.files {
		if !f.Gone() {
			for _, it := range f.Items {
				of[it] = append(of[it], f)
			}
		}
	}
	out := map[CellKey]Fill{}
	for _, it := range w.tree.Items {
		for _, e := range targets.Wanted(w.goals.Editions, w.seen[it.ID]) {
			for _, t := range w.goals.Targets {
				key := CellKey{Item: it.ID, Edition: e, Target: t.ID}
				if fill, ok := w.fillOf(t, key, of[it.ID]); ok {
					out[key] = fill
				}
			}
		}
	}
	return out
}

// fillOf is the fill of key's cell of t among files, oldest first, false
// when none meets t.
func (w world) fillOf(t targets.Target, key CellKey, files []File) (Fill, bool) {
	incumbent := w.fills[key]
	var best, forcing, held *File
	for i := range files {
		f := &files[i]
		if !key.Edition.Takes(f.Cut) || !targets.Judge(t, targets.Scored{Facts: w.scoring.facts(*f)}, nil).Meets() {
			continue
		}
		if w.force[key] == f.ID {
			forcing = f
		}
		if !incumbent.Forced.IsZero() && f.ID == incumbent.File {
			held = f
		}
		if best == nil {
			best = f
			continue
		}
		if c := w.scoring.compare(t, *f, *best); c > 0 || c == 0 && f.ID == incumbent.File {
			best = f
		}
	}
	switch {
	case forcing != nil:
		return Fill{File: forcing.ID, Score: w.scoring.score(t, *forcing), Forced: w.forcedAt}, true
	case held != nil && !w.challenges(*best, *held):
		return Fill{File: held.ID, Score: w.scoring.score(t, *held), Forced: incumbent.Forced}, true
	case best != nil:
		return Fill{File: best.ID, Score: w.scoring.score(t, *best)}, true
	}
	return Fill{}, false
}

// challenges reports whether best, the best file at a cell held's forced
// fill holds, takes the cell: a better file arriving now from another
// Import than held's.
func (w world) challenges(best, held File) bool {
	if best.ID == held.ID || !w.fresh[best.ID] && !w.found[best.ID] {
		return false
	}
	of := w.importOf(best.ID)
	return of.IsZero() || of != w.importOf(held.ID)
}

// importOf is the Import a file of w is of, zero for none.
func (w world) importOf(id ID) storage.UUID {
	if w.fresh[id] {
		return w.importing
	}
	return w.imports[id]
}
