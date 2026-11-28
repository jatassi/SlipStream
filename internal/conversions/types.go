package conversions

import (
	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/order"
	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type (
	ID         = db.ConversionID
	BackfillID = db.BackfillID
)

type Stage string

const (
	StageQueued    Stage = "queued"
	StageCopying   Stage = "copying"
	StageReady     Stage = "ready"
	StageRunning   Stage = "running"
	StagePaused    Stage = "paused"
	StageChecking  Stage = "checking"
	StageReturning Stage = "returning"
	StageLanding   Stage = "landing"
	StageDone      Stage = "done"
	StageFailed    Stage = "failed"
	StageCanceled  Stage = "canceled"
)

// Lane is a queue's part, in run order.
type Lane string

const (
	LaneMovedUp  Lane = "moved_up"
	LaneNew      Lane = "new"
	LaneUpgrade  Lane = "upgrade"
	LaneBackfill Lane = "backfill"
)

// Row is one Conversion: a source file made into one Target's file.
type Row struct {
	ID         ID
	Work       works.WorkID
	Type       works.TypeID
	Items      []works.ItemID
	Edition    targets.Edition
	SourceFile *files.ID
	Target     *targets.TargetID
	Version    int
	Lane       Lane
	Position   order.Key
	Backfill   *BackfillID
	Stage      Stage
	Node       *role.NodeID
	Root       *roots.ID
	// Attempt fences every result. BudgetFrom is the first attempt the
	// retry rule counts, past the last Retry.
	Attempt, BudgetFrom int
	Pin                 *role.NodeID
	Exclude             []role.NodeID
	NotBefore           time.Time
	// PausedBy is the Media Servers whose transcodes pause it.
	PausedBy  []string
	Failure   *Failure
	CancelWhy CancelWhy
	Snapshot  Snapshot
	QueuedAt  time.Time
	StartedAt *time.Time
	// FinishedAt is set once done or canceled. A failed Conversion is
	// unfinished.
	FinishedAt *time.Time
	Output     *files.ID
}

// Snapshot is the names a row shows once its links are gone.
type Snapshot struct {
	Work       string `json:"work"`
	Target     string `json:"target"`
	SourceName string `json:"sourceName"`
}

type Failure struct {
	Code   Code
	Class  Class
	Detail string
}

// Code is a fault's code, a key of the faults table.
type Code string

// Class is what a fault costs: a run fault retries, a source fault fails
// at once, and a put-back fault queues again.
type Class string

const (
	RunClass     Class = "run"
	SourceClass  Class = "source"
	PutBackClass Class = "put_back"
)

type CancelWhy string

const (
	CanceledByAdmin CancelWhy = "admin"
	NotWanted       CancelWhy = "not_wanted"
	SourceGone      CancelWhy = "source_gone"
	ImportEnded     CancelWhy = "import_ended"
	ImportDeclined  CancelWhy = "import_declined"
)

type Attempt struct {
	N        int
	Node     role.NodeID
	NodeName string
	Pipeline encode.Pipeline
	Started  time.Time
	Ended    *time.Time
	// Outcome is empty while the attempt is open.
	Outcome Outcome
	Code    Code
	Class   Class
	Tail    string
}

type Outcome string

const (
	AttemptFailed   Outcome = "failed"
	AttemptStalled  Outcome = "stalled"
	AttemptDiskFull Outcome = "disk_full"
	AttemptPutBack  Outcome = "put_back"
)

// On is what moves a Conversion along the stage table.
type On string

const (
	Placed       On = "placed"
	Unpin        On = "unpin"
	Copied       On = "copied"
	CopyFailed   On = "copy_failed"
	Started      On = "started"
	Paused       On = "paused"
	Resumed      On = "resumed"
	Preempted    On = "preempted"
	Exited       On = "exited"
	Returned     On = "returned"
	ReturnFailed On = "return_failed"
	Checked      On = "checked"
	Fault        On = "fault"
	DiskFull     On = "disk_full"
	Lost         On = "lost"
	PutBack      On = "put_back"
	Landed       On = "landed"
	Ended        On = "ended"
	Unneeded     On = "unneeded"
	Canceled     On = "canceled"
	Retry        On = "retry"
)

// Event is one move along the stage table, with what its On carries.
type Event struct {
	On         On
	Conversion ID
	// Node and Attempt fence a Worker's or a Transfer's event when Fenced.
	Node    role.NodeID
	Attempt int
	Fenced  bool
	Basis   Basis
	// Proof and Copy are Placed's.
	Proof *plan.Proof
	Copy  bool
	// Code, Tail and Bytes are Fault's and Exited's.
	Code  Code
	Tail  string
	Bytes int64
	// Probe and Verdict are Checked's.
	Probe   *media.Record
	Verdict Verdict
	// Why is Unneeded's, Canceled's, Ended's and PutBack's.
	Why CancelWhy
	// File is Landed's.
	File files.ID
}

// Basis is the stage and attempt a snapshot read, which an event built
// from it must still find.
type Basis struct {
	Stage   Stage
	Attempt int
}

// Verdict is the root-side check of an output: a zero Code passes.
type Verdict struct {
	Code  Code
	Class Class
}

// Why is the qualifying rule's answer, also a cause key and a wait reason.
type Why = plan.Why

// Wait is why an unfinished Conversion hasn't started, with its facts.
type Wait = plan.Wait

// Cause is one reason no Node can convert some Conversions.
type Cause struct {
	Key   CauseKey
	Count int
	// Node is the Node its sentence names, when one.
	Node *role.NodeID
	// Notified is when the Admin was told of it.
	Notified time.Time
}

// CauseKey is a Why with the Class and Shape it covers, and the Target
// for RunsElsewhere.
type CauseKey struct {
	Why    Why
	Class  encode.Class
	Shape  string
	Target *targets.TargetID
}
