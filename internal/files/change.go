package files

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Impact is what saving a Change does, as an Admin confirms it.
type Impact struct {
	// Works counts the Works whose cells, fills or file names change.
	Works int
	// OwnList counts the Works that keep their own Edition list through a
	// change to their Profile's.
	OwnList int
	// Unwanted counts the files that fill a cell now and none after, and
	// Bytes is their size.
	Unwanted int
	Bytes    int64
	// Renamed counts the files that change path, Kept files included.
	Renamed int
	// Renames lists the first ImpactRows of them, by Work in title order.
	Renames []Renames
	// NewlyWanted counts the cells that become Wants.
	NewlyWanted int
	// Downloading counts the Grabs in flight for a cell that goes.
	Downloading int
	// Subtitles is what a change of a Profile's Wanted Languages does.
	Subtitles SubtitlesImpact
	// Made is the files a change of a Profile's made-from rows has
	// SlipStream make, and Remake the made files a Target's older
	// Conversion settings made.
	Made   MadeImpact
	Remake RemakeImpact
}

// Unwanted is what becomes of a file a change leaves filling nothing: it
// is deleted, or it stays as a Kept file.
type Unwanted string

const (
	Delete Unwanted = "delete"
	Keep   Unwanted = "keep"
)

// Renames is the rows of one Work's files an Impact renames, in file id
// order.
type Renames struct {
	Work  works.WorkID
	Title string
	Rows  []Row
}

// ImpactRows is how many renamed files an Impact lists, so that a Target
// rename across a whole library is still one response.
const ImpactRows = 500

// Mode is how Change ends. A Preview computes the Impact and rolls the
// change back; a save commits it. Unwanted is what becomes of the files
// it leaves filling nothing. Confirmed, when set, is how many files the
// Admin agreed to delete, ConfirmedSubtitles how many subtitles,
// ConfirmedMade how many files to make and ConfirmedRemake how many to
// make again: a save past any of them is an *ImpactChangedError. Remake
// is the Works whose files a Target's save makes again.
type Mode struct {
	Unwanted           Unwanted
	Preview            bool
	Confirmed          *int
	ConfirmedSubtitles *int
	ConfirmedMade      *int
	ConfirmedRemake    *int
	Remake             []works.WorkID
}

// ImpactChangedError is a save that would delete more files or more
// subtitles, or make more files, than the Admin confirmed, with the Impact
// it would have now. Files, Subtitles, Made and Remake report which count
// grew.
type ImpactChangedError struct {
	Impact                         Impact
	Confirmed, ConfirmedSubtitles  int
	ConfirmedMade, ConfirmedRemake int
	Files, Subtitles, Made, Remake bool
}

func (e *ImpactChangedError) Error() string {
	var grew []string
	if e.Files {
		grew = append(grew, fmt.Sprintf("leaves %d files unwanted, not %d", e.Impact.Unwanted, e.Confirmed))
	}
	if e.Subtitles {
		grew = append(grew, fmt.Sprintf("deletes %d subtitles, not %d", e.Impact.Subtitles.Deleted(), e.ConfirmedSubtitles))
	}
	if e.Made {
		grew = append(grew, fmt.Sprintf("makes %d files, not %d", e.Impact.Made.Files, e.ConfirmedMade))
	}
	if e.Remake {
		grew = append(grew, fmt.Sprintf("makes %d files again, not %d", e.Impact.Remake.Files(), e.ConfirmedRemake))
	}
	return "the change now " + strings.Join(grew, " and ")
}

// refused is the *ImpactChangedError a save of impact under m is, nil
// when it deletes no more than m confirmed.
func (m Mode) refused(impact Impact) error {
	e := &ImpactChangedError{Impact: impact}
	if m.Unwanted == Delete && m.Confirmed != nil && impact.Unwanted > *m.Confirmed {
		e.Files, e.Confirmed = true, *m.Confirmed
	}
	if m.ConfirmedSubtitles != nil && impact.Subtitles.Deleted() > *m.ConfirmedSubtitles {
		e.Subtitles, e.ConfirmedSubtitles = true, *m.ConfirmedSubtitles
	}
	if m.ConfirmedMade != nil && impact.Made.Files > *m.ConfirmedMade {
		e.Made, e.ConfirmedMade = true, *m.ConfirmedMade
	}
	if m.ConfirmedRemake != nil && impact.Remake.Files() > *m.ConfirmedRemake {
		e.Remake, e.ConfirmedRemake = true, *m.ConfirmedRemake
	}
	if !e.Files && !e.Subtitles && !e.Made && !e.Remake {
		return nil
	}
	return e
}

