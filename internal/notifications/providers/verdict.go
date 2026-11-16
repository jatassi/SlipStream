package providers

import "time"

// Outcome is what a send's reply means for the message and its connection.
type Outcome uint8

const (
	Sent Outcome = iota + 1
	// SlowDown: the provider asked for a wait. It isn't a failure.
	SlowDown
	// Transient: a failure that a retry may fix.
	Transient
	// RefusedConnection: only the Admin can fix it, so the connection
	// pauses.
	RefusedConnection
	// RefusedContent: this message can't go, and the next one may.
	RefusedContent
)

type Verdict struct {
	Outcome Outcome
	// Until is when a SlowDown ends.
	Until time.Time
	// Reason is set for every Outcome but Sent.
	Reason Reason
	// WithoutImage: a RefusedContent 413 on a body with a poster.
	WithoutImage bool
	// Response is kept on the given-up delivery for RefusedContent, and
	// logged for a Sent that carries one (Apprise 424).
	Response []byte
	// Accepted are the Email addresses that took it.
	Accepted []string
}
