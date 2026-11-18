package subtitles

import (
	goflag "flag"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var update = goflag.Bool("update", false, "rewrite the rendered messages under testdata/history")

func TestHistoryRowsRender(t *testing.T) {
	id, err := storage.ParseUUID("019a8f5e-0000-7000-8000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	book := entry.Nouns{Work: works.Noun{One: "Book", Many: "Books"}, Item: works.Noun{One: "Chapter", Many: "Chapters"}}
	dune := entry.Work{ID: works.WorkID{UUID: id}, Label: "Dune (1965)", Type: "book", Nouns: book, Poster: artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/cover.jpg"}}
	fetched := SubtitleFact{Language: "English", HI: true, Source: FromProvider, Provider: "SubDL"}
	file := "Dune (1965)/Dune (1965).mkv"
	golden := func(e entry.Entry, facts any, name string) {
		entrytest.Golden(t, e, facts, "testdata/history", name, *update)
	}
	golden(addedRow, SidecarFacts{Work: dune, File: file, Subtitles: []SubtitleFact{fetched, {Language: "French", Forced: true, Source: FromUpload}}}, "added")
	golden(replacedRow, SidecarFacts{Work: dune, File: file, Subtitles: []SubtitleFact{{Language: "English", Source: Found}}}, "replaced")
	golden(removedRow, SidecarFacts{Work: dune, File: file, Subtitles: []SubtitleFact{fetched}, Why: "retime_failed"}, "removed-retime-failed")
	golden(removedRow, SidecarFacts{Work: dune, File: file, Subtitles: []SubtitleFact{{Language: "German", Source: FromProvider}}, Why: "retime_failed"}, "removed-provider-unknown")
	golden(markedBadRow, MarkedBadFacts{Work: dune, File: file, Subtitle: fetched, By: entry.Actor{Name: "Sam"}}, "marked-bad")
	golden(missingRow, SearchFacts{Work: dune, File: file, Language: "English"}, "missing")
	golden(timingFailedRow, SearchFacts{Work: dune, File: file, Language: "English", Forced: true, Found: 3}, "timing-failed")
	golden(timingFailedRow, SearchFacts{Work: dune, File: file, Language: "French", Found: 1}, "timing-failed-one")
	golden(providerOutRow, ProviderOutFacts{Provider: "SubDL", Until: time.Date(2026, 11, 19, 19, 0, 0, 0, time.UTC)}, "provider-out")
}
