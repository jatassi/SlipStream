package files

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
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
	// Renamed counts the files that fill a cell after and change name.
	Renamed int
	// NewlyWanted counts the cells that become Wants.
	NewlyWanted int
	// Downloading counts the unfinished Grabs whose every cell goes; their
	// files follow the Admin's choice.
	Downloading int
}

// Mode is how Change ends. A Preview computes the Impact and rolls the
// change back; a save commits it. Unwanted is what becomes of the files
// it leaves filling nothing, and of the downloads in flight for cells it
// removes. Confirmed, when set, is how many files the Admin agreed to
// delete: a save deleting more is an *ImpactChangedError.
type Mode struct {
	Unwanted  Unwanted
	Preview   bool
	Confirmed *int
}

// ImpactChangedError is a save that would delete more files than the
// Admin confirmed, with the Impact it would have now.
type ImpactChangedError struct {
	Impact    Impact
	Confirmed int
}

func (e *ImpactChangedError) Error() string {
	return fmt.Sprintf("the change now leaves %d files unwanted, not %d", e.Impact.Unwanted, e.Confirmed)
}

var errPreview = errors.New("files: a preview rolls back")

// Change applies c and converges every Work it touches in one Write: it
// reads the Works c touches and their world, applies c, reads their world
// again, refills and names them, writes the fills, names, dooms and moves
// and settles the downloads in flight for cells that go. A Preview then
// rolls it all back, so its Impact is the save's. A save publishes
// targets.changed naming the Works it touched and pokes the disk queue
// once it commits.
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
		case m.Unwanted == Delete && m.Confirmed != nil && impact.Unwanted > *m.Confirmed:
			return &ImpactChangedError{Impact: impact, Confirmed: *m.Confirmed}
		}
		return nil
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
	touched, err := targets.Touches(ctx, tx, c)
	if err != nil {
		return Impact{}, nil, err
	}
	q := db.New(tx)
	tagged, err := q.ListWorksOfCuts(ctx, touched.Cuts)
	if err != nil {
		return Impact{}, nil, err
	}
	var (
		trees []works.Tree
		ids   []works.WorkID
	)
	for _, w := range append(touched.Works, tagged...) {
		if slices.ContainsFunc(trees, func(t works.Tree) bool { return t.ID == w }) {
			continue
		}
		tree, err := f.cfg.Works.TreeIn(ctx, tx, w)
		if err != nil {
			return Impact{}, nil, err
		}
		trees = append(trees, tree)
		ids = append(ids, w)
	}
	before, err := f.worlds(ctx, tx, trees)
	if err != nil {
		return Impact{}, nil, err
	}
	known, err := f.cfg.Targets.CutsIn(ctx, tx)
	if err != nil {
		return Impact{}, nil, err
	}
	if err := targets.Apply(ctx, tx, known, c); err != nil {
		return Impact{}, nil, err
	}
	afters, err := f.worlds(ctx, tx, trees)
	if err != nil {
		return Impact{}, nil, err
	}
	impact := Impact{OwnList: touched.OwnList}
	for i, was := range before {
		after := afters[i]
		p, err := f.converge(ctx, tx, was, after, refill(after), func(File) bool { return m.Unwanted == Delete })
		if err != nil {
			return Impact{}, nil, err
		}
		from, to := cells(was), cells(p.settled())
		impact.add(p, from, to)
		if err := apply(ctx, q, p); err != nil {
			return Impact{}, nil, err
		}
		n, err := f.cfg.Grabs.Settle(ctx, tx, was.tree.ID, gone(from, to), m.Unwanted)
		if err != nil {
			return Impact{}, nil, err
		}
		impact.Downloading += n
	}
	return impact, ids, nil
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
	for _, g := range p.after.files {
		nm := p.names[g.ID]
		if nm.Path != g.Path || nm.Label != g.Label || nm.Extra != g.Extra {
			changed = true
		}
		if is[g.ID] && nm.Path != g.Path {
			i.Renamed++
		}
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
	return fileOf(before) == fileOf(after)
}

func fileOf(c Cell) ID {
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

// HeldOf counts the library's files of cut in r, for targets.DeleteCut.
func HeldOf(ctx context.Context, r storage.Reader[file], cut release.CutKey) (int, error) {
	n, err := db.New(r).CountFilesOfCut(ctx, cut)
	return int(n), err
}
