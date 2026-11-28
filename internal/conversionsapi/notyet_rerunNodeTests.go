package conversionsapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/conversionsapi/notyet"
	"github.com/jatassi/slipstream/internal/identity"
)

func (handlers) rerunNodeTests(context.Context, identity.Admin, *nodeInput) (*struct{}, error) {
	return nil, notyet.Op
}
