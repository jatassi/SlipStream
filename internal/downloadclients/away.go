package downloadclients

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Away is a client whose Node is in step and whose last poll failed: it
// isn't responding, refused the sign-in or needs attention. Health raises
// it, its card turns red and its unfinished Grabs wait on client_away.
func (r ClientReach) Away() bool {
	switch r {
	case ReachNotResponding, ReachSignInRefused, ReachNeedsAttention:
		return true
	case ReachResponding, ReachOff, ReachNodeOffline, ReachUnchecked:
	}
	return false
}

// StaleCause is why a Torrent's row is its last report.
type StaleCause string

const (
	// StaleNode is a Node out of step.
	StaleNode StaleCause = "node"
	// StaleClient is an external client that isn't responding or refused
	// the sign-in.
	StaleClient StaleCause = "client"
	// StaleOff is an external client the Admin turned off.
	StaleOff StaleCause = "off"
)

// staleOf is why a row of client on a Node in step or not is stale, ""
// while it is current. client is nil for the built-in client, and
// externals holds every client that isn't deleted.
func staleOf(inStep bool, client *ClientRef, externals []External) StaleCause {
	if !inStep {
		return StaleNode
	}
	if client == nil {
		return ""
	}
	i := slices.IndexFunc(externals, func(e External) bool { return e.ID == client.ID })
	if i < 0 {
		return ""
	}
	switch ReachOf(externals[i], true) {
	case ReachNotResponding, ReachSignInRefused:
		return StaleClient
	case ReachOff:
		return StaleOff
	case ReachResponding, ReachNeedsAttention, ReachNodeOffline, ReachUnchecked:
	}
	return ""
}

// AwayIn is which of ids an away client holds, for GrabCells'
// client_away.
func AwayIn(ctx context.Context, r storage.Reader[file], ids []ID, inStep func(role.NodeID) bool) (map[ID]bool, error) {
	rows, err := db.New(r).ListClientsOfTorrents(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := map[ID]bool{}
	for _, row := range rows {
		e, err := externalOf(row.DownloadclientsClient, nil)
		if err != nil {
			return nil, err
		}
		if ReachOf(e, inStep(e.Node)).Away() {
			out[row.TorrentID] = true
		}
	}
	return out, nil
}
