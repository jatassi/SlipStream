package library

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
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

// Work reads Work id and what its page shows beside it from one snapshot,
// works.ErrNoWork when no Work has the id.
func (l *Library) Work(ctx context.Context, _ identity.Admin, id works.WorkID) (Work, error) {
	var out Work
	err := l.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		tree, err := l.cfg.Works.TreeIn(ctx, r, id)
		if err != nil {
			return err
		}
		goals, err := targets.GoalsIn(ctx, r, id)
		if err != nil {
			return err
		}
		timings, err := l.cfg.Targets.TimingsIn(ctx, r, tree.ContentType, works.Only([]works.WorkID{id}))
		if err != nil {
			return err
		}
		out = Work{Tree: tree, Links: l.cfg.Works.Links(tree), Profile: goals.Profile, Timings: timings}
		root, err := roots.OfWork(ctx, r, id)
		switch {
		case errors.Is(err, roots.ErrNoRoot):
		case err != nil:
			return err
		default:
			out.Root = &root
		}
		return nil
	})
	return out, err
}
