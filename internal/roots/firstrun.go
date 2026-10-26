package roots

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// RootsStep is first setup's library roots step, done once every enabled
// Content Type has a library root.
func (r *Roots) RootsStep() firstrun.Step {
	return firstrun.Step{
		ID: firstrun.RootsID,
		Done: func(ctx context.Context, rd storage.Reader[file]) (bool, error) {
			all, err := All(ctx, rd)
			if err != nil {
				return false, err
			}
			for _, t := range r.cfg.Types {
				on, err := works.Enabled(ctx, rd, t)
				if err != nil {
					return false, err
				}
				if on && !slices.ContainsFunc(all, func(root Root) bool { return root.ContentType == t }) {
					return false, nil
				}
			}
			return true, nil
		},
		Events: []live.Event{ChangedEvent, works.ContentTypesChangedEvent},
	}
}
