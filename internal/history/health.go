package history

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// healthWords is every label and sentence that names a Health item in an
// entry. The catalog labels and the renders read it.
var healthWords = struct {
	NewLabel     string
	ClearedLabel string
	// NewMany takes the count.
	NewMany string
	// ClearedMany takes the count.
	ClearedMany string
	// NewOne takes the item's subject.
	NewOne string
	// ClearedOne takes the item's subject.
	ClearedOne string
	// OnHealthSince takes the clock time it appeared.
	OnHealthSince string
	// OnHealthFrom takes the clock times it appeared and cleared.
	OnHealthFrom string
}{
	NewLabel:      "New Health item",
	ClearedLabel:  "Health item cleared",
	NewMany:       "%s new Health items",
	ClearedMany:   "%s Health items cleared",
	NewOne:        "New Health item for %s",
	ClearedOne:    "Health item for %s cleared",
	OnHealthSince: "It has been on Health since %s.",
	OnHealthFrom:  "It was on Health from %s to %s.",
}

// The Health rows store no facts: an entry's items are its history_item
// rows, and whoever renders one builds HealthFacts from them.
var (
	NewItemsRow = entry.Row[HealthFacts]{
		Spec:   entry.Spec{Kind: "new_health_item", Label: healthWords.NewLabel, Group: entry.Health, OnByDefault: true, BellByDefault: true},
		Render: renderNewItems,
	}
	ClearedRow = entry.Row[HealthFacts]{
		Spec:   entry.Spec{Kind: "health_item_cleared", Label: healthWords.ClearedLabel, Group: entry.Health, OnByDefault: true, BellByDefault: true},
		Render: renderCleared,
	}
)

// IsHealth is whether k is one of the Health kinds, whose items are kept
// beside the entry.
func IsHealth(k entry.Kind) bool { return k == NewItemsRow.Kind || k == ClearedRow.Kind }

// HealthFacts is the items a Health entry renders.
type HealthFacts struct {
	Items []Item `json:"items"`
}

func (HealthFacts) About() entry.About { return entry.About{} }

// Item is one Health item of an entry.
type Item struct {
	// Key is "<category>.<key>". Category ids hold no ".".
	Key     string    `json:"key"`
	Level   string    `json:"level"`
	Subject string    `json:"subject"`
	Text    string    `json:"text"`
	Since   time.Time `json:"since"`
	Cleared time.Time `json:"cleared,omitzero"`
}

// ItemKey is a Health item's key in History: "<category>.<key>".
func ItemKey(c health.ID, k health.Key) string { return string(c) + "." + string(k) }

func (it Item) Failing() bool { return it.Level == health.Fail.String() }

func (it Item) parts() (category, key string) {
	category, key, _ = strings.Cut(it.Key, ".")
	return category, key
}

func (it Item) params(id EventID) db.InsertItemParams {
	return db.InsertItemParams{
		EventID: id, Key: it.Key, Level: it.Level, Subject: it.Subject, Text: it.Text,
		SinceAt: storage.FromTime(it.Since), ClearedAt: storage.FromTime(it.Cleared),
	}
}

func itemOf(row db.HistoryItem) Item {
	return Item{
		Key: row.Key, Level: row.Level, Subject: row.Subject, Text: row.Text,
		Since: row.SinceAt.Time(), Cleared: row.ClearedAt.Time(),
	}
}

func renderNewItems(f HealthFacts, c entry.Context) entry.Message {
	m := entry.Message{Tone: entry.Warn, Body: entry.Body{Health: bodyItems(f.Items)}}
	for _, it := range f.Items {
		if it.Failing() {
			m.Tone = entry.Fail
		}
	}
	if len(f.Items) == 1 {
		it := f.Items[0]
		m.Title = fmt.Sprintf(healthWords.NewOne, it.Subject)
		m.Lines = []string{it.Text, fmt.Sprintf(healthWords.OnHealthSince, c.Clock(it.Since))}
		m.Link = entry.HealthLink(it.parts())
		return m
	}
	m.Title = fmt.Sprintf(healthWords.NewMany, entry.Grouped(len(f.Items)))
	m.Lines = subjects(f.Items)
	return m
}

func renderCleared(f HealthFacts, c entry.Context) entry.Message {
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

func subjects(items []Item) []string {
	out := make([]string, 0, 3)
	for _, it := range items[:min(len(items), 3)] {
		out = append(out, it.Subject)
	}
	return out
}

func bodyItems(items []Item) []entry.BodyItem {
	out := make([]entry.BodyItem, len(items))
	for i, it := range items {
		category, key := it.parts()
		out[i] = entry.BodyItem{Category: category, Key: key, Level: it.Level, Subject: it.Subject, Text: it.Text}
	}
	return out
}

// Transitions is Health's sink, inside Health's Write. Every Raised and
// Worse of one evaluation is one New Health item entry, and every Cleared
// one Health item cleared entry, as captured. Better is ignored.
func (s *Service) Transitions(ctx context.Context, tx *storage.Tx[storage.Library], ts []health.Transition) error {
	var raised, cleared []Item
	for _, t := range ts {
		key := ItemKey(t.Category, t.Key)
		switch t.Kind {
		case health.Raised, health.Worse:
			raised = append(raised, Item{Key: key, Level: t.To.String(), Subject: t.Subject, Text: t.Text, Since: t.Since})
		case health.Cleared:
			it, ok, err := s.clearOf(ctx, tx, t, key)
			if err != nil {
				return err
			}
			if !ok {
				s.cfg.Log.Warn("skipped a Health clear with no words", "key", key)
				continue
			}
			cleared = append(cleared, it)
		case health.Better:
		}
	}
	for _, e := range []struct {
		row   entry.Row[HealthFacts]
		items []Item
	}{{NewItemsRow, raised}, {ClearedRow, cleared}} {
		if len(e.items) == 0 {
			continue
		}
		if err := s.store(ctx, tx, e.row.Kind, []byte("{}"), entry.About{}, e.items); err != nil {
			return err
		}
	}
	return nil
}

// clearOf is the item a clear of key records. Its words are the newest
// item's for key, else what a target was last told, else the transition's.
// false: there are no words at all.
func (s *Service) clearOf(ctx context.Context, tx *storage.Tx[storage.Library], t health.Transition, key string) (Item, bool, error) {
	it := Item{Key: key, Level: t.From.String(), Cleared: t.At}
	newest, err := db.New(tx).NewestItem(ctx, key)
	switch {
	case err == nil:
		it.Subject, it.Text, it.Since = newest.Subject, newest.Text, newest.SinceAt.Time()
		return it, true, nil
	case !errors.Is(err, storage.ErrNoRows):
		return Item{}, false, err
	}
	told, ok, err := s.cfg.Outbox.Told(ctx, tx, key)
	switch {
	case err != nil:
		return Item{}, false, err
	case ok:
		it.Subject, it.Since = told.Subject, told.Since
	default:
		it.Subject, it.Text, it.Since = t.Subject, t.Text, t.Since
	}
	return it, it.Subject != "", nil
}
