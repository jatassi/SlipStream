package downloadclients

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// settleWriteOffs parts at once, files kept, every Torrent of each Node
// the Admin wrote Downloading off: each leaves its intent, and no removal
// waits on a Node that won't confirm it. A settled Node holds nothing, so
// a second pass parts nothing.
func (c *Clients) settleWriteOffs(ctx context.Context) {
	var holding []role.NodeID
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		holding, err = db.New(r).ListHoldingNodes(ctx)
		return err
	})
	if err != nil {
		c.logFailure(ctx, "read the Nodes holding Torrents", err)
		return
	}
	written := slices.DeleteFunc(holding, func(node role.NodeID) bool { return !c.cfg.Roster.WrittenOff(role.Downloading, node) })
	if len(written) == 0 {
		return
	}
	var parted []ID
	err = c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		parted = nil
		now := storage.Now()
		for _, node := range written {
			ids, err := db.New(tx).WriteOff(ctx, db.WriteOffParams{At: now, NodeID: node})
			if err != nil {
				return err
			}
			parted = append(parted, ids...)
		}
		return nil
	})
	if err != nil {
		c.logFailure(ctx, "part the Torrents of written-off Nodes", err)
		return
	}
	c.committed(written, parted)
}

func (c *Clients) logFailure(ctx context.Context, what string, err error) {
	if err != nil && ctx.Err() == nil {
		c.cfg.Log.Error(what, "err", err.Error())
	}
}
