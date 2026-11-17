package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// WorkStates is a Work's levels and each of its Items with its state on
// the Controller's calendar, in tree order: withdrawn and unmonitored
// Items included.
type WorkStates struct {
	Levels []works.Level
	Items  []ItemState
}

// ItemStates reads them for each of ws a Work has, in r, loading worlds
// once for them all as ItemFacts does. The caller filters.
func (f *Files) ItemStates(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]WorkStates, error) {
	trees, err := works.OutlinesIn(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	bares := make([]works.Bare, len(trees))
	ids := make([]works.WorkID, len(trees))
	levels := make(map[works.WorkID][]works.Level, len(trees))
	for i, t := range trees {
		bares[i], ids[i] = works.BareOf(t), t.ID
		levels[t.ID] = t.Levels
	}
	worlds, err := f.worlds(ctx, r, bares, works.Like(ws, ids), briefFiles)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]WorkStates, len(worlds))
	var cs []Cell
	for _, w := range worlds {
		cs = appendCells(cs[:0], w)
		items := make([]ItemState, 0, len(w.tree.Items))
		eachItem(w.tree.Items, cs, func(it works.Item, its []Cell) {
			s, _ := Status(its)
			items = append(items, ItemState{Item: it, Status: s, Released: w.timing[it.ID].ReleasedBy(w.today)})
		})
		out[w.tree.ID] = WorkStates{Levels: levels[w.tree.ID], Items: items}
	}
	return out, nil
}
