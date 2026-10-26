package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Monitor turns every Item of each of ws on or off.
func (l *Library) Monitor(context.Context, identity.Admin, []works.WorkID, bool) ([]Outcome, error) {
	return nil, ErrNotBuilt
}

// SetProfile gives each of ws the Profile, through one Impact.
func (l *Library) SetProfile(context.Context, identity.Admin, []works.WorkID, targets.ProfileID, files.Mode) (files.Impact, []Outcome, error) {
	return files.Impact{}, nil, ErrNotBuilt
}

// SetRoot homes each of ws in the root.
func (l *Library) SetRoot(context.Context, identity.Admin, []works.WorkID, roots.ID) ([]Outcome, error) {
	return nil, ErrNotBuilt
}

// Refresh refreshes each of ws from its metadata source.
func (l *Library) Refresh(context.Context, identity.Admin, []works.WorkID) ([]Outcome, error) {
	return nil, ErrNotBuilt
}
