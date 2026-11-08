// Package downloadclients owns the Controller's record of every Torrent a
// Node's built-in client or external Download Clients hold, and of the
// external clients themselves. Grabs point at a Torrent and read its
// outcome; nothing here writes a Grab. A Torrent's state word, tabs and
// severity are derived from its record at read time and never stored.
package downloadclients

import (
	"context"
	"embed"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("downloadclients", schemaFiles) }

type ID = db.TorrentID

type Wanted string

const (
	Started Wanted = "started"
	Paused  Wanted = "paused"
)

// Torrent is one payload a Node's client holds by info hash. It outlives
// the Grabs that made it and the Works they were for.
type Torrent struct {
	ID   ID
	Node role.NodeID
	// Client is nil for the Node's built-in client.
	Client *ClientRef
	// Attached is nil until the Node first reports an external torrent:
	// whether it was in its client before SlipStream attached to it.
	Attached *bool
	InfoHash bencode.InfoHash
	// Name is the metainfo's name, how a Torrent whose Work is gone reads.
	Name string
	// Size is the bytes wanted.
	Size    int64
	Private bool
	// Trackers are the announce hosts in announce-list order.
	Trackers []string
	// ContentType is what the Torrent was first grabbed for.
	ContentType works.TypeID
	// SaveFolder is opaque, as the Node reported it; "" until it does.
	SaveFolder string
	Wanted     Wanted
	Order      int64
	AddedAt    time.Time
	Removal    *Removal
	Totals     Totals
	Report     *Report
	Completion *Completion
	Fault      *Fault
}

// ClientRef names a Torrent's external client for readers that show it.
type ClientRef struct {
	ID   ClientID
	Kind clients.Kind
	Name string
}

// Totals never go down: a merge keeps the larger of each counter.
type Totals struct {
	Uploaded, Downloaded int64
	Active, Seeding      time.Duration
	// Announced is the seed time the Torrent's trackers credited it, as its
	// Node counts it.
	Announced time.Duration
	// At is when the Node last reported them.
	At time.Time
}

type RemoveReason string

const (
	ByAdmin        RemoveReason = "admin"
	ByMarkFailed   RemoveReason = "mark_failed"
	NodeWrittenOff RemoveReason = "node_written_off"
	// ByClient is an external client that stopped listing the torrent.
	ByClient RemoveReason = "client"
	// ClientDeleted is the Admin deleting the torrent's client with Stop
	// tracking.
	ClientDeleted RemoveReason = "client_deleted"
	// GoalMet is a seeding goal's remove.
	GoalMet RemoveReason = "goal_met"
)

// RemoveReasons is every RemoveReason, the list the schema's CHECK holds.
var RemoveReasons = []RemoveReason{ByAdmin, ByMarkFailed, NodeWrittenOff, ByClient, ClientDeleted, GoalMet}

// Removal is set when the Torrent leaves its Node's intent. It is Removing
// until Parted: the Node confirmed, or a write-off ended the wait.
type Removal struct {
	At          time.Time
	Reason      RemoveReason
	DeleteFiles bool
	Parted      *Parted
}

type Parted struct {
	At time.Time
	// Freed is the bytes the delete freed, as the Node counted them.
	Freed int64
}

// Report is the Node's latest word on the Torrent, parsed from the wire
// and stored as JSON.
type Report struct {
	// At is the Node's clock, for "Last known at".
	At     time.Time   `json:"at"`
	Engine EngineState `json:"engine"`
	Hold   Hold        `json:"hold,omitempty"`
	// Until is when a HoldWindow starts the Torrent, or when a check
	// resumes it.
	Until    time.Time     `json:"until,omitzero"`
	Complete bool          `json:"complete,omitempty"`
	Stalled  bool          `json:"stalled,omitempty"`
	Error    string        `json:"error,omitempty"`
	Trackers []TrackerLine `json:"trackers,omitempty"`
	// Done is the bytes done at the report, the last known progress.
	Done int64 `json:"done"`
	// Run is what an external client means to do with the torrent, ""
	// for the built-in client, whose HoldWanted says the same.
	Run Wanted `json:"run,omitempty"`
	// ImportAt is when a finished external torrent imports, set while the
	// Import delay runs.
	ImportAt time.Time `json:"importAt,omitzero"`
	// QueuePosition is its place in its external client's queue, from 1,
	// 0 for none.
	QueuePosition int `json:"queuePosition,omitempty"`
	// StoppedAtLimit is an external torrent its client stopped at its
	// seeding limit.
	StoppedAtLimit bool `json:"stoppedAtLimit,omitempty"`
}

