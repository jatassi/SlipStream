package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

const restoredWords = "No Node had this Grab's torrent after the restore."

// reconcilingWait is a fetch's wait while a restored Controller is Reconciling:
// the Grab takes an adopted torrent or fails once it ends.
func (gr *Grabs) reconcilingWait(ctx context.Context) outcome {
	var on bool
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		on, err = gr.cfg.Reconciling(ctx, r)
		return err
	})
	switch {
	case err != nil:
		return local(ctx, err)
	case on:
		return wait{cause: ErrReconciling, rerouted: gr.routes.watch()}
	}
	return nil
}

// DecideFetching is the Write that ends Reconciling's part: each Grab at
// fetching whose Release's torrent a Node has goes on, and its fetch takes
// that torrent. Every other one fails restored.
func (gr *Grabs) DecideFetching(ctx context.Context, tx *storage.Tx[file]) error {
	rows, err := db.New(tx).ListOpenGrabs(ctx)
	if err != nil {
		return err
	}
	open, err := loadAll(ctx, tx, rows)
	if err != nil {
		return err
	}
	for _, g := range open {
		if g.Stage != Fetching {
			continue
		}
		if g.Release.InfoHash != nil {
			held, err := downloadclients.LiveOfHash(ctx, tx, *g.Release.InfoHash)
			if err != nil {
				return err
			}
			if held {
				continue
			}
		}
		o := fail{Restored, restoredWords}
		if _, err := gr.move(ctx, tx, g, o); err != nil {
			return err
		}
		if _, _, err := gr.left(ctx, tx, g, o, nil); err != nil {
			return err
		}
		gr.restored = append(gr.restored, g)
	}
	return nil
}

// Reconciled runs once the Write that ended Reconciling committed: each
// Grab it failed is told, and every waiting fetch and dispatch runs.
func (gr *Grabs) Reconciled(ctx context.Context) {
	for _, g := range gr.restored {
		gr.entered(ctx, g, Failed)
	}
	gr.restored = nil
	gr.Rerouted()
	gr.wake()
}
