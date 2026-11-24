package reachability

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Step is first setup's Remote access step, done only by Continue. It shows
// while something is set up for use away from home or there is a suggestion
// to accept, so a home-only install never sees it. A UI and portal whose
// Address outsiders can't use is no suggestion here: the Address step sets
// it.
func (r *Reachability) Step() firstrun.Step {
	return firstrun.Step{
		ID:     firstrun.RemoteAccessID,
		Shows:  r.stepShows,
		Events: []live.Event{identity.AddressChangedEvent, mediaservers.ChangedEvent, ChangedEvent},
	}
}

func (r *Reachability) stepShows(ctx context.Context, rd storage.Reader[storage.Library]) (bool, error) {
	wanted, err := wantedIn(ctx, rd)
	if err != nil || len(wanted) > 0 {
		return len(wanted) > 0, err
	}
	addr, err := identity.AddressIn(ctx, rd)
	if err != nil {
		return false, err
	}
	dismissed, err := db.New(rd).ListDismissals(ctx)
	if err != nil {
		return false, err
	}
	recs, err := mediaservers.ServersIn(ctx, rd)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(r.suggestions(ctx, nil, dismissed, recs, addr), func(s Suggestion) bool { return !s.AddressPrivate }), nil
}
