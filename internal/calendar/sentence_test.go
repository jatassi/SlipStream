package calendar

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func film(state files.State) *work {
	it := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, Label: "Heatwave", Monitored: true}
	return newWork(works.Row{Bare: works.Bare{Summary: works.Summary{ID: works.WorkID{UUID: storage.NewUUID()}, ContentType: "films", Label: "Heatwave"}}}, 0, single,
		files.WorkStates{Items: []files.ItemState{{Item: it, Status: state}}})
}

func dates(f *work, kinds map[works.DatedKey]Day) []Dated {
	var out []Dated
	for k, d := range kinds {
		for id := range f.items {
			out = append(out, Dated{Work: f.row.ID, Item: id, Kind: k, Day: d})
		}
	}
	return out
}

func TestTheSentenceOtherwiseNamesTheWeekOrNothing(t *testing.T) {
	today := NewDay(2026, time.November, 17)
	s := newShelf(0, "Attic", states(4, files.Unreleased))
	f := film(files.Unreleased)
	all := of(s.w, f)
	soon := kept(joined(s.dated(0, today.Add(6), 1, 2, 3, 4), dates(f, map[works.DatedKey]Day{"out": today.Add(1), "home": today.Add(5)})), all, false)
	got := sentenceOf(today, nil, soon)
	if want := (Sentence{Words: "4 things and 1 film come out this week."}); got == nil || *got != want {
		t.Fatalf("sentence = %+v, want %+v", got, want)
	}
	if got := sentenceOf(today, kept(dates(f, map[works.DatedKey]Day{"out": today.Add(-1)}), all, false), nil); got != nil {
		t.Fatalf("sentence = %+v, want none while nothing needs attention or comes out this week", got)
	}
	if got := sentenceOf(today, nil, kept(dates(f, map[works.DatedKey]Day{"out": today}), all, false)); got == nil || got.Words != "1 film comes out this week." {
		t.Fatalf("sentence = %+v, want 1 film comes out this week.", got)
	}
}

func TestTheSentenceCountsOnlyMonitoredItemsWhenUnmonitoredOnesShow(t *testing.T) {
	today := NewDay(2026, time.November, 17)
	s := newShelf(0, "Attic", states(4, files.Missing))
	for _, id := range s.boxes[0][1:] {
		p := s.w.items[id]
		p.Item.Monitored = false
		s.w.items[id] = p
	}
	all := of(s.w)
	got := sentenceOf(today, kept(s.dated(0, today, 1, 2, 3, 4), all, true), kept(s.dated(0, today.Add(1), 2, 3), all, true))
	if want := (Sentence{NeedsYou: true, Words: "1 thing is out and still missing."}); got == nil || *got != want {
		t.Fatalf("sentence = %+v, want %+v: Wanted, where it links, lists only the monitored Thing 1", got, want)
	}
	if got := sentenceOf(today, nil, kept(s.dated(0, today.Add(1), 2, 3), all, true)); got != nil {
		t.Fatalf("sentence = %+v, want none while only unmonitored Things come out this week", got)
	}
}

func joined(ds ...[]Dated) []Dated {
	var out []Dated
	for _, d := range ds {
		out = append(out, d...)
	}
	return out
}