// Files is how many made files r's Works hold.
func (r RemakeImpact) Files() int {
	n := 0
	for _, row := range r.Rows {
		n += row.Files
	}
	return n
}

var errPreview = errors.New("files: a preview rolls back")

// Change applies c and converges every Work it touches in one Write: it
// reads the Works c touches and their world, applies c, reads their world
// again, refills and names them, and writes the fills, names, dooms and
// moves. A Preview then rolls it all back, so its Impact is the save's. A
// save publishes targets.changed naming the Works it touched and pokes
// the disk queue once it commits.
func (f *Files) Change(ctx context.Context, _ identity.Admin, c targets.Change, m Mode) (Impact, error) {
	var (
		impact  Impact
		touched []works.WorkID
	)
	err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		impact, touched, err = f.change(ctx, tx, c, m)
		switch {
		case err != nil:
			return err
		case m.Preview:
			return errPreview
		}
		return m.refused(impact)
	})
	switch {
	case errors.Is(err, errPreview):
		return impact, nil
	case err != nil:
		return Impact{}, err
	}
	f.cfg.Targets.Publish(ctx, touched)
	f.Poke()
	return impact, nil
}

// change answers the Impact and the Works c touches.
func (f *Files) change(ctx context.Context, tx *storage.Tx[file], c targets.Change, m Mode) (Impact, []works.WorkID, error) {
	touched, moved, err := f.touches(ctx, tx, c)
	if err != nil {
		return Impact{}, nil, err
	}
	q := db.New(tx)
	tagged, err := q.ListWorksOfCuts(ctx, touched.Cuts)
	if err != nil {
		return Impact{}, nil, err
	}
	var (
		trees []works.Bare
		ids   []works.WorkID
	)
	for _, w := range append(touched.Works, tagged...) {
		if slices.ContainsFunc(trees, func(t works.Bare) bool { return t.ID == w }) {
			continue
		}
		tree, err := works.BareIn(ctx, tx, w)
		if err != nil {
			return Impact{}, nil, err
		}
		trees = append(trees, tree)
		ids = append(ids, w)
	}
	before, err := f.worlds(ctx, tx, trees, works.Only(ids), wholeFiles)
	if err != nil {
		return Impact{}, nil, err
	}
	known, err := f.cfg.Targets.CutsIn(ctx, tx)
	if err != nil {
		return Impact{}, nil, err
	}
	subs, err := f.applyTelling(ctx, tx, known, c, m.Preview)
	if err != nil {
		return Impact{}, nil, err
	}
	afters, err := f.after(ctx, tx, trees, ids, before, moved)
	if err != nil {
		return Impact{}, nil, err
	}
	fills, from := make([]map[CellKey]Fill, len(before)), make([][]Cell, len(before))
	inParallel(len(before), func(i int) {
		afters[i].scoring.known = moved.onto(before[i].fills)
		fills[i], from[i] = refill(afters[i]), cells(before[i])
	})
	impact := Impact{OwnList: touched.OwnList, Subtitles: subs}
	var made []MadePair
	for i, was := range before {
		p, err := f.converge(ctx, tx, was, afters[i], fills[i], changing(m))
		if err != nil {
			return Impact{}, nil, err
		}
		settled := p.settled()
		to := cells(settled)
		impact.add(p, from[i], to)
		made = append(made, newlyMade(settled, from[i], to)...)
		if err := f.apply(ctx, tx, p, nil); err != nil {
			return Impact{}, nil, err
		}
		impact.Downloading += downloading(was.grabs, gone(from[i], to))
	}
	impact.Renames = listed(impact.Renames, trees)
	if impact.Made, err = f.made(ctx, tx, c, made); err != nil {
		return Impact{}, nil, err
	}
	if impact.Remake, err = f.remake(ctx, tx, c, m); err != nil {
		return Impact{}, nil, err
	}
	if err := f.queue(ctx, tx, c, ids); err != nil {
		return Impact{}, nil, err
	}
	return impact, ids, nil
}

