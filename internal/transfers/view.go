package transfers

import (
	"time"

	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
)

// State is a Transfer's one state now. A Transfer whose files are all
// placed is gone and has none.
type State uint8

const (
	// Queued: another Transfer copies on the same pair.
	Queued State = iota + 1
	// Waiting: a Wait names why.
	Waiting
	// NeedsSpace: its files don't fit, or the disk filled.
	NeedsSpace
	// Copying: bytes move, or every file is verified and waits for its
	// purpose to place it.
	Copying
	// Failed: a failure is stored.
	Failed
)

func (s State) String() string {
	return [...]string{"", "queued", "waiting", "needs_space", "copying", "failed"}[s]
}

// WaitKind is why a Transfer waits. A pair wait stops every Transfer on
// the pair, so the waiting one keeps its place. A step-aside wait is this
// Transfer's alone, so it goes back in line.
type WaitKind uint8

const (
	// NodeAway: Node is offline or lapsed.
	NodeAway WaitKind = iota + 1
	// NodeOtherBuild: Node runs another build than the Controller.
	NodeOtherBuild
	// Connecting: no lane yet on the current route.
	Connecting
	// NoRoute: two Workers, and both gave up dialing the other.
	NoRoute
	// RootNotReady: Node's root Root isn't ready. Detail is its State, or
	// the receiver's error.
	RootNotReady
	// StorageDraining: Node drains Storage.
	StorageDraining
	// SourceChecking: Node checks the torrent Name, or found it
	// incomplete.
	SourceChecking
	// SourceNotReady: Node can't open the file Name yet. Detail is the
	// error.
	SourceNotReady
)

func (k WaitKind) String() string {
	return [...]string{"", "node_away", "node_other_build", "connecting", "no_route", "root_not_ready", "storage_draining", "source_checking", "source_not_ready"}[k]
}

// Wait is a typed reason. Detail is OS or engine text, never words.
type Wait struct {
	Kind   WaitKind
	Node   role.NodeID
	Root   roots.ID
	Name   string
	Detail string
}

// Space is what a NeedsSpace Transfer needs.
type Space struct {
	Node role.NodeID
	// Volume is the volume's name.
	Volume string
	// Need is the bytes to free before it fits above the error level.
	Need int64
}

// View is one Transfer as judge sees it. Only words.go turns it into
// words.
type View struct {
	Transfer
	State State
	Wait  *Wait
	Space *Space
	// Ahead is the Transfers before a Queued one on its pair.
	Ahead    int
	DestNode role.NodeID
	// Verified and Total are bytes.
	Verified, Total int64
	// Rate is bytes a second, from the receiver's facts while it copies,
	// else the pair's last speed.
	Rate int64
	// Estimate is 0 while unknown.
	Estimate time.Duration
	// Route is unspecified unless the Transfer is active.
	Route   tv1.Route
	Details Details
}

type Details struct{ Lanes, RTTMillis, Retried int }

// Pair is one recent speed and the last route that moved bytes from
// Source to Destination.
type Pair struct {
	Source, Destination role.NodeID
	Rate                int64
	RateAt              time.Time
	// Route is unspecified while none moved bytes.
	Route   tv1.Route
	RouteAt time.Time
}

// End is how one end of a Transfer stands, as judge reads it.
type End struct {
	Standing   nodes.Standing
	Controller bool
	// DrainsStorage: the Node finishes its Storage work and takes no new
	// Transfers into its roots.
	DrainsStorage bool
}

// Board is the roster at one instant. Changed closes when a newer one
// replaces it.
type Board interface {
	End(node role.NodeID) (End, bool)
	Changed() <-chan struct{}
}

// BoardOf reads b's ends.
func BoardOf(b *nodes.Board) Board { return nodesBoard{b} }

type nodesBoard struct{ b *nodes.Board }

func (b nodesBoard) End(node role.NodeID) (End, bool) {
	v, ok := b.b.Node(node)
	if !ok {
		return End{}, false
	}
	return End{Standing: v.Standing, Controller: v.Controller, DrainsStorage: b.b.Who(role.Storage).State(node) == nodes.Draining}, true
}

func (b nodesBoard) Changed() <-chan struct{} { return b.b.Changed() }
