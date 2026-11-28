package targets

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// SplitProfile saves Profile with Spec for all its Works but Rest, which
// first move to To with Profile's old spec, so their cells stay as they
// are. Default, when set, becomes the Content Type's default Profile.
type SplitProfile struct {
	Profile ProfileID
	Spec    ProfileSpec
	Rest    []works.WorkID
	To      ProfileTarget
	Default *ProfileID
}

// ProfileTarget is where a SplitProfile moves the Works it leaves out:
// Profile, of the same Content Type, or else a Profile with the old spec,
// reused when one has it and otherwise created as NewName.
type ProfileTarget struct {
	Profile *ProfileID
	NewName string
}

// touches is every Work of Profile: the Rest keep their cells only when
// To has the old spec.
func (c SplitProfile) touches(ctx context.Context, q *db.Queries, tm formats.Timings) (Touched, error) {
	out, err := UpdateProfile{ID: c.Profile, Spec: c.Spec}.touches(ctx, q, tm)
	if err != nil {
		return Touched{}, err
	}
	out.Works, err = q.ListWorksOfProfile(ctx, c.Profile)
	return out, err
}

func (c SplitProfile) apply(ctx context.Context, q *db.Queries, known Cuts) error {
	if len(c.Rest) > 0 {
		on, err := q.ListWorksOfProfile(ctx, c.Profile)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(c.Rest, func(w works.WorkID) bool { return !slices.Contains(on, w) }) {
			return ErrNotOnProfile
		}
		to, err := c.restTo(ctx, q)
		if err != nil {
			return err
		}
		if err := q.SetProfileOfWorks(ctx, db.SetProfileOfWorksParams{ProfileID: to, WorkIds: c.Rest}); err != nil {
			return err
		}
	}
	if err := (UpdateProfile{ID: c.Profile, Spec: c.Spec}).apply(ctx, q, known); err != nil {
		return err
	}
	if c.Default == nil {
		return nil
	}
	if err := sameType(ctx, q, *c.Default, c.Spec.ContentType); err != nil {
		return err
	}
	_, err := makeDefault(ctx, q, *c.Default)
	return err
}

// restTo is the Profile the Rest move to, created when none has the old
// spec.
func (c SplitProfile) restTo(ctx context.Context, q *db.Queries) (ProfileID, error) {
	if to := c.To.Profile; to != nil {
		if *to == c.Profile {
			return ProfileID{}, ErrRestStays
		}
		return *to, sameType(ctx, q, *to, c.Spec.ContentType)
	}
	all, err := listProfiles(ctx, q)
	if err != nil {
		return ProfileID{}, err
	}
	i := slices.IndexFunc(all, func(p Profile) bool { return p.ID == c.Profile })
	if i < 0 {
		return ProfileID{}, ErrNoProfile
	}
	old := all[i]
	if j := slices.IndexFunc(all, func(p Profile) bool { return p.ID != old.ID && sameSpec(p, old) }); j >= 0 {
		return all[j].ID, nil
	}
	rest := Profile{
		ID: ProfileID{UUID: storage.NewUUID()}, Name: c.To.NewName, ContentType: old.ContentType,
		Entries: old.Entries, Editions: old.Editions, Languages: old.Languages,
	}
	return rest.ID, insertProfile(ctx, q, rest)
}

// sameSpec reports whether a and b want the same: their Content Type,
// rows, Edition list and Wanted Languages.
func sameSpec(a, b Profile) bool {
	return a.ContentType == b.ContentType && slices.EqualFunc(a.Entries, b.Entries, Entry.equal) &&
		a.Editions.Equal(b.Editions) && slices.Equal(a.Languages, b.Languages)
}
