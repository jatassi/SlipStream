package targets

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// Goals is what one Work wants: its Profile, that Profile's Targets in
// Profile order, and its effective Edition list, which is its own when
// Own.
type Goals struct {
	Profile  Profile
	Targets  []Target
	Editions Editions
	Own      bool
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
	gs, err := GoalsOf(ctx, r, works.Only([]works.WorkID{work}))
	return gs[work], err
}

// GoalsOf reads the Goals of each of ws in r, reading every Target and
// Profile once for all of them: works.ErrNoWork when one of ws, listed,
// has no wants row.
func GoalsOf(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]Goals, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.TargetsWant, error) { return q.ListEveryWants(ctx) }, func(ids []works.WorkID) ([]db.TargetsWant, error) {
		rows, err := q.ListWantsOf(ctx, ids)
		if err == nil && len(rows) < len(ids) {
			return nil, works.ErrNoWork
		}
		return rows, err
	})
	if err != nil {
		return nil, err
	}
	cuts, err := works.Read(ws, func() ([]db.TargetsWantsCut, error) { return q.ListEveryWantsCut(ctx) }, func(ids []works.WorkID) ([]db.TargetsWantsCut, error) {
		return q.ListWantsCutsOf(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	named := map[works.WorkID][]release.CutKey{}
	for _, c := range cuts {
		named[c.WorkID] = append(named[c.WorkID], c.Cut)
	}
	profiles, err := listProfiles(ctx, q)
	if err != nil {
		return nil, err
	}
	all, err := listTargets(ctx, q)
	if err != nil {
		return nil, err
	}
	profileOf := make(map[ProfileID]Profile, len(profiles))
	for _, p := range profiles {
		profileOf[p.ID] = p
	}
	targetOf := make(map[TargetID]Target, len(all))
	for _, t := range all {
		targetOf[t.ID] = t
	}
	wanted := make(map[ProfileID][]Target, len(profiles))
	for _, p := range profiles {
		for _, e := range p.Entries {
			wanted[p.ID] = append(wanted[p.ID], targetOf[e.Target])
		}
	}
	out := make(map[works.WorkID]Goals, len(rows))
	for _, row := range rows {
		g := Goals{Profile: profileOf[row.ProfileID], Targets: wanted[row.ProfileID]}
		g.Editions, g.Own = g.Profile.Editions, row.Editions != nil
		if g.Own {
			g.Editions = editionsOf(*row.Editions, named[row.WorkID])
		}
		out[row.WorkID] = g
	}
	return out, nil
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
func (t *Targets) TimingsIn(ctx context.Context, r storage.Reader[file], typ works.TypeID, ws works.Among[works.WorkID]) (map[works.ItemID]Timing, error) {
	facet, ok := t.cfg.Types[typ]
	if !ok {
		return nil, ErrNoType
	}
	return facet.Timings(ctx, r, ws)
}

// Timings answers each Item of work, a Work of Content Type typ.
func (t *Targets) Timings(ctx context.Context, _ identity.Admin, typ works.TypeID, work works.WorkID) (map[works.ItemID]Timing, error) {
	var out map[works.ItemID]Timing
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = t.TimingsIn(ctx, r, typ, works.Only([]works.WorkID{work}))
		return err
	})
	return out, err
}

// NameOfFirst is the name of the first of ids a Target still has in r,
// empty when none does.
func NameOfFirst(ctx context.Context, r storage.Reader[file], ids []TargetID) (string, error) {
	q := db.New(r)
	for _, id := range ids {
		t, err := q.GetTarget(ctx, id)
		switch {
		case err == nil:
			return t.Name, nil
		case !errors.Is(err, storage.ErrNoRows):
			return "", err
		}
	}
	return "", nil
}
