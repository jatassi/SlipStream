package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// ItemFacts reads, for each of ws a Work has, the facts the monitoring
// presets read of each of its live Items, in r: Released on the
// Controller's calendar, and Filled once a library file fills any of its
// cells, Met or Upgradable. It loads timings, files and cells once for
// them all, as Rollups does.
func (f *Files) ItemFacts(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]works.Facts, error) {
	trees, err := works.OutlinesIn(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	bares := make([]works.Bare, len(trees))
	ids := make([]works.WorkID, len(trees))
	for i, t := range trees {
		bares[i], ids[i] = works.BareOf(t), t.ID
	}
	worlds, err := f.worlds(ctx, r, bares, works.Like(ws, ids), briefFiles)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]works.Facts, len(worlds))
	var cs []Cell
	for _, w := range worlds {
		facts := make(works.Facts, len(w.tree.Items))
		for _, it := range w.tree.Items {
			if it.Live() {
				facts[it.ID] = works.ItemFacts{Released: w.timing[it.ID].ReleasedBy(w.today)}
			}
		}
		cs = appendCells(cs[:0], w)
		for _, c := range cs {
			if fact, ok := facts[c.Item.ID]; ok && c.File != nil {
				fact.Filled = true
				facts[c.Item.ID] = fact
			}
		}
		out[w.tree.ID] = facts
	}
	return out, nil
}

// UnfilledFacts is the facts of Work t while it holds no file, as an
// add's preview does: each live Item's Released under timing, on the
// Controller's calendar, and none Filled.
func (f *Files) UnfilledFacts(t works.Tree, timing map[works.ItemID]targets.Timing) works.Facts {
	today := f.cfg.Today()
	out := make(works.Facts, len(t.Items))
	for _, it := range t.Items {
		if it.Live() {
			out[it.ID] = works.ItemFacts{Released: timing[it.ID].ReleasedBy(today)}
		}
	}
	return out
}
