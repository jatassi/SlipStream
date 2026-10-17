package files

import (
	"context"
	"errors"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// incomingDir is the folder in each root where a file's bytes wait between
// their download and its name, and between an old name and a new one.
const incomingDir = ".slipstream-incoming"

// HoldingOf is the root-relative path where the bytes of the file or Grab
// with id wait: a name only it uses. ext is the file's, with its dot.
func HoldingOf(id storage.UUID, ext string) string { return incomingDir + "/" + id.String() + ext }

// Incoming is a downloaded file whose bytes sit at Holding under its root,
// placed there before the Write that imports it.
type Incoming struct {
	WorkID  works.WorkID
	ItemID  works.ItemID
	Edition Edition
	RootID  roots.ID
	// Holding is root-relative, '/' separated; its extension is the file's.
	Holding string
	Size    int64
	Probe   Probe
	Quality release.Quality
	GrabID  storage.UUID
}

// Placement is where a probed file would go and the Targets it would
// fill, in Profile order. Kept is a file that would fill none.
type Placement struct {
	Fills []targets.TargetID
	Path  string
	Kept  bool
}

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
	p := placed{before: before, file: File{
		ID: ID{UUID: storage.NewUUID()}, WorkID: in.WorkID, ItemID: in.ItemID, Edition: in.Edition, RootID: in.RootID, Path: in.Holding,
		Size: in.Size, Probe: in.Probe, GrabID: in.GrabID, Quality: in.Quality,
	}}
	p.after = before
	p.after.files = append(slices.Clone(before.files), p.file)
	p.after.fresh = p.file.ID
	p.fills = refill(p.after)
	was, is := filling(before.fills), filling(p.fills)
	p.after.files = slices.DeleteFunc(p.after.files, func(g File) bool {
		if g.ID == p.file.ID || !was[g.ID] || is[g.ID] {
			return false
		}
		p.doomed = append(p.doomed, g)
		return true
	})
	if err := f.named(ctx, r, &p.after, p.fills); err != nil {
		return placed{}, err
	}
	p.names, err = names(p.after, p.fills)
	return p, err
}

// Place answers, in r, where in would go and what it would fill, as
// Import would decide now. A file that fills nothing is a
// *FillsNothingError unless keep makes it a Kept file.
func (f *Files) Place(ctx context.Context, r storage.Reader[file], in Incoming, keep bool) (Placement, error) {
	p, err := f.place(ctx, r, in)
	if err != nil {
		return Placement{}, err
	}
	out := Placement{Path: p.names[p.file.ID].Path}
	for _, t := range p.after.goals.Targets {
		if p.fills[CellKey{Item: in.ItemID, Edition: in.Edition, Target: t.ID}] == p.file.ID {
			out.Fills = append(out.Fills, t.ID)
		}
	}
	out.Kept = len(out.Fills) == 0
	if out.Kept && !keep {
		return Placement{}, &FillsNothingError{Quality: in.Quality}
	}
	return out, nil
}

// Import records in and converges its Work in tx: fills follow refill, a
// file that loses its last fill to in is doomed (the one delete that
// never asks, ADR 0002), files whose names change are renamed, and in is
// queued to move from Holding to its name. A file a race left filling
// nothing lands Kept. The disk queue does the IO once tx commits; the
// caller pokes it.
func (f *Files) Import(ctx context.Context, tx *storage.Tx[file], in Incoming) (File, error) {
	p, err := f.place(ctx, tx, in)
	if err != nil {
		return File{}, err
	}
	if err := apply(ctx, db.New(tx), p); err != nil {
		return File{}, err
	}
	nm := p.names[p.file.ID]
	p.file.Path, p.file.Label, p.file.Extra = nm.Path, nm.Label, nm.Extra
	return p.file, nil
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
