package works_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/works"
)

func TestMonitoringCountsLiveItemsAndAsidesOnlyWhenWanted(t *testing.T) {
	h := newHarness(t)
	winter := func(things ...works.Entry) works.Entry {
		return works.Entry{Key: "b1", Label: "Winter", Children: append([]works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Gloves"}}, things...)}
	}
	button := works.Entry{Key: "c9", Label: "Button"}
	stock := func(boxes ...works.Entry) {
		h.fx.Put("attic", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: boxes}})
	}
	monitored := func() string {
		t.Helper()
		for _, r := range h.rows(t) {
			if r.Key == "attic" {
				return r.Monitored.String()
			}
		}
		t.Fatal("the Library lists no attic")
		return ""
	}

	stock(winter(), works.Entry{Key: "b9", Label: "Spares", Aside: true, Children: []works.Entry{button}})
	tree, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "attic", works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := monitored(); got != "all" {
		t.Fatalf("a Work added monitoring all with asides off reads %s, want all: its unmonitored aside Button isn't asked for", got)
	}

	stock(winter(button))
	if _, err := h.works.Refresh(t.Context(), admin, tree.ID); err != nil {
		t.Fatal(err)
	}
	if got := monitored(); got != "some" {
		t.Fatalf("once Button moves out of the aside box, still unmonitored, the Work reads %s, want some", got)
	}

	stock(winter())
	if _, err := h.works.Refresh(t.Context(), admin, tree.ID); err != nil {
		t.Fatal(err)
	}
	if got := monitored(); got != "all" {
		t.Fatalf("once the source stops listing Button, the Work reads %s, want all: a withdrawn Item counts nowhere", got)
	}

	h.fx.Put("cellar", contenttypetest.Record{Tree: works.Fetched{Label: "The Cellar", Children: []works.Entry{winter()}}})
	if _, _, err := h.works.Add(t.Context(), admin, contenttypetest.ID, "cellar", works.Adding{}); err != nil {
		t.Fatal(err)
	}
	for _, r := range h.rows(t) {
		if r.Key == "cellar" && r.Monitored != works.MonitoredNone {
			t.Fatalf("a Work monitoring nothing reads %s, want none", r.Monitored)
		}
	}
}
