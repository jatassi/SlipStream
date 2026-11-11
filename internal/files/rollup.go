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

// Rollup is what one Work's cells and files add up to. The Library's row
// and the Work page read the same one.
type Rollup struct {
	// Status is the first State any of the Work's counted Items holds, in
	// the glossary's order, zero while none holds a cell.
	Status State
	// Cells counts the cells of the Work's counted Items by State: each
	// Edition it wants at each Target of each. A listed cut not seen for
	// an Item makes no cell, so it never counts.
	Cells Counts
	// Items counts the Work's counted Items.
	Items int
	// ByItem counts counted Items by status. An Item with no cell isn't
	// counted.
	ByItem Counts
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
	for _, it := range w.tree.Items {
		if it.Counts() {
			out.Items++
		}
	}
	of := map[works.ItemID]State{}
	for _, c := range cs {
		if !c.Item.Counts() {
			continue
		}
		out.Cells[c.State]++
		if s, ok := of[c.Item.ID]; !ok || c.State < s {
			of[c.Item.ID] = c.State
		}
	}
	for _, s := range of {
		out.ByItem[s]++
		if out.Status == 0 || s < out.Status {
			out.Status = s
		}
	}
	var held Held
	for _, file := range w.files {
		if !file.Gone() {
			held.add(file)
		}
	}
	out.Files, out.Kept, out.Bytes = held.Files, held.Kept, held.Bytes
	return out
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
