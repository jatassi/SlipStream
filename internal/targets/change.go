package targets

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// Change is a rule change that can leave files filling nothing, rename
// files or make new cells. Only targets defines one, and only files
// applies one (lint: targets-apply-confined), in the Write that refills
// every Work it touches.
type Change interface {
	touches(ctx context.Context, q *db.Queries) (Touched, error)
	apply(ctx context.Context, q *db.Queries, known Cuts) error
}

// UpdateTarget replaces Target ID's name and rules with Spec.
type UpdateTarget struct {
	ID   TargetID
	Spec Spec
}

// UpdateProfile replaces Profile ID's name, Targets and Edition list with
// Spec, whose Content Type must be the Profile's.
type UpdateProfile struct {
	ID   ProfileID
	Spec ProfileSpec
}

// DeleteProfile deletes Profile ID, which isn't its type's default, and
// moves its Works to To, of the same type. They keep their own lists.
type DeleteProfile struct{ ID, To ProfileID }

// SetWants has Work want from Profile, of its type, with Own as its own
// Edition list, or its Profile's when Own is nil.
type SetWants struct {
	Work    works.WorkID
	Profile ProfileID
	Own     *Editions
}

// UpdateCut replaces custom cut Key's name and aliases with Spec. Its key
// stays, and every file of it takes the new name.
type UpdateCut struct {
	Key  release.CutKey
	Spec CutSpec
}

// Touched is what a Change may change, read before it applies: the Works
// whose Goals it changes, the cuts whose files it renames and, for a
// Profile's Edition change, how many Works keep their own list.
type Touched struct {
	Works   []works.WorkID
	Cuts    []release.CutKey
	OwnList int
}

// Touches reads in r, before c applies, what c may change.
func Touches(ctx context.Context, r storage.Reader[file], c Change) (Touched, error) {
	return c.touches(ctx, db.New(r))
}

// Apply checks c as a create checks its record, against known, every cut
// in tx, and writes it in tx. Only files.Change calls it: a rule written
// without a refill leaves fills and names that no longer match their
// cells.
func Apply(ctx context.Context, tx *storage.Tx[file], known Cuts, c Change) error {
	return c.apply(ctx, db.New(tx), known)
}

func (c UpdateTarget) touches(ctx context.Context, q *db.Queries) (Touched, error) {
	ws, err := q.ListWorksOfTarget(ctx, c.ID)
	return Touched{Works: ws}, err
}

func (c UpdateTarget) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	t, err := NewTarget(c.Spec)
	if err != nil {
		return err
	}
	t.ID = c.ID
	switch has, err := q.HasTarget(ctx, t.ID); {
	case err != nil:
		return err
	case !has:
		return ErrNoTarget
	}
	switch taken, err := q.TargetNameTakenByOther(ctx, db.TargetNameTakenByOtherParams{Name: t.Name, ID: t.ID}); {
	case err != nil:
		return err
	case taken:
		return ErrTargetNameTaken
	}
	if err := q.UpdateTarget(ctx, db.UpdateTargetParams{ID: t.ID, Name: t.Name, CutoffTier: int64(t.Cutoff)}); err != nil {
		return err
	}
	if err := q.DeleteQualities(ctx, t.ID); err != nil {
		return err
	}
	if err := q.DeleteFilters(ctx, t.ID); err != nil {
		return err
	}
	return insertRules(ctx, q, t)
}

func (c UpdateProfile) touches(ctx context.Context, q *db.Queries) (Touched, error) {
	p, err := profileIn(ctx, q, c.ID)
	if err != nil {
		return Touched{}, err
	}
	ws, err := q.ListWorksOfProfile(ctx, c.ID)
	out := Touched{Works: ws}
	if err != nil || p.Editions.Equal(c.Spec.Editions) {
		return out, err
	}
	own, err := q.CountOwnLists(ctx, c.ID)
	out.OwnList = int(own)
	return out, err
}

func (c UpdateProfile) apply(ctx context.Context, q *db.Queries, known Cuts) error {
	p, err := profileIn(ctx, q, c.ID)
	switch {
	case err != nil:
		return err
	case p.ContentType != c.Spec.ContentType:
		return ErrProfileWrongType
	}
	if err := known.check(c.Spec.Editions, p.ContentType); err != nil {
		return err
	}
	switch taken, err := q.ProfileNameTakenByOther(ctx, db.ProfileNameTakenByOtherParams{Name: c.Spec.Name, ID: c.ID}); {
	case err != nil:
		return err
	case taken:
		return ErrProfileNameTaken
	}
	if err := checkEntries(ctx, q, c.Spec.Entries); err != nil {
		return err
	}
	if err := q.UpdateProfile(ctx, db.UpdateProfileParams{ID: c.ID, Name: c.Spec.Name, Editions: listOf(c.Spec.Editions)}); err != nil {
		return err
	}
	if err := q.DeleteProfileCuts(ctx, c.ID); err != nil {
		return err
	}
	if err := insertProfileCuts(ctx, q, c.ID, c.Spec.Editions); err != nil {
		return err
	}
	if err := q.DeleteProfileTargets(ctx, c.ID); err != nil {
		return err
	}
	return insertEntries(ctx, q, c.ID, c.Spec.Entries)
}

