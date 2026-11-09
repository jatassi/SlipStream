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
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("files", schemaFiles) }

type ID = db.FileID

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
	Media  media.Record
	// ProbedAt is when a scan last read Media, zero when the file's import
	// brought it. ProbeError is ffprobe's first line when that read failed;
	// Media is then the record before it.
	ProbedAt   storage.Time
	ProbeError string
	// Seen is what a scan or an import last read of its bytes besides
	// their size, nil until one did.
	Seen *storing.Seen
	// GoneAt is when a scan found its bytes absent from its Ready root,
	// zero while they are there. A gone file fills nothing.
	GoneAt storage.Time
	// GrabID is the Grab that brought it, zero for none.
	GrabID storage.UUID
	// Quality is what its import or a scan judged of its Title and its
	// media record.
	Quality release.Quality
	// Label is the Target name its {Target} renders, before any suffix.
	Label string
	// Extra is N in " (extra N)", set exactly when the file fills nothing.
	Extra int64
	// ReleaseName is its Grab's Release title, or its own file name when
	// no Grab brought it, and Title is that name's reading.
	ReleaseName string
	Title       release.Read
	// SourceName is its name, extension included, in its download or
	// where a scan found it, "" when unknown. It never changes.
	SourceName string
}

// Gone reports whether a scan found f's bytes absent from its root.
func (f File) Gone() bool { return !f.GoneAt.IsZero() }

// Kept reports whether f fills no Target and stays, flagged. A gone file
// is gone, not Kept.
func (f File) Kept() bool { return f.Extra > 0 && !f.Gone() }

// Facts is f as Judge reads it: its Title's facts with what its media
// record measured winning, at the quality its import or scan judged, its
// cut and its size.
func (f File) Facts() release.Facts {
	facts := f.Title.Facts.WithProbe(f.Media.Probed())
	facts.Quality, facts.Cut, facts.Size = release.Known(f.Quality), f.Cut, release.Known(f.Size)
	return facts
}

func fileOf(row db.FilesFile) File {
	f := File{
		ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Cut: row.Cut, RootID: row.RootID, Path: row.Path, Size: row.Size, Media: row.Probe,
		ProbedAt: row.ProbedAt, Seen: seenOf(row.ModifiedAt, row.NodeFileID, row.NodeVolumeID), GoneAt: row.GoneAt,
		GrabID: row.GrabID, Quality: row.Quality, Label: row.Label, Extra: row.Extra, ReleaseName: row.ReleaseName, Title: row.TitleRead.Read,
	}
	if row.ProbeError != nil {
		f.ProbeError = *row.ProbeError
	}
	if row.SourceName != nil {
		f.SourceName = *row.SourceName
	}
	return f
}

// seenOf is what a row records of a file's bytes, nil while nothing read
// them.
func seenOf(modified storage.Time, file, volume *string) *storing.Seen {
	if modified.IsZero() {
		return nil
	}
	s := &storing.Seen{Modified: modified.Time()}
	if file != nil {
		s.ID = storing.FileID{Volume: *volume, File: *file}
	}
	return s
}

// seenColumns is s as a row records it.
func seenColumns(s *storing.Seen) (modified storage.Time, file, volume *string) {
	if s == nil {
		return storage.Time{}, nil, nil
	}
	if !s.ID.IsZero() {
		file, volume = &s.ID.File, &s.ID.Volume
	}
	return storage.FromTime(s.Modified), file, volume
}

type Changed struct {
	WorkID string `json:"workId" format:"uuid"`
}

var ChangedEvent = live.AdminEvent[Changed]("files.changed")

type Publisher interface {
	Publish(ctx context.Context, c Changed)
}

// Moved names the Works whose files the disk queue moved to their paths,
// or put back because another file was at one, in one pass.
type Moved struct {
	WorkIDs []string `json:"workIds" nullable:"false"`
}

var MovedEvent = live.AdminEvent[Moved]("files.moved")

type MovedPublisher interface {
	Publish(ctx context.Context, m Moved)
}

// MovedOf is Moved of ws, each named once.
func MovedOf(ws []works.WorkID) Moved {
	out := Moved{WorkIDs: []string{}}
	for _, w := range ws {
		if s := w.String(); !slices.Contains(out.WorkIDs, s) {
			out.WorkIDs = append(out.WorkIDs, s)
		}
	}
	return out
}

// Add records f, whose bytes are already at its Path, filling each of
// fills for its Item at the Edition its cut fills at score 0, and labels
// it after the first of them in its Work's Profile order. It trusts fills
// and their scores: an import goes through Import, which decides them.
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
		err := q.InsertFill(ctx, db.InsertFillParams{FileID: f.ID, TargetID: t, ItemID: f.ItemID, Edition: goals.Editions.Filling(f.Cut), Score: 0})
		if err != nil {
			return File{}, err
		}
	}
	return f, nil
}

func insertFile(ctx context.Context, q *db.Queries, f File) error {
	modified, file, volume := seenColumns(f.Seen)
	return q.InsertFile(ctx, db.InsertFileParams{
		ID: f.ID, WorkID: f.WorkID, ItemID: f.ItemID, Cut: f.Cut, RootID: f.RootID, Path: f.Path, Size: f.Size, Probe: f.Media,
		GrabID: f.GrabID, Quality: f.Quality, Label: f.Label, Extra: f.Extra, ReleaseName: f.ReleaseName, TitleRead: db.Reading{Read: f.Title},
		ModifiedAt: modified, NodeFileID: file, NodeVolumeID: volume, ProbedAt: f.ProbedAt,
	})
}

