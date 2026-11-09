package files

import (
	"context"
	"errors"
	"maps"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/media"
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
	// Seen is what placing the bytes at Holding read of them.
	Seen    storing.Seen
	Media   media.Record
	Quality release.Quality
	// ReleaseName is its Grab's Release title, and Title its reading.
	ReleaseName string
	Title       release.Read
	GrabID      storage.UUID
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

// placed is a Work converged: the world with its new files and without
// the files it dooms or forgets, and the fills and names refill and names
// give that world.
type placed struct {
	before world
	after  world
	file   File
	fills  map[CellKey]Fill
	doomed []File
	// forgot are gone files whose rows go: no bytes are left to remove.
	forgot []File
	names  map[ID]name
}

// rule is how a converge treats the files a refill leaves behind.
type rule struct {
	// doom says whether a file that lost its last fill, or the fresh file
	// filling none, is deleted; false keeps it as a Kept file.
	doom func(File) bool
	// mode is which files take a path from the templates: a library scan
	// never renames.
	mode renaming
	// forgetGone forgets each gone file that would win back no cell if it
	// returned.
	forgetGone bool
}

// importing is an import's rule: it dooms the files the fresh file
// displaces, and the fresh file itself when it fills nothing and its
// Grab's choice is Delete.
func importing(fresh ID, u *Unwanted) rule {
	return rule{doom: func(g File) bool { return g.ID != fresh || u != nil && *u == Delete }, forgetGone: true}
}

// changing is a Targets change's rule: the files it leaves filling
// nothing go as m chose.
func changing(m Mode) rule { return rule{doom: func(File) bool { return m.Unwanted == Delete }} }

// scanned is a library scan's rule: it never dooms a file and never
// renames one.
var scanned = rule{doom: func(File) bool { return false }, mode: keepEvery}

// place converges in's Work with in added, in r.
func (f *Files) place(ctx context.Context, r storage.Reader[file], in Incoming) (placed, error) {
	before, err := f.load(ctx, r, in.WorkID)
	if err != nil {
		return placed{}, err
	}
	fresh := File{
		ID: ID{UUID: storage.NewUUID()}, WorkID: in.WorkID, ItemID: in.ItemID, Cut: in.Cut, RootID: in.RootID, Path: in.Holding,
		Size: in.Size, Seen: &in.Seen, Media: in.Media, GrabID: in.GrabID, Quality: in.Quality, ReleaseName: in.ReleaseName, Title: in.Title,
	}
	after := before
	after.files = append(slices.Clone(before.files), fresh)
	after.fresh, after.beside = fresh.ID, before.files
	after.seen = maps.Clone(before.seen)
	if !slices.Contains(after.seen[in.ItemID], in.Cut) {
		after.seen[in.ItemID] = append(slices.Clone(after.seen[in.ItemID]), in.Cut)
	}
	if after.meant, err = targets.NameOfFirst(ctx, r, in.Meant); err != nil {
		return placed{}, err
	}
	p, err := f.converge(ctx, r, before, after, refill(after), importing(fresh.ID, in.Unwanted))
	p.file = fresh
	return p, err
}

// converge is a Work going from before to after, whose files are before's
// and any fresh or found ones, refilled as fills: a file that filled a
// cell before and fills none after, or the fresh file filling none, goes
// when the rule dooms it and is named Kept otherwise. A gone file is
// forgotten when the rule forgets those that would win nothing back, and
// when a live file takes its path, where its bytes could no longer
// return. Every file is named under fills and after's cut names.
func (f *Files) converge(ctx context.Context, r storage.Reader[file], before, after world, fills map[CellKey]Fill, how rule) (placed, error) {
	p := placed{before: before, after: after, fills: fills}
	p.after.retagged = retagged(before.cuts, after.cuts)
	p.after.mode = how.mode
	was, is := filling(before.fills), filling(fills)
	p.after.files = slices.DeleteFunc(slices.Clone(after.files), func(g File) bool {
		lost := g.ID == after.fresh || was[g.ID]
		if !lost || is[g.ID] || !how.doom(g) {
			return false
		}
		p.doomed = append(p.doomed, g)
		return true
	})
	if how.forgetGone {
		forget(&p, func(g File) bool { return !winsBack(p.after, fills, g) })
	}
	if err := f.named(ctx, r, &p.after, p.fills); err != nil {
		return placed{}, err
	}
	var err error
	if p.names, _, err = names(p.after, p.fills); err != nil {
		return placed{}, err
	}
	taken := map[place]bool{}
	for _, g := range p.after.files {
		if !g.Gone() {
			taken[place{g.RootID, p.names[g.ID].Path}] = true
		}
	}
	forget(&p, func(g File) bool { return taken[place{g.RootID, g.Path}] })
	return p, nil
}

// forget takes the gone files which picks out of p's world: their rows
// go when p is applied.
func forget(p *placed, which func(File) bool) {
	p.after.files = slices.DeleteFunc(p.after.files, func(g File) bool {
		if !g.Gone() || !which(g) {
			return false
		}
		p.forgot = append(p.forgot, g)
		return true
	})
}

// winsBack reports whether g, a gone file of w, would fill a cell were it
// back, with fills' files holding their cells on a tie.
func winsBack(w world, fills map[CellKey]Fill, g File) bool {
	w.files = slices.Clone(w.files)
	for i := range w.files {
		if w.files[i].ID == g.ID {
			w.files[i].GoneAt = storage.Time{}
		}
	}
	w.fills = fills
	return filling(refill(w))[g.ID]
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
		if p.fills[CellKey{Item: in.ItemID, Edition: e, Target: t.ID}].File == p.file.ID {
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

// apply writes p: its fills and their scores, its dooms, the gone files
// it forgets, its found files, every name that changed, and a move for
// each file whose bytes must reach a new path. A row's path changes in two
// steps, through the file's holding path, so files may trade names in one
// Write; the disk queue moves the bytes straight unless they trade paths.
func apply(ctx context.Context, q *db.Queries, p placed) error {
	if err := unfill(ctx, q, p); err != nil {
		return err
	}
	if err := drop(ctx, q, p); err != nil {
		return err
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
	for key, fill := range p.fills {
		if p.before.fills[key] != fill {
			err := q.InsertFill(ctx, db.InsertFillParams{FileID: fill.File, TargetID: key.Target, ItemID: key.Item, Edition: key.Edition, Score: int64(fill.Score)})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// drop dooms p's doomed files and deletes the rows of the gone files it
// forgets.
func drop(ctx context.Context, q *db.Queries, p placed) error {
	for _, d := range p.doomed {
		if err := doom(ctx, q, d); err != nil {
			return err
		}
	}
	for _, g := range p.forgot {
		if err := q.DeleteFile(ctx, g.ID); err != nil {
			return err
		}
	}
	return nil
}

// unfill deletes every fill p changes or rescores, before any file it
// held goes.
func unfill(ctx context.Context, q *db.Queries, p placed) error {
	for key, fill := range p.before.fills {
		if p.fills[key] != fill {
			if err := q.DeleteFill(ctx, db.DeleteFillParams{ItemID: key.Item, Edition: key.Edition, TargetID: key.Target}); err != nil {
				return err
			}
		}
	}
	return nil
}

// rename writes g's new label and N, and reports whether its path changes:
// then g's row holds its holding path until apply's last pass names it,
// and a move takes its bytes there.
func rename(ctx context.Context, q *db.Queries, p placed, g File) (bool, error) {
	nm := p.names[g.ID]
	switch {
	case g.ID == p.after.fresh:
		g.Path, g.Label, g.Extra = holding(g), nm.Label, nm.Extra
		if err := insertFile(ctx, q, g); err != nil {
			return false, err
		}
		return true, q.InsertMove(ctx, db.InsertMoveParams{FileID: g.ID, Sits: p.file.Path})
	case p.after.found[g.ID]:
		g.Label, g.Extra = nm.Label, nm.Extra
		return false, insertFile(ctx, q, g)
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
	case at.Sits == to:
		_, err := q.DeleteMove(ctx, db.DeleteMoveParams{FileID: g.ID, Sits: at.Sits})
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
		moving = &at.Sits
	case !errors.Is(err, storage.ErrNoRows):
		return err
	}
	if err := q.InsertDoomed(ctx, db.InsertDoomedParams{FileID: d.ID, RootID: d.RootID, Path: d.Path, Sits: moving}); err != nil {
		return err
	}
	return q.DeleteFile(ctx, d.ID)
}