// made is the MadeImpact of making made, the pairs a save of c makes.
func (f *Files) made(ctx context.Context, tx *storage.Tx[file], c targets.Change, made []MadePair) (MadeImpact, error) {
	var out MadeImpact
	if split, ok := c.(targets.SplitProfile); ok {
		out.Matching = len(split.Rest)
	}
	if len(made) == 0 {
		return out, nil
	}
	cost, err := f.cfg.Conversions.Cost(ctx, tx, made)
	if err != nil {
		return MadeImpact{}, err
	}
	out.Files, out.Works, out.Seconds, out.Bytes, out.Nodes, out.Unconvertible, out.Rows =
		len(made), cost.Works, cost.Seconds, cost.Bytes, cost.Nodes, cost.Unconvertible, cost.Rows
	return out, nil
}

// remake is what an UpdateTarget of a Target with Conversion settings
// makes again: a preview lists every Work its older settings made a file
// of, and a save remakes m's.
func (f *Files) remake(ctx context.Context, tx *storage.Tx[file], c targets.Change, m Mode) (RemakeImpact, error) {
	u, ok := c.(targets.UpdateTarget)
	if !ok || !m.Preview && len(m.Remake) == 0 {
		return RemakeImpact{}, nil
	}
	all, err := targets.All(ctx, tx)
	if err != nil {
		return RemakeImpact{}, err
	}
	i := slices.IndexFunc(all, func(t targets.Target) bool { return t.ID == u.ID })
	if i < 0 || all[i].Conversion == nil {
		return RemakeImpact{}, nil
	}
	var ws []works.WorkID
	if !m.Preview {
		ws = m.Remake
	}
	return f.cfg.Conversions.Remakes(ctx, tx, u.ID, all[i].Conversion.Version, ws, m.Preview)
}

// newlyMade is the pairs to, a Work's cells after a Change, makes wanted
// that from, its cells before, didn't: each made-from cell of a counted,
// released Item, not met, with a file to make it from, whose cell before
// wasn't made from the same Target. A file filling two cells is one pair.
func newlyMade(w world, from, to []Cell) []MadePair {
	was := map[CellKey]targets.TargetID{}
	for _, c := range from {
		if c.From != nil {
			was[c.key()] = *c.From
		}
	}
	at := make(map[CellKey]Cell, len(to))
	for _, c := range to {
		at[c.key()] = c
	}
	byID := w.byID()
	var out []MadePair
	for _, c := range to {
		if c.From == nil || !c.Item.Counts() || !c.Released || c.State == Met {
			continue
		}
		if t, ok := was[c.key()]; ok && t == *c.From {
			continue
		}
		src := sourceOf(w, byID, at, c)
		if src == nil {
			continue
		}
		pair := MadePair{Work: w.tree.ID, Source: src.ID, Target: c.Target.ID}
		if !slices.Contains(out, pair) {
			out = append(out, pair)
		}
	}
	return out
}

// applyTelling applies c in tx and tells subtitles of a change of its
// Profile's Wanted Languages: an UpdateProfile's before it applies, and a
// SplitProfile's once its rest have left the Profile.
func (f *Files) applyTelling(ctx context.Context, tx *storage.Tx[file], known targets.Cuts, c targets.Change, preview bool) (SubtitlesImpact, error) {
	was, err := f.wantedLanguages(ctx, tx, c)
	if err != nil {
		return SubtitlesImpact{}, err
	}
	_, split := c.(targets.SplitProfile)
	var subs SubtitlesImpact
	if !split {
		if subs, err = f.languages(ctx, tx, c, was, preview); err != nil {
			return SubtitlesImpact{}, err
		}
	}
	if err := targets.Apply(ctx, tx, known, c); err != nil {
		return SubtitlesImpact{}, err
	}
	if split {
		return f.languages(ctx, tx, c, was, preview)
	}
	return subs, nil
}

