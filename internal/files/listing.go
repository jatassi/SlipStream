package files

import (
	"cmp"
	"context"
	"path"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Listed is one library file of a Work with the cells it fills, in cell
// order, none for a Kept file.
type Listed struct {
	File
	Fills []CellKey
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
// one its naming template gives its first Item.
func (f *Files) WorkFiles(ctx context.Context, _ identity.Admin, work works.WorkID) (Listing, error) {
	var out Listing
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		w, err := f.load(ctx, r, work)
		if err != nil {
			return err
		}
		out = listing(w)
		if len(out.Files) > 0 || len(w.tree.Items) == 0 {
			return nil
		}
		facet := f.cfg.Naming[w.tree.ContentType]
		tokens, err := facet.NamingTokens(ctx, r, w.tree.Items[0].ID)
		if err != nil {
			return err
		}
		out.Folder, err = naming.Render(facet.DefaultTemplates().Folder, tokens, naming.Core{})
		return err
	})
	return out, err
}

func listing(w world) Listing {
	fills := map[ID][]CellKey{}
	for _, c := range cells(w) {
		if id, ok := w.fills[c.key()]; ok {
			fills[id] = append(fills[id], c.key())
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
