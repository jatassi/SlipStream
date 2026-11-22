package entry_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type deletedFacts struct {
	Work  entry.Work  `json:"work"`
	By    entry.Actor `json:"by"`
	Count int         `json:"count"`
	First []string    `json:"first"`
}

func (f deletedFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

type noFacts struct{}

func (noFacts) About() entry.About { return entry.About{Quiet: true} }

var deletedRow = entry.Row[deletedFacts]{
	Spec: entry.Spec{Kind: "works_deleted", Label: "Works deleted", Group: entry.Library},
	Render: func(f deletedFacts, c entry.Context) entry.Message {
		return entry.Message{
			Title:  "Deleted " + entry.Count(f.Count, f.Work.Nouns.Work),
			Lines:  append(entry.Titles(f.First, f.Count, "deleted"), f.By.Subject()+" deleted them at "+c.Clock(c.Now)+"."),
			Link:   entry.WorkLink(f.Work.ID),
			Poster: f.Work.Poster,
		}
	},
}

var otherRow = entry.Row[noFacts]{Spec: entry.Spec{Kind: "other"}}

var heat = deletedFacts{
	Work: entry.Work{
		ID: works.WorkID{UUID: storage.NewUUID()}, Label: "Heat (1995)", Type: "book",
		Nouns:  entry.Nouns{Work: works.Noun{One: "book", Many: "books"}, Item: works.Noun{One: "book", Many: "books"}, Single: true},
		Poster: artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/heat.jpg"},
	},
	By:    entry.Actor{Name: "Sam"},
	Count: 20000,
	First: []string{"Alien (1979)", "Heat (1995)", "Ran (1985)"},
}

func TestARecordedEventRendersAsItsFactsDo(t *testing.T) {
	var stored []byte
	var about entry.About
	rec := entry.Sink("library", []entry.Entry{deletedRow}, func(_ context.Context, _ *storage.Tx[storage.Library], k entry.Kind, subject []byte, a entry.About) error {
		if k != deletedRow.Kind {
			t.Fatalf("kind = %s, want %s", k, deletedRow.Kind)
		}
		stored, about = subject, a
		return nil
	})
	if err := deletedRow.Record(t.Context(), nil, rec, heat); err != nil {
		t.Fatal(err)
	}
	if want := (entry.About{Type: "book", Works: []works.WorkID{heat.Work.ID}}); !reflect.DeepEqual(about, want) {
		t.Fatalf("put was told the entry is about %+v, want %+v", about, want)
	}
	c := entry.Context{Now: time.Date(2026, 11, 16, 14, 2, 0, 0, time.UTC), Zone: time.UTC}
	got, err := entry.Entry(deletedRow).RenderStored(stored, c)
	if err != nil {
		t.Fatal(err)
	}
	want := entry.Message{
		Title:  "Deleted 20,000 books",
		Lines:  []string{"Alien (1979)", "Heat (1995)", "Ran (1985)", "19,997 more were deleted.", "Sam deleted them at 14:02."},
		Link:   entry.Link{Path: "/works/" + heat.Work.ID.String()},
		Poster: heat.Work.Poster,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rendered %+v, want %+v", got, want)
	}
	if _, err := entry.Entry(deletedRow).RenderStored([]byte(`{"count":"many"}`), c); err == nil {
		t.Fatal("a subject of the wrong shape rendered")
	}
}

func TestRecordPanicsOnAWiringBug(t *testing.T) {
	refused := func(name string, rec entry.Recorder) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: Record didn't panic", name)
			}
		}()
		_ = deletedRow.Record(t.Context(), nil, rec, heat)
	}
	refused("a zero Recorder", entry.Recorder{})
	refused("a kind its owner didn't register", entry.Sink("grabs", []entry.Entry{otherRow}, func(context.Context, *storage.Tx[storage.Library], entry.Kind, []byte, entry.About) error {
		t.Fatal("put called")
		return nil
	}))
	rec, _ := entrytest.New("grabs", []entry.Entry{otherRow})
	refused("entrytest with a kind its owner didn't register", rec)
}

func TestTitlesListsThreeThenCountsTheRest(t *testing.T) {
	first := []string{"Alien", "Heat", "Ran", "Up"}
	for _, c := range []struct {
		first []string
		total int
		want  []string
	}{
		{first[:1], 1, []string{"Alien"}},
		{first[:3], 3, []string{"Alien", "Heat", "Ran"}},
		{first, 4, []string{"Alien", "Heat", "Ran", "1 more was added."}},
		{first[:3], 12, []string{"Alien", "Heat", "Ran", "9 more were added."}},
		{first[:3], 20003, []string{"Alien", "Heat", "Ran", "20,000 more were added."}},
	} {
		if got := entry.Titles(c.first, c.total, "added"); !slices.Equal(got, c.want) {
			t.Errorf("Titles(%v, %d) = %q, want %q", c.first, c.total, got, c.want)
		}
	}
	if !slices.Equal(first, []string{"Alien", "Heat", "Ran", "Up"}) {
		t.Fatalf("Titles changed its argument: %v", first)
	}
	episode := works.Noun{One: "episode", Many: "episodes"}
	if got := entry.Count(1, episode) + ", " + entry.Count(10, episode) + ", " + entry.Count(1200, episode); got != "1 episode, 10 episodes, 1,200 episodes" {
		t.Fatalf("Count = %q", got)
	}
}
