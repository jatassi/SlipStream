package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/role"
)

// settleWriteOffs fails each unimported Grab on a Node the Admin wrote
// Downloading off with node_written_off, so its Items are wanted again,
// and cancels its Transfer in the same Write; downloadclients parts that
// Node's Torrents. A fetching Grab is not the Node's yet: the fetch routes
// it afresh. It reports whether it changed anything.
func (gr *Grabs) settleWriteOffs(ctx context.Context, open []Grab) (bool, error) {
	settled := false
	for _, g := range open {
		if g.Stage == Fetching || !gr.cfg.Roster.WrittenOff(role.Downloading, g.Node) {
			continue
		}
		if err := gr.commit(ctx, g, fail{NodeWrittenOff, writtenOffWords}); err != nil {
			return settled, err
		}
		settled = true
	}
	return settled, nil
}
