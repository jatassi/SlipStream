package imports

import (
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type ImportID = db.ImportID

type Source string

const (
	FromGrab   Source = "grab"
	FromRoot   Source = "root"
	FromFolder Source = "folder"
)

// Import is files from one source into one Work: a finished download, a
// folder a library scan found in a root and couldn't place, or a folder
// the Admin picks. Work is zero for a root Import, whose Suggestion names
// the Work it may be.
type Import struct {
	ID       ImportID
	Source   Source
	Grab     grabs.GrabID      // grab only; one unfinished Import per Grab
	Node     role.NodeID       // grab: the downloading Node; folder: the Node
	InfoHash *bencode.InfoHash // grab only
	Root     roots.ID          // root only
	// Folder is root-relative for a root Import: its Work folder, or the
	// file's path when it sits loose at the root's top. Absolute on the
	// Node for a folder Import.
	Folder      string
	ContentType works.TypeID
	Hint        scanning.Hint
	Suggestion  *Suggestion
	Work        works.WorkID
	Edition     *release.CutKey
	MoveFiles   bool // folder only
	Failure     *FailureCode
	// Ended is how a source removal, a Cancel or Not this Release ended
	// it, so a later converge fails its Grab with that code.
	Ended    *Ending
	Created  storage.Time
	Ignored  storage.Time // zero while it needs the Admin
	Finished storage.Time
	Files    []File // by position
}

func (im Import) IsIgnored() bool { return !im.Ignored.IsZero() }

// FailureCode is the release fault that failed an Import.
type FailureCode string

const (
	NoVideoFile    FailureCode = "no_video_file"
	UnreadableFile FailureCode = "unreadable"
)

type Ending struct {
	Reason ReasonCode `json:"reason"`
	Code   string     `json:"code"`
}

type Outcome string

const (
	Pending   Outcome = "pending"
	LeftOut   Outcome = "left_out"
	NotBetter Outcome = "not_better"
	Waiting   Outcome = "waiting"
	Sending   Outcome = "sending"
	Placing   Outcome = "placing"
	Placed    Outcome = "placed"
	Declined  Outcome = "declined"
	FailedOut Outcome = "failed"
)

// Outcomes is every Outcome, in the order of the outcome CHECK.
var Outcomes = []Outcome{Pending, LeftOut, NotBetter, Waiting, Sending, Placing, Placed, Declined, FailedOut}

// Final outcomes change only through an Admin's Retry of a failed file,
// or never.
func (o Outcome) Final() bool {
	switch o {
	case LeftOut, NotBetter, Placed, Declined, FailedOut:
		return true
	case Pending, Waiting, Sending, Placing:
	}
	return false
}

// File is one video of an Import. Path is torrent-relative for a grab,
// root-relative for a root and absolute on the Node for a folder. Items
// are in tree order. Probe is nil until read, and for a file ffprobe
// can't read.
type File struct {
	Position    int
	Path        string
	SourceIndex *int
	Size        int64
	Seen        *storing.Seen
	Probe       *media.Record
	Quality     release.Quality
	Cut         release.CutKey
	ReleaseName string
	SourceName  string
	Items       []works.ItemID
	Outcome     Outcome
	Reason      *Reason
	Hold        *Hold
	Confirmed   bool
	// Fills are the cells it fills: decided, or as it landed, or for a
	// confirm wait the ones it would.
	Fills []Fill
	// Dest is the root and holding path its bytes go to, or for a root
	// Import the path they already sit at.
	Dest          *Dest
	Sent          *Sent
	PlacedBy      PlacedBy
	FileID        files.ID
	SourceRemoved storage.Time
}

// Scan is the S22 (Library scan and file probe) reason a root Import's
// file waits for, zero for none.
func (f File) Scan() scanning.Reason {
	if f.Reason == nil || f.Reason.Scan == nil {
		return scanning.Reason{}
	}
	return *f.Reason.Scan
}

// waitFor makes f wait for r, a library scan's reason.
func (f *File) waitFor(r scanning.Reason) {
	f.Outcome, f.Reason = Waiting, &Reason{Code: ReasonCode(r.Code), Scan: &r}
}

type Dest struct {
	Root    roots.ID
	Holding storing.Rel
}

type Sent struct {
	Transfer transfers.ID
	Order    int
}

// PlacedBy is how a file's bytes reached the library.
type PlacedBy string

const (
	ByLink     PlacedBy = "link"
	ByClone    PlacedBy = "clone"
	ByCopy     PlacedBy = "copy"
	ByTransfer PlacedBy = "transfer"
	ByMove     PlacedBy = "move"
	InPlace    PlacedBy = "in_place"
)

type Stage string

const (
	StageProbing Stage = "probing"
	StageWaiting Stage = "waiting"
	StagePlacing Stage = "placing"
	StageDone    Stage = "done"
	StageFailed  Stage = "failed"
)

// Stage is derived, never stored: failed when Failure is set, probing
// while any file is pending, placing while any is sending or placing,
// waiting while any waits, else done.
func (im Import) Stage() Stage {
	if im.Failure != nil {
		return StageFailed
	}
	has := map[Outcome]bool{}
	for _, f := range im.Files {
		has[f.Outcome] = true
	}
	switch {
	case has[Pending]:
		return StageProbing
	case has[Sending] || has[Placing]:
		return StagePlacing
	case has[Waiting]:
		return StageWaiting
	}
	return StageDone
}
