package movie_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheMovieConformsToCalendarWithTheAvailabilityRule(t *testing.T) {
	s := catalog(t)
	s.TMDB.Movies = append(s.TMDB.Movies,
		scenario.TMDBMovie{ID: 9200001, IMDbID: "tt9200001", Title: "Lantern Fall", Dates: []scenario.TMDBDate{
			{Country: "US", Kind: "premiere", Date: "2026-11-02"},
			{Country: "US", Kind: "limited", Date: "2026-11-03"},
			{Country: "US", Kind: "tv", Date: "2026-11-05"},
			{Country: "US", Kind: "cinema", Date: "2026-11-10"},
			{Country: "US", Kind: "digital", Date: "2026-11-20"},
			{Country: "US", Kind: "physical", Date: "2026-11-28"},
		}},
		scenario.TMDBMovie{ID: 9200002, IMDbID: "tt9200002", Title: "Far Shore", Dates: []scenario.TMDBDate{
			{Country: "FR", Kind: "digital", Date: "2026-11-15"},
			{Country: "DE", Kind: "digital", Date: "2026-11-12"},
			{Country: "FR", Kind: "physical", Date: "2026-11-04"},
			{Country: "US", Kind: "physical", Date: "2026-11-25"},
		}},
		scenario.TMDBMovie{ID: 9200003, IMDbID: "tt9200003", Title: "Cold Open", Dates: []scenario.TMDBDate{
			{Country: "US", Kind: "digital", Date: "2026-12-20"},
			{Country: "GB", Kind: "digital", Date: "2026-11-08"},
		}},
		scenario.TMDBMovie{ID: 9200004, IMDbID: "tt9200004", Title: "Edges", Dates: []scenario.TMDBDate{
			{Country: "US", Kind: "digital", Date: "2026-10-31"},
			{Country: "US", Kind: "physical", Date: "2026-12-01"},
			{Country: "US", Kind: "premiere", Date: "2026-11-11"},
		}},
	)
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): serve(t, s).open()}))
	lantern, shore := sh.add(t, "9200001"), sh.add(t, "9200002")
	sh.add(t, "9200003")
	sh.add(t, "9200004")
	day := func(d int) calendar.Day { return calendar.NewDay(2026, time.November, d) }
	span, err := calendar.NewSpan(day(1), day(1).Add(30))
	if err != nil {
		t.Fatal(err)
	}
	at := func(tree works.Tree, kind works.DatedKey, d int) calendar.Dated {
		return calendar.Dated{Work: tree.ID, Item: tree.Items[0].ID, Kind: kind, Day: day(d)}
	}
	contenttypetest.Calendar(t, contenttypetest.CalendarCase{Library: sh.lib, Type: sh.m, Span: span, Index: "movie_release_day", Want: []calendar.Dated{
		at(lantern, "cinema", 3), at(lantern, "digital", 20), at(lantern, "physical", 28),
		at(shore, "digital", 12), at(shore, "physical", 25),
	}})
}
