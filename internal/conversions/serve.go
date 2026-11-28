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
func (c *Conversions) Derive(context.Context, storage.Reader[storage.Library], role.NodeID) (*cv1.Intent, error) {
	return &cv1.Intent{}, nil
}

// Apply applies one of from's results, fenced by its Node and attempt.
func (c *Conversions) Apply(context.Context, *storage.Tx[storage.Library], role.NodeID, *cv1.Result) error {
	return errNotBuilt
}
