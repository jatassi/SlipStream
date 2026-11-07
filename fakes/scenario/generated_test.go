package scenario

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestAGeneratedMovieIsAPureFunctionOfSeedAndIndex(t *testing.T) {
	g := TMDBGenerated{Count: 20000, Seed: "scale"}
	for _, i := range []int{0, 1, 19, 20, 9999, 19999} {
		if a, b := g.Movie(i), (TMDBGenerated{Count: 3, Seed: "scale"}).Movie(i); !reflect.DeepEqual(a, b) {
			t.Fatalf("movie %d differs by count:\n%+v\n%+v", i, a, b)
		}
	}
	if a, b := g.Movie(5), (TMDBGenerated{Count: 20000, Seed: "other"}).Movie(5); a.ID != b.ID || reflect.DeepEqual(a, b) {
		t.Fatalf("seed other gives movie 5 %+v, want the same id with other fields than %+v", b, a)
	}
	if m := g.Movie(1); m.ID != 50000002 || m.IMDbID != "tt50000002" {
		t.Fatalf("movie 1 has ids %d and %s, want 50000002 and tt50000002", m.ID, m.IMDbID)
	}
}

func TestGeneratedMoviesSpreadAsALibraryDoes(t *testing.T) {
	g := TMDBGenerated{Count: 20000, Seed: "scale"}
	var accented, alternative, future int
	years := map[int]bool{}
	titles := map[string]bool{}
	now := time.Now().Format(time.DateOnly)
	for i := range g.Count {
		m := g.Movie(i)
		titles[m.Title] = true
		if strings.ContainsFunc(m.Title, func(r rune) bool { return r >= utf8.RuneSelf }) {
			accented++
		}
		if len(m.Titles) > 0 {
			alternative++
		}
		if len(m.Dates) != 1 || m.Dates[0].Date >= m.Released || len(m.Genres) != 2 || m.Genres[0] == m.Genres[1] || m.Runtime < 80*time.Minute {
			t.Fatalf("movie %d = %+v, want one cinema date before its digital date, two genres and a runtime", i, m)
		}
		cinema, _ := time.Parse(time.DateOnly, m.Dates[0].Date)
		years[cinema.Year()] = true
		if m.Dates[0].Date > now {
			future++
		}
	}
	if accented != 1000 || alternative != 2000 {
		t.Fatalf("%d accented titles and %d alternative titles in 20,000, want 1,000 and 2,000", accented, alternative)
	}
	if len(years) != 71 || future == 0 {
		t.Fatalf("cinema dates span %d years with %d in the future, want 1960 to 2030 with some ahead", len(years), future)
	}
	if len(titles) < 5000 {
		t.Fatalf("%d distinct titles in 20,000, want at least 5,000", len(titles))
	}
}

func TestGeneratedIDsAreDisjointFromListedOnes(t *testing.T) {
	g := TMDBGenerated{Count: 3, Seed: "s"}
	c := TMDB{Movies: []TMDBMovie{{ID: 9000001, IMDbID: "tt9000001", Title: "The Quiet Orchard"}}, Generated: &g}
	var ids []int64
	for m := range c.All() {
		ids = append(ids, m.ID)
	}
	if want := []int64{9000001, 50000001, 50000002, 50000003}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("All = %v, want %v", ids, want)
	}
	for _, id := range ids {
		if m, ok := c.Movie(id); !ok || m.ID != id {
			t.Fatalf("Movie(%d) = %+v, %t", id, m, ok)
		}
	}
	for _, id := range []int64{50000000, 50000004} {
		if _, ok := c.Movie(id); ok {
			t.Fatalf("Movie(%d) found a movie outside the catalog", id)
		}
	}

	s := Scenario{Name: "x", Nodes: []Node{{Name: "home"}}, TMDB: &TMDB{
		Movies: []TMDBMovie{
			{ID: 50000002, IMDbID: "tt9000002", Title: "Taken id"},
			{ID: 9000003, IMDbID: "tt50049999", Title: "Taken IMDb id"},
		},
		Generated: &TMDBGenerated{Count: 50001},
	}}
	want := strings.Join([]string{
		`scenario x: tmdb generated has count 50001 and seed ""; want a count from 1 to 50000 and a seed`,
		"scenario x: tmdb movie 50000002 uses an id from 50000001 to 50050000, which generated movies hold",
		"scenario x: tmdb movie 9000003 uses an id from 50000001 to 50050000, which generated movies hold",
	}, "\n")
	if err := s.Validate(); err == nil || err.Error() != want {
		t.Fatalf("Validate =\n%v\nwant\n%s", err, want)
	}
}
