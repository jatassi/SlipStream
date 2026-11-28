package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) getConversionGroup(context.Context, identity.Admin, *groupInput) (*groupOutput, error) {
	return nil, notyet.Op
}
