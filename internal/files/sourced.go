package files

import (
	"context"
	"path"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Sourced is one made-from cell: its source Target and grace, the file
// filling it, and the downloaded file behind its source cell. Items are
// that file's, or the cell's Item while there is none.
type Sourced struct {
	Work     works.WorkID
	Items    []works.ItemID
	Edition  targets.Edition
	Target   targets.TargetID
	Sourcing targets.Sourcing
	From     targets.TargetID
	// Grace is a "Downloaded, or made from" cell's wait for a download.
	Grace             time.Duration
	Counted, Released bool
	File              *File
	// SourceFile is the live file filling the source cell, or the one a
	// made file there was made from; nil while there is none.
	SourceFile *File
}

// Source is the file it is made from.
func (s Sourced) Source() (File, bool) {
	if s.SourceFile == nil {
		return File{}, false
	}
	return *s.SourceFile, true
}

// FilledByDownload reports whether a download fills the cell.
func (s Sourced) FilledByDownload() bool { return s.File != nil && s.File.Derived == nil }

// Located is a file with where it lies: its root, the Node holding that
// root, its path in it and the volume its Node last read it on, "" until
// one did. Name is its file name.
type Located struct {
	ID     ID
	Root   roots.ID
	Rel    storing.Rel
	Node   role.NodeID
	Volume string
	Name   string
	Size   int64
	Media  media.Record
	Read   release.Read
}

// Sourced is the made-from cells of ws, read in r, by Work in tree order
// then Profile order. One read per batch.
func (f *Files) Sourced(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID][]Sourced, error) {
	trees, err := works.OutlinesIn(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	bares := make([]works.Bare, len(trees))
	ids := make([]works.WorkID, len(trees))
	for i, t := range trees {
		bares[i], ids[i] = works.BareOf(t), t.ID
	}
	worlds, err := f.worlds(ctx, r, bares, works.Like(ws, ids), wholeFiles)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID][]Sourced, len(worlds))
	for _, w := range worlds {
		if s := sourcedOf(w, cells(w)); len(s) > 0 {
			out[w.tree.ID] = s
		}
	}
	return out, nil
}

// sourcedOf is the made-from cells of cs, w's cells.
func sourcedOf(w world, cs []Cell) []Sourced {
	at := make(map[CellKey]Cell, len(cs))
	grace := make(map[targets.TargetID]time.Duration, len(w.goals.Profile.Entries))
	for _, c := range cs {
		at[c.key()] = c
	}
	for _, e := range w.goals.Profile.Entries {
		grace[e.Target] = e.Grace
	}
	byID := w.byID()
	var out []Sourced
	for _, c := range cs {
		if c.From == nil {
			continue
		}
		s := Sourced{
			Work: w.tree.ID, Items: []works.ItemID{c.Item.ID}, Edition: c.Edition, Target: c.Target.ID, Sourcing: c.Sourcing, From: *c.From,
			Grace: grace[c.Target.ID], Counted: c.Item.Counts(), Released: c.Released, File: c.File,
		}
		if src := sourceOf(w, byID, at, c); src != nil {
			s.SourceFile, s.Items = src, src.Items
		}
		out = append(out, s)
	}
	return out
}

// sourceOf is the live file c, a made-from cell of w, is made from: the
// file filling its source cell in at, or the one a made file there was
// made from. byID is w's files.
func sourceOf(w world, byID map[ID]*File, at map[CellKey]Cell, c Cell) *File {
	src := at[CellKey{Item: c.Item.ID, Edition: c.Edition, Target: *c.From}].File
	if src != nil && src.Derived != nil {
		src = byID[src.Derived.From]
	}
	if src == nil || src.Gone() || w.lost[src.RootID] {
		return nil
	}
	return src
}

// Located is the live file id names, read in r: ErrNoFile when no file
// has it or its bytes are gone.
func (f *Files) Located(ctx context.Context, r storage.Reader[file], id ID) (Located, error) {
	rows, err := db.New(r).ListFilesByID(ctx, []ID{id})
	if err != nil {
		return Located{}, err
	}
	if len(rows) == 0 || !rows[0].GoneAt.IsZero() {
		return Located{}, ErrNoFile
	}
	g := fileOf(rows[0])
	root, err := roots.Get(ctx, r, g.RootID)
	if err != nil {
		return Located{}, err
	}
	rel, err := storing.ParseRel(g.Path)
	if err != nil {
		return Located{}, err
	}
	out := Located{ID: g.ID, Root: g.RootID, Rel: rel, Node: root.Node, Name: path.Base(g.Path), Size: g.Size, Media: g.Media, Read: g.Title}
	if g.Seen != nil {
		out.Volume = g.Seen.ID.Volume
	}
	return out, nil
}
