// Package roots owns the library roots: one folder per Content Type that
// its Works' files are placed under.
package roots

import (
	"context"
	"embed"
	"errors"
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

// Root is the folder on Node that holds ContentType's Works.
type Root struct {
	ID          ID
	ContentType works.TypeID
	Node        role.NodeID
	Path        string
}

type Changed struct{}

var ChangedEvent = live.AdminEvent[Changed]("roots.changed")

type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

var (
	ErrNoType = errors.New("no Content Type has that id")
	ErrExists = errors.New("the Content Type already has a root")
)

// UnusableError is a folder that can't be a root, and why.
type UnusableError struct{ Reason string }

func (e *UnusableError) Error() string { return "the folder can't be a root: " + e.Reason }

type Config struct {
	Library *storage.DB[file]
	// Types is every Content Type this binary registers.
	Types []works.TypeID
	// Self is this Node, which holds every root.
	Self    role.NodeID
	Changed Publisher
}

type Roots struct{ cfg Config }

func New(cfg Config) *Roots { return &Roots{cfg: cfg} }

// Create makes path the root of t once it proves usable: absolute, an
// existing folder, and one this Node can create and delete a file in.
func (r *Roots) Create(ctx context.Context, _ identity.Admin, t works.TypeID, path string) (Root, error) {
	if !slices.Contains(r.cfg.Types, t) {
		return Root{}, ErrNoType
	}
	if err := usable(path); err != nil {
		return Root{}, err
	}
	root := Root{ID: ID{UUID: storage.NewUUID()}, ContentType: t, Node: r.cfg.Self, Path: filepath.Clean(path)}
	err := r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		switch has, err := q.HasRoot(ctx, t); {
		case err != nil:
			return err
		case has:
			return ErrExists
		}
		return q.InsertRoot(ctx, db.InsertRootParams{ID: root.ID, ContentType: t, NodeID: root.Node, Path: root.Path})
	})
	if err != nil {
		return Root{}, err
	}
	r.cfg.Changed.Publish(ctx, Changed{})
	return root, nil
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
		return err
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

// List lists every root by Content Type.
func (r *Roots) List(ctx context.Context, _ identity.Admin) ([]Root, error) {
	rows, err := db.New(r.cfg.Library).ListRoots(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Root, len(rows))
	for i, row := range rows {
		out[i] = Root{ID: row.ID, ContentType: row.ContentType, Node: row.NodeID, Path: row.Path}
	}
	return out, nil
}
