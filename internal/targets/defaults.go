package targets

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// StarterName is the starter Target's name, which keeps the walking
// skeleton's "- 1080p" file names.
const StarterName = "1080p"

// Open makes sure, in one idempotent Write, that a library that never had
// a starter Target gets one, that every registered Content Type has a
// default Profile, and that every Work has a wants row. It runs at claim
// and at every Controller start, and publishes only when it wrote.
func Open(ctx context.Context, cfg Config) (*Targets, error) {
	wrote := false
	err := cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		wrote = false
		q := db.New(tx)
		starter, made, err := ensureStarter(ctx, q)
		if err != nil {
			return err
		}
		defaults, named, err := ensureDefaults(ctx, q, cfg, starter)
		if err != nil {
			return err
		}
		assigned, err := ensureWants(ctx, tx, q, defaults)
		wrote = made || named || assigned
		return err
	})
	if err != nil {
		return nil, err
	}
	if wrote {
		cfg.Changed.Publish(ctx, Changed{})
	}
	return &Targets{cfg: cfg}, nil
}

// Starter is the starter Target: the 720p and 1080p qualities under the
// disc-or-resolution preset at their floors, final at the top tier.
func Starter() Spec {
	var chosen []release.Quality
	for _, q := range release.Qualities() {
		if r := q.Info().Resolution; r == release.R720p || r == release.R1080p {
			chosen = append(chosen, q)
		}
	}
	s := Layout(DiscOrResolution, chosen, true)
	s.Name = StarterName
	return s
}

// ensureStarter makes the starter Target unless the library recorded one,
// and answers the starter that still exists, nil once it is deleted.
func ensureStarter(ctx context.Context, q *db.Queries) (starter *TargetID, made bool, err error) {
	row, err := q.GetStarter(ctx)
	if err == nil {
		return row.TargetID, false, nil
	}
	if !errors.Is(err, storage.ErrNoRows) {
		return nil, false, err
	}
	t, err := NewTarget(Starter())
	if err != nil {
		return nil, false, err
	}
	t.ID, t.CreatedAt = TargetID{UUID: storage.NewUUID()}, storage.Now()
	if err := insertTarget(ctx, q, t); err != nil {
		return nil, false, err
	}
	return &t.ID, true, q.InsertStarter(ctx, &t.ID)
}

// ensureDefaults makes a default Profile for each registered Content Type
// lacking one, named after the type and holding the starter if it still
// exists, and answers every type's default.
func ensureDefaults(ctx context.Context, q *db.Queries, cfg Config, starter *TargetID) (out map[works.TypeID]ProfileID, made bool, err error) {
	out = map[works.TypeID]ProfileID{}
	for _, typ := range slices.Sorted(maps.Keys(cfg.Types)) {
		row, err := q.DefaultProfile(ctx, typ)
		if err == nil {
			out[typ] = row.ID
			continue
		}
		if !errors.Is(err, storage.ErrNoRows) {
			return nil, false, err
		}
		desc, ok := cfg.Works.Describe(typ)
		if !ok {
			return nil, false, fmt.Errorf("targets: Content Type %s has a facet and no description", typ)
		}
		name, err := freeProfileName(ctx, q, desc.Name)
		if err != nil {
			return nil, false, err
		}
		p := Profile{ID: ProfileID{UUID: storage.NewUUID()}, Name: name, ContentType: typ, Default: true}
		if starter != nil {
			p.Entries = []Entry{{Target: *starter, Sourcing: Download}}
		}
		if err := insertProfile(ctx, q, p); err != nil {
			return nil, false, err
		}
		out[typ], made = p.ID, true
	}
	return out, made, nil
}

// freeProfileName is name, or the first of "<name> 2", "<name> 3" and on
// that no Profile has.
func freeProfileName(ctx context.Context, q *db.Queries, name string) (string, error) {
	for n := 1; ; n++ {
		try := name
		if n > 1 {
			try = fmt.Sprintf("%s %d", name, n)
		}
		switch taken, err := q.ProfileNameTaken(ctx, try); {
		case err != nil:
			return "", err
		case !taken:
			return try, nil
		}
	}
}

// ensureWants gives every Work with no wants row its type's default.
func ensureWants(ctx context.Context, tx *storage.Tx[file], q *db.Queries, defaults map[works.TypeID]ProfileID) (bool, error) {
	all, err := works.Summaries(ctx, tx)
	if err != nil {
		return false, err
	}
	rows, err := q.ListWants(ctx)
	if err != nil {
		return false, err
	}
	has := make(map[works.WorkID]bool, len(rows))
	for _, row := range rows {
		has[row.WorkID] = true
	}
	made := false
	for _, w := range all {
		if has[w.ID] {
			continue
		}
		profile, ok := defaults[w.ContentType]
		if !ok {
			return false, fmt.Errorf("targets: Work %s is of Content Type %s, which isn't registered", w.ID, w.ContentType)
		}
		if err := q.InsertWants(ctx, db.InsertWantsParams{WorkID: w.ID, ProfileID: profile}); err != nil {
			return false, err
		}
		made = true
	}
	return made, nil
}
