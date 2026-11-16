package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// SearchWork queues each search group of work within scope that has a
// wanted cell, gone and stopped cells included, in the now lane, and
// answers each group: queued, or merged into a row already queued or
// searching with its scope. Searching a cell clears its failure count, so
// Search on a Failed cell is Search again. An id scope names that work
// doesn't have is a *works.NotInWorkError.
func (gr *Grabs) SearchWork(ctx context.Context, _ identity.Admin, work works.WorkID, scope Scope) ([]Queued, error) {
	interval, err := gr.interval(ctx)
	if err != nil {
		return nil, err
	}
	var out []Queued
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.fence(ctx, tx); err != nil {
			return err
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		items, err := scope.items(tree)
		if err != nil {
			return err
		}
		root, err := gr.searchable(ctx, tx, tree)
		if err != nil {
			return err
		}
		if err := gr.cfg.Roots.Reachable(ctx, tx, root); err != nil {
			return err
		}
		cells, err := gr.cfg.Files.Cells(ctx, tx, work)
		if err != nil {
			return err
		}
		ids := make([]works.ItemID, len(items))
		for i, it := range items {
			ids[i] = it.ID
		}
		pressed := slices.DeleteFunc(within(cells, ids), func(c files.Cell) bool { return !c.Wanted() })
		if len(pressed) == 0 {
			return ErrNothingWanted
		}
		if err := uncount(ctx, tx, pressed); err != nil {
			return err
		}
		out, err = gr.enqueue(ctx, tx, work, tree.ContentType, pressed, landing(searches.Search, false), interval)
		return err
	})
	if err != nil {
		return nil, err
	}
	gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: []string{work.String()}})
	gr.wake()
	return out, nil
}