func (c DeleteProfile) touches(ctx context.Context, q *db.Queries) (Touched, error) {
	ws, err := q.ListWorksOfProfile(ctx, c.ID)
	return Touched{Works: ws}, err
}

func (c DeleteProfile) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	gone, err := profileIn(ctx, q, c.ID)
	switch {
	case err != nil:
		return err
	case gone.Default:
		return ErrProfileIsDefault
	case c.To == c.ID:
		return ErrMovesToItself
	}
	switch err := sameType(ctx, q, c.To, gone.ContentType); {
	case errors.Is(err, ErrNoProfile):
		return ErrNoReplacement
	case err != nil:
		return err
	}
	if err := q.MoveWants(ctx, db.MoveWantsParams{FromProfile: c.ID, ToProfile: c.To}); err != nil {
		return err
	}
	return q.DeleteProfile(ctx, c.ID)
}

func (c SetWants) touches(context.Context, *db.Queries) (Touched, error) { //nolint:unparam // the other kinds read to answer
	return Touched{Works: []works.WorkID{c.Work}}, nil
}

func (c SetWants) apply(ctx context.Context, q *db.Queries, known Cuts) error {
	row, err := q.GetWants(ctx, c.Work)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return works.ErrNoWork
	case err != nil:
		return err
	}
	now, err := profileIn(ctx, q, row.ProfileID)
	if err != nil {
		return err
	}
	if err := sameType(ctx, q, c.Profile, now.ContentType); err != nil {
		return err
	}
	var own *db.List
	if c.Own != nil {
		if err := known.check(*c.Own, now.ContentType); err != nil {
			return err
		}
		own = new(listOf(*c.Own))
	}
	if err := q.SetWants(ctx, db.SetWantsParams{WorkID: c.Work, ProfileID: c.Profile, Editions: own}); err != nil {
		return err
	}
	if err := q.DeleteWantsCuts(ctx, c.Work); err != nil {
		return err
	}
	if c.Own == nil {
		return nil
	}
	for i, cut := range c.Own.cuts {
		if err := q.InsertWantsCut(ctx, db.InsertWantsCutParams{WorkID: c.Work, Position: int64(i + 1), Cut: cut}); err != nil {
			return err
		}
	}
	return nil
}

func (c UpdateCut) touches(context.Context, *db.Queries) (Touched, error) { //nolint:unparam // the other kinds read to answer
	return Touched{Cuts: []release.CutKey{c.Key}}, nil
}

func (c UpdateCut) apply(ctx context.Context, q *db.Queries, known Cuts) error {
	if err := known.custom(c.Key); err != nil {
		return err
	}
	aliases, err := c.Spec.aliases()
	if err != nil {
		return err
	}
	if known.claimed(CutSpec{Name: c.Spec.Name, Aliases: aliases}, c.Key) {
		return ErrCutNameTaken
	}
	if err := q.RenameCut(ctx, db.RenameCutParams{Key: c.Key, Name: c.Spec.Name}); err != nil {
		return err
	}
	if err := q.DeleteCutAliases(ctx, c.Key); err != nil {
		return err
	}
	return insertAliases(ctx, q, c.Key, aliases)
}

// profileIn reads Profile id, ErrNoProfile when none has it.
func profileIn(ctx context.Context, q *db.Queries, id ProfileID) (Profile, error) {
	row, err := q.GetProfile(ctx, id)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return Profile{}, ErrNoProfile
	case err != nil:
		return Profile{}, err
	}
	cuts, err := q.ListCutsOfProfile(ctx, id)
	if err != nil {
		return Profile{}, err
	}
	return Profile{ID: row.ID, Name: row.Name, ContentType: row.ContentType, Default: row.IsDefault, Editions: editionsOf(row.Editions, cuts)}, nil
}

// sameType refuses Profile id unless it exists and is for typ.
func sameType(ctx context.Context, q *db.Queries, id ProfileID, typ works.TypeID) error {
	p, err := profileIn(ctx, q, id)
	switch {
	case err != nil:
		return err
	case p.ContentType != typ:
		return ErrProfileWrongType
	}
	return nil
}
