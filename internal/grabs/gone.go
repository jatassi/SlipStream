package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// settleGone settles each Grab past fetching whose Node the Admin wrote
// Downloading off, or, once importing, whose Torrent is Gone. A downloading
// Grab fails by failOfGone, so its Items are wanted again. An importing
// Grab's Import ends, which fails it the same way with nothing placed, and
// lets a placing file finish. None of these blocklists. A fetching Grab
// has no Torrent yet: the fetch routes it afresh, and a downloading Grab's
// own step reads its Torrent gone. It reports whether it changed anything.
func (gr *Grabs) settleGone(ctx context.Context, open []Grab) (bool, error) {
	settled := false
	for _, g := range open {
		if g.Stage == Fetching {
			continue
		}
		gone, ok := g.Torrent.Outcome.(downloadclients.Gone)
		switch {
		case ok && g.Stage == Importing:
		case gr.cfg.Roster.WrittenOff(role.Downloading, g.Node):
			gone = downloadclients.Gone{Reason: downloadclients.NodeWrittenOff}
		default:
			continue
		}
		var err error
		if g.Stage == Importing {
			err = gr.endImport(ctx, g, GoneFailure(gone))
		} else {
			err = gr.commit(ctx, g, failOfGone(gone))
		}
		if err != nil {
			return settled, err
		}
		settled = true
	}
	return settled, nil
}

// endImport ends importing Grab g's Import for why, and publishes the
// Grab's move.
func (gr *Grabs) endImport(ctx context.Context, g Grab, why Failure) error {
	var now Grab
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.importer.End(ctx, tx, g.ID, why, false); err != nil {
			return err
		}
		var err error
		now, err = grabIn(ctx, tx, g.ID)
		return err
	})
	if err != nil {
		return err
	}
	if now.Stage != Importing {
		gr.entered(ctx, now, now.Stage)
	}
	return nil
}
