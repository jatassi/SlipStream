// Package files owns the library's files, the Targets each fills, and the
// cells they and the in-flight Grabs make of a Work.
package files

import (
	"context"
	"embed"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("files", schemaFiles) }

type (
	ID      = db.FileID
	Edition = db.Edition
	Probe   = db.Probe
)

const Standard = db.Standard

// File is one library file of an Item, at Path under its root, '/'
// separated.
type File struct {
	ID      ID
	WorkID  works.WorkID
	ItemID  works.ItemID
	Edition Edition
	RootID  roots.ID
	Path    string
	Size    int64
	Probe   Probe
	// GrabID is the Grab that brought it, zero for none.
	GrabID storage.UUID
	// Quality is release.File's over its Grab's release title, or its own
	// name, and its probe.
	Quality release.Quality
}

// Facts is what f's row knows about it, for Judge.
func (f File) Facts() release.Facts {
	return release.Facts{Quality: release.Known(f.Quality), Cut: release.Standard, Size: release.Known(f.Size)}
}

type Changed struct {
	WorkID string `json:"workId" format:"uuid"`
}

var ChangedEvent = live.AdminEvent[Changed]("files.changed")

// Add records f with ID set, and that it fills each of targets for its
// Item and Edition.
func Add(ctx context.Context, tx *storage.Tx[file], f File, fills []targets.TargetID) (File, error) {
	f.ID = ID{UUID: storage.NewUUID()}
	q := db.New(tx)
	err := q.InsertFile(ctx, db.InsertFileParams{
		ID: f.ID, WorkID: f.WorkID, ItemID: f.ItemID, Edition: f.Edition, RootID: f.RootID, Path: f.Path, Size: f.Size, Probe: f.Probe,
		GrabID: f.GrabID, Quality: f.Quality,
	})
	if err != nil {
		return File{}, err
	}
	for _, t := range fills {
		if err := q.InsertFill(ctx, db.InsertFillParams{FileID: f.ID, TargetID: t, ItemID: f.ItemID, Edition: f.Edition}); err != nil {
			return File{}, err
		}
	}
	return f, nil
}

type State uint8

// State is a cell's state, in the glossary's order.
const (
	Missing State = iota + 1
	Downloading
	Failed
	Upgradable
	Met
)

var stateNames = [...]string{Missing: "missing", Downloading: "downloading", Failed: "failed", Upgradable: "upgradable", Met: "met"}

func (s State) String() string { return stateNames[s] }

// CellKey names one cell: an Edition at a Target of an Item.
type CellKey struct {
	Item    works.ItemID
	Edition Edition
	Target  targets.TargetID
}

// Cell is one Edition at one Target of one Item, in the State computed on
// read. File is set when a file fills the cell: Met once the file's
// quality is at or past the Target's cutoff, Upgradable below it.
type Cell struct {
	Item    works.Item
	Edition Edition
	Target  targets.Target
	State   State
	File    *File
}

// Wanted is whether a search should try to fill the cell. An upgradable
// cell isn't, until an import can replace the file it holds.
func (c Cell) Wanted() bool {
	return c.Item.Monitored && (c.State == Missing || c.State == Failed)
}

// GrabState is what the latest Grab for a cell makes of it.
type GrabState uint8

const (
	GrabDownloading GrabState = iota + 1
	GrabFailed
)

// GrabCells reads the latest Grab state of each of a Work's cells that a
// Grab is in flight for or failed.
type GrabCells interface {
	GrabCells(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (map[CellKey]GrabState, error)
}

// NoGrabs is GrabCells while nothing grabs.
type NoGrabs struct{}

func (NoGrabs) GrabCells(context.Context, storage.Reader[storage.Library], works.WorkID) (map[CellKey]GrabState, error) {
	return nil, nil
}

type Config struct {
	Library *storage.DB[file]
	Works   *works.Works
	Grabs   GrabCells
}

type Files struct{ cfg Config }

func New(cfg Config) *Files { return &Files{cfg: cfg} }

// WorkCells reads a Work's cells, works.ErrNoWork when it has none.
func (f *Files) WorkCells(ctx context.Context, _ identity.Admin, work works.WorkID) ([]Cell, error) {
	var cells []Cell
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		cells, err = f.Cells(ctx, r, work)
		return err
	})
	return cells, err
}

// final reports whether f is at or past t's cutoff. A file whose quality t
// doesn't rank is not.
func final(t targets.Target, f File) bool {
	r, ok := t.RankOf(f.Quality)
	return ok && t.Final(r)
}

// Cells computes a Work's cells in r: each Item in tree order, at each
// Target of its Content Type's default Profile in rank order.
func (f *Files) Cells(ctx context.Context, r storage.Reader[file], work works.WorkID) ([]Cell, error) {
	tree, err := f.cfg.Works.TreeIn(ctx, r, work)
	if err != nil {
		return nil, err
	}
	wanted, err := targets.Wanted(ctx, r, tree.ContentType)
	if err != nil {
		return nil, err
	}
	rows, err := db.New(r).ListFillsOfWork(ctx, work)
	if err != nil {
		return nil, err
	}
	filled := map[CellKey]*File{}
	for _, row := range rows {
		ff := File(row.FilesFile)
		filled[CellKey{Item: ff.ItemID, Edition: ff.Edition, Target: row.TargetID}] = &ff
	}
	grabbed, err := f.cfg.Grabs.GrabCells(ctx, r, work)
	if err != nil {
		return nil, err
	}
	var cells []Cell
	for _, it := range tree.Items {
		for _, t := range wanted {
			key := CellKey{Item: it.ID, Edition: Standard, Target: t.ID}
			c := Cell{Item: it, Edition: Standard, Target: t, State: Missing, File: filled[key]}
			switch {
			case c.File != nil && final(t, *c.File):
				c.State = Met
			case c.File != nil:
				c.State = Upgradable
			case grabbed[key] == GrabDownloading:
				c.State = Downloading
			case grabbed[key] == GrabFailed:
				c.State = Failed
			}
			cells = append(cells, c)
		}
	}
	return cells, nil
}
