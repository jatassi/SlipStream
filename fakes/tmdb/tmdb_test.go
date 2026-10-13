package tmdb_test

import (
	"io"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tmdb"
)

type exchange struct {
	path, want string
}

func get(t *testing.T, f faketest.Served, path string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL(path), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.Status + " " + string(body)
}

func inBothTiers(t *testing.T, s scenario.Scenario, exchanges []exchange) {
	check := func(t *testing.T, f faketest.Served) {
		for _, e := range exchanges {
			if got := get(t, f, e.path); got != e.want {
				t.Errorf("GET %s = %q, want %q", e.path, got, e.want)
			}
		}
	}
	t.Run("loopback", func(t *testing.T) { check(t, faketest.Loopback(t, tmdb.Fake, s)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { check(t, faketest.InMemory(t, tmdb.Fake, s)) })
	})
}

const (
	notFound = `404 Not Found {"success":false,"status_code":34,"status_message":"The resource you requested could not be found."}` + "\n"
	orchard  = `{"adult":false,"id":9000001,"original_title":"The Quiet Orchard","release_date":"2021-04-16","title":"The Quiet Orchard"}`
	none     = `200 OK {"page":1,"results":[],"total_pages":0,"total_results":0}` + "\n"
)

func TestTheFakeAnswersAsTMDBDoes(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: 9000002, IMDbID: "tt9000002", Title: "Orchard Road", Released: "2019-01-05"})
	road := `{"adult":false,"id":9000002,"original_title":"Orchard Road","release_date":"2019-01-05","title":"Orchard Road"}`
	inBothTiers(t, s, []exchange{
		{"/3/search/movie?query=QUIET+orchard", `200 OK {"page":1,"results":[` + orchard + `],"total_pages":1,"total_results":1}` + "\n"},
		{"/3/search/movie?query=orchard", `200 OK {"page":1,"results":[` + orchard + `,` + road + `],"total_pages":1,"total_results":2}` + "\n"},
		{"/3/search/movie?query=orchard&year=2019", `200 OK {"page":1,"results":[` + road + `],"total_pages":1,"total_results":1}` + "\n"},
		{"/3/search/movie?query=orchard&year=2020", none},
		{"/3/search/movie?query=", none},
		{"/3/movie/9000001", `200 OK {"adult":false,"id":9000001,"imdb_id":"tt9000001","original_title":"The Quiet Orchard","release_date":"2021-04-16","runtime":104,"status":"Released","title":"The Quiet Orchard"}` + "\n"},
		{"/3/movie/9000001?append_to_response=external_ids,release_dates", `200 OK {"adult":false,"id":9000001,"imdb_id":"tt9000001","original_title":"The Quiet Orchard","release_date":"2021-04-16","runtime":104,"status":"Released","title":"The Quiet Orchard",` +
			`"external_ids":{"imdb_id":"tt9000001"},"release_dates":{"results":[{"iso_3166_1":"US","release_dates":[{"certification":"","iso_639_1":"","note":"","release_date":"2021-04-16T00:00:00.000Z","type":3}]}]}}` + "\n"},
		{"/3/movie/9000003", notFound},
		{"/3/movie/orchard", notFound},
		{"/3/tv/1", notFound},
	})
}

func TestAScenarioWithoutTMDBServesAnEmptyCatalog(t *testing.T) {
	s, err := scenario.Load("first-run")
	if err != nil {
		t.Fatal(err)
	}
	inBothTiers(t, s, []exchange{
		{"/3/search/movie?query=orchard", none},
		{"/3/movie/9000001", notFound},
	})
}
