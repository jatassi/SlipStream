package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

// Search answers every Work whose titles match q, best first, and the rows
// of the first limit of them. A nil t searches every Content Type.
func (l *Library) Search(context.Context, identity.Admin, string, int, *works.TypeID) ([]works.WorkID, []Row, error) {
	return nil, nil, ErrNotBuilt
}
