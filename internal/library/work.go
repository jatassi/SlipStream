package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Work is a Work's tree with what its page shows beside it. Root is nil
// while the Work is rootless.
type Work struct {
	works.Tree
	Links   []works.WebLink
	Profile targets.Profile
	Root    *roots.Root
	Timings map[works.ItemID]targets.Timing
}

func (l *Library) Work(context.Context, identity.Admin, works.WorkID) (Work, error) {
	return Work{}, ErrNotBuilt
}
