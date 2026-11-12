package imports

import (
	"context"
	"errors"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	ErrOutside     = errors.New("the folder isn't in one of the Node's library roots or download folders")
	ErrNotOffered  = errors.New("the file doesn't offer that action now")
	ErrMoveSeeding = errors.New("a tracked torrent seeds one of these files")
)

// Opened is what opening a folder answers: its unfinished Imports, and
// whether this opening created them.
type Opened struct {
	IDs     []ImportID
	Created bool
}

// ImportFolder opens the folder abs on node for a manual import: one
// folder Import per Work its files' names match in the library, and one
// with no Work for the files no name matches. Each matched file probes and
// decides as any Import's, then waits confirm with what it would fill. An
// open folder answers the Imports it already has. A folder in a library
// root has its root's Library scan run instead, and answers the root
// Imports of its Work folder. ErrOutside when abs is in none of node's
// library roots and download folders.
func (im *Imports) ImportFolder(ctx context.Context, by identity.Admin, node role.NodeID, abs string) (Opened, error) {
	listed, err := im.cfg.Roots.ListVideos(ctx, node, abs)
	if errors.As(err, new(*storing.OutsideError)) {
		return Opened{}, ErrOutside
	}
	if err != nil {
		return Opened{}, err
	}
	if !listed.Root.IsZero() {
		return im.rootFolder(ctx, by, roots.ID{UUID: listed.Root}, listed.Rel)
	}
	var out Opened
	err = im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		out, err = im.openIn(ctx, tx, node, listed)
		return err
	})
	if err != nil {
		return Opened{}, err
	}
	if out.Created {
		im.cfg.Changed.Publish(ctx, changedOf())
		im.Touch()
	}
	return out, nil
}

// openIn answers the open folder Imports of listed, or creates them.
func (im *Imports) openIn(ctx context.Context, tx *storage.Tx[file], node role.NodeID, listed storing.Listed) (Opened, error) {
	open, err := db.New(tx).ListOpenFolderImports(ctx, db.ListOpenFolderImportsParams{NodeID: node, Folder: &listed.Path})
	if err != nil {
		return Opened{}, err
	}
	var out Opened
	for _, row := range open {
		out.IDs = append(out.IDs, row.ID)
	}
	if len(open) > 0 {
		return out, nil
	}
	imps, err := im.opened(ctx, tx, node, listed)
	if err != nil {
		return Opened{}, err
	}
	for _, imp := range imps {
		if err := insertImport(ctx, tx, imp); err != nil {
			return Opened{}, err
		}
		if _, err := im.writeIn(ctx, tx, imp.ID, &after{}, func(context.Context, *storage.Tx[file], *Import, *after) error { return nil }); err != nil {
			return Opened{}, err
		}
		out.IDs = append(out.IDs, imp.ID)
	}
	out.Created = len(imps) > 0
	return out, nil
}

// rootFolder runs root's Library scan for a folder in it, which places
// its untracked files in place, and answers the root Imports of rel's
// Work folder.
func (im *Imports) rootFolder(ctx context.Context, by identity.Admin, root roots.ID, rel storing.Rel) (Opened, error) {
	if _, err := im.Scan(ctx, by, root); err != nil {
		return Opened{}, err
	}
	folder, _ := folderOf(rel.String())
	var out Opened
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		e, err := rootImportAt(ctx, r, db.GetRootImportAtParams{RootID: root, Folder: &folder})
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		out.IDs = []ImportID{e.ID}
		return err
	})
	return out, err
}

