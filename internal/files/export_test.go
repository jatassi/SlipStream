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
