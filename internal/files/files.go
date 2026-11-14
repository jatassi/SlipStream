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

// File is one library file of one or more Items of a Work. Path is where
// it belongs under its root, '/' separated; until the disk queue moves it
// there, its bytes sit where only the queue looks.
type File struct {
	ID     ID
	WorkID works.WorkID
	// Items are the Items it holds, never none, in tree order: the first
	// names its folder.
	Items []works.ItemID
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
		ID: row.ID, WorkID: row.WorkID, Cut: row.Cut, RootID: row.RootID, Path: row.Path, Size: row.Size, Media: row.Probe,
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

// Moved names the Works whose files a confirmed rename plan queued to
// move, or the disk queue moved to their paths or put back because
// another file was at one, in one pass.
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
// fills for each of its Items at the Edition its cut fills at score 0, and labels
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
	for _, it := range f.Items {
		for _, t := range fills {
			err := q.InsertFill(ctx, db.InsertFillParams{FileID: f.ID, TargetID: t, ItemID: it, Edition: goals.Editions.Filling(f.Cut), Score: 0})
			if err != nil {
				return File{}, err
			}
		}
	}
	return f, nil
}

func insertFile(ctx context.Context, q *db.Queries, f File) error {
	modified, file, volume := seenColumns(f.Seen)
	var source *string
	if f.SourceName != "" {
		source = &f.SourceName
	}
	err := q.InsertFile(ctx, db.InsertFileParams{
		ID: f.ID, WorkID: f.WorkID, Cut: f.Cut, RootID: f.RootID, Path: f.Path, Size: f.Size, Probe: f.Media,
		GrabID: f.GrabID, Quality: f.Quality, Label: f.Label, Extra: f.Extra, ReleaseName: f.ReleaseName, TitleRead: db.Reading{Read: f.Title},
		ModifiedAt: modified, NodeFileID: file, NodeVolumeID: volume, ProbedAt: f.ProbedAt, SourceName: source,
	})
	if err != nil {
		return err
	}
	for i, it := range f.Items {
		if err := q.InsertFileItem(ctx, db.InsertFileItemParams{FileID: f.ID, ItemID: it, Position: int64(i)}); err != nil {
			return err
		}
	}
	return nil
}

// State is a cell's state, in the glossary's order: what needs attention
// first.
type State uint8

const (
	Failed State = iota + 1
	Missing
	Downloading
	Upgradable
	Unreleased
	Met
)

var stateNames = [...]string{Failed: "failed", Missing: "missing", Downloading: "downloading", Upgradable: "upgradable", Unreleased: "unreleased", Met: "met"}

func (s State) String() string { return stateNames[s] }

// States names every State, in the glossary's order.
func States() []string { return slices.Clone(stateNames[Failed:]) }

// CellKey names one cell: an Edition at a Target of an Item.
type CellKey struct {
	Item    works.ItemID
	Edition targets.Edition
	Target  targets.TargetID
}

// Cell is one Edition at one Target of one Item, in the State computed on
// read. File is set when a file fills the cell and Score is its score at
// the Target: Met once the file's quality is at or past the Target's
// cutoff and Score at or above its upgrade-until. Short of that, a
// Downloading or Failed cell keeps the file its Grab would replace, and
// is Upgradable with no such Grab. Grab is the Grab a Downloading or
// Failed cell reads from, or a Met cell's Grab in flight. Doing is what
// that Grab in flight does for a Downloading or Met cell. Timing is the
// Item's, and Released whether it is out today.
type Cell struct {
	Item     works.Item
	Edition  targets.Edition
	Target   targets.Target
	State    State
	File     *File
	Score    int
	Grab     storage.UUID
	Doing    Doing
	Timing   targets.Timing
	Released bool
}

// Step is what a Grab in flight, or its Import, does for a cell, in the
// order a download walks it.
type Step string

const (
	StepFetching    Step = "fetching"
	StepDownloading Step = "downloading"
	StepReading     Step = "reading"
	StepSending     Step = "sending"
	StepPlacing     Step = "placing"
	StepWaiting     Step = "waiting"
)

// Doing is the step a Grab in flight is at for a cell, empty until it is
// known. From is when the Grab entered its stage. Until is when its
// client's Import delay ends, zero unless it waits for it.
type Doing struct {
	Step        Step
	From, Until time.Time
}

