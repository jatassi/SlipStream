package grabs

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/role"
)

// settleGone fails each unimported Grab past fetching whose Node the Admin
// wrote Downloading off, or whose Torrent is Gone for a reason that fails
// it at any stage, by failOfGone, so its Items are wanted again, and
// cancels its Transfer in the same Write. None of these blocklists. A
// fetching Grab has no Torrent yet: the fetch routes it afresh. It
// reports whether it changed anything.
func (gr *Grabs) settleGone(ctx context.Context, open []Grab) (bool, error) {
	settled := false
	for _, g := range open {
		if g.Stage == Fetching {
			continue
		}
		gone, ok := g.goneAtAnyStage()
		if !ok && !gr.cfg.Roster.WrittenOff(role.Downloading, g.Node) {
			continue
		}
		if !ok {
			gone = downloadclients.Gone{Reason: downloadclients.NodeWrittenOff}
		}
		if err := gr.commit(ctx, g, failOfGone(gone)); err != nil {
			return settled, err
		}
		settled = true
	}
	return settled, nil
}

// failsAtAnyStage is the reasons that fail a Grab past downloading too: an
// Admin's remove is refused while a Grab imports, and these can't be.
var failsAtAnyStage = []downloadclients.RemoveReason{downloadclients.NodeWrittenOff, downloadclients.ByClient, downloadclients.ClientDeleted}

// goneAtAnyStage is g's Torrent's removal when its reason fails g at any
// stage.
func (g Grab) goneAtAnyStage() (downloadclients.Gone, bool) {
	if g.Torrent == nil {
		return downloadclients.Gone{}, false
	}
	gone, ok := g.Torrent.Outcome.(downloadclients.Gone)
	return gone, ok && slices.Contains(failsAtAnyStage, gone.Reason)
}
