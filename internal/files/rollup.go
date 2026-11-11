package files

import (
	"context"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Counts is a count for each State, indexed by it.
type Counts [Met + 1]int

// Counted is what a Work's or a level's counted Items add up to.
type Counted struct {
	// Cells counts their cells by State: each Edition an Item wants at
	// each Target. A listed cut not seen for an Item makes no cell, so it
	// never counts.
	Cells Counts
	Items int
	// ByItem counts them by status. An Item with no cell isn't counted.
	ByItem Counts
}

func (t *Counted) add(cells []Cell) (State, bool) {
	t.Items++
	for _, c := range cells {
		t.Cells[c.State]++
	}
	s, ok := Status(cells)
	if ok {
		t.ByItem[s]++
	}
	return s, ok
}

// Rollup is what one Work's cells and files add up to. The Library's row
// and the Work page read the same one.
type Rollup struct {
	// Status is the first State any of the Work's counted Items holds, in
	// the glossary's order, zero while none holds a cell.
	Status State
	Counted
	// Levels rolls up each level the Work's Items sit in, in the order its
	// first Item comes. A Work of a type with no levels has none.
	Levels []LevelRollup
	Needs  []Need
	// Files, Kept and Bytes count the Work's live library files, Kept
	// files included.
	Files, Kept int
	Bytes       int64
	// Available is the one Item's availability date, or for many Items the
	// earliest not yet out, else the latest out, read over the counted
	// Items, or over the live ones while none counts. It is unknown while
	// none of those has a date.
	Available release.Fact[works.Date]
	Profile   targets.ProfileID
	// Root is zero while the Work has none.
	Root roots.ID
}

// LevelRollup is what one level's Items add up to, counted as the Work's
// Rollup counts them.
type LevelRollup struct {
	Level works.LevelID
	Counted
	// Withdrawn counts the cells of the level's Items its source no longer
	// lists that a file still fills, by State.
	Withdrawn Counts
}

// Rollups reads the Rollup of each of ws, the Works among names, in r and
// in ws' order, loading each kind of row once for them all.
func (f *Files) Rollups(ctx context.Context, r storage.Reader[file], ws []works.Bare, among works.Among[works.WorkID]) ([]Rollup, error) {
	worlds, err := f.worlds(ctx, r, ws, among, briefFiles)
	if err != nil {
		return nil, err
	}
	out := make([]Rollup, len(worlds))
	var cs []Cell
	for i, w := range worlds {
		cs = appendCells(cs[:0], w)
		out[i] = rollup(w, cs)
	}
	return out, nil
}

func rollup(w world, cs []Cell) Rollup {
	out := Rollup{
		Needs:     needs(w.tree.Items, cs, w.grabs, !w.root.IsZero()),
		Available: available(w.tree.Items, w.timing, w.today), Profile: w.goals.Profile.ID, Root: w.root,
	}
	eachItem(w.tree.Items, cs, func(it works.Item, its []Cell) {
		if !it.Counts() {
			return
		}
		if s, ok := out.add(its); ok && (out.Status == 0 || s < out.Status) {
			out.Status = s
		}
	})
	out.Levels = levelsOf(w.tree.Items, cs)
	var held Held
	for _, file := range w.files {
		if !file.Gone() {
			held.add(file)
		}
	}
	out.Files, out.Kept, out.Bytes = held.Files, held.Kept, held.Bytes
	return out
}

func levelsOf(items []works.Item, cs []Cell) []LevelRollup {
	var out []LevelRollup
	at := map[works.LevelID]int{}
	eachItem(items, cs, func(it works.Item, its []Cell) {
		if it.LevelID.IsZero() {
			return
		}
		i, ok := at[it.LevelID]
		if !ok {
			i = len(out)
			at[it.LevelID] = i
			out = append(out, LevelRollup{Level: it.LevelID})
		}
		switch {
		case it.Counts():
			out[i].add(its)
		case !it.Live() && slices.ContainsFunc(its, func(c Cell) bool { return c.File != nil }):
			for _, c := range its {
				out[i].Withdrawn[c.State]++
			}
		}
	})
	return out
}

func eachItem(items []works.Item, cs []Cell, f func(works.Item, []Cell)) {
	i := 0
	for _, it := range items {
		j := i
		for j < len(cs) && cs[j].Item.ID == it.ID {
			j++
		}
		f(it, cs[i:j])
		i = j
	}
}

func available(items []works.Item, timing map[works.ItemID]targets.Timing, today time.Time) release.Fact[works.Date] {
	read := works.Item.Live
	if slices.ContainsFunc(items, works.Item.Counts) {
		read = works.Item.Counts
	}
	var next, last release.Fact[works.Date]
	var nextFrom, lastFrom time.Time
	for _, it := range items {
		if !read(it) {
			continue
		}
		t := timing[it.ID]
		from, known := t.From()
		switch {
		case !known:
		case !t.ReleasedBy(today):
			if nextFrom.IsZero() || from.Before(nextFrom) {
				next, nextFrom = t.Released, from
			}
		case lastFrom.IsZero() || from.After(lastFrom):
			last, lastFrom = t.Released, from
		}
	}
	if !nextFrom.IsZero() {
		return next
	}
	return last
}
