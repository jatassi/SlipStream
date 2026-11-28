// Package plan is placement and the forecast as one pure function over a
// snapshot of the queue, the Nodes and their proofs.
package plan

import (
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

type Snapshot struct {
	Now   time.Time
	Nodes []Node
	// Jobs are every unfinished Conversion, in lane then position order.
	Jobs []Job
	// Rates are bytes a second per (from, to) Node pair, from transfers_pair.
	Rates map[[2]role.NodeID]float64
	// Tests are the test runs in flight per Node.
	Tests map[role.NodeID]int
}

type Node struct {
	ID   role.NodeID
	Name string
	// May is in step and not draining.
	Holds, Away, Lapsed, May bool
	AtOnce                   int
	// Hours is nil while the Node converts at any hour.
	Hours     *setting.Span
	Zone      *time.Location
	Processor bool
	Proofs    map[encode.Key]Proof
	// ScratchFree is the scratch volume's free bytes, net of what placed
	// Conversions reserve there.
	ScratchFree int64
	// Hold is set while a Media Server on the Node transcodes video.
	Hold    bool
	Pauses  bool
	Running []Running
}

// Running is a started Conversion and the time its run has left.
type Running struct {
	Job  storage.UUID
	Left time.Duration
}

type Job struct {
	ID      storage.UUID
	Stage   string
	Attempt int
	Target  storage.UUID
	// Node is where it is placed, Pin the Node its retry waits for, RunOn
	// its Target's Run on.
	Node, Pin, RunOn *role.NodeID
	Exclude          map[role.NodeID]bool
	Key              encode.Key
	NotBefore        time.Time
	Source           storage.UUID
	SourceNode       role.NodeID
	SourceBytes      int64
	OutBytes         int64
	Root             storage.UUID
	RootNode         role.NodeID
	// RootRoom is the free bytes on the root's volume.
	RootRoom  int64
	Length    time.Duration
	OutPixels int
}

// Proof is a Node's test run of one Key. Speed is a real-time factor at
// 1080p.
type Proof struct {
	Pipeline             encode.Pipeline
	Speed                float64
	Pauses, Failed, Lost bool
}

// Why is the first clause of the qualifying rule a Node fails for a job,
// or Fits. Each is also a cause key, a wait reason and an editor warning.
type Why string

const (
	Fits            Why = "fits"
	NoRole          Why = "no_role"
	Lapsed          Why = "lapsed"
	RunsElsewhere   Why = "runs_elsewhere"
	PinnedElsewhere Why = "pinned_elsewhere"
	FailedHere      Why = "failed_here"
	Untested        Why = "untested"
	TestFailed      Why = "test_failed"
	Retesting       Why = "retesting"
	ProofLost       Why = "proof_lost"
	ProcessorOff    Why = "processor_off"
	NoneOnline      Why = "none_online"
	NoHolder        Why = "no_holder"
)

// Step is what the reconciler writes now: at most AtOnce+1 source units
// placed per Node, with room, the test runs to ask, and the causes.
type Step struct {
	Placed []Placed
	Tests  []Test
	Causes []Cause
}

// Placed is a job placed on Node, at the stage and attempt the snapshot
// read. Copy is set when its source is copied to the Node first.
type Placed struct {
	Job     storage.UUID
	Stage   string
	Attempt int
	Node    role.NodeID
	Proof   Proof
	Copy    bool
}

type Test struct {
	Node role.NodeID
	Key  encode.Key
}

// Cause is a due job no Node qualifies for and none is testing, with the
// Why of the Node closest to qualifying, and that Node.
type Cause struct {
	Job  storage.UUID
	Why  Why
	Node *role.NodeID
}

type Forecast map[storage.UUID]Plan

type Plan struct {
	Node          *role.NodeID
	Start, Finish *time.Time
	Wait          Wait
}

// WaitCode is why a job hasn't started.
type WaitCode string

const (
	InLine    WaitCode = "in_line"
	Hours     WaitCode = "hours"
	Room      WaitCode = "room"
	Copying   WaitCode = "copying"
	Away      WaitCode = "away"
	Download  WaitCode = "download"
	Testing   WaitCode = "testing"
	Transcode WaitCode = "transcode"
	NoNode    WaitCode = "no_node"
	NoSource  WaitCode = "no_source"
)

// Wait is a WaitCode with the facts its words name.
type Wait struct {
	Code  WaitCode
	Node  *role.NodeID
	Until *time.Time
	// Place is the job's place in line, from 1.
	Place      int
	Free, Need int64
	Cause      Why
}
