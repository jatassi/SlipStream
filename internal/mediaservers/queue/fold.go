package queue

import (
	"reflect"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

// effect is what a refresh change does to the file at its path, whether
// the file is a media file or a Sidecar.
type effect int

const (
	nothing effect = iota
	add
	replace
	remove
)

// merged is the spec's merge table for two refresh changes on one path,
// earlier then later. nothing deletes the row: the server never saw the file.
var merged = map[[2]effect]effect{
	{add, add}:         add,
	{add, replace}:     add,
	{add, remove}:      nothing,
	{replace, add}:     replace,
	{replace, replace}: replace,
	{replace, remove}:  remove,
	{remove, add}:      replace,
	{remove, replace}:  replace,
	{remove, remove}:   remove,
}

// families are the refresh kinds by effect. A media file and a Sidecar
// fold by the same table and keep their own kinds.
var families = [][4]link.ChangeKind{
	{add: link.ChangeAdded, replace: link.ChangeReplaced, remove: link.ChangeRemoved},
	{add: link.ChangeSidecarAdded, replace: link.ChangeSidecarReplaced, remove: link.ChangeSidecarRemoved},
}

func effectOf(k link.ChangeKind) ([4]link.ChangeKind, effect) {
	for _, f := range families {
		if e := slices.Index(f[:], k); e > 0 {
			return f, effect(e)
		}
	}
	panic("queue: " + string(k) + " is not a refresh kind")
}

// Open is every foldable row of one server that a new change can fold into.
type Open struct {
	// AtPath holds the refresh rows at the change's paths and at the origin
	// of its file's carry. The library row of a root sits at the zero path.
	AtPath map[storing.Rel]Change
	// Carry is the foldable carry of the change's file.
	Carry *Change
	// Partner is the other half of an upgrade. For a removed with
	// ReplacedBy it is the added of that file with Upgrade, and for an
	// added it is the removed whose ReplacedBy is its file.
	Partner *Change
}

// Do is what an Op does to its row.
type Do int

const (
	Delete Do = iota
	Update
	Insert
)

// Op is one write that applies a fold. Fold and Reopen order deletes before
// updates before inserts, so no write meets the unique index of a row a
// later write removes.
type Op struct {
	Do     Do
	Change Change
}

// Fold merges in, a new change of one server, into the open rows it meets.
func Fold(open Open, in Change) []Op {
	var r rows
	r.fold(open, in)
	return r.ops()
}

// Reopen returns back, a row whose send failed or was cut off, to due with
// its attempts, wait and problem already set. A back that is foldable again
// folds newer into it: the row that took its place while it was sending, at
// its path for a refresh and of its file for a carry. The server may have
// seen back, so the fold reads its added as replaced and its moved as a
// replaced carry, which never folds to nothing.
func Reopen(back Change, newer *Change) []Op {
	r := rows{was: []Change{back}}
	back.State = Due
	if newer == nil || !back.Foldable() {
		r.now = []Change{back}
		return r.ops()
	}
	if k, ok := seen[back.Kind]; ok {
		back.Kind = k
	}
	r.was = append(r.was, *newer)
	r.now = []Change{back}
	open := Open{AtPath: map[storing.Rel]Change{back.Path: back}}
	if back.Carry() {
		open = Open{Carry: &back}
	}
	r.fold(open, *newer)
	return r.ops()
}

var seen = map[link.ChangeKind]link.ChangeKind{
	link.ChangeAdded:        link.ChangeReplaced,
	link.ChangeSidecarAdded: link.ChangeSidecarReplaced,
	link.ChangeMoved:        link.ChangeReplaced,
}

// rows is a fold's working set: the rows it read as they were, and every
// row as the fold leaves it.
type rows struct{ was, now []Change }

func same(id link.ChangeID) func(Change) bool {
	return func(c Change) bool { return c.ID == id }
}

func (r *rows) read(c Change) {
	if !slices.ContainsFunc(r.was, same(c.ID)) {
		r.was = append(r.was, c)
	}
	if !slices.ContainsFunc(r.now, same(c.ID)) {
		r.now = append(r.now, c)
	}
}

func (r *rows) put(c Change) {
	if i := slices.IndexFunc(r.now, same(c.ID)); i >= 0 {
		r.now[i] = c
		return
	}
	r.now = append(r.now, c)
}

func (r *rows) drop(id link.ChangeID) { r.now = slices.DeleteFunc(r.now, same(id)) }

func (r *rows) ops() []Op {
	var deletes, updates, inserts []Op
	for _, w := range r.was {
		if !slices.ContainsFunc(r.now, same(w.ID)) {
			deletes = append(deletes, Op{Delete, w})
		}
	}
	for _, n := range r.now {
		switch i := slices.IndexFunc(r.was, same(n.ID)); {
		case i < 0:
			inserts = append(inserts, Op{Insert, n})
		case !reflect.DeepEqual(r.was[i], n):
			updates = append(updates, Op{Update, n})
		}
	}
	return slices.Concat(deletes, updates, inserts)
}

func (r *rows) fold(open Open, in Change) {
	if in.Kind == link.ChangeLibrary {
		r.library(open, in)
		return
	}
	if in.Kind == link.ChangeMoved {
		r.move(open, in)
		return
	}
	r.refresh(open, in)
}

// library keeps one library row per root: a second adds nothing.
func (r *rows) library(open Open, in Change) {
	if c, ok := open.AtPath[storing.Rel{}]; ok {
		r.read(c)
		return
	}
	r.put(in)
}

func (r *rows) refresh(open Open, in Change) {
	if c := open.Carry; c != nil {
		r.read(*c)
		_, e := effectOf(in.Kind)
		switch {
		case e == remove && c.State == Prepared && c.Kind == link.ChangeReplaced:
			r.prepared(open, *c)
			return
		case e == remove:
			row, ok := r.at(open, gone(*c, in), true)
			r.pair(open, row, ok)
			return
		case c.Path == in.Path:
			if e == replace {
				c.Kind = link.ChangeReplaced
				r.put(*c)
			}
			return
		}
	}
	row, ok := r.at(open, in, false)
	r.pair(open, row, ok)
}

// gone is in, the removal of a carry's file, at the carry's origin: the
// path the server holds the item at. Any row there came after the file
// left it, so gone folds as the earlier change.
func gone(carry, in Change) Change {
	c := carry
	c.Kind, c.Path, c.From = link.ChangeRemoved, carry.From, storing.Rel{}
	c.File, c.ReplacedBy = in.File, in.ReplacedBy
	c.State, c.Step, c.StepAt, c.Snapshot = Due, link.StepNone, time.Time{}, nil
	return c
}

// prepared is a new-name upgrade's doomed file leaving: its prepared carry
// to the replacer becomes the change, and the replacer's added folds in.
func (r *rows) prepared(open Open, c Change) {
	c.State, c.File, c.ReplacedBy = Due, c.ReplacedBy, files.ID{}
	if p := open.Partner; p != nil {
		r.read(*p)
		r.drop(p.ID)
	}
	r.put(c)
}

// at folds in into the refresh row at its path, as the later change, or as
// the earlier when first. It reports the row in became, false when the two
// folded to nothing.
func (r *rows) at(open Open, in Change, first bool) (Change, bool) {
	e, ok := open.AtPath[in.Path]
	if !ok {
		r.put(in)
		return in, true
	}
	r.read(e)
	r.drop(e.ID)
	r.drop(in.ID)
	earlier, later := e, in
	if first {
		earlier, later = in, e
	}
	c, ok := merge(earlier, later)
	if ok {
		r.put(c)
	}
	return c, ok
}

// merge folds later into earlier at one path. The row keeps earlier's place
// in the queue: its id, its age and its backoff.
func merge(earlier, later Change) (Change, bool) {
	_, was := effectOf(earlier.Kind)
	family, e := effectOf(later.Kind)
	m := merged[[2]effect{was, e}]
	if m == nothing {
		return Change{}, false
	}
	c := earlier
	c.Kind, c.File = family[m], later.File
	c.ReplacedBy, c.Upgrade = files.ID{}, false
	switch m {
	case add:
		c.Upgrade = earlier.Upgrade || later.Upgrade
	case remove:
		c.ReplacedBy = later.ReplacedBy
	case nothing, replace:
	}
	return c, true
}

// pair joins c, an upgrade's half as it now stands, with its partner into one
// replaced: a carry from the old file's path when the new file's name
// differs. When the old file's change folded to nothing, the server never
// saw it, and the new file goes as a plain added.
func (r *rows) pair(open Open, c Change, ok bool) {
	p := open.Partner
	if p == nil || (ok && p.ID == c.ID) {
		return
	}
	r.read(*p)
	if !ok {
		p.Upgrade = false
		r.put(*p)
		return
	}
	if c.Kind != link.ChangeAdded && c.Kind != link.ChangeRemoved {
		return
	}
	old, fresh := c, *p
	if c.Kind == link.ChangeAdded {
		old, fresh = *p, c
	}
	j := *p
	j.Kind, j.File, j.Path, j.From = link.ChangeReplaced, fresh.File, fresh.Path, storing.Rel{}
	if old.Path != fresh.Path {
		j.From = old.Path
	}
	j.ReplacedBy, j.Upgrade = files.ID{}, false
	r.drop(c.ID)
	r.put(j)
}

// move folds in, a rename of a file from in.From to in.Path.
func (r *rows) move(open Open, in Change) {
	if c := open.Carry; c != nil {
		r.read(*c)
		switch {
		case c.State == Prepared:
			c.State, c.Path = Due, in.Path
			r.put(*c)
		case in.Path != c.From:
			c.Path = in.Path
			r.put(*c)
		case c.Kind == link.ChangeMoved:
			r.drop(c.ID)
		default:
			back := *c
			back.Path, back.From = c.From, storing.Rel{}
			back.Step, back.StepAt, back.Snapshot = link.StepNone, time.Time{}, nil
			r.at(open, back, false)
		}
		return
	}
	if e, ok := open.AtPath[in.From]; ok && e.Kind != link.ChangeRemoved {
		r.read(e)
		if e.Kind == link.ChangeAdded {
			e.Path = in.Path
			r.at(open, e, false)
			return
		}
		e.From, e.Path = in.From, in.Path
		r.put(e)
		return
	}
	r.put(in)
}
