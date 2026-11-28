package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// OutputsIn is the bytes unfinished Conversions on node will write to
// volume, for imports.OtherCopies.
func (c *Conversions) OutputsIn(context.Context, storage.Reader[storage.Library], role.NodeID, string) (int64, error) {
	return 0, errNotBuilt
}
