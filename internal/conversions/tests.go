package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/role"
)

// Rerun deletes n's proofs, so its test runs run again.
func (c *Conversions) Rerun(context.Context, role.NodeID) error { return errNotBuilt }
