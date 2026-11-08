// Package imports owns what a library scan finds and can't place: Pending
// imports and their actions, each root's last comparison, the Library
// Health category and the scan's settings. S33 (Import pipeline) adds
// downloads as a second source of pending entries.
package imports

import (
	"context"
	"embed"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("imports", schemaFiles) }

type EntryID = db.EntryID

const TaskID tasks.TaskID = "library-scan"

type Changed struct {
	RootIDs []string `json:"rootIds" nullable:"false"`
}

func changedOf(ids ...roots.ID) Changed {
	out := Changed{RootIDs: make([]string, len(ids))}
	for i, id := range ids {
		out.RootIDs[i] = id.String()
	}
	return out
}

// ChangedEvent names the roots whose pending imports changed.
var ChangedEvent = live.AdminEvent[Changed]("imports.changed")

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

type Config struct {
	Library *storage.DB[file]
	Roots   *roots.Roots
	Files   *files.Files
	Works   *works.Works
	Facets  map[works.TypeID]scanning.Facet
	// Add is library.Add. The library is built after imports, which is its
	// Leaver.
	Add func(ctx context.Context, by identity.Admin, a library.Adding) (library.Added, error)
	// Steps is the Scheduler's, which opens after imports.
	Steps        func(task tasks.TaskID) (run tasks.RunID, steps map[tasks.UnitKey]tasks.Step, open bool)
	Changed      Publisher[Changed]
	FilesChanged Publisher[files.Changed]
}

type Imports struct {
	cfg Config
}

func New(cfg Config) *Imports { return &Imports{cfg: cfg} }

func unitKey(root roots.ID) tasks.UnitKey { return tasks.UnitKey("root:" + root.String()) }

// folderOf is a path's Work folder: its first folder, or the path itself
// when it lies loose at the root's top.
func folderOf(path string) (folder string, loose bool) {
	if folder, _, ok := strings.Cut(path, "/"); ok {
		return folder, false
	}
	return path, true
}

// inside is a path inside its Work folder, as a Placer reads it: a loose
// file's own name.
func inside(path string) string {
	if _, rest, ok := strings.Cut(path, "/"); ok {
		return rest
	}
	return path
}
