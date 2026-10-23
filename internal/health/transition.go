package health

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/live"
)

// Kind is how an item changed.
type Kind uint8

const (
	// Raised: the item appeared.
	Raised Kind = iota + 1
	Worse
	Better
	// Cleared: the item went and its row with it.
	Cleared
)

func (k Kind) String() string { return [...]string{"", "raised", "worse", "better", "cleared"}[k] }

// Transition is one item's change of level, seen once: Config.Transitions
// takes it inside the Write that records it.
type Transition struct {
	Category ID
	Key      Key
	Kind     Kind
	// From is None for Raised; To is None for Cleared.
	From, To Level
	// Subject and Text are the words at the change: the new answer's, or
	// for Cleared the last answer this process saw.
	Subject, Text string
	At            time.Time
}

// Level is the item's level after the change, or before it when cleared.
func (t Transition) Level() Level {
	if t.Kind == Cleared {
		return t.From
	}
	return t.To
}

// Changed names the categories whose shown items changed, and storage
// after its levels change.
type Changed struct {
	Categories []string `json:"categories" nullable:"false"`
}

var ChangedEvent = live.AdminEvent[Changed]("health.changed")

type ChangedPublisher interface {
	Publish(ctx context.Context, v Changed)
}
