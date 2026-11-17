package notifications

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history/entry"
)

// The Health rows' facts are built at release from the event's items, one
// subset per target, so Record never stores them.
var (
	newItemsRow = entry.Row[healthFacts]{
		Spec:   entry.Spec{Kind: "new_health_item", Label: healthWords.NewLabel, Group: entry.Health, OnByDefault: true, BellByDefault: true},
		Render: renderNewItems,
	}
	clearedRow = entry.Row[healthFacts]{
		Spec:   entry.Spec{Kind: "health_item_cleared", Label: healthWords.ClearedLabel, Group: entry.Health, OnByDefault: true, BellByDefault: true},
		Render: renderCleared,
	}
	testRow = entry.Row[testFacts]{
		Spec:   entry.Spec{Kind: "test", Label: "Test", Group: systemGroup, Fixed: true},
		Render: renderTest,
	}
)

// systemGroup is Test's group on a connection's event list. History has
// no such group: Test records nothing.
const systemGroup entry.Group = "system"

// healthFacts is one target's subset of an event's Health items.
type healthFacts struct {
	Items []item `json:"items"`
}

type item struct {
	Category health.ID `json:"category"`
	Key      string    `json:"key"`
	Level    string    `json:"level"`
	Subject  string    `json:"subject"`
	Text     string    `json:"text"`
	Since    time.Time `json:"since"`
	Cleared  time.Time `json:"cleared,omitzero"`
}

type testFacts struct{}

func (healthFacts) About() entry.About { return entry.About{} }

func (testFacts) About() entry.About { return entry.About{} }

func renderNewItems(f healthFacts, c entry.Context) entry.Message {
	m := entry.Message{Tone: entry.Warn, Body: entry.Body{Health: bodyItems(f.Items)}}
	for _, it := range f.Items {
		if it.Level == health.Fail.String() {
			m.Tone = entry.Fail
		}
	}
	if len(f.Items) == 1 {
		it := f.Items[0]
		m.Title = it.Subject
		m.Lines = []string{it.Text, fmt.Sprintf(healthWords.OnHealthSince, c.Clock(it.Since))}
		m.Link = entry.HealthLink(string(it.Category), it.Key)
		return m
	}
	m.Title = fmt.Sprintf(healthWords.NewMany, entry.Grouped(len(f.Items)))
	m.Lines = subjects(f.Items)
	return m
}

func renderCleared(f healthFacts, c entry.Context) entry.Message {
	m := entry.Message{Tone: entry.Good, Body: entry.Body{Health: bodyItems(f.Items)}}
	if len(f.Items) == 1 {
		it := f.Items[0]
		m.Title = fmt.Sprintf(healthWords.ClearedOne, it.Subject)
		m.Lines = []string{fmt.Sprintf(healthWords.OnHealthFrom, c.Clock(it.Since), c.Clock(it.Cleared))}
		return m
	}
	m.Title = fmt.Sprintf(healthWords.ClearedMany, entry.Grouped(len(f.Items)))
	m.Lines = subjects(f.Items)
	return m
}

func renderTest(_ testFacts, c entry.Context) entry.Message {
	return entry.Message{
		Title: "Test from SlipStream",
		Lines: []string{"This connection works.", fmt.Sprintf("%s sent this message at %s.", c.Sender, c.Clock(c.Now))},
	}
}

func subjects(items []item) []string {
	out := make([]string, 0, 3)
	for _, it := range items[:min(len(items), 3)] {
		out = append(out, it.Subject)
	}
	return out
}

func bodyItems(items []item) []entry.BodyItem {
	out := make([]entry.BodyItem, len(items))
	for i, it := range items {
		out[i] = entry.BodyItem{Category: string(it.Category), Key: it.Key, Level: it.Level, Subject: it.Subject, Text: it.Text}
	}
	return out
}

// catalog is every row an event can be recorded or rendered from:
// notifications' Health rows, then the owners' in registration order, then
// Test. Owners register while the node wires the Controller, and Run seals
// it.
type catalog struct {
	mu     sync.Mutex
	rows   []entry.Entry
	sealed bool
}

func newCatalog() *catalog {
	return &catalog{rows: []entry.Entry{newItemsRow, clearedRow}}
}

func (c *catalog) register(owner string, rows []entry.Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sealed {
		panic(fmt.Sprintf("notifications: %s registered its events after the dispatcher started", owner))
	}
	for _, e := range rows {
		k := e.Declared().Kind
		if k == testRow.Kind || slices.ContainsFunc(c.rows, func(r entry.Entry) bool { return r.Declared().Kind == k }) {
			panic(fmt.Sprintf("notifications: %s registers %s, which is already registered", owner, k))
		}
		c.rows = append(c.rows, e)
	}
}

// seal ends registration. Run calls it before its first pass.
func (c *catalog) seal() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sealed = true
}

func (c *catalog) all() []entry.Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append(slices.Clone(c.rows), testRow)
}

// Events is the catalog, for the events API and the web's toggles.
func (s *Service) Events() []entry.Spec {
	rows := s.catalog.all()
	out := make([]entry.Spec, len(rows))
	for i, e := range rows {
		out[i] = e.Declared()
	}
	return out
}
