package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) cancelConversionGroup(context.Context, identity.Admin, *groupCancelInput) (*canceledOutput, error) {
	return nil, notyet.Op
}
