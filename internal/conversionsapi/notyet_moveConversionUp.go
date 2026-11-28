package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) moveConversionUp(context.Context, identity.Admin, *conversionInput) (*rowOutput, error) {
	return nil, notyet.Op
}
