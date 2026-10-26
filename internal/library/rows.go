package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

// Scope is every Work, one Content Type's, or the named ones.
type Scope struct {
	Type  *works.TypeID
	Works []works.WorkID
}

// Row is one Work as the Library lists it.
type Row struct{}

// Rows answers scope from one snapshot, by sort label then id.
func (l *Library) Rows(context.Context, identity.Admin, Scope) ([]Row, error) {
	return nil, ErrNotBuilt
}
