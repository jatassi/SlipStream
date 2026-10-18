package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// settleWriteOffs gives up the Downloading work of each Node the Admin
// wrote it off on: its unimported Grabs fail with node_written_off, so
// their Items are wanted again, then every one of its torrents leaves its
// intent, files kept. It reads what is left each time, so a pass a crash
// cut short finishes on the next, and a settled Node has nothing left.
// A fetching Grab is not the Node's yet: the fetch routes it afresh. It
// reports whether it changed anything.
func (gr *Grabs) settleWriteOffs(ctx context.Context, open []Grab) (bool, error) {
	var holding []role.NodeID
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		holding, err = db.New(r).ListDownloadingNodes(ctx)
		return err
	})
	if err != nil {
		return false, err
	}
	settled := false
	for _, node := range holding {
		if !gr.cfg.Roster.WrittenOff(role.Downloading, node) {
			continue
		}
		for _, g := range open {
			if g.Node != node || g.Stage == Fetching {
				continue
			}
			if err := gr.commit(ctx, g, fail{NodeWrittenOff, "the Admin wrote off what this Grab's Node downloads"}); err != nil {
				return settled, err
			}
			settled = true
		}
		var dropped int64
		err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			var err error
			dropped, err = db.New(tx).DropSeeding(ctx, db.DropSeedingParams{NodeID: node, SeedDroppedAt: storage.Now()})
			return err
		})
		if err != nil {
			return settled, err
		}
		if dropped > 0 {
			gr.cfg.Downloads.Touch(node)
			settled = true
		}
	}
	return settled, nil
}