// Fill is the file filling a cell and its score at the cell's Target.
// Forced is when an Admin's Force gave the cell its file, zero when
// refill chose it.
type Fill struct {
	File   ID
	Score  int
	Forced storage.Time
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

// GrabState is what a cell's Grabs make of it: in flight while any
// unfinished Grab holds it, fetching included, else failed when its latest
// Grab failed, else none.
type GrabState uint8

const (
	GrabNone GrabState = iota
	GrabInFlight
	GrabFailed
)

// GrabCell is the Grab a cell reads from: its latest unfinished Grab, else
// its latest. Waiting is the code of what an unfinished Grab waits on that
// only an Admin resolves, such as node_away, empty while it waits on
// nothing. FailedAt is the stage a failed Grab failed at, such as
// importing. ClientError is the error a Grab waiting client_error waits
// on, and Hold the root a Grab waiting no_space or root_not_ready waits
// on, nil for any other wait. Doing is the step an unfinished Grab is at.
type GrabCell struct {
	State       GrabState
	Grab        storage.UUID
	Waiting     string
	ClientError *ClientError
	Hold        *ImportHold
	FailedAt    string
	Doing       Doing
}

// ImportHold is the root an importing Grab's file waits on, and for
// no_space the bytes Free above the root's Storage error level and the
// bytes the file Needs, both 0 when a clone found the volume full.
type ImportHold struct {
	Root        roots.ID
	Free, Needs int64
}

// ClientError is the error an external client reports on a Grab's Torrent:
// Client on Node words it Message, and the Grab fails at FailsAt unless it
// clears.
type ClientError struct {
	Client, Node, Message string
	FailsAt               time.Time
}

func (g GrabCell) InFlight() bool { return g.State == GrabInFlight }

// Inflight is files' view of the Grabs, which grabs implements.
type Inflight interface {
	// GrabCells reads the Grab each cell of ws reads from, for each cell a
	// Grab is in flight for or failed.
	GrabCells(ctx context.Context, r storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[CellKey]GrabCell, error)
	// Holding is every path in a root's holding folder an unfinished
	// import claims, by root.
	Holding(ctx context.Context, r storage.Reader[storage.Library]) (map[roots.ID]map[string]bool, error)
	// ReleaseTitles reads the Release title of every Grab with a record.
	ReleaseTitles(ctx context.Context, r storage.Reader[storage.Library]) (map[storage.UUID]string, error)
}

// NoGrabs is Inflight while nothing grabs.
type NoGrabs struct{}

func (NoGrabs) GrabCells(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[CellKey]GrabCell, error) {
	return nil, nil
}

func (NoGrabs) Holding(context.Context, storage.Reader[storage.Library]) (map[roots.ID]map[string]bool, error) {
	return nil, nil
}

func (NoGrabs) ReleaseTitles(context.Context, storage.Reader[storage.Library]) (map[storage.UUID]string, error) {
	return nil, nil
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
	// Moved publishes files.moved once per confirmed rename plan that
	// queued a move, and once per pass of the disk queue that finished or
	// put back a move.
	Moved MovedPublisher
	// MediaServers is told every change of a file's bytes, NoMediaServers
	// when nil.
	MediaServers MediaServers
	Timings      Timings
	Rand         *rand.Rand
	Log          *slog.Logger
}

type Files struct {
	cfg  Config
	poke chan struct{}
}

// ItemState is an Item's status, zero for an Item with no cell, and
// whether it is out today.
type ItemState struct {
	Item     works.Item
	Status   State
	Released bool
}

// WorkCells reads each of a Work's Items with its state, its cells and
// what they add up to, works.ErrNoWork when no Work has the id.
func (f *Files) WorkCells(ctx context.Context, _ identity.Admin, work works.WorkID) ([]ItemState, []Cell, Rollup, error) {
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
	items := make([]ItemState, 0, len(w.tree.Items))
	eachItem(w.tree.Items, cs, func(it works.Item, its []Cell) {
		s, _ := Status(its)
		items = append(items, ItemState{Item: it, Status: s, Released: w.timing[it.ID].ReleasedBy(w.today)})
	})
	return items, cs, rollup(w, cs), nil
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
