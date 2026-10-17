package files

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
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
// targets.changed and pokes the disk queue once it commits.
func (f *Files) Change(ctx context.Context, _ identity.Admin, c targets.Change, m Mode) (Impact, error) {
	var impact Impact
	err := f.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		impact, err = f.change(ctx, tx, c, m)
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
	f.cfg.Targets.Publish(ctx)
	f.Poke()
	return impact, nil
}

func (f *Files) change(ctx context.Context, tx *storage.Tx[file], c targets.Change, m Mode) (Impact, error) {
	touched, err := targets.Touches(ctx, tx, c)
	if err != nil {
		return Impact{}, err
	}
	before := make([]world, len(touched.Works))
	for i, w := range touched.Works {
		if before[i], err = f.load(ctx, tx, w); err != nil {
			return Impact{}, err
		}
	}
	if err := targets.Apply(ctx, tx, c); err != nil {
		return Impact{}, err
	}
	impact := Impact{OwnList: touched.OwnList}
	q := db.New(tx)
	for _, was := range before {
		after, err := f.load(ctx, tx, was.tree.ID)
		if err != nil {
			return Impact{}, err
		}
		p, err := f.converge(ctx, tx, was, after, refill(after), func(File) bool { return m.Unwanted == Delete })
		if err != nil {
			return Impact{}, err
		}
		from, to := cells(was), cells(p.settled())
		impact.add(p, from, to)
		if err := apply(ctx, q, p); err != nil {
			return Impact{}, err
		}
		n, err := f.cfg.Grabs.Settle(ctx, tx, was.tree.ID, gone(from, to), m.Unwanted)
		if err != nil {
			return Impact{}, err
		}
		impact.Downloading += n
	}
	return impact, nil
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
	wanted := map[CellKey]bool{}
	for _, c := range from {
		wanted[keyOf(c)] = c.Wanted()
	}
	for _, c := range to {
		if c.Wanted() && !wanted[keyOf(c)] {
			i.NewlyWanted++
		}
	}
	if changed {
		i.Works++
	}
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
