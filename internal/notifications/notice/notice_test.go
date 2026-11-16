package notice_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/notice/noticetest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type deletedFacts struct {
	Work  notice.Work  `json:"work"`
	By    notice.Actor `json:"by"`
	Count int          `json:"count"`
	First []string     `json:"first"`
}

var deletedRow = notice.Row[deletedFacts]{
	Spec: notice.Spec{Kind: "works_deleted", Label: "Works deleted", Group: notice.Library},
	Render: func(f deletedFacts, c notice.Context) notice.Message {
		return notice.Message{
			Title:  "Deleted " + notice.Count(f.Count, f.Work.Nouns.Work),
			Lines:  append(notice.Titles(f.First, f.Count, "deleted"), f.By.Subject()+" deleted them at "+c.Clock(c.Now)+"."),
			Link:   notice.WorkLink(f.Work.ID),
			Poster: f.Work.Poster,
		}
	},
}

var otherRow = notice.Row[struct{}]{Spec: notice.Spec{Kind: "other"}}

var heat = deletedFacts{
	Work: notice.Work{
		ID: works.WorkID{UUID: storage.NewUUID()}, Label: "Heat (1995)", Type: "movie",
		Nouns:  notice.Nouns{Work: works.Noun{One: "movie", Many: "movies"}, Item: works.Noun{One: "movie", Many: "movies"}, Single: true},
		Poster: artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/heat.jpg"},
	},
	By:    notice.Actor{Name: "Sam"},
	Count: 20000,
	First: []string{"Alien (1979)", "Heat (1995)", "Ran (1985)"},
}

func TestARecordedEventRendersAsItsFactsDo(t *testing.T) {
	var stored []byte
	rec := notice.Sink("library", []notice.Entry{deletedRow}, func(_ context.Context, _ *storage.Tx[storage.Library], k notice.Kind, subject []byte) error {
		if k != deletedRow.Kind {
			t.Fatalf("kind = %s, want %s", k, deletedRow.Kind)
		}
		stored = subject
		return nil
	})
	if err := deletedRow.Record(t.Context(), nil, rec, heat); err != nil {
		t.Fatal(err)
	}
	c := notice.Context{Now: time.Date(2026, 11, 16, 14, 2, 0, 0, time.UTC), Zone: time.UTC}
	got, err := notice.Entry(deletedRow).RenderStored(stored, c)
	if err != nil {
		t.Fatal(err)
	}
	want := notice.Message{
		Title:  "Deleted 20000 movies",
		Lines:  []string{"Alien (1979)", "Heat (1995)", "Ran (1985)", "19997 more were deleted.", "Sam deleted them at 14:02."},
		Link:   notice.Link{Path: "/works/" + heat.Work.ID.String()},
		Poster: heat.Work.Poster,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rendered %+v, want %+v", got, want)
	}
	if _, err := notice.Entry(deletedRow).RenderStored([]byte(`{"count":"many"}`), c); err == nil {
		t.Fatal("a subject of the wrong shape rendered")
	}
}

func TestRecordPanicsOnAWiringBug(t *testing.T) {
	refused := func(name string, rec notice.Recorder) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: Record didn't panic", name)
			}
		}()
		_ = deletedRow.Record(t.Context(), nil, rec, heat)
	}
	refused("a zero Recorder", notice.Recorder{})
	refused("a kind its owner didn't register", notice.Sink("grabs", []notice.Entry{otherRow}, func(context.Context, *storage.Tx[storage.Library], notice.Kind, []byte) error {
		t.Fatal("put called")
		return nil
	}))
	rec, _ := noticetest.New("grabs", []notice.Entry{otherRow})
	refused("noticetest with a kind its owner didn't register", rec)
}

func TestNoticetestKeepsWhatAnOwnerRecorded(t *testing.T) {
	rec, kept := noticetest.New("library", []notice.Entry{deletedRow, otherRow})
	if err := deletedRow.Record(t.Context(), nil, rec, heat); err != nil {
		t.Fatal(err)
	}
	if err := otherRow.Record(t.Context(), nil, rec, struct{}{}); err != nil {
		t.Fatal(err)
	}
	if got := noticetest.Of(kept, deletedRow); !reflect.DeepEqual(got, []deletedFacts{heat}) {
		t.Fatalf("kept %+v, want only %+v", got, heat)
	}
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
	} {
		if got := notice.Titles(c.first, c.total, "added"); !slices.Equal(got, c.want) {
			t.Errorf("Titles(%v, %d) = %q, want %q", c.first, c.total, got, c.want)
		}
	}
	if !slices.Equal(first, []string{"Alien", "Heat", "Ran", "Up"}) {
		t.Fatalf("Titles changed its argument: %v", first)
	}
	episode := works.Noun{One: "episode", Many: "episodes"}
	if got := notice.Count(1, episode) + ", " + notice.Count(10, episode); got != "1 episode, 10 episodes" {
		t.Fatalf("Count = %q", got)
	}
}
