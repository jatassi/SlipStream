package targets

import (
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// Sourcing is how a Profile's Target gets its file. Conversion adds
// derive and download-then-derive.
type Sourcing = db.Sourcing

const Download = db.Download

// Entry is one Target of a Profile and how it is sourced.
type Entry struct {
	Target   TargetID
	Sourcing Sourcing
}

// Profile is the Targets a Work wants, in Profile order: the first a file
// fills names it. Each Content Type has one default Profile, which a Work
// added without one takes.
type Profile struct {
	ID          ProfileID
	Name        string
	ContentType works.TypeID
	Default     bool
	Entries     []Entry
	Editions    Editions
	// Works counts the Works wanting from it. Read only.
	Works int
}

// ProfileSpec is a Profile as an Admin writes it. It may hold no Target,
// and then its Works want nothing.
type ProfileSpec struct {
	Name        string
	ContentType works.TypeID
	Entries     []Entry
	Editions    Editions
}

// CreateProfile checks s and creates its Profile, which is never the
// default.
func (t *Targets) CreateProfile(ctx context.Context, _ identity.Admin, s ProfileSpec) (Profile, error) {
	if _, ok := t.cfg.Types[s.ContentType]; !ok {
		return Profile{}, ErrNoType
	}
	p := Profile{ID: ProfileID{UUID: storage.NewUUID()}, Name: s.Name, ContentType: s.ContentType, Entries: s.Entries, Editions: s.Editions}
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		known, err := t.CutsIn(ctx, tx)
		if err != nil {
			return err
		}
		if err := known.check(p.Editions, p.ContentType); err != nil {
			return err
		}
		return insertProfile(ctx, db.New(tx), p)
	})
	if err != nil {
		return Profile{}, err
	}
	t.cfg.Changed.Publish(ctx, changedOf(nil))
	return p, nil
}

func insertProfile(ctx context.Context, q *db.Queries, p Profile) error {
	switch taken, err := q.ProfileNameTaken(ctx, p.Name); {
	case err != nil:
		return err
	case taken:
		return ErrProfileNameTaken
	}
	if err := checkEntries(ctx, q, p.Entries); err != nil {
		return err
	}
	err := q.InsertProfile(ctx, db.InsertProfileParams{ID: p.ID, Name: p.Name, ContentType: p.ContentType, IsDefault: p.Default, Editions: listOf(p.Editions)})
	if err != nil {
		return err
	}
	if err := insertProfileCuts(ctx, q, p.ID, p.Editions); err != nil {
		return err
	}
	return insertEntries(ctx, q, p.ID, p.Entries)
}

func insertProfileCuts(ctx context.Context, q *db.Queries, id ProfileID, e Editions) error {
	for i, c := range e.cuts {
		if err := q.InsertProfileCut(ctx, db.InsertProfileCutParams{ProfileID: id, Position: int64(i + 1), Cut: c}); err != nil {
			return err
		}
	}
	return nil
}

// checkEntries refuses a Target named twice or that doesn't exist.
func checkEntries(ctx context.Context, q *db.Queries, entries []Entry) error {
	for i, e := range entries {
		if slices.ContainsFunc(entries[:i], func(o Entry) bool { return o.Target == e.Target }) {
			return ErrTargetTwice
		}
		switch has, err := q.HasTarget(ctx, e.Target); {
		case err != nil:
			return err
		case !has:
			return ErrNoTarget
		}
	}
	return nil
}

func insertEntries(ctx context.Context, q *db.Queries, id ProfileID, entries []Entry) error {
	for i, e := range entries {
		err := q.InsertProfileTarget(ctx, db.InsertProfileTargetParams{ProfileID: id, Position: int64(i + 1), TargetID: e.Target, Sourcing: e.Sourcing})
		if err != nil {
			return err
		}
	}
	return nil
}

// ListProfiles lists every Profile by Content Type, then name.
func (t *Targets) ListProfiles(ctx context.Context, _ identity.Admin) ([]Profile, error) {
	var out []Profile
	err := t.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		out, err = listProfiles(ctx, db.New(r))
		return err
	})
	return out, err
}

// GetProfile reads Profile id, ErrNoProfile when none has it.
func (t *Targets) GetProfile(ctx context.Context, by identity.Admin, id ProfileID) (Profile, error) {
	all, err := t.ListProfiles(ctx, by)
	if err != nil {
		return Profile{}, err
	}
	i := slices.IndexFunc(all, func(p Profile) bool { return p.ID == id })
	if i < 0 {
		return Profile{}, ErrNoProfile
	}
	return all[i], nil
}

// MakeDefault makes Profile id its Content Type's default. Existing Works
// keep the Profile they have.
func (t *Targets) MakeDefault(ctx context.Context, by identity.Admin, id ProfileID) (Profile, error) {
	var changed bool
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		changed, err = MakeDefaultIn(ctx, tx, id)
		return err
	})
	if err != nil {
		return Profile{}, err
	}
	if changed {
		t.cfg.Changed.Publish(ctx, changedOf(nil))
	}
	return t.GetProfile(ctx, by, id)
}

// MakeDefaultIn makes Profile id its Content Type's default in tx, writing
// only when it isn't already. changed is whether it wrote. Its caller
// publishes targets.changed after commit when it did.
func MakeDefaultIn(ctx context.Context, tx *storage.Tx[file], id ProfileID) (changed bool, err error) {
	q := db.New(tx)
	p, err := q.GetProfile(ctx, id)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return false, ErrNoProfile
	case err != nil:
		return false, err
	case p.IsDefault:
		return false, nil
	}
	if err := q.ClearDefault(ctx, p.ContentType); err != nil {
		return false, err
	}
	return true, q.SetDefault(ctx, id)
}

func listProfiles(ctx context.Context, q *db.Queries) ([]Profile, error) {
	rows, err := q.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}
	members, err := q.ListProfileTargets(ctx)
	if err != nil {
		return nil, err
	}
	cuts, err := q.ListProfileCuts(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := q.CountWants(ctx)
	if err != nil {
		return nil, err
	}
	held := map[ProfileID][]Entry{}
	for _, m := range members {
		held[m.ProfileID] = append(held[m.ProfileID], Entry{Target: m.TargetID, Sourcing: m.Sourcing})
	}
	named := map[ProfileID][]release.CutKey{}
	for _, c := range cuts {
		named[c.ProfileID] = append(named[c.ProfileID], c.Cut)
	}
	wanting := map[ProfileID]int{}
	for _, c := range counts {
		wanting[c.ProfileID] = int(c.Works)
	}
	out := make([]Profile, len(rows))
	for i, row := range rows {
		out[i] = Profile{
			ID: row.ID, Name: row.Name, ContentType: row.ContentType, Default: row.IsDefault, Entries: held[row.ID],
			Editions: editionsOf(row.Editions, named[row.ID]), Works: wanting[row.ID],
		}
	}
	return out, nil
}
