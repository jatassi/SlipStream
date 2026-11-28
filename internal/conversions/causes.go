package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/targets"
)

// Causes is the open causes of each of ts.
func (c *Conversions) Causes(context.Context, []targets.TargetID) (map[targets.TargetID][]Cause, error) {
	return nil, errNotBuilt
}
