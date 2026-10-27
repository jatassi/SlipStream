package indexers

import (
	"context"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
)

var HealthAbout = health.About{
	ID:     health.IndexersID,
	Label:  "Indexers",
	Phrase: "Indexers",
	Events: []live.Event{ChangedEvent, StatusChangedEvent},
}

// HealthSource is the indexers category's items. It raises none yet.
func (x *Indexers) HealthSource() health.Source {
	return health.Source{Answer: func(context.Context) (health.Answer, error) { return health.Answer{}, nil }}
}
