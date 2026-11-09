package files

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ScoresFromScratch is the score of each cell of work that refill fills
// in r when no file fills any cell yet.
func (f *Files) ScoresFromScratch(ctx context.Context, r storage.Reader[file], work works.WorkID) (map[CellKey]int, error) {
	w, err := f.load(ctx, r, work)
	if err != nil {
		return nil, err
	}
	w.fills = nil
	out := map[CellKey]int{}
	for key, fill := range refill(w) {
		out[key] = fill.Score
	}
	return out, nil
}

// Plans is the plan of each of ids's Works, every Work of the library,
// before any Node is asked.
func (f *Files) Plans(ctx context.Context, ids ...works.WorkID) ([]WorkPlan, error) {
	var out []WorkPlan
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		trees := make([]works.Bare, len(ids))
		for i, id := range ids {
			var err error
			if trees[i], err = works.BareIn(ctx, r, id); err != nil {
				return err
			}
		}
		var err error
		out, err = f.plans(ctx, r, trees, works.Every(ids))
		return err
	})
	return out, err
}
