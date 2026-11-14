package mstest

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// hubEvents are the events the World publishes on its hub, which Health
// follows.
var hubEvents = []live.Event{
	mediaservers.ChangedEvent, mediaservers.SessionsChangedEvent, mediaservers.UnhealthyEvent,
	mediaservers.RecoveredEvent, mediaservers.ItemsFoundEvent, roots.ChangedEvent, nodes.ChangedEvent,
}

// Transition is one media-servers.unhealthy, or media-servers.recovered
// when Recovered, as published.
type Transition struct {
	ServerID, Kind, Reason string
	Recovered              bool
}

// Transitions is every media-servers.unhealthy and .recovered published so
// far, in order.
func (w *World) Transitions() []Transition {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]Transition(nil), w.transitions...)
}

type unhealthyEvents struct{ w *World }

func (e unhealthyEvents) Publish(ctx context.Context, v mediaservers.WentUnhealthy) {
	e.w.mu.Lock()
	e.w.transitions = append(e.w.transitions, Transition{ServerID: v.ServerID, Kind: kindOf(v.Kind), Reason: v.Reason})
	e.w.mu.Unlock()
	mediaservers.UnhealthyEvent.Bind(e.w.hub).Publish(ctx, v)
}

type recoveredEvents struct{ w *World }

func (e recoveredEvents) Publish(ctx context.Context, v mediaservers.Recovered) {
	e.w.mu.Lock()
	e.w.transitions = append(e.w.transitions, Transition{ServerID: v.ServerID, Kind: kindOf(v.Kind), Reason: v.Reason, Recovered: true})
	e.w.mu.Unlock()
	mediaservers.RecoveredEvent.Bind(e.w.hub).Publish(ctx, v)
}

// Health is home's Health with the Media Servers category, every other
// category answering nothing, running until the test ends.
func (w *World) Health(t *testing.T) *health.Health {
	t.Helper()
	none := health.Source{Answer: func(context.Context) (health.Answer, error) { return health.Answer{}, nil }}
	var cats []health.Category
	for _, id := range health.IDs()[1:] {
		c := health.Category{About: health.About{ID: id, Label: string(id), Phrase: string(id)}, Source: none}
		if id == health.MediaServersID {
			c = health.Category{About: mediaservers.HealthAbout, Source: w.ms.HealthSource()}
		}
		cats = append(cats, c)
	}
	h, err := health.Open(t.Context(), health.Config{
		Library: w.Lib, Hub: w.hub, Changed: discard{},
		Storage:    health.StorageParts{Roots: none},
		Categories: cats,
		Names: func(context.Context, storage.Reader[storage.Library]) (map[role.NodeID]string, error) {
			return map[role.NodeID]string{w.Home: "home", w.Seedbox: "seedbox"}, nil
		},
		Zone: time.UTC, Timings: health.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return h
}

func kindOf(k api.Enum[mediaservers.Kind]) string {
	b, _ := k.MarshalJSON()
	return strings.Trim(string(b), `"`)
}

type discard struct{}

func (discard) Publish(context.Context, health.Changed) {}
