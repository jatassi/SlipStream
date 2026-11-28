package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) listConversions(context.Context, identity.Admin, *listInput) (*listOutput, error) {
	return nil, notyet.Op
}
