package files

import (
	"cmp"
	"context"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Listed is one library file of a Work with the cells it fills, in cell
// order, none for a Kept file, and how its last import placed it, "" for
// a file no import recorded.
type Listed struct {
	File
	Fills    []CellKey
	PlacedBy string
}

// Listing is a Work's library files and the folder under its root they
// sit in.
type Listing struct {
	Folder string
	Files  []Listed
}

// WorkFiles lists every library file of work, works.ErrNoWork when no
// Work has the id: the files that fill a cell first, in Profile order of
// the first Target each fills, then the rest, Kept files among them,
// oldest first. Folder is the first file's, or while the Work has none the
// one its folder template gives it.
func (f *Files) WorkFiles(ctx context.Context, _ identity.Admin, work works.WorkID) (Listing, error) {
	var out Listing
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		w, err := f.load(ctx, r, work)
		if err != nil {
			return err
		}
		out = listing(w)
		if len(out.Files) > 0 {
			return placedBy(ctx, db.New(r), out.Files)
		}
		scheme, err := f.cfg.Naming.SchemeIn(ctx, r, w.tree.ContentType)
		if err != nil {
			return err
		}
		values, err := f.cfg.Naming.ValuesIn(ctx, r, w.tree.ContentType, works.Only([]works.WorkID{work}))
		if err != nil {
			return err
		}
		out.Folder, err = scheme.Folder(values[work].Work)
		return err
	})
	return out, err
}

func listing(w world) Listing {
	fills := map[ID][]CellKey{}
	for _, c := range cells(w) {
		if fill, ok := w.fills[c.key()]; ok {
			fills[fill.File] = append(fills[fill.File], c.key())
		}
	}
	order := make(map[ID]int, len(w.files))
	for _, f := range w.files {
		order[f.ID] = len(w.goals.Targets)
		for _, k := range fills[f.ID] {
			order[f.ID] = min(order[f.ID], slices.IndexFunc(w.goals.Targets, func(t targets.Target) bool { return t.ID == k.Target }))
		}
	}
	out := Listing{Files: make([]Listed, len(w.files))}
	for i, f := range w.files {
		out.Files[i] = Listed{File: f, Fills: fills[f.ID]}
	}
	slices.SortStableFunc(out.Files, func(a, b Listed) int { return cmp.Compare(order[a.ID], order[b.ID]) })
	if len(out.Files) > 0 {
		out.Folder = path.Dir(out.Files[0].Path)
	}
	return out
}

// placedBy sets each of fs's PlacedBy from its last import.
func placedBy(ctx context.Context, q *db.Queries, fs []Listed) error {
	ids := make([]ID, len(fs))
	for i, f := range fs {
		ids[i] = f.ID
	}
	rows, err := q.ListPlacedByOfFiles(ctx, ids)
	if err != nil {
		return err
	}
	by := make(map[ID]string, len(rows))
	for _, row := range rows {
		by[row.FileID] = row.PlacedBy
	}
	for i := range fs {
		fs[i].PlacedBy = by[fs[i].ID]
	}
	return nil
}
