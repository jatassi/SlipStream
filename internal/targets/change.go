package targets

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/formats"
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

// SetWants gives Work Own as its own Edition list, or its Profile's when
// Own is nil. Its Profile changes only through SetProfiles.
type SetWants struct {
	Work works.WorkID
	Own  *Editions
}

// SetProfiles has each of Works want from Profile, of their Content Type,
// with its Edition list, except a Work with its own list, which keeps it.
type SetProfiles struct {
	Works   []works.WorkID
	Profile ProfileID
}

// UpdateCut replaces custom cut Key's name and aliases with Spec. Its key
// stays, and every file of it takes the new name.
type UpdateCut struct {
	Key  release.CutKey
	Spec CutSpec
}

// UpdateFormat replaces Custom Format ID's name, renaming flag and
// conditions with Format's. It keeps its dialect and pasted document, and
// its timeout count starts again.
type UpdateFormat struct {
	ID     formats.ID
	Format formats.Format
}

// DeleteFormat deletes Custom Format ID and every Target's score for it.
type DeleteFormat struct{ ID formats.ID }

// AddFormats adds each of Add and replaces each format of Replace's ids,
// all in Dialect. Each of Add carries the id NewFormatID minted. "New
// Custom Format" is an Add of one, in the slipstream dialect. A replaced
// format's timeout count starts again.
type AddFormats struct {
	Dialect formats.Dialect
	Add     []formats.Format
	Replace map[formats.ID]formats.Format
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
	err = q.UpdateTarget(ctx, db.UpdateTargetParams{
		ID: t.ID, Name: t.Name, CutoffTier: int64(t.Cutoff), MinScore: int64(t.Scores.Minimum), UpgradeUntil: int64(t.Scores.UpgradeUntil),
		ScoreIncrement: int64(t.Scores.Increment), Propers: t.Propers.String(),
	})
	if err != nil {
		return err
	}
	if err := q.DeleteQualities(ctx, t.ID); err != nil {
		return err
	}
	if err := q.DeleteFilters(ctx, t.ID); err != nil {
		return err
	}
	if err := q.DeleteFormatScores(ctx, t.ID); err != nil {
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
	var own *db.List
	if c.Own != nil {
		if err := known.check(*c.Own, now.ContentType); err != nil {
			return err
		}
		own = new(listOf(*c.Own))
	}
	if err := q.SetWants(ctx, db.SetWantsParams{WorkID: c.Work, Editions: own}); err != nil {
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

func (c SetProfiles) touches(ctx context.Context, q *db.Queries) (Touched, error) {
	own, err := q.CountOwnListsOf(ctx, c.Works)
	return Touched{Works: c.Works, OwnList: int(own)}, err
}

func (c SetProfiles) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	p, err := profileIn(ctx, q, c.Profile)
	if err != nil {
		return err
	}
	switch other, err := q.CountWantsOfOtherType(ctx, db.CountWantsOfOtherTypeParams{ContentType: p.ContentType, WorkIds: c.Works}); {
	case err != nil:
		return err
	case other > 0:
		return ErrProfileWrongType
	}
	return q.SetProfileOfWorks(ctx, db.SetProfileOfWorksParams{ProfileID: c.Profile, WorkIds: c.Works})
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

// worksScoring is the Works whose Profiles hold a Target that scores one
// of ids. A format no Target scores touches none, since no cell reads it.
func worksScoring(ctx context.Context, q *db.Queries, ids ...formats.ID) (Touched, error) {
	ws, err := q.ListWorksScoring(ctx, rowIDs(ids))
	return Touched{Works: ws}, err
}

func (c UpdateFormat) touches(ctx context.Context, q *db.Queries) (Touched, error) {
	return worksScoring(ctx, q, c.ID)
}

func (c UpdateFormat) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	stored, err := formatIn(ctx, q, c.ID)
	if err != nil {
		return err
	}
	f := c.Format
	f.ID, f.Dialect, f.Source = c.ID, stored.Dialect, stored.Source
	return saveFormat(ctx, q, f, true)
}

func (c DeleteFormat) touches(ctx context.Context, q *db.Queries) (Touched, error) {
	return worksScoring(ctx, q, c.ID)
}

func (c DeleteFormat) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	switch n, err := q.DeleteFormat(ctx, rowID(c.ID)); {
	case err != nil:
		return err
	case n == 0:
		return ErrNoFormat
	}
	return nil
}

func (c AddFormats) touches(ctx context.Context, q *db.Queries) (Touched, error) {
	return worksScoring(ctx, q, slices.SortedFunc(maps.Keys(c.Replace), byFormatID)...)
}

// apply replaces before it adds, so an Add may take a name a Replace
// gives up.
func (c AddFormats) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	for _, id := range slices.SortedFunc(maps.Keys(c.Replace), byFormatID) {
		f := c.Replace[id]
		f.ID, f.Dialect = id, c.Dialect
		if err := saveFormat(ctx, q, f, true); err != nil {
			return err
		}
	}
	for _, f := range c.Add {
		f.Dialect = c.Dialect
		if err := saveFormat(ctx, q, f, false); err != nil {
			return err
		}
	}
	return nil
}
