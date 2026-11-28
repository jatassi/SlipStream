package files

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Reason is why Wanted lists a Work, in the order a row names them.
type Reason uint8

const (
	// ReasonFailed is a stopped cell, or one a failed Grab left for the
	// Admin.
	ReasonFailed Reason = iota + 1
	ReasonMissing
	ReasonUpgrade
	ReasonGone
)

var reasonNames = [...]string{ReasonFailed: "failed", ReasonMissing: "missing", ReasonUpgrade: "upgrade", ReasonGone: "gone"}

func (r Reason) String() string { return reasonNames[r] }

// Reasons names every Reason, in order.
func Reasons() []string { return slices.Clone(reasonNames[ReasonFailed:]) }

// reasonOf is the Reason c is wanted for, false when it isn't.
func reasonOf(c Cell) (Reason, bool) {
	switch {
	case !c.Wanted():
		return 0, false
	case c.Gone:
		return ReasonGone, true
	case c.State == Failed || c.Search.Stopped:
		return ReasonFailed, true
	case c.State == Missing || c.State == Converting && c.File == nil:
		return ReasonMissing, true
	}
	return ReasonUpgrade, true
}

// ReasonOf is the first Reason of cells, a Work's in tree order, and how
// many Items each other Reason is the first of, false when no cell is
// wanted. Each Item counts once, under its own first Reason.
func ReasonOf(cells []Cell) (Reason, map[Reason]int, bool) {
	items := map[Reason]int{}
	for len(cells) > 0 {
		n := 1 + slices.IndexFunc(cells[1:], func(c Cell) bool { return c.Item.ID != cells[0].Item.ID })
		if n == 0 {
			n = len(cells)
		}
		var first Reason
		for _, c := range cells[:n] {
			if r, ok := reasonOf(c); ok && (first == 0 || r < first) {
				first = r
			}
		}
		if first != 0 {
			items[first]++
		}
		cells = cells[n:]
	}
	if len(items) == 0 {
		return 0, nil, false
	}
	lead := ReasonGone
	for r := range items {
		lead = min(lead, r)
	}
	delete(items, lead)
	return lead, items, true
}

// WorkWanted is what one Work wants: its wanted cells, gone and stopped
// ones included, in tree and rank order, and its counted, released Items
// whose Edition list names a cut not yet seen for them.
type WorkWanted struct {
	Work   works.WorkID
	Type   works.TypeID
	Title  string
	Cells  []Cell
	Unseen []works.ItemID
}

// Wanted is the WorkWanted of each Work among that wants anything, in r,
// loading each kind of row once for them all, as Rollups does.
func (f *Files) Wanted(ctx context.Context, r storage.Reader[file], among works.Among[works.WorkID]) ([]WorkWanted, error) {
	trees, err := works.OutlinesIn(ctx, r, among)
	if err != nil {
		return nil, err
	}
	bares := make([]works.Bare, len(trees))
	ids := make([]works.WorkID, len(trees))
	for i, t := range trees {
		bares[i], ids[i] = works.BareOf(t), t.ID
	}
	worlds, err := f.worlds(ctx, r, bares, works.Like(among, ids), briefFiles)
	if err != nil {
		return nil, err
	}
	var out []WorkWanted
	var cs []Cell
	for _, w := range worlds {
		ww := WorkWanted{Work: w.tree.ID, Type: w.tree.ContentType, Title: w.tree.Label}
		cs = appendCells(cs[:0], w)
		for _, c := range cs {
			if c.Wanted() {
				ww.Cells = append(ww.Cells, c)
			}
		}
		for _, it := range w.tree.Items {
			if it.Counts() && w.timing[it.ID].ReleasedBy(w.today) && unseen(w.goals.Editions, w.seen[it.ID]) {
				ww.Unseen = append(ww.Unseen, it.ID)
			}
		}
		if len(ww.Cells) > 0 || len(ww.Unseen) > 0 {
			out = append(out, ww)
		}
	}
	return out, nil
}

// unseen reports whether list names a cut besides Standard that seen
// lacks, which makes no cell until a Release of it is seen.
func unseen(list targets.Editions, seen []release.CutKey) bool {
	cuts := list.Cuts()
	return slices.Contains(cuts, release.Standard) && slices.ContainsFunc(cuts, func(c release.CutKey) bool {
		return c != release.Standard && !slices.Contains(seen, c)
	})
}
