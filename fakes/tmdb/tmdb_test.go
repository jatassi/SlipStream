package tmdb_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/tmdb"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	provider "github.com/jatassi/slipstream/internal/providers/tmdb"
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

func check(t *testing.T, f faketest.Served, exchanges []exchange) {
	t.Helper()
	for _, e := range exchanges {
		if got := get(t, f, e.path); got != e.want {
			t.Errorf("GET %s =\n%q\nwant\n%q", e.path, got, e.want)
		}
	}
}

func inBothTiers(t *testing.T, s scenario.Scenario, test func(t *testing.T, f faketest.Served)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, tmdb.Fake, s)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, tmdb.Fake, s)) })
	})
}

func load(t *testing.T, name string) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load(name)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

const (
	notFound     = `404 Not Found {"success":false,"status_code":34,"status_message":"The resource you requested could not be found."}` + "\n"
	unauthorized = `401 Unauthorized {"status_code":7,"status_message":"Invalid API key: You must be granted a valid key.","success":false}` + "\n"
	none         = `200 OK {"page":1,"results":[],"total_pages":0,"total_results":0}` + "\n"
	orchard      = `{"adult":false,"backdrop_path":"/znm9omh75o26.jpg","id":9000001,"original_language":"en","original_title":"The Quiet Orchard","overview":"","poster_path":"/znvorf0peat3.jpg","release_date":"2021-04-16","title":"The Quiet Orchard"}`
)

func TestTheFakeAnswersAsTMDBDoes(t *testing.T) {
	s := load(t, "single-machine")
	s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: 9000002, IMDbID: "tt9000002", Title: "Orchard Road", Released: "2019-01-05", Images: []scenario.TMDBImage{{Shape: "poster"}}})
	road := `{"adult":false,"backdrop_path":null,"id":9000002,"original_language":"en","original_title":"Orchard Road","overview":"","poster_path":"/3vnaflt6hssi0.jpg","release_date":"2019-01-05","title":"Orchard Road"}`
	inBothTiers(t, s, func(t *testing.T, f faketest.Served) {
		check(t, f, []exchange{
			{"/3/search/movie?query=QUIET+orchard", `200 OK {"page":1,"results":[` + orchard + `],"total_pages":1,"total_results":1}` + "\n"},
			{"/3/search/movie?query=orchard", `200 OK {"page":1,"results":[` + orchard + `,` + road + `],"total_pages":1,"total_results":2}` + "\n"},
			{"/3/search/movie?query=orchard&primary_release_year=2019", `200 OK {"page":1,"results":[` + road + `],"total_pages":1,"total_results":1}` + "\n"},
			{"/3/search/movie?query=orchard&page=2", `200 OK {"page":2,"results":[],"total_pages":1,"total_results":2}` + "\n"},
			{"/3/search/movie?query=orchard&primary_release_year=2020", none},
			{"/3/search/movie?query=", none},
			{"/3/movie/9000001", `200 OK {"adult":false,"backdrop_path":"/znm9omh75o26.jpg","belongs_to_collection":null,"genres":[],"id":9000001,"imdb_id":"tt9000001","original_language":"en","original_title":"The Quiet Orchard","overview":"","poster_path":"/znvorf0peat3.jpg","production_companies":[],"release_date":"2021-04-16","runtime":104,"status":"Released","tagline":"","title":"The Quiet Orchard","vote_average":0,"vote_count":0}` + "\n"},
			{"/3/movie/9000003", notFound},
			{"/3/movie/orchard", notFound},
			{"/3/tv/1", notFound},
		})
	})
}

func TestSearchReadsPagesOfTwenty(t *testing.T) {
	s := load(t, "single-machine")
	for i := range 24 {
		s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: int64(9100000 + i), IMDbID: "tt9100000", Title: "Orchard", Runtime: time.Hour})
	}
	inBothTiers(t, s, func(t *testing.T, f faketest.Served) {
		for page, want := range map[string]string{"1": `"page":1,`, "2": `"page":2,`} {
			got := get(t, f, "/3/search/movie?query=orchard&page="+page)
			if !strings.Contains(got, want) || !strings.Contains(got, `"total_pages":2,"total_results":25}`) {
				t.Errorf("page %s = %s", page, got)
			}
		}
		if got := get(t, f, "/3/search/movie?query=orchard&page=2"); strings.Count(got, `"id":`) != 5 {
			t.Errorf("page 2 holds %d results, want 5: %s", strings.Count(got, `"id":`), got)
		}
	})
}

