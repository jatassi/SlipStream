package notifications

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/storage"
)

// State is a connection's delivery state.
type State string

const (
	OK      State = "ok"
	Failing State = "failing"
	// Waiting: the provider asked for a wait.
	Waiting State = "waiting"
	// Paused: a refusal only the Admin can fix.
	Paused State = "paused"
)

// Line is one connection's delivery state. Only the dispatcher writes it.
// A connection with no stored line is ok at its current revision.
type Line struct {
	State State
	// Since is kept from failing to waiting and back, and set by ok and
	// paused.
	Since    time.Time
	Reason   providers.Reason
	Attempts int
	Next     time.Time
	// Revision is the connection's revision this line was judged at. A line
	// below its connection's starts fresh, which is how an edit or a
	// passing Test unpauses it.
	Revision int64
	LastSent time.Time
	// GivenUp counts deliveries given up since the revision last moved.
	GivenUp int
}

// fresh is l once its connection's revision moved: ok, nothing to wait
// for and nothing counted.
func (l Line) fresh(rev int64, now time.Time) Line {
	return Line{State: OK, Since: now, Next: now, Revision: rev, LastSent: l.LastSent}
}

// DeliveryState is where one message to one connection is. Storage keeps
// only due and given-up deliveries: a sent or dropped one is deleted in
// the Write that decides it.
type DeliveryState string

const (
	Due     DeliveryState = "due"
	Sent    DeliveryState = "sent"
	GivenUp DeliveryState = "given_up"
	Dropped DeliveryState = "dropped"
)

type Delivery struct {
	Event    EventID
	State    DeliveryState
	Attempts int
	// Items are the Health keys it carries, nil for other kinds.
	Items []string
	// Accepted are the Email addresses that already took it.
	Accepted []string
	// Reason and Response are set once it is given up. Response is at most
	// 1 KiB with the connection's secret values masked.
	Reason   providers.Reason
	Response []byte
}

// applyVerdict is the one transition table of a connection and the
// delivery a send answered for. rev is the revision the send ran at.
func applyVerdict(l Line, d Delivery, rev int64, v providers.Verdict, now time.Time, ladder []time.Duration, rnd *rand.Rand) (Line, Delivery) {
	switch v.Outcome {
	case providers.Sent:
		d.State = Sent
		return Line{State: OK, Since: now, Next: now, Revision: l.Revision, LastSent: now, GivenUp: l.GivenUp}, d
	case providers.SlowDown:
		d.Attempts++
		return Line{
			State: Waiting, Since: troubleSince(l, now), Reason: providers.Reason{Code: providers.RateLimited, Until: v.Until},
			Next: v.Until, Revision: l.Revision, LastSent: l.LastSent, GivenUp: l.GivenUp,
		}, d
	case providers.Transient:
		d.Attempts++
		d.Accepted = union(d.Accepted, v.Accepted)
		rung := ladder[min(l.Attempts, len(ladder)-1)]
		return Line{
			State: Failing, Since: troubleSince(l, now), Reason: v.Reason, Attempts: l.Attempts + 1,
			Next: now.Add(backoff.Delay(rung, rung, 0, rnd)), Revision: l.Revision, LastSent: l.LastSent, GivenUp: l.GivenUp,
		}, d
	case providers.RefusedConnection:
		return Line{State: Paused, Since: now, Reason: v.Reason, Next: now, Revision: rev, LastSent: l.LastSent, GivenUp: l.GivenUp}, d
	case providers.RefusedContent:
		d.State, d.Reason, d.Response = GivenUp, v.Reason, v.Response
		return Line{State: OK, Since: now, Next: now, Revision: l.Revision, LastSent: l.LastSent, GivenUp: l.GivenUp + 1}, d
	}
	panic(fmt.Sprintf("notifications: a verdict with outcome %d", v.Outcome))
}

// troubleSince is when l's trouble began: kept while it fails or waits.
func troubleSince(l Line, now time.Time) time.Time {
	if l.State == Failing || l.State == Waiting {
		return l.Since
	}
	return now
}

func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, s := range b {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func lineOf(row db.NotificationsLine) (Line, error) {
	l := Line{
		State: State(row.State), Since: row.StateAt.Time(), Attempts: int(row.Attempts), Next: row.NextAt.Time(),
		Revision: row.Revision, LastSent: row.LastSentAt.Time(), GivenUp: int(row.GivenUp),
	}
	if row.Reason != nil {
		if err := json.Unmarshal([]byte(*row.Reason), &l.Reason); err != nil {
			return Line{}, fmt.Errorf("decode the reason of a line: %w", err)
		}
	}
	return l, nil
}

func (l Line) params(id ConnectionID) db.UpsertLineParams {
	p := db.UpsertLineParams{
		ConnectionID: id, State: string(l.State), StateAt: storage.FromTime(l.Since), Attempts: int64(l.Attempts),
		NextAt: storage.FromTime(l.Next), Revision: l.Revision, LastSentAt: storage.FromTime(l.LastSent), GivenUp: int64(l.GivenUp),
	}
	if l.State != OK {
		p.Reason = jsonText(l.Reason)
	}
	return p
}

// jsonText is v as stored JSON text.
func jsonText(v any) *string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("notifications: encode %T: %v", v, err))
	}
	s := string(b)
	return &s
}