// wantedLanguages is, in tx before c applies, the Wanted Languages of the
// Profile an UpdateProfile or SplitProfile saves, nil for any other.
func (f *Files) wantedLanguages(ctx context.Context, tx *storage.Tx[file], c targets.Change) ([]targets.WantedLanguage, error) {
	switch u := c.(type) {
	case targets.UpdateProfile:
		return f.cfg.Targets.LanguagesIn(ctx, tx, u.ID)
	case targets.SplitProfile:
		return f.cfg.Targets.LanguagesIn(ctx, tx, u.Profile)
	}
	return nil, nil
}

// languages tells subtitles of an UpdateProfile or SplitProfile that
// changes its Profile's Wanted Languages from was, and answers what it
// does.
func (f *Files) languages(ctx context.Context, tx *storage.Tx[file], c targets.Change, was []targets.WantedLanguage, preview bool) (SubtitlesImpact, error) {
	var (
		id  targets.ProfileID
		now []targets.WantedLanguage
	)
	switch u := c.(type) {
	case targets.UpdateProfile:
		id, now = u.ID, u.Spec.Languages
	case targets.SplitProfile:
		id, now = u.Profile, u.Spec.Languages
	default:
		return SubtitlesImpact{}, nil
	}
	if slices.Equal(was, now) {
		return SubtitlesImpact{}, nil
	}
	return f.cfg.Subtitles.Languages(ctx, tx, id, was, now, preview)
}

// queue asks, in tx, for the search c calls for of the Works it touched,
// and for their Conversions to converge when c moves Works onto a Profile
// or changes a Profile's rows.
func (f *Files) queue(ctx context.Context, tx *storage.Tx[file], c targets.Change, touched []works.WorkID) error {
	if asks := profileAsks(c, touched); len(asks) > 0 {
		if err := f.cfg.Conversions.Queue(ctx, tx, asks...); err != nil {
			return err
		}
	}
	t, ok := searchesOf(c)
	if !ok || len(touched) == 0 {
		return nil
	}
	asks := make([]searches.Ask, len(touched))
	for i, w := range touched {
		asks[i] = searches.Ask{Work: w}
	}
	return f.cfg.Queue.Queue(ctx, tx, t, asks...)
}

// profileAsks is an Ask for each of touched whose Profile c moves it onto
// or changes. A SplitProfile's rest keep their cells, and the Works it
// converts are a backfill.
func profileAsks(c targets.Change, touched []works.WorkID) []Ask {
	var rest []works.WorkID
	backfill := false
	switch u := c.(type) {
	case targets.SetProfiles, targets.DeleteProfile, targets.UpdateProfile:
	case targets.SplitProfile:
		rest, backfill = u.Rest, true
	default:
		return nil
	}
	now := storage.Now().Time()
	var out []Ask
	for _, w := range touched {
		if !slices.Contains(rest, w) {
			out = append(out, Ask{Work: w, Why: ArrivalProfile, Since: now, Backfill: backfill})
		}
	}
	return out
}

// searchesOf is the search a save of c asks for the Works it touches,
// false for none. A Target, cut or format edit only resets their gaps.
func searchesOf(c targets.Change) (searches.Trigger, bool) {
	switch c.(type) {
	case targets.SetProfiles, targets.DeleteProfile:
		return searches.Profile, true
	case targets.SetWants:
		return searches.Editions, true
	case targets.UpdateTarget, targets.UpdateProfile, targets.SplitProfile, targets.UpdateCut, targets.UpdateFormat, targets.DeleteFormat, targets.AddFormats:
		return searches.Retarget, true
	case targets.SetHold:
		return "", false
	}
	panic(fmt.Sprintf("files: no search rule for a %T", c))
}

// touches is what c may change, narrowed by its format edits to the
// Works holding a file whose score they move, and those moves, nil for a
// Change with no format edits.
func (f *Files) touches(ctx context.Context, tx *storage.Tx[file], c targets.Change) (targets.Touched, moves, error) {
	touched, err := f.cfg.Targets.Touches(ctx, tx, c)
	if err != nil || touched.Narrow == nil {
		return touched, nil, err
	}
	var moved moves
	touched.Works, moved, err = f.flips(ctx, tx, touched.Works, touched.Narrow)
	return touched, moved, err
}

// settled is p's world once applied: its files and their new fills.
func (p placed) settled() world {
	w := p.after
	w.fills = p.fills
	return w
}

