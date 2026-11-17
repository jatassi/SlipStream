package notifications

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/notifications/notice"
)

// The Health rows' facts are built at release from the event's items, one
// subset per target, so Record never stores them.
var (
	newItemsRow = notice.Row[healthFacts]{
		Spec:   notice.Spec{Kind: "new_health_item", Label: healthWords.NewLabel, Group: notice.Health, OnByDefault: true, BellByDefault: true},
		Render: renderNewItems,
	}
	clearedRow = notice.Row[healthFacts]{
		Spec:   notice.Spec{Kind: "health_item_cleared", Label: healthWords.ClearedLabel, Group: notice.Health, OnByDefault: true, BellByDefault: true},
		Render: renderCleared,
	}
	testRow = notice.Row[testFacts]{
		Spec:   notice.Spec{Kind: "test", Label: "Test", Group: notice.System, Fixed: true},
		Render: renderTest,
	}
)

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

func renderNewItems(f healthFacts, c notice.Context) notice.Message {
	m := notice.Message{Tone: notice.Warn, Body: notice.Body{Health: bodyItems(f.Items)}}
	for _, it := range f.Items {
		if it.Level == health.Fail.String() {
			m.Tone = notice.Fail
		}
	}
	if len(f.Items) == 1 {
		it := f.Items[0]
		m.Title = it.Subject
		m.Lines = []string{it.Text, fmt.Sprintf(healthWords.OnHealthSince, c.Clock(it.Since))}
		m.Link = notice.HealthLink(string(it.Category), it.Key)
		return m
	}
	m.Title = fmt.Sprintf(healthWords.NewMany, notice.Grouped(len(f.Items)))
	m.Lines = subjects(f.Items)
	return m
}

func renderCleared(f healthFacts, c notice.Context) notice.Message {
	m := notice.Message{Tone: notice.Good, Body: notice.Body{Health: bodyItems(f.Items)}}
	if len(f.Items) == 1 {
		it := f.Items[0]
		m.Title = fmt.Sprintf(healthWords.ClearedOne, it.Subject)
		m.Lines = []string{fmt.Sprintf(healthWords.OnHealthFrom, c.Clock(it.Since), c.Clock(it.Cleared))}
		return m
	}
	m.Title = fmt.Sprintf(healthWords.ClearedMany, notice.Grouped(len(f.Items)))
	m.Lines = subjects(f.Items)
	return m
}

func renderTest(_ testFacts, c notice.Context) notice.Message {
	return notice.Message{
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

func bodyItems(items []item) []notice.BodyItem {
	out := make([]notice.BodyItem, len(items))
	for i, it := range items {
		out[i] = notice.BodyItem{Category: string(it.Category), Key: it.Key, Level: it.Level, Subject: it.Subject, Text: it.Text}
	}
	return out
}

// catalog is every row an event can be recorded or rendered from:
// notifications' Health rows, then the owners' in registration order, then
// Test. Owners register while the node wires the Controller, and Run seals
// it.
type catalog struct {
	mu     sync.Mutex
	rows   []notice.Entry
	sealed bool
}

func newCatalog() *catalog {
	return &catalog{rows: []notice.Entry{newItemsRow, clearedRow}}
}

func (c *catalog) register(owner string, rows []notice.Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sealed {
		panic(fmt.Sprintf("notifications: %s registered its events after the dispatcher started", owner))
	}
	for _, e := range rows {
		k := e.Declared().Kind
		if k == testRow.Kind || slices.ContainsFunc(c.rows, func(r notice.Entry) bool { return r.Declared().Kind == k }) {
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

func (c *catalog) all() []notice.Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append(slices.Clone(c.rows), testRow)
}

// Events is the catalog, for the events API and the web's toggles.
func (s *Service) Events() []notice.Spec {
	rows := s.catalog.all()
	out := make([]notice.Spec, len(rows))
	for i, e := range rows {
		out[i] = e.Declared()
	}
	return out
}
