package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Enqueue lands t on every wanted cell of work in its own Write, as a
// call for one Work does, and wakes the reconciler.
func (gr *Grabs) Enqueue(ctx context.Context, work works.WorkID, t searches.Trigger) error {
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		cells, err := gr.cfg.Files.Cells(ctx, tx, work)
		if err != nil {
			return err
		}
		wanted := slices.DeleteFunc(cells, func(c files.Cell) bool { return !c.Wanted() })
		_, err = gr.enqueue(ctx, tx, tree, wanted, landing(t, false))
		return err
	})
	gr.wake()
	return err
}
