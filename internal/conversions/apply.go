package conversions

import (
	"context"
	"errors"

	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// apply applies one of from's reports, in tx. It moves the Conversion
// only while it is placed on from at the report's attempt. A report past
// its caps, or for a Conversion that is gone, changes nothing.
func (c *Conversions) apply(ctx context.Context, tx *storage.Tx[storage.Library], from role.NodeID, r *cv1.Result) error {
	ev, err := eventOf(r)
	if err != nil {
		c.cfg.Log.Warn("refused a Converting report", "node", from, "err", err)
		return nil
	}
	ev.Node, ev.Attempt, ev.Fenced = from, int(r.GetAttempt()), true
	if err := c.advance(ctx, tx, ev.Conversion, ev); !errors.Is(err, ErrNoConversion) {
		return err
	}
	return nil
}
