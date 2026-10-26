package files

import (
	"context"
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
	// Status is the first State any of the Work's Items holds, in the
	// glossary's order, zero for a Work with no cell.
	Status State
	// Cells counts the cells the Work wants by State: each Edition it
	// wants at each Target of each Item. A listed cut not seen for an Item
	// makes no cell, so it never counts.
	Cells Counts
	// Failed counts the wanted cells whose download failed, which the
	// Library's Failed chip shows.
	Failed int
	Items  int
	// ByItem counts Items by status. An Item with no cell isn't counted.
	ByItem Counts
	Needs  []Need
	// Files, Kept and Bytes count the Work's library files, Kept files
	// included.
	Files, Kept int
	Bytes       int64
	// Available is the one Item's availability date, or for many Items the
	// earliest not yet out, else the latest out. It is unknown while no
	// Item has a date.
	Available release.Fact[works.Date]
	Profile   targets.ProfileID
	// Root is zero while the Work has none.
	Root roots.ID
}

// Rollups reads the Rollup of each of ws, the Works among names, in r,
// loading each kind of row once for them all.
func (f *Files) Rollups(ctx context.Context, r storage.Reader[file], ws []works.Bare, among works.Among[works.WorkID]) (map[works.WorkID]Rollup, error) {
	worlds, err := f.worlds(ctx, r, ws, among)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]Rollup, len(worlds))
	var cs []Cell
	for _, w := range worlds {
		cs = appendCells(cs[:0], w)
		out[w.tree.ID] = rollup(w, cs)
	}
	return out, nil
}

func rollup(w world, cs []Cell) Rollup {
	out := Rollup{
		Items: len(w.tree.Items), Needs: needs(w.tree.Items, cs, w.grabs, !w.root.IsZero()),
		Available: available(w.tree.Items, w.timing, w.today), Profile: w.goals.Profile.ID, Root: w.root,
	}
	of := map[works.ItemID]State{}
	for _, c := range cs {
		out.Cells[c.State]++
		if c.State == Failed && c.Wanted() {
			out.Failed++
		}
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
	for _, file := range w.files {
		out.Files++
		out.Bytes += file.Size
		if file.Kept() {
			out.Kept++
		}
	}
	return out
}

func available(items []works.Item, timing map[works.ItemID]targets.Timing, today time.Time) release.Fact[works.Date] {
	var next, last release.Fact[works.Date]
	var nextFrom, lastFrom time.Time
	for _, it := range items {
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
