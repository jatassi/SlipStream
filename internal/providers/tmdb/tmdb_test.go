package tmdb_test

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketmdb "github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

func inEachTier(t *testing.T, test func(t *testing.T, c *tmdb.Client), opts ...faketest.Option) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	client := func(f faketest.Served) *tmdb.Client {
		return tmdb.New(f.Client, tmdb.Endpoints{API: endpoint.MustParseURL(f.Address.URL("/3"))}, "SlipStream/test")
	}
	t.Run("loopback", func(t *testing.T) { test(t, client(faketest.Loopback(t, faketmdb.Fake, s, opts...))) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, client(faketest.InMemory(t, faketmdb.Fake, s, opts...))) })
	})
}

func TestTheClientReadsTheFakesMovie(t *testing.T) {
	inEachTier(t, func(t *testing.T, c *tmdb.Client) {
		found, err := c.SearchMovies(t.Context(), "quiet orchard")
		if want := []tmdb.MovieResult{{ID: 9000001, Title: "The Quiet Orchard", ReleaseDate: "2021-04-16"}}; err != nil || !reflect.DeepEqual(found, want) {
			t.Fatalf("SearchMovies = %+v, %v; want %+v", found, err, want)
		}
		m, err := c.Movie(t.Context(), "9000001")
		want := tmdb.Movie{
			ID: 9000001, Title: "The Quiet Orchard", ReleaseDate: "2021-04-16", Runtime: 104,
			ExternalIDs:  tmdb.ExternalIDs{IMDbID: "tt9000001"},
			ReleaseDates: tmdb.ReleaseDates{Results: []tmdb.CountryReleases{{Country: "US", Releases: []tmdb.Release{{Date: "2021-04-16T00:00:00.000Z", Type: tmdb.Theatrical}}}}},
		}
		if err != nil || !reflect.DeepEqual(m, want) {
			t.Fatalf("Movie = %+v, %v; want %+v", m, err, want)
		}
		if got := m.Released(); got != "2021-04-16" {
			t.Fatalf("Released = %q, want 2021-04-16", got)
		}
	})
}

func TestAnUnknownMovieIsNotFound(t *testing.T) {
	inEachTier(t, func(t *testing.T, c *tmdb.Client) {
		_, err := c.Movie(t.Context(), "9000002")
		if want := "GET /movie/9000002: 404 Not Found: the metadata source has no record with that key"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
			t.Fatalf("Movie(9000002) = %v, want %s", err, want)
		}
	})
}

func TestReleasedIsTheEarliestTheatricalOrDigitalDate(t *testing.T) {
	release := func(country, date string, kind int) tmdb.CountryReleases {
		return tmdb.CountryReleases{Country: country, Releases: []tmdb.Release{{Date: date, Type: kind}}}
	}
	for _, c := range []struct {
		results []tmdb.CountryReleases
		want    string
	}{
		{nil, ""},
		{[]tmdb.CountryReleases{release("US", "2021-04-16T00:00:00.000Z", tmdb.Theatrical), release("GB", "2021-03-01T00:00:00.000Z", tmdb.Digital)}, "2021-03-01"},
		{[]tmdb.CountryReleases{release("US", "2021-04-16T00:00:00.000Z", tmdb.Theatrical), release("FR", "2020-09-12T00:00:00.000Z", 1)}, "2021-04-16"},
		{[]tmdb.CountryReleases{release("US", "2022-01-01T00:00:00.000Z", 5), release("DE", "2021-11-20T00:00:00.000Z", tmdb.TheatricalLimited)}, "2021-11-20"},
		{[]tmdb.CountryReleases{release("US", "2022-01-01T00:00:00.000Z", 6)}, ""},
	} {
		m := tmdb.Movie{ReleaseDates: tmdb.ReleaseDates{Results: c.results}}
		if got := m.Released(); got != c.want {
			t.Errorf("Released of %+v = %q, want %q", c.results, got, c.want)
		}
	}
}

func TestAFaultedAnswerIsUnavailable(t *testing.T) {
	inEachTier(t, func(t *testing.T, c *tmdb.Client) {
		var got []string
		for range 3 {
			_, err := c.Movie(t.Context(), "9000001")
			got = append(got, fmt.Sprintf("%v %t", err, errors.Is(err, works.ErrUnavailable)))
		}
		want := []string{
			"GET /movie/9000001: 503 Service Unavailable: the metadata source didn't answer true",
			"GET /movie/9000001: decode: invalid character '\\x00' looking for beginning of value: the metadata source didn't answer true",
			"<nil> false",
		}
		if !slices.Equal(got, want) {
			t.Fatalf("three lookups gave\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	},
		faketest.WithFaults(
			fault.Rule{ID: "down", Target: "tmdb", Route: "movie", Fault: fault.Status{Code: 503}, Times: 1},
			fault.Rule{ID: "garbled", Target: "tmdb", Route: "movie", Params: map[string]string{"id": "9000001"}, Fault: fault.Malformed{}, Times: 1},
		))
}