type EngineState string

const (
	EngineChecking    EngineState = "checking"
	EngineDownloading EngineState = "downloading"
	EngineSeeding     EngineState = "seeding"
	EngineFailed      EngineState = "failed"
)

// Hold is why a Torrent the Node wants to run doesn't.
type Hold string

const (
	HoldNone   Hold = ""
	HoldWanted Hold = "wanted"
	// HoldWindow waits out a tracker's window or the start spread.
	HoldWindow   Hold = "window"
	HoldDiskFull Hold = "disk_full"
	// HoldSlot waits for a free download slot.
	HoldSlot      Hold = "slot"
	HoldCheckSlot Hold = "check_slot"
	// HoldDiskGuard is the Controller holding new downloads.
	HoldDiskGuard Hold = "disk_guard"
	HoldDraining  Hold = "draining"
	// HoldController is a Node out of step with the Controller.
	HoldController Hold = "controller"
	// HoldClientError is a local error in an external client, with the
	// client's words in Report.Error. It clears when the client clears
	// it, and it is never a release fault.
	HoldClientError Hold = "client_error"
)

type TrackerLine struct {
	// Index is into the metainfo's tracker list.
	Index   int           `json:"index"`
	Status  TrackerStatus `json:"status"`
	Message string        `json:"message,omitempty"`
	// RetryAt is set while Status is error or unregistered.
	RetryAt time.Time `json:"retryAt,omitzero"`
}

type TrackerStatus string

const (
	TrackerWorking      TrackerStatus = "working"
	TrackerUpdating     TrackerStatus = "updating"
	TrackerError        TrackerStatus = "error"
	TrackerUnregistered TrackerStatus = "unregistered"
	TrackerNotContacted TrackerStatus = "not_contacted"
)

// Completion is the first complete report's files. It is sticky.
type Completion struct {
	At    time.Time     `json:"-"`
	Files []engine.File `json:"files"`
}

type FaultKind string

const (
	FaultEngineError  FaultKind = "engine_error"
	FaultUnregistered FaultKind = "unregistered"
	FaultNoPayload    FaultKind = "no_payload"
)

// Fault is the first release fault. It is sticky.
type Fault struct {
	Kind FaultKind
	At   time.Time
}

// MergeTotals records a Node's totals for its Torrent id, keeping the
// larger of each counter, so a resent or restarted report never lowers
// them. An id that is not node's Torrent changes nothing.
func MergeTotals(ctx context.Context, tx *storage.Tx[file], node role.NodeID, id ID, t Totals) error {
	return db.New(tx).MergeTotals(ctx, db.MergeTotalsParams{
		Uploaded:         t.Uploaded,
		Downloaded:       t.Downloaded,
		ActiveSeconds:    int64(t.Active / time.Second),
		SeedingSeconds:   int64(t.Seeding / time.Second),
		AnnouncedSeconds: int64(t.Announced / time.Second),
		TotalsAt:         storage.FromTime(t.At),
		ID:               id,
		NodeID:           node,
	})
}

// Ratio is uploaded over the larger of downloaded and size, so a Torrent
// that came complete from another client reads against its size.
func Ratio(t Totals, size int64) float64 {
	d := max(t.Downloaded, size)
	if d == 0 {
		return 0
	}
	return float64(t.Uploaded) / float64(d)
}
