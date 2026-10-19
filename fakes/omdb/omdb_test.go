package omdb_test

import (
	"io"
	"math/rand/v2"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/omdb"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	provider "github.com/jatassi/slipstream/internal/providers/omdb"
)

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

type exchange struct {
	path, want string
}

func check(t *testing.T, f faketest.Served, exchanges []exchange) {
	t.Helper()
	for _, e := range exchanges {
		if got := get(t, f, e.path); got != e.want {
			t.Errorf("GET %s =\n%q\nwant\n%q", e.path, got, e.want)
		}
	}
}

func inBothTiers(t *testing.T, s scenario.Scenario, test func(t *testing.T, f faketest.Served)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, omdb.Fake, s)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, omdb.Fake, s)) })
	})
}

func ptr[T any](v T) *T { return &v }

// rated is single-machine with The Quiet Orchard rated everywhere and an
// unrated title OMDb knows but TMDB doesn't.
func rated(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.OMDb = &scenario.OMDb{Titles: []scenario.OMDbTitle{
		{IMDbID: "tt9000001", IMDbRating: ptr(7.4), IMDbVotes: ptr(int64(1234567)), RottenTomatoes: ptr(91), Metacritic: ptr(78), Awards: "Won 1 Oscar. 12 wins & 30 nominations total"},
		{IMDbID: "tt9000009"},
	}}
	return s
}

const (
	badKey  = `401 Unauthorized {"Response":"False","Error":"Invalid API key!"}`
	limited = `401 Unauthorized {"Response":"False","Error":"Request limit reached!"}`
	orchard = `200 OK {"Title":"The Quiet Orchard","Year":"2021","Awards":"Won 1 Oscar. 12 wins & 30 nominations total","Ratings":[{"Source":"Internet Movie Database","Value":"7.4/10"},{"Source":"Rotten Tomatoes","Value":"91%"},{"Source":"Metacritic","Value":"78/100"}],"Metascore":"78","imdbRating":"7.4","imdbVotes":"1,234,567","imdbID":"tt9000001","Type":"movie","Response":"True"}`
)

func TestTheFakeAnswersAsOMDbDoes(t *testing.T) {
	inBothTiers(t, rated(t), func(t *testing.T, f faketest.Served) {
		check(t, f, []exchange{
			{"/?i=tt9000001&apikey=yours", orchard},
			{"/?i=tt9000001", orchard},
			{"/?i=tt9000009", `200 OK {"Title":"N/A","Year":"N/A","Awards":"N/A","Ratings":[],"Metascore":"N/A","imdbRating":"N/A","imdbVotes":"N/A","imdbID":"tt9000009","Type":"movie","Response":"True"}`},
			{"/?i=tt9000002", `200 OK {"Response":"False","Error":"Movie not found!"}`},
			{"/?i=orchard", `200 OK {"Response":"False","Error":"Incorrect IMDb ID."}`},
			{"/", `200 OK {"Response":"False","Error":"Incorrect IMDb ID."}`},
			{"/v1/title", "404 Not Found 404 page not found\n"},
		})
	})
}

func TestRevokeKeyAnswersOMDbsBadKey(t *testing.T) {
	inBothTiers(t, rated(t), func(t *testing.T, f faketest.Served) {
		get(t, f, "/?i=tt9000001&apikey=bundled")
		f.Cue(t, omdb.RevokeKey.Name, omdb.Key{})
		check(t, f, []exchange{{"/?i=tt9000001&apikey=bundled", badKey}, {"/?i=tt9000002&apikey=bundled", badKey}})
		check(t, f, []exchange{{"/?i=tt9000001&apikey=yours", orchard}})
		f.Cue(t, omdb.RevokeKey.Name, omdb.Key{Key: ptr("yours")})
		check(t, f, []exchange{{"/?i=tt9000001&apikey=yours", badKey}})
	})
}

func TestCapAnswersRequestLimitReachedUntilUTCMidnight(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := rated(t)
		s.OMDb.DailyLimit = 2
		f := faketest.InMemory(t, omdb.Fake, s)
		check(t, f, []exchange{{"/?i=tt9000001&apikey=a", orchard}, {"/?i=tt9000002&apikey=a", `200 OK {"Response":"False","Error":"Movie not found!"}`}})
		check(t, f, []exchange{{"/?i=tt9000001&apikey=a", limited}, {"/?i=tt9000001&apikey=b", orchard}})

		f.Cue(t, omdb.Cap.Name, omdb.Key{Key: ptr("b")})
		check(t, f, []exchange{{"/?i=tt9000001&apikey=b", limited}, {"/?i=tt9000001&apikey=c", orchard}})
		f.Cue(t, omdb.Cap.Name, omdb.Key{})
		check(t, f, []exchange{{"/?i=tt9000001&apikey=c", limited}, {"/?i=tt9000001", limited}})

		time.Sleep(time.Until(time.Now().UTC().Truncate(24 * time.Hour).Add(24*time.Hour - time.Second)))
		check(t, f, []exchange{{"/?i=tt9000001&apikey=b", limited}})
		time.Sleep(time.Second)
		check(t, f, []exchange{{"/?i=tt9000001&apikey=a", orchard}, {"/?i=tt9000001&apikey=b", orchard}, {"/?i=tt9000001", orchard}})
	})
}

func TestTheFakeConforms(t *testing.T) {
	inBothTiers(t, rated(t), func(t *testing.T, f faketest.Served) {
		conformance.Run(t, conformance.Target{
			Endpoints: provider.Endpoints{API: endpoint.MustParseURL(f.Address.URL(""))},
			Client:    f.Client,
			Known:     "tt9000001",
			Lines:     lines,
			Timings:   provider.DefaultTimings(),
		})
	})
}

func lines(c *http.Client) *metadata.Lines {
	return metadata.NewLines(c, "SlipStream/conformance ( github.com/jatassi/SlipStream )", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}
