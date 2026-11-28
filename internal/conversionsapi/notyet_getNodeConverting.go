package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) getNodeConverting(context.Context, identity.Admin, *nodeInput) (*nodeOutput, error) {
	return nil, notyet.Op
}
