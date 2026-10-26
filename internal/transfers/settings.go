package transfers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
)

// maxSend is downloading.uploadLimit's maximum, which libtorrent keeps in
// an int, so one Node's two send caps share one range.
const maxSend = 2000 * 1024 * 1024

// SendLimit is a Node's cap on the bytes a second its Transfers send,
// across all of them.
var SendLimit = setting.Rate{Key: "transfers.sendLimit", Label: "Send limit", Max: maxSend, Zero: "No limit"}

// NodeOverrides is one Node's stored overrides, each nil at its default.
type NodeOverrides struct{ SendLimit *int64 }

// NodeChange is the per-Node settings to store, each nil to leave.
type NodeChange struct{ SendLimit *setting.Stored[int64] }

// NodeSettings is node's stored overrides.
func (t *Transfers) NodeSettings(ctx context.Context, node role.NodeID) (NodeOverrides, error) {
	return nodeOverridesOf(ctx, t.cfg.Library, node)
}

// SetNodeSettings stores ch as node's settings, which node applies from
// its next intent, and names node in the next transfers.changed.
func (t *Transfers) SetNodeSettings(ctx context.Context, node role.NodeID, ch NodeChange) (NodeOverrides, error) {
	var o NodeOverrides
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if o, err = nodeOverridesOf(ctx, tx, node); err != nil {
			return err
		}
		if ch.SendLimit != nil {
			o.SendLimit = ch.SendLimit.Override()
		}
		return ledger.New(tx).SetNodeSettings(ctx, ledger.SetNodeParams{NodeID: node, SendLimit: o.SendLimit})
	})
	if err != nil {
		return NodeOverrides{}, err
	}
	t.mu.Lock()
	t.limited = append(t.limited, node)
	t.mu.Unlock()
	t.Touch()
	return o, nil
}

// nodeOverridesOf reads node's overrides. A Node with no row has none.
func nodeOverridesOf(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (NodeOverrides, error) {
	row, err := ledger.New(r).GetNodeSettings(ctx, node)
	if errors.Is(err, storage.ErrNoRows) {
		return NodeOverrides{}, nil
	}
	return NodeOverrides{SendLimit: row.SendLimit}, err
}
