package metadata

import (
	"errors"
	"time"
)

// OutcomeKind is how one provider call ended. Provider status reads it.
type OutcomeKind uint8

const (
	OK OutcomeKind = iota + 1
	NotFound
	Refused
	// Limited is rate limited until Until.
	Limited
	// Capped is out of a daily allowance until Until. Calls wait like
	// Limited; status shows it apart.
	Capped
	Unreachable
)

// Outcome is one provider call's result as a holder reports it.
type Outcome struct {
	Provider ProviderID
	Kind     OutcomeKind
	// Until is when a Limited or Capped provider takes calls again.
	Until time.Time
	// Words say what failed, in plain words.
	Words string
	At    time.Time
}

// ErrNoChanges answers a changes list asked from further back than the
// source lists, or from a source that keeps none. The caller relies on its
// other refresh tiers instead.
var ErrNoChanges = errors.New("the metadata source lists no changes that far back")