func TestScenarioReleasedIsOneUSDigitalDate(t *testing.T) {
	inBothTiers(t, load(t, "single-machine"), func(t *testing.T, f faketest.Served) {
		check(t, f, []exchange{{
			"/3/movie/9000001?append_to_response=external_ids,release_dates",
			`200 OK {"adult":false,"backdrop_path":"/znm9omh75o26.jpg","belongs_to_collection":null,"genres":[],"id":9000001,"imdb_id":"tt9000001","original_language":"en","original_title":"The Quiet Orchard","overview":"","poster_path":"/znvorf0peat3.jpg","production_companies":[],"release_date":"2021-04-16","runtime":104,"status":"Released","tagline":"","title":"The Quiet Orchard","vote_average":0,"vote_count":0,` +
				`"external_ids":{"imdb_id":"tt9000001"},"release_dates":{"results":[{"iso_3166_1":"US","release_dates":[{"certification":"","descriptors":[],"iso_639_1":"","note":"","release_date":"2021-04-16T00:00:00.000Z","type":4}]}]}}` + "\n",
		}})
	})
}

func TestPerCountryDatesAndThePrimaryDate(t *testing.T) {
	s := load(t, "single-machine")
	s.TMDB.Movies[0].Dates = []scenario.TMDBDate{
		{Country: "GB", Kind: "cinema", Date: "2021-03-01", Certification: "15"},
		{Country: "FR", Kind: "premiere", Date: "2021-02-10", Note: "Berlinale"},
		{Country: "GB", Kind: "physical", Date: "2021-06-07"},
	}
	inBothTiers(t, s, func(t *testing.T, f faketest.Served) {
		got := get(t, f, "/3/movie/9000001?append_to_response=release_dates")
		for _, want := range []string{
			`"release_date":"2021-02-10",`,
			`{"iso_3166_1":"FR","release_dates":[{"certification":"","descriptors":[],"iso_639_1":"","note":"Berlinale","release_date":"2021-02-10T00:00:00.000Z","type":1}]}`,
			`{"iso_3166_1":"GB","release_dates":[{"certification":"15","descriptors":[],"iso_639_1":"","note":"","release_date":"2021-03-01T00:00:00.000Z","type":3},{"certification":"","descriptors":[],"iso_639_1":"","note":"","release_date":"2021-06-07T00:00:00.000Z","type":5}]}`,
			`{"iso_3166_1":"US","release_dates":[{"certification":"","descriptors":[],"iso_639_1":"","note":"","release_date":"2021-04-16T00:00:00.000Z","type":4}]}`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("movie = %s\nwant it to hold %s", got, want)
			}
		}
	})
}

func TestAnUndatedMovieIsAllowed(t *testing.T) {
	s, err := scenario.Parse("undated", []byte("seed: x\nnodes: [{name: home, setup: none}]\ntmdb:\n  movies:\n    - {id: 9000005, imdb_id: tt9000005, title: Someday, runtime: 0s, status: Planned}\n"))
	if err != nil {
		t.Fatal(err)
	}
	inBothTiers(t, s, func(t *testing.T, f faketest.Served) {
		check(t, f, []exchange{
			{"/3/movie/9000005?append_to_response=release_dates", `200 OK {"adult":false,"backdrop_path":"/1vlup5f0bwq4y.jpg","belongs_to_collection":null,"genres":[],"id":9000005,"imdb_id":"tt9000005","original_language":"en","original_title":"Someday","overview":"","poster_path":"/1vm4487ju5cvv.jpg","production_companies":[],"release_date":"","runtime":0,"status":"Planned","tagline":"","title":"Someday","vote_average":0,"vote_count":0,"release_dates":{"results":[]}}` + "\n"},
			{"/3/search/movie?query=someday&primary_release_year=2021", none},
		})
	})
}

