package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Liftable is the soft refusals a Manual pick may grab past. covers_more
// and not_an_upgrade wait for S33 (Import pipeline). cut_not_wanted is
// lifted only by the Edition override.
var Liftable = []targets.ReasonCode{targets.WrongTitle, targets.SizeBelow, targets.SizeAbove, targets.ScoreBelowMinimum, targets.RepackOtherGroup}

// ManualPick is one ticked row, or one Release's Grab, as the sheet sends
// it.
type ManualPick struct {
	Release ReleaseRef
	// Lifts is the soft refusals the row said it gets past, by code and
	// Target.
	Lifts []LiftKey
	// Edition is the Edition override, zero for none.
	Edition targets.Edition
	// Client is the client chosen, nil for where Tracker routing sends it.
	Client  *downloadclients.Choice
	Unblock bool
	// Cancel is the Grab downloading a cell the pick fills, which the same
	// Write cancels first.
	Cancel *GrabID
}

type LiftKey struct {
	Code   targets.ReasonCode
	Target targets.TargetID
}

// PickResult is a pick's Grab, or why it was refused.
type PickResult struct {
	Grab GrabID
	Err  *PickError
}

type PickCode string

const (
	// PickConflict is two picks that fill one cell.
	PickConflict PickCode = "pick_conflict"
	// CellHeld is a pick for a cell a Grab downloads.
	CellHeld PickCode = "cell_held"
	// PickChanged is a pick whose cells or refusals changed since the plan.
	PickChanged PickCode = "pick_changed"
	// NotTaken is a pick its Indexer won't take now.
	NotTaken PickCode = "not_taken"
	// NotGrabbable is a hard refusal, covers_more or not_an_upgrade.
	NotGrabbable PickCode = "not_grabbable"
	// PickBusy is a pick whose Grab to cancel is placing its file.
	PickBusy PickCode = "grab_busy"
	// ClientRefused is a chosen client SlipStream can't use.
	ClientRefused PickCode = "download_client_field_refused"
	// NotDownloading is a built-in client on a Node that doesn't hold
	// Downloading.
	NotDownloading PickCode = "node_not_downloading"
)

// PickCodes is every PickCode.
var PickCodes = []PickCode{PickConflict, CellHeld, PickChanged, NotTaken, NotGrabbable, PickBusy, ClientRefused, NotDownloading}

// PickError is a refused pick in words. Held is the Grab downloading the
// cell, for CellHeld.
type PickError struct {
	Code  PickCode
	Words string
	Held  GrabID
}

// GrabFromManual grabs picks from Manual search id of work in one Write.
// Each is judged again in that Write, and a refused pick is refused
// alone. Two picks with one info hash make one Grab. It fails as a whole
// with ErrManualSearchReplaced, ErrReconciling, ErrNoWork or a library
// error.
func (gr *Grabs) GrabFromManual(context.Context, identity.Admin, works.WorkID, ManualID, []ManualPick) ([]PickResult, error) {
	return nil, ErrNotBuilt
}
