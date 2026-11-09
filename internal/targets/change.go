package targets

import (
	"context"
	"errors"
	"maps"
	"reflect"
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
	touches(ctx context.Context, q *db.Queries, t formats.Timings) (Touched, error)
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
	// Narrow, when set, is every format edit of a Change that moves only
	// scores: of Works, only those holding a file whose score at some
	// Target one of them moves change.
	Narrow *Narrow
}

// Narrow is a Change's formats before and after, compiled together, and
// each one's score at each Target that scores it before and after, in
// the Edits' order.
type Narrow struct {
	Match  formats.Edits
	Scores []map[TargetID][2]int
}

// Moves is how far the edits move rd's score at each Target, leaving out
// the Targets they don't move.
func (n *Narrow) Moves(rd release.Read) map[TargetID]int {
	var out map[TargetID]int
	for i, flip := range n.Match.Match(rd) {
		for t, s := range n.Scores[i] {
			move := 0
			if flip.Before {
				move -= s[0]
			}
			if flip.After {
				move += s[1]
			}
			if move == 0 {
				continue
			}
			if out == nil {
				out = map[TargetID]int{}
			}
			out[t] += move
		}
	}
	maps.DeleteFunc(out, func(_ TargetID, n int) bool { return n == 0 })
	return out
}

// Touches reads in r, before c applies, what c may change.
func (t *Targets) Touches(ctx context.Context, r storage.Reader[file], c Change) (Touched, error) {
	return c.touches(ctx, db.New(r), t.cfg.Formats)
}

// Apply checks c as a create checks its record, against known, every cut
// in tx, and writes it in tx. Only files.Change calls it: a rule written
// without a refill leaves fills and names that no longer match their
// cells.
func Apply(ctx context.Context, tx *storage.Tx[file], known Cuts, c Change) error {
	return c.apply(ctx, db.New(tx), known)
}

// touches narrows to the formats whose scores c changes when it changes
// nothing else, and touches nothing when it changes nothing.
func (c UpdateTarget) touches(ctx context.Context, q *db.Queries, tm formats.Timings) (Touched, error) {
	ws, err := q.ListWorksOfTarget(ctx, c.ID)
	if err != nil {
		return Touched{}, err
	}
	all, err := listTargets(ctx, q)
	if err != nil {
		return Touched{}, err
	}
	i := slices.IndexFunc(all, func(t Target) bool { return t.ID == c.ID })
	if i < 0 {
		return Touched{}, ErrNoTarget
	}
	was, err := NewTarget(all[i].Spec())
	if err != nil {
		return Touched{}, err
	}
	next, err := NewTarget(c.Spec)
	if err != nil {
		return Touched{}, err
	}
	before, after := was.Scores.Formats, next.Scores.Formats
	was.Scores.Formats, next.Scores.Formats = nil, nil
	if !reflect.DeepEqual(was, next) {
		return Touched{Works: ws}, nil
	}
	var moved []formats.ID
	for id, n := range before {
		if after[id] != n {
			moved = append(moved, id)
		}
	}
	for id := range after {
		if _, ok := before[id]; !ok {
			moved = append(moved, id)
		}
	}
	slices.SortFunc(moved, byFormatID)
	var edits []versions
	for _, id := range moved {
		f, err := formatIn(ctx, q, id)
		if err != nil {
			return Touched{}, err
		}
		edits = append(edits, versions{id: id, Versions: formats.Versions{Before: &f.Format, After: &f.Format}})
	}
	if edits == nil {
		return Touched{}, nil
	}
	return Touched{Works: ws, Narrow: narrowed(edits, tm, func(e versions) map[TargetID][2]int {
		return map[TargetID][2]int{c.ID: {before[e.id], after[e.id]}}
	})}, nil
}

