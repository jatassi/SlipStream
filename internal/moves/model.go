package moves

import (
	"path"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type (
	BatchID = db.BatchID
	UnitID  = db.UnitID
)

// Origin is where the Admin started a batch.
type Origin string

const (
	OriginWork    Origin = "work"
	OriginBulk    Origin = "bulk"
	OriginRoot    Origin = "root"
	OriginDrain   Origin = "drain"
	OriginClient  Origin = "client"
	OriginTorrent Origin = "torrent"
)

// Phase is a unit's stage as its row stores it.
type Phase string

const (
	PhaseQueued   Phase = "queued"
	PhaseTransfer Phase = "transfer"
	PhasePlace    Phase = "place"
	PhaseSwitch   Phase = "switch"
	PhaseCleanup  Phase = "cleanup"
	PhaseDone     Phase = "done"
	PhaseFailed   Phase = "failed"
	PhaseCanceled Phase = "canceled"
)

// Phases is every Phase in the order a unit walks them.
func Phases() []Phase {
	return []Phase{PhaseQueued, PhaseTransfer, PhasePlace, PhaseSwitch, PhaseCleanup, PhaseDone, PhaseFailed, PhaseCanceled}
}

// Wait is why a unit isn't moving on. It is judged when the unit is due
// and never fails a unit.
type Wait string

const (
	WaitPairBusy      Wait = "pair_busy"
	WaitLowPriority   Wait = "low_priority"
	WaitNodeOffline   Wait = "node_offline"
	WaitNoRoute       Wait = "no_route"
	WaitSpace         Wait = "space"
	WaitRenames       Wait = "renames"
	WaitDownloading   Wait = "downloading"
	WaitImporting     Wait = "importing"
	WaitOpenFiles     Wait = "open_files"
	WaitPlaying       Wait = "playing"
	WaitMediaServers  Wait = "media_servers"
	WaitGuard         Wait = "guard"
	WaitLinkedTorrent Wait = "linked_torrent"
	// WaitReconciling is derived by viewOf only and never stored.
	WaitReconciling Wait = "reconciling"
)

// Waits is every stored Wait.
func Waits() []Wait {
	return []Wait{WaitPairBusy, WaitLowPriority, WaitNodeOffline, WaitNoRoute, WaitSpace, WaitRenames, WaitDownloading,
		WaitImporting, WaitOpenFiles, WaitPlaying, WaitMediaServers, WaitGuard, WaitLinkedTorrent}
}

// FailureCode is why a unit failed.
type FailureCode string

const (
	SourceGone             FailureCode = "source_gone"
	SourceFailed           FailureCode = "source_failed"
	Damaged                FailureCode = "damaged"
	SourceWrittenOff       FailureCode = "source_written_off"
	DestinationGone        FailureCode = "destination_gone"
	DestinationCheckFailed FailureCode = "destination_check_failed"
	SwitchRefused          FailureCode = "switch_refused"
	DestinationUnreachable FailureCode = "destination_unreachable"
)

// FailureCodes is every FailureCode.
func FailureCodes() []FailureCode {
	return []FailureCode{SourceGone, SourceFailed, Damaged, SourceWrittenOff, DestinationGone, DestinationCheckFailed, SwitchRefused, DestinationUnreachable}
}

// Lane is the pair a unit runs on: "pair:<from>><to>" across Nodes,
// "vol:<node>:<fromVolume>><toVolume>" on one Node. One unit holds a lane
// from its Transfer, or its first Place, until its Place ends.
type Lane string

// End is one side of a Work unit: a root and the Node it was on at the
// plan.
type End struct {
	Root roots.ID
	Node role.NodeID
}

// laneOf is the lane between from and to: their Node pair, or on one Node
// the pair of volumes the bytes sit on.
func laneOf(from, to End, fromVolume, toVolume string) Lane {
	if from.Node != to.Node {
		return Lane("pair:" + from.Node.String() + ">" + to.Node.String())
	}
	return Lane("vol:" + from.Node.String() + ":" + fromVolume + ">" + toVolume)
}

// sameNode is whether l is a lane on one Node, where no Transfer runs.
func (l Lane) sameNode() bool { return len(l) > 4 && l[:4] == "vol:" }

// Subject is what a unit moves.
type Subject interface{ subject() }

// WorkSubject is a Work's live files moving from one root to another of
// its Content Type.
type WorkSubject struct {
	Work     works.WorkID
	Type     works.TypeID
	From, To End
}

func (WorkSubject) subject() {}

// Side is a torrent's Node and Download Client, nil for the built-in one.
type Side struct {
	Node   role.NodeID
	Client *downloadclients.ClientID
}

// TorrentSubject is a seeding torrent moving to another Download Client.
// Folder is an external destination's folder once placed, Arriving its
// destination Torrent once inserted.
type TorrentSubject struct {
	Torrent  downloadclients.ID
	From, To Side
	Folder   string
	Arriving *downloadclients.ID
}

func (TorrentSubject) subject() {}

// Handoff is a torrent unit's step at its switch: it waits out the guard,
// stops the source, then starts the destination.
type Handoff string

const (
	HandoffGuard    Handoff = "guard"
	HandoffStopping Handoff = "stopping"
	HandoffStarting Handoff = "starting"
)

// HandoffAt is a handoff's step and when it began.
type HandoffAt struct {
	Step Handoff
	At   time.Time
}

// Stage is a unit's phase with exactly the facts that phase has. rows.go
// parses it from the row; nothing else builds one from columns.
type Stage interface{ Phase() Phase }

// Queued is a unit waiting for its lane, or back from the switch's stale
// check or a Retry with Pass > 0. Wait is empty while it heads its lane.
type Queued struct {
	Wait  Wait
	Since time.Time
	Pass  int
}

// Copying is a unit whose Transfer copies the files across Nodes. Wait is
// mirrored from the Transfer's view.
type Copying struct {
	Transfer transfers.ID
	Pass     int
	Wait     Wait
	Since    time.Time
}

// Placing is a unit putting its files at their Rels in the destination
// root, and then comparing the source again in its final pass. Transfer is
// the Transfer the files were staged by, nil on one Node.
type Placing struct {
	Pass     int
	Transfer *transfers.ID
	Wait     Wait
	Since    time.Time
}

// Switching is a placed unit waiting for its switch. SwitchNow passes the
// playing and media_servers waits. Handoff is a torrent unit's.
type Switching struct {
	Wait      Wait
	Since     time.Time
	SwitchNow *time.Time
	Handoff   *HandoffAt
}

// Cleaning is a switched unit removing its source files.
type Cleaning struct {
	SwitchedAt time.Time
	Wait       Wait
	Since      time.Time
}

type Done struct{ SwitchedAt, CleanedAt time.Time }

// Failed is a unit that can't go on before its switch. Undone is when its
// undo finished, nil until then.
type Failed struct {
	Code FailureCode
	// At is the phase it failed in.
	At     Phase
	Detail string
	// KeepsCopy is a torrent that failed at its start: its arriving
	// Torrent stays, paused, for a Retry.
	KeepsCopy bool
	Undone    *time.Time
}

// Canceled is a unit the Admin, or a delete of its Work, canceled before
// its switch.
type Canceled struct {
	At     Phase
	Undone *time.Time
}

func (Queued) Phase() Phase    { return PhaseQueued }
func (Copying) Phase() Phase   { return PhaseTransfer }
func (Placing) Phase() Phase   { return PhasePlace }
func (Switching) Phase() Phase { return PhaseSwitch }
func (Cleaning) Phase() Phase  { return PhaseCleanup }
func (Done) Phase() Phase      { return PhaseDone }
func (Failed) Phase() Phase    { return PhaseFailed }
func (Canceled) Phase() Phase  { return PhaseCanceled }

// Planned is what the plan, or the last final pass, read of the unit.
type Planned struct {
	Bytes int64
	// At is the time of the last final pass: a file a scan marked gone
	// before it is no difference at the switch.
	At time.Time
}

type Unit struct {
	ID       UnitID
	Batch    BatchID
	Seq      int64
	Position int
	Lane     Lane
	// Subject is a Work unit's, zero for a torrent unit, whose Torrent is
	// set.
	Subject WorkSubject
	Torrent *TorrentSubject
	Stage   Stage
	// Due is when the runner looks again with no event; nil: only an
	// event or a hook.
	Due     *time.Time
	Planned Planned
	// PairsWith is the torrent unit whose payload a Work unit places its
	// linked files from.
	PairsWith *UnitID
	Notes     Notes
	Reconcile bool
	// Files are loaded only where a step needs them.
	Files []File
}

// nodes is u's source and destination Nodes.
func (u Unit) nodes() (from, to role.NodeID) {
	if u.Torrent != nil {
		return u.Torrent.From.Node, u.Torrent.To.Node
	}
	return u.Subject.From.Node, u.Subject.To.Node
}

// open is whether u is before its switch or cleaning, or has an undo to
// run.
func (u Unit) open() bool {
	switch s := u.Stage.(type) {
	case Done:
		return false
	case Failed:
		return s.Undone == nil
	case Canceled:
		return s.Undone == nil
	}
	return true
}

// beforeSwitch is whether u is in a phase from queued to switch.
func (u Unit) beforeSwitch() bool {
	switch u.Stage.(type) {
	case Queued, Copying, Placing, Switching:
		return true
	}
	return false
}

// pass is the unit's pass: 0 at the plan, one more for each final pass
// that found its files changed.
func (u Unit) pass() int {
	switch s := u.Stage.(type) {
	case Queued:
		return s.Pass
	case Copying:
		return s.Pass
	case Placing:
		return s.Pass
	}
	return 0
}

// Via is where a Work unit's file comes from: its library root, or the
// payload its linked torrent's unit placed at the destination.
type Via string

const (
	ViaLibrary  Via = "library"
	ViaArriving Via = "arriving"
)

type FileKind string

const (
	KindLibrary   FileKind = "library"
	KindCompanion FileKind = "companion"
	KindSidecar   FileKind = "sidecar"
	// KindPayload is a torrent's payload file: Sits is its path in the
	// torrent, Index its index.
	KindPayload FileKind = "payload"
)

type Cleanup string

const (
	CleanupPending Cleanup = "pending"
	CleanupDone    Cleanup = "done"
	CleanupSkipped Cleanup = "skipped"
)

// File is one file a unit moves. Sits is where its bytes sit at the
// source, and the Rel they take at the destination.
type File struct {
	Position int
	Kind     FileKind
	Via      Via
	// ID is a library file's or a sidecar's row, zero for a companion.
	ID   files.ID
	Sits storing.Rel
	Size int64
	// Seen is the source's, at the plan or the last final pass.
	Seen storing.Seen
	// Placed is the destination's, once placed.
	Placed *storing.Seen
	// Ledger is a sidecar its source root's ledger marked SlipStream's.
	Ledger bool
	// Index is a payload file's index in its torrent, and the payload
	// file a file via arriving places from.
	Index   int
	Cleanup Cleanup
	// Copy is the bytes of a same-Node copy grant held, 0 for none.
	Copy int64
	// Dropped is when a final pass found the file gone at the source; its
	// placed copy is still to remove.
	Dropped *time.Time
}

// holdingOf is file position pos's holding path in the destination root:
// the unit's id names it, because a companion has no file id.
func holdingOf(u UnitID, pos int, sits storing.Rel) storing.Rel {
	rel, err := storing.ParseRel(storing.HoldingDir + "/" + u.String() + "." + strconv.Itoa(pos) + path.Ext(sits.String()))
	if err != nil {
		panic("moves: a holding path doesn't parse: " + err.Error())
	}
	return rel
}

// Notes are what a unit records for the Admin beside its stage.
type Notes struct {
	// Skipped is source files cleanup left in place because they changed.
	Skipped []string `json:"skipped,omitempty"`
	// Unsafe is the Plex servers not safe when the bound passed.
	Unsafe []string `json:"unsafe,omitempty"`
	// Lost is the servers a Switch now passed.
	Lost []string `json:"lost,omitempty"`
	// Released is set once the Media Servers were told the source is
	// gone at the bound.
	Released bool `json:"released,omitempty"`
	// Amber is the preview's amber reasons.
	Amber []Reason `json:"amber,omitempty"`
}
