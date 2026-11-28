package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) updateNodeConverting(context.Context, identity.Admin, *updateNodeInput) (*nodeOutput, error) {
	return nil, notyet.Op
}
