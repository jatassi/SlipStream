package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Adding is an add's choices. A nil choice takes its Content Type's
// default. Remember makes each given choice the default in the add's
// Write.
type Adding struct {
	Type     works.TypeID
	Key      works.Key
	Profile  *targets.ProfileID
	Root     *roots.ID
	Monitor  *bool
	Remember bool
}

// Added is the Work an add made or found. Created is false when it was
// already in the library.
type Added struct {
	Work    Work
	Created bool
}

func (l *Library) Add(context.Context, identity.Admin, Adding) (Added, error) {
	return Added{}, ErrNotBuilt
}