func TestTheFakeAnswersInTheAskedLanguage(t *testing.T) {
	s := load(t, "single-machine")
	s.TMDB.Movies[0].Overview = "Two sisters keep an orchard."
	s.TMDB.Movies[0].Translations = []scenario.TMDBTranslation{{Language: "fr-FR", Title: "Le Verger tranquille", Overview: "Deux sœurs gardent un verger."}}
	inBothTiers(t, s, func(t *testing.T, f faketest.Served) {
		for path, want := range map[string]string{
			"/3/search/movie?query=verger&language=fr-FR": `"title":"Le Verger tranquille"`,
			"/3/movie/9000001?language=fr-CA":             `"overview":"Deux sœurs gardent un verger.",`,
			"/3/movie/9000001?language=de-DE":             `"overview":"","poster_path"`,
			"/3/movie/9000001?language=en-GB":             `"overview":"Two sisters keep an orchard.",`,
			"/3/movie/9000001?append_to_response=translations": `"translations":{"translations":[{"iso_3166_1":"US","iso_639_1":"en","data":{"title":"The Quiet Orchard","overview":"Two sisters keep an orchard.","tagline":""}},` +
				`{"iso_3166_1":"FR","iso_639_1":"fr","data":{"title":"Le Verger tranquille","overview":"Deux sœurs gardent un verger.","tagline":""}}]}`,
		} {
			if got := get(t, f, path); !strings.Contains(got, want) {
				t.Errorf("GET %s = %s\nwant it to hold %s", path, got, want)
			}
		}
	})
}

func TestImagesAndVideosFollowTheirLanguageLists(t *testing.T) {
	s := load(t, "single-machine")
	s.TMDB.Movies[0].Trailer = "dQw4w9WgXcQ"
	s.TMDB.Movies[0].Images = []scenario.TMDBImage{{Shape: "poster", Language: "fr"}, {Shape: "poster", Language: "en"}, {Shape: "backdrop"}}
	inBothTiers(t, s, func(t *testing.T, f faketest.Served) {
		en := get(t, f, "/3/movie/9000001?append_to_response=images,videos&include_image_language=en,null&include_video_language=en")
		fr := get(t, f, "/3/movie/9000001?append_to_response=images,videos&include_image_language=fr&include_video_language=fr")
		if strings.Count(en, `"file_path"`) != 2 || !strings.Contains(en, `"key":"dQw4w9WgXcQ","site":"YouTube"`) {
			t.Errorf("en,null = %s; want the English poster, the backdrop and the trailer", en)
		}
		if strings.Count(fr, `"file_path"`) != 1 || !strings.Contains(fr, `"iso_639_1":"fr"`) || !strings.Contains(fr, `"videos":{"results":[]}`) {
			t.Errorf("fr = %s; want only the French poster and no video", fr)
		}
	})
}

func TestRevokeKeyAnswersTMDBs401Body(t *testing.T) {
	inBothTiers(t, load(t, "single-machine"), func(t *testing.T, f faketest.Served) {
		before := get(t, f, "/3/configuration?api_key=bundled")
		if !strings.Contains(before, "200 OK") || !strings.Contains(get(t, f, "/3/configuration"), "200 OK") {
			t.Fatalf("configuration = %s, want any key and none taken", before)
		}
		if got := f.Cue(t, tmdb.RevokeKey.Name, tmdb.Key{}); got == nil {
			t.Fatal("revoke-key answered nothing")
		}
		check(t, f, []exchange{
			{"/3/configuration?api_key=bundled", unauthorized},
			{"/3/search/movie?query=orchard&api_key=bundled", unauthorized},
			{"/3/movie/9000001", unauthorized},
		})
		if got := get(t, f, "/3/movie/9000001?api_key=yours"); !strings.Contains(got, "200 OK") {
			t.Errorf("a key the fake never saw = %s, want 200", got)
		}
		yours := "yours"
		f.Cue(t, tmdb.RevokeKey.Name, tmdb.Key{Key: &yours})
		check(t, f, []exchange{{"/3/movie/changes?api_key=yours", unauthorized}})
	})
}