// add counts what converging p changes, whose cells go from from to to.
func (i *Impact) add(p placed, from, to []Cell) {
	was, is := filling(p.before.fills), filling(p.fills)
	changed := len(p.doomed) > 0 || !maps.Equal(p.before.fills, p.fills) || !sameCells(from, to)
	for _, g := range p.before.files {
		if was[g.ID] && !is[g.ID] {
			i.Unwanted++
			i.Bytes += g.Size
		}
	}
	var rows []Row
	for _, g := range p.after.files {
		nm := p.names[g.ID]
		if nm.Path != g.Path || nm.Label != g.Label || nm.Extra != g.Extra {
			changed = true
		}
		if nm.Path != g.Path {
			rows = append(rows, Row{Work: p.after.tree.ID, File: g.ID, From: g.Path, To: nm.Path, root: g.RootID})
		}
	}
	if len(rows) > 0 {
		group(rows)
		i.Renamed += len(rows)
		i.Renames = append(i.Renames, Renames{Work: p.after.tree.ID, Title: p.after.tree.Label, Rows: rows})
	}
	type spot struct {
		item   works.ItemID
		target targets.TargetID
	}
	prior := map[spot][]Cell{}
	for _, c := range from {
		at := spot{c.Item.ID, c.Target.ID}
		prior[at] = append(prior[at], c)
	}
	for _, c := range to {
		if c.Wanted() && !slices.ContainsFunc(prior[spot{c.Item.ID, c.Target.ID}], func(b Cell) bool { return b.Wanted() && continues(b, c) }) {
			i.NewlyWanted++
		}
	}
	if changed {
		i.Works++
	}
}

// listed is the first ImpactRows rows of rs, by Work in the title order
// of trees.
func listed(rs []Renames, trees []works.Bare) []Renames {
	sortLabel := make(map[works.WorkID]string, len(trees))
	for _, t := range trees {
		sortLabel[t.ID] = t.SortLabel
	}
	slices.SortFunc(rs, func(a, b Renames) int {
		return cmp.Or(cmp.Compare(sortLabel[a.Work], sortLabel[b.Work]), a.Work.Compare(b.Work.UUID))
	})
	left := ImpactRows
	for i := range rs {
		if len(rs[i].Rows) >= left {
			rs[i].Rows = rs[i].Rows[:left]
			return rs[:i+1]
		}
		left -= len(rs[i].Rows)
	}
	return rs
}

// continues reports whether after, a cell at before's Item and Target,
// is the cell before was: one of the same Edition or, across a switch
// between any cut and named cuts, one the same file fills, or none does.
func continues(before, after Cell) bool {
	switch {
	case before.Edition == after.Edition:
		return true
	case before.Edition != targets.AnyEdition && after.Edition != targets.AnyEdition:
		return false
	}
	return fileIDOf(before) == fileIDOf(after)
}

func fileIDOf(c Cell) ID {
	if c.File == nil {
		return ID{}
	}
	return c.File.ID
}

func keyOf(c Cell) CellKey { return CellKey{Item: c.Item.ID, Edition: c.Edition, Target: c.Target.ID} }

// sameCells reports whether a and b hold the same cells in the same
// states, filled by the same files.
func sameCells(a, b []Cell) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if keyOf(a[i]) != keyOf(b[i]) || a[i].State != b[i].State || (a[i].File == nil) != (b[i].File == nil) ||
			a[i].File != nil && a[i].File.ID != b[i].File.ID {
			return false
		}
	}
	return true
}

// gone is every cell of from that to lacks.
func gone(from, to []Cell) map[CellKey]bool {
	out := map[CellKey]bool{}
	for _, c := range from {
		out[keyOf(c)] = true
	}
	for _, c := range to {
		delete(out, keyOf(c))
	}
	return out
}

// downloading counts the Grabs in flight for a cell of gone.
func downloading(grabs map[CellKey]GrabCell, gone map[CellKey]bool) int {
	in := map[storage.UUID]bool{}
	for key := range gone {
		if g := grabs[key]; g.InFlight() {
			in[g.Grab] = true
		}
	}
	return len(in)
}

// HeldOf counts the library's files of cut in r, for targets.DeleteCut.
func HeldOf(ctx context.Context, r storage.Reader[file], cut release.CutKey) (int, error) {
	n, err := db.New(r).CountFilesOfCut(ctx, cut)
	return int(n), err
}
