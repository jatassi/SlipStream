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
	// Allowance is what the answer reported of the key's allowance, nil
	// when it reported none.
	Allowance *Allowance
}

// Allowance is what a provider last reported of a key's allowance. Every
// number is the provider's own; SlipStream assumes none.
type Allowance struct {
	Downloads, Searches *Window
}

// Window is one counted allowance.
type Window struct {
	Used, Limit, Remaining int
	Resets                 time.Time
	// Exact is whether the provider counts this window up to the call, or
	// reports it with a lag.
	Exact bool
}
