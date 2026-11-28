package conversions

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// putBack queues every Conversion placed on node again, in tx: its
// Role is gone, or it lapsed or was written off. checking and landing
// stay, because their bytes are in the root.
func (c *Conversions) putBack(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID) error {
	rows, err := db.New(tx).ListPlacedOn(ctx, node)
	if err != nil {
		return fmt.Errorf("read the Conversions on the Node: %w", err)
	}
	for _, r := range rows {
		if err := c.advance(ctx, tx, r.ID, Event{On: PutBack}); err != nil {
			return err
		}
	}
	return nil
}

// unpin moves every retry pinned to node on to another Node, in tx.
func (c *Conversions) unpin(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID) error {
	ids, err := db.New(tx).ListPinnedTo(ctx, node)
	if err != nil {
		return fmt.Errorf("read the retries pinned to the Node: %w", err)
	}
	for _, id := range ids {
		if err := c.advance(ctx, tx, id, Event{On: Unpin, Node: node}); err != nil {
			return err
		}
	}
	return nil
}
