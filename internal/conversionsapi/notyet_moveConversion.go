package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) moveConversion(context.Context, identity.Admin, *moveInput) (*rowOutput, error) {
	return nil, notyet.Op
}