// State is a cell's state, in the glossary's order.
type State uint8

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

// States names every State, in the glossary's order.
func States() []string { return slices.Clone(stateNames[Unreleased:]) }

// CellKey names one cell: an Edition at a Target of an Item.
type CellKey struct {
	Item    works.ItemID
	Edition targets.Edition
	Target  targets.TargetID
}

// Cell is one Edition at one Target of one Item, in the State computed on
// read. File is set when a file fills the cell and Score is its score at
// the Target: Met once the file's quality is at or past the Target's
// cutoff and Score at or above its upgrade-until, Upgradable otherwise.
// Timing is the Item's, and Released whether it is out today.
type Cell struct {
	Item     works.Item
	Edition  targets.Edition
	Target   targets.Target
	State    State
	File     *File
	Score    int
	Timing   targets.Timing
	Released bool
}

// Fill is the file filling a cell and its score at the cell's Target.
type Fill struct {
	File  ID
	Score int
}

// Wanted is whether a search should try to fill the cell: a missing,
// failed or upgradable cell of a counted, released Item.
func (c Cell) Wanted() bool {
	return c.Item.Counts() && c.Released && (c.State == Missing || c.State == Failed || c.State == Upgradable)
}

// Status rolls one Item's cells up to the first State any holds, in the
// glossary's order, false for an Item with no cell.
func Status(cells []Cell) (State, bool) {
	if len(cells) == 0 {
		return 0, false
	}
	return slices.MinFunc(cells, func(a, b Cell) int { return cmp.Compare(a.State, b.State) }).State, true
}

// GrabState is what the latest Grab for a cell makes of it, zero while
// the Grab fetches.
type GrabState uint8

const (
	GrabDownloading GrabState = iota + 1
	GrabFailed
)

// GrabCell is the latest Grab of a cell. Waiting is the code of what an
// unfinished Grab waits on that only an Admin resolves, such as
// node_away, empty while it waits on nothing.
type GrabCell struct {
	State   GrabState
	Grab    storage.UUID
	Waiting string
}

// Inflight is files' view of the Grabs, which grabs implements.
type Inflight interface {
	// GrabCells reads the latest Grab of each cell of ws that a Grab
	// fetches, is in flight for or failed.
	GrabCells(ctx context.Context, r storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[CellKey]GrabCell, error)
	// Unfinished lists the ids of every Grab not yet imported or failed,
	// whose download may sit in a root's holding folder.
	Unfinished(ctx context.Context, r storage.Reader[storage.Library]) ([]storage.UUID, error)
	// ReleaseTitles reads the Release title of every Grab with a record.
	ReleaseTitles(ctx context.Context, r storage.Reader[storage.Library]) (map[storage.UUID]string, error)
	// Settle records u on each unfinished Grab of work whose every cell is
	// in gone, for its probe or import to follow if its file fills
	// nothing, and counts them.
	Settle(ctx context.Context, tx *storage.Tx[storage.Library], work works.WorkID, gone map[CellKey]bool, u Unwanted) (int, error)
}

// NoGrabs is Inflight while nothing grabs.
type NoGrabs struct{}

func (NoGrabs) GrabCells(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[CellKey]GrabCell, error) {
	return nil, nil
}

func (NoGrabs) Unfinished(context.Context, storage.Reader[storage.Library]) ([]storage.UUID, error) {
	return nil, nil
}

func (NoGrabs) ReleaseTitles(context.Context, storage.Reader[storage.Library]) (map[storage.UUID]string, error) {
	return nil, nil
}

func (NoGrabs) Settle(context.Context, *storage.Tx[storage.Library], works.WorkID, map[CellKey]bool, Unwanted) (int, error) {
	return 0, nil
}

type Config struct {
	Library *storage.DB[file]
	Targets *targets.Targets
	Grabs   Inflight
	// Transfers is every Transfer with a record, read in r: the staging
	// folders the sweep keeps. transfers.LiveIn is it.
	Transfers func(ctx context.Context, r storage.Reader[storage.Library]) (map[transfers.ID]bool, error)
	// Naming names each Content Type's files.
	Naming *naming.Naming
	// Names read each Content Type's release and file names.
	Names map[works.TypeID]release.Names
	// Today is the time now in the Controller's zone, whose calendar date
	// decides which Items are out.
	Today func() time.Time
	// Disk is what the disk queue moves and removes files with.
	Disk Disk
	// Changed publishes files.changed for each Work whose files Forget deletes.
	Changed Publisher
	// Moved publishes files.moved once per pass of the disk queue that
	// finished or put back a move.
	Moved   MovedPublisher
	Timings Timings
	Rand    *rand.Rand
	Log     *slog.Logger
}

type Files struct {
	cfg  Config
	poke chan struct{}
}

// WorkCells reads a Work's Items, its cells and what they add up to,
// works.ErrNoWork when no Work has the id.
func (f *Files) WorkCells(ctx context.Context, _ identity.Admin, work works.WorkID) ([]works.Item, []Cell, Rollup, error) {
	var w world
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		w, err = f.load(ctx, r, work)
		return err
	})
	if err != nil {
		return nil, nil, Rollup{}, err
	}
	cs := cells(w)
	return w.tree.Items, cs, rollup(w, cs), nil
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