func TestDeleteAnswers404Status34(t *testing.T) {
	s := load(t, "single-machine")
	s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: 9000006, IMDbID: "tt9000006", Title: "Orchard Lost", Deleted: true})
	inBothTiers(t, s, func(t *testing.T, f faketest.Served) {
		check(t, f, []exchange{{"/3/movie/9000006", notFound}})
		f.Cue(t, tmdb.Delete.Name, tmdb.Movie{ID: 9000001})
		today := time.Now().UTC().Format(time.DateOnly)
		check(t, f, []exchange{
			{"/3/movie/9000001", notFound},
			{"/3/search/movie?query=orchard", none},
			{"/3/movie/changes?start_date=" + today + "&end_date=" + today, `200 OK {"results":[{"id":9000001,"adult":false}],"page":1,"total_pages":1,"total_results":1}` + "\n"},
		})
	})
}

func TestChangeAltersTheRecordAndListsIt(t *testing.T) {
	inBothTiers(t, load(t, "single-machine"), func(t *testing.T, f faketest.Served) {
		before := get(t, f, "/3/movie/9000001")
		f.Cue(t, tmdb.Change.Name, tmdb.Movie{ID: 9000001})
		after := get(t, f, "/3/movie/9000001")
		if before == after || !strings.Contains(after, `"overview":"Revision 1."`) || !strings.Contains(after, `"vote_count":1}`) {
			t.Errorf("after a change the record is %s", after)
		}
		day := time.Now().UTC()
		yesterday := day.Add(-24 * time.Hour).Format(time.DateOnly)
		check(t, f, []exchange{
			{"/3/movie/changes?start_date=" + day.Format(time.DateOnly), `200 OK {"results":[{"id":9000001,"adult":false}],"page":1,"total_pages":1,"total_results":1}` + "\n"},
			{"/3/movie/changes?start_date=" + yesterday + "&end_date=" + yesterday, `200 OK {"results":[],"page":1,"total_pages":0,"total_results":0}` + "\n"},
		})
	})
}

func TestAScenarioWithoutTMDBServesAnEmptyCatalog(t *testing.T) {
	inBothTiers(t, load(t, "first-run"), func(t *testing.T, f faketest.Served) {
		check(t, f, []exchange{
			{"/3/search/movie?query=orchard", none},
			{"/3/movie/9000001", notFound},
		})
	})
}

// rich gives s's first movie every block the conformance suite reads.
func rich(s scenario.Scenario) scenario.Scenario {
	m := &s.TMDB.Movies[0]
	m.Overview = "Two sisters keep an orchard through one long summer."
	m.Genres = []string{"Drama"}
	m.Studio = &scenario.TMDBNamed{ID: 90001, Name: "Lantern Field Pictures"}
	m.Titles = []scenario.TMDBTitle{{Country: "GB", Title: "Quiet Orchard"}}
	m.Credits = scenario.TMDBCredits{
		Cast: []scenario.TMDBCast{{ID: 91001, Name: "Mara Vell", Character: "June"}},
		Crew: []scenario.TMDBCrew{{ID: 91002, Name: "Osei Brandt", Department: "Directing", Job: "Director"}},
	}
	m.Trailer = "qOrchard01"
	return s
}

func TestTheFakeConforms(t *testing.T) {
	inBothTiers(t, rich(load(t, "single-machine")), func(t *testing.T, f faketest.Served) {
		conformance.Run(t, conformance.Target{
			Endpoints: provider.Endpoints{API: endpoint.MustParseURL(f.Address.URL("/3")), Images: endpoint.MustParseURL(f.Address.URL("/t/p"))},
			Client:    f.Client,
			Known:     conformance.Known{ID: 9000001, Title: "The Quiet Orchard", IMDbID: "tt9000001", Year: 2021},
		})
	})
}
