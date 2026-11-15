package library_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/works"
)

// unmonitored adds each key, monitoring nothing and searching nothing.
func (s shelf) unmonitored(t *testing.T, l *library.Library, keys ...works.Key) []works.Tree {
	t.Helper()
	var out []works.Tree
	for _, key := range keys {
		s.stock(key)
		added, err := l.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: key, Preset: new(works.PresetNone)})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, added.Tree)
	}
	return out
}

func TestMonitoringOneWorkQueuesOneCallWithOneAsk(t *testing.T) {
	t.Parallel()
	s := newShelf(t)
	l := s.library(nil)
	w := s.unmonitored(t, l, "w1")[0]
	if err := l.SetMonitored(t.Context(), identity.Admin{}, w.ID, works.Within{Items: idsOf(w.Items[1])}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SetMonitoring(t.Context(), identity.Admin{}, []works.WorkID{w.ID}, works.Choice{Preset: works.PresetAll}, false); err != nil {
		t.Fatal(err)
	}
	if err := l.SetMonitored(t.Context(), identity.Admin{}, w.ID, works.Within{Items: idsOf(w.Items[0])}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := l.SetMonitoring(t.Context(), identity.Admin{}, []works.WorkID{w.ID}, works.Choice{Preset: works.PresetFirst}, true); err != nil {
		t.Fatal(err)
	}
	want := []call{
		{trigger: searches.Monitor, asks: []searches.Ask{{Work: w.ID, Items: idsOf(w.Items[1])}}},
		{trigger: searches.Monitor, asks: []searches.Ask{{Work: w.ID, Items: idsOf(w.Items[0], w.Items[2])}}},
	}
	if got := s.queue.took(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("turning Clip 2 on, then All, then Clip 1 off, then previewing First queued %v, want %v", got, want)
	}
}

func TestAPresetOverManyWorksQueuesOneCallWithAnAskEach(t *testing.T) {
	t.Parallel()
	s := newShelf(t)
	l := s.library(nil)
	ws := s.unmonitored(t, l, "w1", "w2", "w3")
	for _, c := range []struct {
		preset works.Preset
		of     []works.Tree
	}{
		{works.PresetAll, ws[:2]},
		{works.PresetFirst, ws},
	} {
		var ids []works.WorkID
		for _, w := range c.of {
			ids = append(ids, w.ID)
		}
		if _, err := l.SetMonitoring(t.Context(), identity.Admin{}, ids, works.Choice{Preset: c.preset}, false); err != nil {
			t.Fatal(err)
		}
	}
	want := []call{
		{trigger: searches.Monitor, asks: []searches.Ask{{Work: ws[0].ID, Items: idsOf(ws[0].Items...)}, {Work: ws[1].ID, Items: idsOf(ws[1].Items...)}}},
		{trigger: searches.Monitor, asks: []searches.Ask{{Work: ws[2].ID, Items: idsOf(ws[2].Items...)}}},
	}
	if got := s.queue.took(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("All over two Works, then First over all three, queued %v, want %v", got, want)
	}
}

func TestMonitoringWhoseQueueFailsMonitorsNothing(t *testing.T) {
	t.Parallel()
	s := newShelf(t)
	l := s.library(nil)
	ws := s.unmonitored(t, l, "w1", "w2")
	s.queue.refuse = errors.New("the queue is closed")
	if _, err := l.SetMonitoring(t.Context(), identity.Admin{}, []works.WorkID{ws[0].ID, ws[1].ID}, works.Choice{Preset: works.PresetAll}, false); !errors.Is(err, s.queue.refuse) {
		t.Fatalf("a preset whose Queue fails answers %v, want its error", err)
	}
	if err := l.SetMonitored(t.Context(), identity.Admin{}, ws[0].ID, works.Within{Items: idsOf(ws[0].Items[0])}, true); !errors.Is(err, s.queue.refuse) {
		t.Fatalf("turning an Item on whose Queue fails answers %v, want its error", err)
	}
	if got := s.queue.took(t); len(got) != 0 {
		t.Fatalf("monitoring whose Queue failed left %v queued, want nothing", got)
	}
	for _, w := range ws {
		tree, err := s.works.Get(t.Context(), identity.Admin{}, w.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range tree.Items {
			if it.Monitored {
				t.Fatalf("after monitoring whose Queue failed %s monitors %s, want nothing", w.Label, it.Label)
			}
		}
	}
}
