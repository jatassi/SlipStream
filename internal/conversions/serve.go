package conversions

import (
	"context"

	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Derive is node's Converting intent: its ready, running and paused
// Conversions, whether a Media Server there transcodes video, and the
// scratch names to keep.
func (c *Conversions) Derive(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (*cv1.Intent, error) {
	return c.derive(ctx, r, node)
}

// Apply applies one of from's results, fenced by its Node and attempt,
// and wakes the reconciler to tell Transfers, imports and the tab once it
// commits.
func (c *Conversions) Apply(ctx context.Context, tx *storage.Tx[storage.Library], from role.NodeID, r *cv1.Result) error {
	if err := c.apply(ctx, tx, from, r); err != nil {
		return err
	}
	if u, err := storage.UUIDFromBytes(r.GetConversionId()); err == nil {
		c.noteMoved(ID{UUID: u})
	}
	c.cfg.Poke.Wake()
	return nil
}

// noteMoved has the next pass tell the holders, Transfers, imports and
// the tab of ids, which a Write outside the reconciler moved.
func (c *Conversions) noteMoved(ids ...ID) {
	c.rec.mu.Lock()
	c.rec.applied = append(c.rec.applied, ids...)
	c.rec.mu.Unlock()
}
