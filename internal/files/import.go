package files

import (
	"context"
	"errors"
	"maps"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// HoldingOf is the root-relative path where the bytes of the file or Grab
// with id wait, between their download and its name and between an old
// name and a new one: a name only it uses. ext is the file's, with its dot.
func HoldingOf(id storage.UUID, ext string) string {
	return storing.HoldingDir + "/" + id.String() + ext
}

// Incoming is a downloaded file whose bytes sit at Holding under its root,
// placed there before the Write that imports it.
type Incoming struct {
	WorkID works.WorkID
	ItemID works.ItemID
	// Cut is the file's own, its Grab's release's.
	Cut    release.CutKey
	RootID roots.ID
	// Holding is root-relative, '/' separated; its extension is the file's.
	Holding string
	Size    int64
	Probe   Probe
	Quality release.Quality
	GrabID  storage.UUID
	// Meant is the Targets its Grab was meant to fill, in Profile order.
	Meant []targets.TargetID
	// Unwanted is what its Grab's import does if the file fills nothing,
	// as an Admin chose when a save left its cells unwanted; nil when no
	// save did.
	Unwanted *Unwanted
}

// Unwanted is what becomes of a file a change leaves filling nothing: it
// is deleted, or it stays as a Kept file.
type Unwanted string

const (
	Delete Unwanted = "delete"
	Keep   Unwanted = "keep"
)

// Placement is where a probed file would go and the Targets it would
// fill, in Profile order. Kept is a file that would fill none.
type Placement struct {
	Fills []targets.TargetID
	Path  string
	Kept  bool
}

var ErrRootChanged = errors.New("the Work moved to another library root after the file was planned")

// FillsNothingError is a probed file no cell would take.
type FillsNothingError struct{ Quality release.Quality }

func (e *FillsNothingError) Error() string {
	return "a " + e.Quality.Info().Name + " file fills no Target the Work still wants"
}

// placed is a Work converged with one more file: the world with it and
// without the files it dooms, and the fills and names refill and names
// give that world.
type placed struct {
	before world
	after  world
	file   File
	fills  map[CellKey]ID
	doomed []File
	names  map[ID]name
}

// place converges in's Work with in added, in r.
func (f *Files) place(ctx context.Context, r storage.Reader[file], in Incoming) (placed, error) {
	before, err := f.load(ctx, r, in.WorkID)
	if err != nil {
		return placed{}, err
	}
	fresh := File{
		ID: ID{UUID: storage.NewUUID()}, WorkID: in.WorkID, ItemID: in.ItemID, Cut: in.Cut, RootID: in.RootID, Path: in.Holding,
		Size: in.Size, Probe: in.Probe, GrabID: in.GrabID, Quality: in.Quality,
	}
	after := before
	after.files = append(slices.Clone(before.files), fresh)
	after.fresh = fresh.ID
	after.seen = maps.Clone(before.seen)
	if !slices.Contains(after.seen[in.ItemID], in.Cut) {
		after.seen[in.ItemID] = append(slices.Clone(after.seen[in.ItemID]), in.Cut)
	}
	if after.meant, err = targets.NameOfFirst(ctx, r, in.Meant); err != nil {
		return placed{}, err
	}
	p, err := f.converge(ctx, r, before, after, refill(after), func(g File) bool {
		return g.ID != fresh.ID || in.Unwanted != nil && *in.Unwanted == Delete
	})
	p.file = fresh
	return p, err
}

// converge is a Work going from before to after, whose files are before's
// and any fresh one, refilled as fills: a file that filled a cell before
// and fills none after, or the fresh file filling none, goes when doom
// says so and is named Kept otherwise. Every file is named under fills
// and after's cut names.
func (f *Files) converge(ctx context.Context, r storage.Reader[file], before, after world, fills map[CellKey]ID, doom func(File) bool) (placed, error) {
	p := placed{before: before, after: after, fills: fills}
	p.after.retagged = retagged(before.cuts, after.cuts)
	was, is := filling(before.fills), filling(fills)
	p.after.files = slices.DeleteFunc(slices.Clone(after.files), func(g File) bool {
		lost := g.ID == after.fresh || was[g.ID]
		if !lost || is[g.ID] || !doom(g) {
			return false
		}
		p.doomed = append(p.doomed, g)
		return true
	})
	if err := f.named(ctx, r, &p.after, p.fills); err != nil {
		return placed{}, err
	}
	var err error
	p.names, err = names(p.after, p.fills)
	return p, err
}

// retagged is each cut of is whose name differs in was.
func retagged(was, is targets.Cuts) map[release.CutKey]bool {
	out := map[release.CutKey]bool{}
	for _, c := range is.All() {
		if old, ok := was.Get(c.Key); ok && old.Name != c.Name {
			out[c.Key] = true
		}
	}
	return out
}

// Place answers, in r, where in would go and what it would fill, as
// Import would decide now. A file that fills nothing is a
// *FillsNothingError unless its Grab's choice keeps it as a Kept file.
func (f *Files) Place(ctx context.Context, r storage.Reader[file], in Incoming) (Placement, error) {
	p, err := f.place(ctx, r, in)
	if err != nil {
		return Placement{}, err
	}
	var out Placement
	e := p.after.goals.Editions.Filling(in.Cut)
	for _, t := range p.after.goals.Targets {
		if p.fills[CellKey{Item: in.ItemID, Edition: e, Target: t.ID}] == p.file.ID {
			out.Fills = append(out.Fills, t.ID)
		}
	}
	out.Kept = len(out.Fills) == 0
	if out.Kept && (in.Unwanted == nil || *in.Unwanted != Keep) {
		return Placement{}, &FillsNothingError{Quality: in.Quality}
	}
	out.Path = p.names[p.file.ID].Path
	return out, nil
}

// Import records in and converges its Work in tx: fills follow refill, a
// file that loses its last fill to in is doomed (the one delete that
// never asks, ADR 0002), files whose names change are renamed, and in is
// queued to move from Holding to its name. A file that fills nothing is
// doomed when its Grab's choice is Delete and lands Kept otherwise, as a
// race leaves it; Import answers false for a doomed one. The disk queue
// does the IO once tx commits; the caller pokes it. A file planned into a
// root its Work has since left is ErrRootChanged.
func (f *Files) Import(ctx context.Context, tx *storage.Tx[file], in Incoming) (File, bool, error) {
	root, err := roots.OfWork(ctx, tx, in.WorkID)
	switch {
	case errors.Is(err, roots.ErrNoRoot) || err == nil && root.ID != in.RootID:
		return File{}, false, ErrRootChanged
	case err != nil:
		return File{}, false, err
	}
	p, err := f.place(ctx, tx, in)
	if err != nil {
		return File{}, false, err
	}
	if err := apply(ctx, db.New(tx), p); err != nil {
		return File{}, false, err
	}
	nm, landed := p.names[p.file.ID]
	p.file.Path, p.file.Label, p.file.Extra = nm.Path, nm.Label, nm.Extra
	return p.file, landed, nil
}

// apply writes p: its fills, its dooms, every name that changed, and a
// move for each file whose bytes must reach a new path. A path changes
// in two steps, through the file's holding path, so files may trade
// names in one Write.
func apply(ctx context.Context, q *db.Queries, p placed) error {
	if err := unfill(ctx, q, p); err != nil {
		return err
	}
	for _, d := range p.doomed {
		if err := doom(ctx, q, d); err != nil {
			return err
		}
	}
	var moved []File
	for _, g := range p.after.files {
		moves, err := rename(ctx, q, p, g)
		if err != nil {
			return err
		}
		if moves {
			moved = append(moved, g)
		}
	}
	for _, g := range moved {
		nm := p.names[g.ID]
		if err := q.RenameFile(ctx, db.RenameFileParams{ID: g.ID, Path: nm.Path, Label: nm.Label, Extra: nm.Extra}); err != nil {
			return err
		}
	}
	for key, id := range p.fills {
		if p.before.fills[key] != id {
			if err := q.InsertFill(ctx, db.InsertFillParams{FileID: id, TargetID: key.Target, ItemID: key.Item, Edition: key.Edition}); err != nil {
				return err
			}
		}
	}
	return nil
}

// unfill deletes every fill p changes, before any file it held goes.
func unfill(ctx context.Context, q *db.Queries, p placed) error {
	for key, id := range p.before.fills {
		if p.fills[key] != id {
			if err := q.DeleteFill(ctx, db.DeleteFillParams{ItemID: key.Item, Edition: key.Edition, TargetID: key.Target}); err != nil {
				return err
			}
		}
	}
	return nil
}

// rename writes g's new label and N, and reports whether its path changes:
// then g sits at its holding path until apply's last pass names it, and a
// move takes its bytes there.
func rename(ctx context.Context, q *db.Queries, p placed, g File) (bool, error) {
	nm := p.names[g.ID]
	switch {
	case g.ID == p.after.fresh:
		g.Path, g.Label, g.Extra = holding(g), nm.Label, nm.Extra
		if err := insertFile(ctx, q, g); err != nil {
			return false, err
		}
		return true, q.InsertMove(ctx, db.InsertMoveParams{FileID: g.ID, Sits: p.file.Path})
	case nm.Path != g.Path:
		if err := q.RenameFile(ctx, db.RenameFileParams{ID: g.ID, Path: holding(g), Label: nm.Label, Extra: nm.Extra}); err != nil {
			return false, err
		}
		return true, queueMove(ctx, q, g, nm.Path)
	case nm.Label != g.Label || nm.Extra != g.Extra:
		return false, q.RenameFile(ctx, db.RenameFileParams{ID: g.ID, Path: g.Path, Label: nm.Label, Extra: nm.Extra})
	}
	return false, nil
}

// holding is where g's bytes wait between two names.
func holding(g File) string { return HoldingOf(g.ID.UUID, path.Ext(g.Path)) }

// queueMove records that g's bytes, at its old path unless a move already
// took them elsewhere, must reach to. Bytes already at to need no move.
func queueMove(ctx context.Context, q *db.Queries, g File, to string) error {
	at, err := q.GetMove(ctx, g.ID)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return q.InsertMove(ctx, db.InsertMoveParams{FileID: g.ID, Sits: g.Path})
	case err != nil:
		return err
	case at == to:
		_, err := q.DeleteMove(ctx, db.DeleteMoveParams{FileID: g.ID, Sits: at})
		return err
	}
	return nil
}

// doom removes d from the library and queues its bytes' removal from
// wherever they sit: its path, or where its move had taken them.
func doom(ctx context.Context, q *db.Queries, d File) error {
	var moving *string
	at, err := q.GetMove(ctx, d.ID)
	switch {
	case err == nil:
		moving = &at
	case !errors.Is(err, storage.ErrNoRows):
		return err
	}
	if err := q.InsertDoomed(ctx, db.InsertDoomedParams{FileID: d.ID, RootID: d.RootID, Path: d.Path, Sits: moving}); err != nil {
		return err
	}
	return q.DeleteFile(ctx, d.ID)
}
