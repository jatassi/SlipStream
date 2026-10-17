// Package files owns the library's files, the Targets each fills, their
// names, and the disk queue that moves and removes them. Its pure core is
// three functions over one Work's world: cells, refill and names.
package files

import (
	"cmp"
	"context"
	"embed"
	"log/slog"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/naming"
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
	ID    = db.FileID
	Probe = db.Probe
)

// File is one library file of an Item. Path is where it belongs under its
// root, '/' separated; until the disk queue moves it there, its bytes sit
// where only the queue looks.
type File struct {
	ID     ID
	WorkID works.WorkID
	ItemID works.ItemID
	// Cut is the file's own cut, which its name tags; the cells it can
	// fill are its Work's list's.
	Cut    release.CutKey
	RootID roots.ID
	Path   string
	Size   int64
	Probe  Probe
	// GrabID is the Grab that brought it, zero for none.
	GrabID storage.UUID
	// Quality is release.File's over its Grab's release title, or its own
	// name, and its probe.
	Quality release.Quality
	// Label is the Target name its {Target} renders, before any suffix.
	Label string
	// Extra is N in " (extra N)", set exactly when the file fills nothing.
	Extra int64
}

// Kept reports whether f fills no Target and stays, flagged.
func (f File) Kept() bool { return f.Extra > 0 }

// Facts is what f's row knows about it, for Judge: its quality, cut and
// size, and the video codec its probe read.
func (f File) Facts() release.Facts {
	return release.Facts{Quality: release.Known(f.Quality), Cut: f.Cut, Video: release.VideoCodecOf(f.Probe.Codec), Size: release.Known(f.Size)}
}

type Changed struct {
	WorkID string `json:"workId" format:"uuid"`
}

var ChangedEvent = live.AdminEvent[Changed]("files.changed")

// Add records f, whose bytes are already at its Path, filling each of
// fills for its Item at the Edition its cut fills, and labels it after
// the first of them in its Work's Profile order. It trusts fills: an
// import goes through Import, which decides them.
func Add(ctx context.Context, tx *storage.Tx[file], f File, fills []targets.TargetID) (File, error) {
	goals, err := targets.GoalsIn(ctx, tx, f.WorkID)
	if err != nil {
		return File{}, err
	}
	f.ID = ID{UUID: storage.NewUUID()}
	for _, t := range goals.Targets {
		if slices.Contains(fills, t.ID) {
			f.Label = t.Name
			break
		}
	}
	q := db.New(tx)
	if err := insertFile(ctx, q, f); err != nil {
		return File{}, err
	}
	for _, t := range fills {
		err := q.InsertFill(ctx, db.InsertFillParams{FileID: f.ID, TargetID: t, ItemID: f.ItemID, Edition: goals.Editions.Filling(f.Cut)})
		if err != nil {
			return File{}, err
		}
	}
	return f, nil
}

func insertFile(ctx context.Context, q *db.Queries, f File) error {
	return q.InsertFile(ctx, db.InsertFileParams{
		ID: f.ID, WorkID: f.WorkID, ItemID: f.ItemID, Cut: f.Cut, RootID: f.RootID, Path: f.Path, Size: f.Size, Probe: f.Probe,
		GrabID: f.GrabID, Quality: f.Quality, Label: f.Label, Extra: f.Extra,
	})
}

type State uint8

// State is a cell's state, in the glossary's order.
const (
	Unreleased State = iota + 1
	Missing
	Downloading
	Failed
	Upgradable
	Met
)

var stateNames = [...]string{Unreleased: "unreleased", Missing: "missing", Downloading: "downloading", Failed: "failed", Upgradable: "upgradable", Met: "met"}

func (s State) String() string { return stateNames[s] }

// CellKey names one cell: an Edition at a Target of an Item.
type CellKey struct {
	Item    works.ItemID
	Edition targets.Edition
	Target  targets.TargetID
}