// opened is the folder Imports of listed: its files by the Work their
// names match, each Import's files at positions in listing order.
func (im *Imports) opened(ctx context.Context, tx *storage.Tx[file], node role.NodeID, listed storing.Listed) ([]Import, error) {
	rules, err := RulesIn(ctx, tx)
	if err != nil {
		return nil, err
	}
	m, err := im.matcher(ctx, tx)
	if err != nil {
		return nil, err
	}
	byWork, parts := map[works.WorkID]*Import{}, map[works.WorkID][]int{}
	var order []works.WorkID
	for _, lf := range listed.Files {
		var part int
		base := path.Base(strings.ReplaceAll(lf.Path, `\`, "/"))
		f := File{
			Path: lf.Path, Size: lf.Size, Seen: &lf.Seen, Seeded: lf.Seeded, ReleaseName: strings.TrimSuffix(base, path.Ext(base)), SourceName: base,
			Outcome: Pending,
		}
		t, work := m.match(f.ReleaseName, parentName(lf.Path, listed.Path))
		switch {
		case work.IsZero():
			f.Outcome, f.Reason = Waiting, &Reason{Code: NoMatch}
		case lf.Size < rules.Smallest:
			f.Outcome, f.Reason = LeftOut, &Reason{Code: TooSmall, Value: rules.Smallest}
		default:
			place, err := im.cfg.Facets[t].Place(ctx, tx, work)
			if err != nil {
				return nil, err
			}
			read := place(base)
			items, ok := read.Items()
			f.Items, part = items, read.Part()
			if !ok {
				f.Outcome, f.Reason = Waiting, &Reason{Code: NoRule}
			}
		}
		imp, ok := byWork[work]
		if !ok {
			if work.IsZero() {
				t = m.types[0]
			}
			imp = &Import{
				ID: ImportID{UUID: storage.NewUUID()}, Source: FromFolder, Node: node, Folder: listed.Path, ContentType: t, Work: work,
				Hint: im.cfg.Facets[t].Hint(f.ReleaseName), Created: storage.Now(),
			}
			byWork[work] = imp
			order = append(order, work)
		}
		f.Position = len(imp.Files)
		imp.Files = append(imp.Files, f)
		parts[work] = append(parts[work], part)
	}
	out := make([]Import, len(order))
	for i, w := range order {
		waitParts(byWork[w].Files, parts[w])
		out[i] = *byWork[w]
	}
	return out, nil
}

// parentName is the name of the folder p lies in when that folder is
// below top, empty when p lies in top itself.
func parentName(p, top string) string {
	dir := path.Dir(strings.ReplaceAll(p, `\`, "/"))
	if dir == path.Clean(strings.ReplaceAll(top, `\`, "/")) {
		return ""
	}
	return path.Base(dir)
}

// matcher matches a name to a library Work by the library rules of each
// Content Type, in type order.
type matcher struct {
	im      *Imports
	types   []works.TypeID
	shelves map[works.TypeID]*scanning.Shelf
}

func (im *Imports) matcher(ctx context.Context, r storage.Reader[file]) (matcher, error) {
	m := matcher{im: im, types: slices.Sorted(maps.Keys(im.cfg.Facets)), shelves: map[works.TypeID]*scanning.Shelf{}}
	for _, t := range m.types {
		ws, err := im.shelved(ctx, r, t)
		if err != nil {
			return matcher{}, err
		}
		m.shelves[t] = scanning.NewShelf(roots.ID{}, ws, nil)
	}
	return m, nil
}

// match is the type and Work the first of names that matches one names,
// the zero Work when none does.
func (m matcher) match(names ...string) (t works.TypeID, work works.WorkID) {
	for _, name := range names {
		if name == "" {
			continue
		}
		for _, t := range m.types {
			if w := m.shelves[t].Match("", m.im.cfg.Facets[t].Hint(name)).Work; !w.IsZero() {
				return t, w
			}
		}
	}
	return "", works.WorkID{}
}

// SetMoveFiles turns Move files on or off for the folder Imports ids. On
// is refused whole with ErrMoveSeeding when a torrent held a file of one
// that isn't final yet, and ErrNotOffered for an Import of no folder.
func (im *Imports) SetMoveFiles(ctx context.Context, _ identity.Admin, ids []ImportID, on bool) error {
	return im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		for _, id := range ids {
			_, err := im.writeIn(ctx, tx, id, &after{}, func(_ context.Context, _ *storage.Tx[file], cur *Import, _ *after) error {
				switch {
				case cur.Source != FromFolder:
					return ErrNotOffered
				case on && slices.ContainsFunc(cur.Files, func(f File) bool { return f.Seeded && !f.Outcome.Final() }):
					return ErrMoveSeeding
				}
				cur.MoveFiles = on
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// removeSource deletes a moved file's source once it is placed, and
// records that it went.
func (im *Imports) removeSource(ctx context.Context, imp Import, f File) error {
	if err := im.cfg.Roots.RemoveSource(ctx, imp.Node, f.Path); err != nil {
		return err
	}
	_, err := im.write(ctx, imp.ID, func(_ context.Context, _ *storage.Tx[file], cur *Import, _ *after) error {
		fileAt(cur, f.Position).SourceRemoved = storage.Now()
		return nil
	})
	return err
}
