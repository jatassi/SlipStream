// Package roots owns the library roots: folders on a Node that hold a
// Content Type's Works, one of each type its default, and the root each
// Work lives in.
package roots

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("roots", schemaFiles) }

type ID = db.RootID

// Root is a folder on Node that holds Works of ContentType.
type Root struct {
	ID          ID
	ContentType works.TypeID
	Node        role.NodeID
	Path        string
	// Default is where its type's new Works go.
	Default bool
}

func rootOf(row db.RootsRoot) Root {
	return Root{ID: row.ID, ContentType: row.ContentType, Node: row.NodeID, Path: row.Path, Default: row.IsDefault}
}

// View is a root as the Admin lists it.
type View struct {
	Root
	Works int64
	Use   Use
}

// Use is the files a root holds, doomed ones counted, and the bytes of the
// live ones.
type Use struct{ Files, Bytes int64 }

type NewRoot struct {
	ContentType works.TypeID
	Path        string
	// MakeDefault makes it its type's default; a type's first root is
	// whatever it says.
	MakeDefault bool
}

type Changed struct{}

var ChangedEvent = live.AdminEvent[Changed]("roots.changed")

type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

var (
	ErrNoType    = errors.New("no Content Type has that id")
	ErrNoRoot    = errors.New("the Work has no root")
	ErrNotFound  = errors.New("no root has that id")
	ErrWrongType = errors.New("the root holds another Content Type's Works")
	ErrHasFiles  = errors.New("the Work has files, so changing its root is a Move")
)

// UnusableError is a folder that can't be a root, and why.
type UnusableError struct{ Reason string }

func (e *UnusableError) Error() string { return "the folder can't be a root: " + e.Reason }

// Contents is what files knows about roots. files.RootContents implements
// it.
type Contents interface {
	// Use counts the files of each of ids and their bytes.
	Use(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]Use, error)
	// HasFiles reports whether work has a file.
	HasFiles(ctx context.Context, r storage.Reader[file], work works.WorkID) (bool, error)
}

type Config struct {
	Library *storage.DB[file]
	// Types is every Content Type this binary registers.
	Types []works.TypeID
	// Self is this Node, which holds every root.
	Self     role.NodeID
	Contents Contents
	Changed  Publisher
}

type Roots struct{ cfg Config }

// Open homes, in one Write, every rootless Work whose type has a default
// root, as the first root of a type does, and publishes only when it
// wrote.
func Open(ctx context.Context, cfg Config) (*Roots, error) {
	homed := false
	err := cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		homed, err = homeRootless(ctx, tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	if homed {
		cfg.Changed.Publish(ctx, Changed{})
	}
	return &Roots{cfg: cfg}, nil
}

// Create makes path a root of n's type once it proves usable: absolute, an
// existing folder, and one this Node can create and delete a file in. A
// type's first root is its default and homes the type's rootless Works in
// the same Write.
func (r *Roots) Create(ctx context.Context, _ identity.Admin, n NewRoot) (View, error) {
	if !slices.Contains(r.cfg.Types, n.ContentType) {
		return View{}, ErrNoType
	}
	if err := usable(n.Path); err != nil {
		return View{}, err
	}
	root := Root{ID: ID{UUID: storage.NewUUID()}, ContentType: n.ContentType, Node: r.cfg.Self, Path: filepath.Clean(n.Path)}
	var view View
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		others, err := q.CountRootsOf(ctx, n.ContentType)
		if err != nil {
			return err
		}
		root.Default = others == 0 || n.MakeDefault
		if root.Default {
			if err := q.ClearDefaultRoot(ctx, n.ContentType); err != nil {
				return err
			}
		}
		err = q.InsertRoot(ctx, db.InsertRootParams{ID: root.ID, ContentType: root.ContentType, NodeID: root.Node, Path: root.Path, IsDefault: root.Default})
		if err != nil {
			return err
		}
		if others == 0 {
			if _, err := homeRootless(ctx, tx); err != nil {
				return err
			}
		}
		views, err := r.views(ctx, tx, []Root{root})
		view = views[0]
		return err
	})
	if err != nil {
		return View{}, err
	}
	r.cfg.Changed.Publish(ctx, Changed{})
	return view, nil
}

