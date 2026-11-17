package notifications

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

type grabbedFacts struct {
	Release string `json:"release"`
}

func (grabbedFacts) About() entry.About { return entry.About{} }

var (
	grabbedRow = entry.Row[grabbedFacts]{Spec: entry.Spec{Kind: "grabbed", Label: "Grabbed", Group: entry.Grabs}}
	failedRow  = entry.Row[grabbedFacts]{Spec: entry.Spec{Kind: "grab_failed", Label: "Grab failed", Group: entry.Grabs}}
	deletedRow = entry.Row[grabbedFacts]{Spec: entry.Spec{Kind: "works_deleted", Label: "Works deleted", Group: entry.Library}}
)

func TestTheEventsListHealthThenOwnersThenTest(t *testing.T) {
	cat := entry.NewCatalog()
	s := New(Config{Catalog: cat})
	h := history.New(history.Config{Catalog: cat, Outbox: s, Log: slog.New(slog.DiscardHandler)})
	h.Recorder("grabs", []entry.Entry{grabbedRow, failedRow})
	h.Recorder("library", []entry.Entry{deletedRow})
	var got []entry.Kind
	for _, spec := range s.Events() {
		got = append(got, spec.Kind)
	}
	if want := []entry.Kind{"new_health_item", "health_item_cleared", "grabbed", "grab_failed", "works_deleted", "test"}; !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestTwoProvidersWithOneIDPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New didn't panic")
		}
	}()
	New(Config{Providers: []providers.Provider{{ID: "discord"}, {ID: "slack"}, {ID: "discord"}}})
}
