package link

import (
	"errors"
	"fmt"
)

// Reason is why a server's own response failed a call.
type Reason string

const (
	// ReasonUnreachable is no response at the address: refused, timed out,
	// a TLS failure or a name that doesn't resolve.
	ReasonUnreachable Reason = "unreachable"
	// ReasonRefused is the server refusing the credential.
	ReasonRefused Reason = "refused"
	// ReasonOtherServer is a different identity responding at the address.
	ReasonOtherServer Reason = "other_server"
	// ReasonUnsupported is a route or behavior this server's version lacks.
	ReasonUnsupported Reason = "unsupported"
	// ReasonFailed is any other error response.
	ReasonFailed Reason = "failed"
)

func Reasons() []Reason {
	return []Reason{ReasonUnreachable, ReasonRefused, ReasonOtherServer, ReasonUnsupported, ReasonFailed}
}

func ParseReason(s string) (Reason, error) { return parse(s, Reasons(), "problem reason") }

// Problem is a failed call as the server responded to it. It is stored on
// the change it failed and crosses the wire as a call's answer.
type Problem struct {
	Reason Reason `json:"reason"`
	// Status is the HTTP status, 0 when there was no response.
	Status int `json:"status,omitempty"`
	// Found is, for ReasonOtherServer, the identity that responded.
	Found string `json:"found,omitempty"`
	// Detail is the server's own message or the transport error.
	Detail string `json:"detail,omitempty"`
}

// ServerError is a step's error when the server itself failed the call. It
// counts an attempt. Any other error, such as an rpc error, a canceled
// call or the deadline, is the link's: the change keeps its step and no
// attempt is counted.
type ServerError struct{ Problem }

func (e *ServerError) Error() string {
	switch e.Reason {
	case ReasonUnreachable:
		return "the Media Server didn't respond: " + e.Detail
	case ReasonRefused:
		return fmt.Sprintf("the Media Server refused SlipStream's credential (status %d)", e.Status)
	case ReasonOtherServer:
		return "a different Media Server responded, with id " + e.Found
	case ReasonUnsupported:
		return "the Media Server doesn't support this: " + e.Detail
	case ReasonFailed:
	}
	return fmt.Sprintf("the Media Server responded with status %d: %s", e.Status, e.Detail)
}

// ServerProblem is err's Problem when the server itself failed the call,
// false for a link error.
func ServerProblem(err error) (Problem, bool) {
	if se, ok := errors.AsType[*ServerError](err); ok {
		return se.Problem, true
	}
	return Problem{}, false
}
