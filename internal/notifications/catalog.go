package notifications

import (
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
)

// systemGroup is Test's group on a connection's event list. History has
// no such group: Test records nothing.
const systemGroup entry.Group = "system"

var testRow = entry.Row[testFacts]{
	Spec:   entry.Spec{Kind: "test", Label: "Test", Group: systemGroup, Fixed: true},
	Render: renderTest,
}

type testFacts struct{}

func (testFacts) About() entry.About { return entry.About{} }

func renderTest(_ testFacts, c entry.Context) entry.Message {
	return entry.Message{
		Title: "Test from SlipStream",
		Lines: []string{"This connection works.", fmt.Sprintf("%s sent this message at %s.", c.Sender, c.Clock(c.Now))},
	}
}

// Events is History's kinds and then Test, for the events API and the
// web's toggles.
func (s *Service) Events() []entry.Spec {
	return append(s.cfg.Catalog.Specs(), testRow.Spec)
}

// spec is k's declaration. Every kind notifications holds has one: History
// records only registered kinds.
func (s *Service) spec(k entry.Kind) entry.Spec {
	if k == testRow.Kind {
		return testRow.Spec
	}
	spec, ok := s.cfg.Catalog.Spec(k)
	if !ok {
		panic(fmt.Sprintf("notifications: %s isn't in the catalog", k))
	}
	return spec
}

// render is e's message as a provider or the bell takes it.
func (s *Service) render(e event, now time.Time, sender string) (entry.Message, error) {
	c := entry.Context{Now: now, Zone: s.cfg.Zone, Sender: sender}
	if e.Kind == testRow.Kind {
		return testRow.RenderStored(e.Subject, c)
	}
	return s.cfg.Catalog.Render(e.Kind, e.Subject, c)
}