// narrowed compiles edits together, each scored at each Target as score
// answers.
func narrowed(edits []versions, tm formats.Timings, score func(versions) map[TargetID][2]int) *Narrow {
	n := &Narrow{Scores: make([]map[TargetID][2]int, len(edits))}
	vs := make([]formats.Versions, len(edits))
	for i, e := range edits {
		vs[i], n.Scores[i] = e.Versions, score(e)
	}
	n.Match = formats.Diff(vs, tm)
	return n
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
	folded, err := claimName(ctx, q, t.ID, t.Name)
	if err != nil {
		return err
	}
	err = q.UpdateTarget(ctx, db.UpdateTargetParams{
		ID: t.ID, Name: t.Name, NameFolded: &folded, CutoffTier: int64(t.Cutoff), MinScore: int64(t.Scores.Minimum), UpgradeUntil: int64(t.Scores.UpgradeUntil),
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

func (c UpdateProfile) touches(ctx context.Context, q *db.Queries, _ formats.Timings) (Touched, error) {
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

func (c DeleteProfile) touches(ctx context.Context, q *db.Queries, _ formats.Timings) (Touched, error) {
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

func (c SetWants) touches(context.Context, *db.Queries, formats.Timings) (Touched, error) { //nolint:unparam // the other kinds read to answer
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

func (c SetProfiles) touches(ctx context.Context, q *db.Queries, _ formats.Timings) (Touched, error) {
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

func (c UpdateCut) touches(context.Context, *db.Queries, formats.Timings) (Touched, error) { //nolint:unparam // the other kinds read to answer
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

// edited touches the Works whose Profiles hold a Target that scores one
// of the formats edits change, narrowed by those edits. A format no
// Target scores touches none, since no cell reads it.
func edited(ctx context.Context, q *db.Queries, tm formats.Timings, edits []versions) (Touched, error) {
	rows, err := q.ListFormatScores(ctx)
	if err != nil {
		return Touched{}, err
	}
	scores := map[formats.ID]map[TargetID][2]int{}
	for _, row := range rows {
		id := formatID(row.FormatID)
		if scores[id] == nil {
			scores[id] = map[TargetID][2]int{}
		}
		scores[id][row.TargetID] = [2]int{int(row.Score), int(row.Score)}
	}
	var (
		ids    []formats.ID
		scored []versions
	)
	for _, e := range edits {
		if _, ok := scores[e.id]; ok {
			ids = append(ids, e.id)
			scored = append(scored, e)
		}
	}
	if ids == nil {
		return Touched{}, nil
	}
	ws, err := q.ListWorksScoring(ctx, rowIDs(ids))
	if err != nil {
		return Touched{}, err
	}
	return Touched{Works: ws, Narrow: narrowed(scored, tm, func(e versions) map[TargetID][2]int { return scores[e.id] })}, nil
}

// versions is format id before and after a change.
type versions struct {
	id formats.ID
	formats.Versions
}

func (c UpdateFormat) touches(ctx context.Context, q *db.Queries, tm formats.Timings) (Touched, error) {
	stored, err := formatIn(ctx, q, c.ID)
	if err != nil {
		return Touched{}, err
	}
	after := c.saved(stored)
	return edited(ctx, q, tm, []versions{{id: c.ID, Versions: formats.Versions{Before: &stored.Format, After: &after}}})
}

// saved is the format c saves over stored.
func (c UpdateFormat) saved(stored StoredFormat) formats.Format {
	f := c.Format
	f.ID, f.Dialect, f.Source = c.ID, stored.Dialect, stored.Source
	return f
}

func (c UpdateFormat) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	stored, err := formatIn(ctx, q, c.ID)
	if err != nil {
		return err
	}
	return saveFormat(ctx, q, c.saved(stored), true)
}

func (c DeleteFormat) touches(ctx context.Context, q *db.Queries, tm formats.Timings) (Touched, error) {
	stored, err := formatIn(ctx, q, c.ID)
	if err != nil {
		return Touched{}, err
	}
	return edited(ctx, q, tm, []versions{{id: c.ID, Versions: formats.Versions{Before: &stored.Format}}})
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

func (c AddFormats) touches(ctx context.Context, q *db.Queries, tm formats.Timings) (Touched, error) {
	var edits []versions
	for _, id := range slices.SortedFunc(maps.Keys(c.Replace), byFormatID) {
		stored, err := formatIn(ctx, q, id)
		if err != nil {
			return Touched{}, err
		}
		after := c.replacing(id)
		edits = append(edits, versions{id: id, Versions: formats.Versions{Before: &stored.Format, After: &after}})
	}
	return edited(ctx, q, tm, edits)
}

// replacing is the format c saves over format id.
func (c AddFormats) replacing(id formats.ID) formats.Format {
	f := c.Replace[id]
	f.ID, f.Dialect = id, c.Dialect
	return f
}

// apply replaces before it adds, so an Add may take a name a Replace
// gives up.
func (c AddFormats) apply(ctx context.Context, q *db.Queries, _ Cuts) error {
	for _, id := range slices.SortedFunc(maps.Keys(c.Replace), byFormatID) {
		if err := saveFormat(ctx, q, c.replacing(id), true); err != nil {
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
