package targets

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// Goals is what one Work wants: its Profile and that Profile's Targets in
// Profile order.
type Goals struct {
	Profile Profile
	Targets []Target
}

// Assign gives w its wants row in tx: Profile profile, or its Content
// Type's default when profile is nil. A Profile of another Content Type is
// ErrProfileWrongType.
func Assign(ctx context.Context, tx *storage.Tx[file], w works.Summary, profile *ProfileID) error {
	q := db.New(tx)
	var p db.TargetsProfile
	var err error
	if profile == nil {
		p, err = q.DefaultProfile(ctx, w.ContentType)
	} else {
		p, err = q.GetProfile(ctx, *profile)
	}
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return ErrNoProfile
	case err != nil:
		return err
	case p.ContentType != w.ContentType:
		return ErrProfileWrongType
	}
	return q.InsertWants(ctx, db.InsertWantsParams{WorkID: w.ID, ProfileID: p.ID})
}

// GoalsIn reads work's Goals in r, works.ErrNoWork when no Work has id
// work.
func GoalsIn(ctx context.Context, r storage.Reader[file], work works.WorkID) (Goals, error) {
	q := db.New(r)
	row, err := q.GetWants(ctx, work)
	if errors.Is(err, storage.ErrNoRows) {
		return Goals{}, works.ErrNoWork
	}
	if err != nil {
		return Goals{}, err
	}
	profiles, err := listProfiles(ctx, q)
	if err != nil {
		return Goals{}, err
	}
	all, err := listTargets(ctx, q)
	if err != nil {
		return Goals{}, err
	}
	g := Goals{Profile: profiles[slices.IndexFunc(profiles, func(p Profile) bool { return p.ID == row.ProfileID })]}
	for _, e := range g.Profile.Entries {
		g.Targets = append(g.Targets, all[slices.IndexFunc(all, func(t Target) bool { return t.ID == e.Target })])
	}
	return g, nil
}

// Goals reads work's Goals, works.ErrNoWork when no Work has id work.
func (t *Targets) Goals(ctx context.Context, _ identity.Admin, work works.WorkID) (Goals, error) {
	var g Goals
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		g, err = GoalsIn(ctx, r, work)
		return err
	})
	return g, err
}

// WorksOf lists the Works wanting from Profile id in r.
func WorksOf(ctx context.Context, r storage.Reader[file], id ProfileID) ([]works.WorkID, error) {
	return db.New(r).ListWorksOfProfile(ctx, id)
}

// TimingsIn answers each Item of ws, Works of Content Type typ, from its
// facet in one call.
func (t *Targets) TimingsIn(ctx context.Context, r storage.Reader[file], typ works.TypeID, ws []works.WorkID) (map[works.ItemID]Timing, error) {
	facet, ok := t.cfg.Types[typ]
	if !ok {
		return nil, ErrNoType
	}
	return facet.Timings(ctx, r, ws)
}
