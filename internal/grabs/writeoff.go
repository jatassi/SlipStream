package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// settleWriteOffs gives up the Downloading work of each Node the Admin
// wrote it off on: its unimported Grabs fail with node_written_off, so
// their Items are wanted again, then every one of its Torrents is parted,
// files kept. It reads what is left each time, so a pass a crash cut
// short finishes on the next, and a settled Node has nothing left. A
// fetching Grab is not the Node's yet: the fetch routes it afresh. It
// reports whether it changed anything.
func (gr *Grabs) settleWriteOffs(ctx context.Context, open []Grab) (bool, error) {
	var holding []role.NodeID
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		holding, err = downloadclients.Holding(ctx, r)
		return err
	})
	if err != nil {
		return false, err
	}
	for _, g := range open {
		if g.Stage != Fetching && !slices.Contains(holding, g.Node) {
			holding = append(holding, g.Node)
		}
	}
	settled := false
	for _, node := range holding {
		if !gr.cfg.Roster.WrittenOff(role.Downloading, node) {
			continue
		}
		changed, err := gr.settleWriteOff(ctx, node, open)
		settled = settled || changed
		if err != nil {
			return settled, err
		}
	}
	return settled, nil
}

func (gr *Grabs) settleWriteOff(ctx context.Context, node role.NodeID, open []Grab) (bool, error) {
	settled := false
	for _, g := range open {
		if g.Node != node || g.Stage == Fetching {
			continue
		}
		if err := gr.commit(ctx, g, fail{NodeWrittenOff, writtenOffWords}); err != nil {
			return settled, err
		}
		settled = true
	}
	var parted int64
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		parted, err = downloadclients.WriteOff(ctx, tx, node, storage.Now())
		return err
	})
	if err != nil {
		return settled, err
	}
	if parted > 0 {
		gr.cfg.Downloads.Touch(node)
	}
	return settled || parted > 0, nil
}
