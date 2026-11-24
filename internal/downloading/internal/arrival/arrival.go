// Package arrival is a Move's start ask as its destination Node judges
// it: the Node starts the torrent only inside the window the Controller
// gave, judges the start by the tracker's answer within the reply wait,
// and holds the torrent paused once the window ends with the ask still
// open. It is pure: every time comes from its caller, on the Node's
// monotonic clock.
package arrival

import (
	"time"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
)

// The reasons of a refused start the Node gives itself. A tracker's
// failure reason goes verbatim.
const (
	// Late is an ask whose window was shorter than the reply wait before
	// the torrent could start.
	Late = "late"
	// NoAnswer is no tracker answer within the reply wait, or a window
	// that ended with the ask still open.
	NoAnswer = "no_answer"
)

// Ask is one start ask of one torrent.
type Ask struct {
	// asked is the Controller's time of the ask, the echo key.
	asked int64
	until time.Time
	wait  time.Duration
	// started is when the Node resumed the torrent, zero before.
	started time.Time
	// held is the torrent kept or put paused: the ask came late or its
	// window ended.
	held   bool
	result *dlv1.Arrival
}

// Of is the ask a Torrent of the intent carries, kept from was when it is
// the same ask, so its window runs from its first receipt at now. A
// Torrent with no ask is nil.
func Of(was *Ask, t *dlv1.Torrent, wait time.Duration, now time.Time) *Ask {
	asked := t.GetStartAskedUnixMs()
	switch {
	case asked == 0:
		return nil
	case was != nil && was.asked == asked:
		return was
	}
	a := &Ask{asked: asked, until: now.Add(time.Duration(t.GetStartWindowMs()) * time.Millisecond), wait: wait}
	a.Tick(now)
	return a
}

// Held is whether the torrent stays paused whatever its wanted state.
func (a *Ask) Held() bool { return a != nil && a.held }

// Start notes the torrent resumed at now. Only the first counts.
func (a *Ask) Start(now time.Time) {
	if a != nil && a.started.IsZero() && !a.held {
		a.started = now
	}
}

// Replied notes a tracker's answer at at, which judges a start it
// answered.
func (a *Ask) Replied(at time.Time) {
	if a.judging(at) {
		a.result = &dlv1.Arrival{AskedUnixMs: a.asked, Result: dlv1.ArrivalResult_ARRIVAL_RESULT_STARTED, AnnouncedUnixMs: at.UnixMilli()}
	}
}

// Failed notes a tracker's failure answer at at with its reason, which
// refuses a start it answered.
func (a *Ask) Failed(at time.Time, reason string) {
	if a.judging(at) {
		a.refuse(reason)
	}
}

func (a *Ask) judging(at time.Time) bool {
	return a != nil && !a.started.IsZero() && a.result == nil && !at.Before(a.started)
}

func (a *Ask) refuse(reason string) {
	a.result = &dlv1.Arrival{AskedUnixMs: a.asked, Result: dlv1.ArrivalResult_ARRIVAL_RESULT_REFUSED, Reason: reason}
}

// Tick judges the ask at now: a torrent not yet started whose window
// left is under the reply wait is refused late, a start no tracker
// answered within the reply wait is refused, and the window's end holds
// the torrent paused. It answers when the ask next needs a Tick, zero for
// never.
func (a *Ask) Tick(now time.Time) time.Time {
	if a == nil || a.held {
		return time.Time{}
	}
	if a.started.IsZero() {
		if a.until.Sub(now) < a.wait {
			a.held = true
			a.refuse(Late)
			return time.Time{}
		}
		return a.until.Add(-a.wait)
	}
	if !now.Before(a.until) {
		a.held = true
		if a.result.GetResult() != dlv1.ArrivalResult_ARRIVAL_RESULT_REFUSED {
			a.refuse(NoAnswer)
		}
		return time.Time{}
	}
	if a.result == nil {
		if replyBy := a.started.Add(a.wait); now.Before(replyBy) {
			return replyBy
		}
		a.refuse(NoAnswer)
	}
	return a.until
}

// Result is the ask's arrival so far, nil before one.
func (a *Ask) Result() *dlv1.Arrival {
	if a == nil {
		return nil
	}
	return a.result
}

// Started is when the torrent resumed for the ask, zero before.
func (a *Ask) Started() time.Time {
	if a == nil {
		return time.Time{}
	}
	return a.started
}
