package notifications

import (
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/role"
)

// Shown is what a connection's row, its sentence and its Health item say.
type Shown string

const (
	ShownOff     Shown = "off"
	ShownOK      Shown = "ok"
	ShownFailing Shown = "failing"
	ShownWaiting Shown = "waiting"
	ShownPaused  Shown = "paused"
	// ShownHolding: a Custom Script whose Node is offline. Its messages
	// wait, and Nodes already says the Node is offline.
	ShownHolding Shown = "holding"
)

// Status is the one reading of a connection, shared by the dispatcher,
// Health and the API.
type Status struct {
	Shown Shown
	Since time.Time
	// Until is when a waiting connection sends again.
	Until time.Time
	// Reason is worded when read, by wording.reason.
	Reason   providers.Reason
	Level    health.Level
	Waiting  int
	GivenUp  int
	LastSent time.Time
}

// NodeView is what statusOf reads of a Custom Script's Node.
type NodeView struct {
	ID     role.NodeID
	Name   string
	InStep bool
	// Removed: the Admin removed the Node or wrote off its Notifying Role.
	Removed bool
}

// statusOf reads, in order: off; a script whose Node was removed; a line
// paused at the connection's revision; a line below it, which starts
// fresh; a script whose Node is offline; then the line's own state.
// Paused is an Error at once. Failing or waiting is a Warning once it has
// lasted warn.
func statusOf(c Connection, l Line, n NodeView, waiting int, now time.Time, warn time.Duration) Status {
	s := Status{Waiting: waiting, GivenUp: l.GivenUp, LastSent: l.LastSent}
	switch {
	case !c.On:
		s.Shown = ShownOff
		return s
	case c.script() && n.Removed:
		s.Shown, s.Level = ShownPaused, health.Fail
		s.Reason = providers.Reason{Code: nodeRemoved, Node: n.ID}
		return s
	case l.Revision < c.Revision:
		s.Shown, s.GivenUp = ShownOK, 0
		return s
	case l.State == Paused:
		s.Shown, s.Since, s.Reason, s.Level = ShownPaused, l.Since, l.Reason, health.Fail
		return s
	case c.script() && !n.InStep:
		s.Shown = ShownHolding
		s.Reason = providers.Reason{Code: nodeOffline, Node: n.ID}
		return s
	case l.State == OK:
		s.Shown = ShownOK
		return s
	}
	s.Shown, s.Since, s.Reason = ShownFailing, l.Since, l.Reason
	if l.State == Waiting {
		s.Shown, s.Until = ShownWaiting, l.Next
	}
	if now.Sub(l.Since) >= warn {
		s.Level = health.Warn
	}
	return s
}

// MaySend is whether the dispatcher may claim the connection's next
// message now.
func (s Status) MaySend(l Line, now time.Time) bool {
	switch s.Shown {
	case ShownOK:
		return true
	case ShownFailing, ShownWaiting:
		return !now.Before(l.Next)
	case ShownOff, ShownPaused, ShownHolding:
	}
	return false
}