// MakeDefault makes root id where its type's new Works go.
func (r *Roots) MakeDefault(ctx context.Context, _ identity.Admin, id ID) error {
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		root, err := Get(ctx, tx, id)
		if err != nil {
			return err
		}
		q := db.New(tx)
		if err := q.ClearDefaultRoot(ctx, root.ContentType); err != nil {
			return err
		}
		return q.SetDefaultRoot(ctx, id)
	})
	if err != nil {
		return err
	}
	r.cfg.Changed.Publish(ctx, Changed{})
	return nil
}

func usable(path string) error {
	if !filepath.IsAbs(path) {
		return &UnusableError{Reason: "the path is not absolute"}
	}
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return &UnusableError{Reason: "no folder is at the path"}
	case err != nil:
		return &UnusableError{Reason: "the folder can't be read: " + cause(err)}
	case !info.IsDir():
		return &UnusableError{Reason: "the path is a file, not a folder"}
	}
	probe, err := os.CreateTemp(path, ".slipstream-check-*")
	if err != nil {
		return &UnusableError{Reason: "SlipStream can't create a file in the folder: " + cause(err)}
	}
	if err := probe.Close(); err != nil {
		return fmt.Errorf("close the check file: %w", err)
	}
	if err := os.Remove(probe.Name()); err != nil {
		return &UnusableError{Reason: "SlipStream can't delete a file in the folder: " + cause(err)}
	}
	return nil
}

func cause(err error) string {
	if pe := (*fs.PathError)(nil); errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

// List lists every root by Content Type, oldest first in each.
func (r *Roots) List(ctx context.Context, _ identity.Admin) ([]View, error) {
	var out []View
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		all, err := All(ctx, rd)
		if err != nil {
			return err
		}
		out, err = r.views(ctx, rd, all)
		return err
	})
	return out, err
}

func (r *Roots) views(ctx context.Context, rd storage.Reader[file], roots []Root) ([]View, error) {
	counts, err := db.New(rd).CountWorksByRoot(ctx)
	if err != nil {
		return nil, err
	}
	homed := make(map[ID]int64, len(counts))
	for _, c := range counts {
		homed[c.RootID] = c.Works
	}
	ids := make([]ID, len(roots))
	for i, root := range roots {
		ids[i] = root.ID
	}
	use, err := r.cfg.Contents.Use(ctx, rd, ids)
	if err != nil {
		return nil, err
	}
	out := make([]View, len(roots))
	for i, root := range roots {
		out[i] = View{Root: root, Works: homed[root.ID], Use: use[root.ID]}
	}
	return out, nil
}

// All lists every root in r by Content Type, oldest first in each.
func All(ctx context.Context, r storage.Reader[file]) ([]Root, error) {
	rows, err := db.New(r).ListRoots(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Root, len(rows))
	for i, row := range rows {
		out[i] = rootOf(row)
	}
	return out, nil
}

// Get reads root id in r, ErrNotFound when no root has it.
func Get(ctx context.Context, r storage.Reader[file], id ID) (Root, error) {
	return found(db.New(r).GetRoot(ctx, id))
}

// OfWork reads the root work lives in, ErrNoRoot while it is rootless.
func OfWork(ctx context.Context, r storage.Reader[file], work works.WorkID) (Root, error) {
	row, err := db.New(r).GetRootOfWork(ctx, work)
	if errors.Is(err, storage.ErrNoRows) {
		return Root{}, ErrNoRoot
	}
	if err != nil {
		return Root{}, err
	}
	return rootOf(row), nil
}

func found(row db.RootsRoot, err error) (Root, error) {
	if errors.Is(err, storage.ErrNoRows) {
		return Root{}, ErrNotFound
	}
	if err != nil {
		return Root{}, err
	}
	return rootOf(row), nil
}
