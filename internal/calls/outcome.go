package calls

import "time"

// OutcomeKind is how one provider call ended.
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
	// PinRefused is a login the provider refused for its PIN, not its key.
	PinRefused
)

// Outcome is one provider call's result as a holder reports it.
type Outcome struct {
	Provider ID
	Kind     OutcomeKind
	// Until is when a Limited or Capped provider takes calls again.
	Until time.Time
	// Words say what failed, in plain words.
	Words string
	At    time.Time
}