// Cell is one Edition at one Target of one Item, in the State computed on
// read. File is set when a file fills the cell: Met once the file's
// quality is at or past the Target's cutoff, Upgradable below it. Timing
// is the Item's, and Released whether it is out today.
type Cell struct {
	Item     works.Item
	Edition  targets.Edition
	Target   targets.Target
	State    State
	File     *File
	Timing   targets.Timing
	Released bool
}

// Wanted is whether a search should try to fill the cell: a missing,
// failed or upgradable cell of a monitored, released Item.
func (c Cell) Wanted() bool {
	return c.Item.Monitored && c.Released && (c.State == Missing || c.State == Failed || c.State == Upgradable)
}

// Status rolls one Item's cells up to the first State any holds, in the
// glossary's order, false for an Item with no cell.
func Status(cells []Cell) (State, bool) {
	if len(cells) == 0 {
		return 0, false
	}
	return slices.MinFunc(cells, func(a, b Cell) int { return cmp.Compare(a.State, b.State) }).State, true
}

// GrabState is what the latest Grab for a cell makes of it.
type GrabState uint8

const (
	GrabDownloading GrabState = iota + 1
	GrabFailed
)

// Inflight is files' view of the Grabs, which grabs implements.
type Inflight interface {
	// GrabCells reads the latest Grab state of each cell of ws that a Grab
	// is in flight for or failed.
	GrabCells(ctx context.Context, r storage.Reader[storage.Library], ws []works.WorkID) (map[CellKey]GrabState, error)
	// Unfinished lists the ids of every Grab not yet imported or failed,
	// whose download may sit in a root's holding folder.
	Unfinished(ctx context.Context, r storage.Reader[storage.Library]) ([]storage.UUID, error)
	// Settle records u on each unfinished Grab of work whose every cell is
	// in gone, for its probe or import to follow if its file fills
	// nothing, and counts them.
	Settle(ctx context.Context, tx *storage.Tx[storage.Library], work works.WorkID, gone map[CellKey]bool, u Unwanted) (int, error)
}

// NoGrabs is Inflight while nothing grabs.
type NoGrabs struct{}

func (NoGrabs) GrabCells(context.Context, storage.Reader[storage.Library], []works.WorkID) (map[CellKey]GrabState, error) {
	return nil, nil
}

func (NoGrabs) Unfinished(context.Context, storage.Reader[storage.Library]) ([]storage.UUID, error) {
	return nil, nil
}

func (NoGrabs) Settle(context.Context, *storage.Tx[storage.Library], works.WorkID, map[CellKey]bool, Unwanted) (int, error) {
	return 0, nil
}

type Config struct {
	Library *storage.DB[file]
	Works   *works.Works
	Targets *targets.Targets
	Grabs   Inflight
	// Naming names each Content Type's files.
	Naming map[works.TypeID]naming.Facet
	// Today is the time now in the Controller's zone, whose calendar date
	// decides which Items are out.
	Today func() time.Time
	// Disk is what the disk queue moves and removes files with.
	Disk    Disk
	Timings Timings
	Rand    *rand.Rand
	Log     *slog.Logger
}

type Files struct {
	cfg  Config
	poke chan struct{}
}

func New(cfg Config) *Files { return &Files{cfg: cfg, poke: make(chan struct{}, 1)} }

// WorkCells reads a Work's Items and its cells, works.ErrNoWork when it
// has none.
func (f *Files) WorkCells(ctx context.Context, _ identity.Admin, work works.WorkID) ([]works.Item, []Cell, error) {
	var w world
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		w, err = f.load(ctx, r, work)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return w.tree.Items, cells(w), nil
}

// Cells computes a Work's cells in r: each Item in tree order, at each
// Edition it wants in list order, at each Target of the Work's Profile in
// Profile order.
func (f *Files) Cells(ctx context.Context, r storage.Reader[file], work works.WorkID) ([]Cell, error) {
	w, err := f.load(ctx, r, work)
	if err != nil {
		return nil, err
	}
	return cells(w), nil
}
