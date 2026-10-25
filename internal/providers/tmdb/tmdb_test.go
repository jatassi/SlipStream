package tmdb_test

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketmdb "github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

func client(f faketest.Served) *tmdb.Client {
	lines := metadata.NewLines(f.Client, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	return tmdb.New(lines, tmdb.Endpoints{API: endpoint.MustParseURL(f.Address.URL("/3")), Images: endpoint.MustParseURL(f.Address.URL("/t/p"))}, tmdb.DefaultTimings())
}

func inEachTier(t *testing.T, s scenario.Scenario, test func(t *testing.T, c *tmdb.Client, f faketest.Served), opts ...faketest.Option) {
	t.Run("loopback", func(t *testing.T) {
		f := faketest.Loopback(t, faketmdb.Fake, s, opts...)
		test(t, client(f), f)
	})
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			f := faketest.InMemory(t, faketmdb.Fake, s, opts...)
			test(t, client(f), f)
		})
	})
}

func singleMachine(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var english = metadata.NewAsk(metadata.AskOptions{Language: "en-US", Region: "US"})

func TestTMDBMovieDecodesEveryAppendedBlock(t *testing.T) {
	s := singleMachine(t)
	s.TMDB.Movies[0] = scenario.TMDBMovie{
		ID: 9000001, IMDbID: "tt9000001", Title: "The Quiet Orchard", Released: "2021-06-01", Runtime: 104 * time.Minute,
		Dates:            []scenario.TMDBDate{{Country: "US", Kind: "cinema", Date: "2021-04-16", Certification: "PG-13", Note: "wide"}},
		OriginalTitle:    "Le Verger tranquille",
		OriginalLanguage: "fr",
		Overview:         "Two sisters keep an orchard.",
		Tagline:          "Every tree remembers.",
		Genres:           []string{"Drama"},
		Status:           "Released",
		Titles:           []scenario.TMDBTitle{{Country: "GB", Title: "Quiet Orchard", Kind: "short title"}},
		Translations:     []scenario.TMDBTranslation{{Language: "fr-FR", Title: "Le Verger tranquille", Overview: "Deux sœurs."}},
		Credits: scenario.TMDBCredits{
			Cast: []scenario.TMDBCast{{ID: 91001, Name: "Mara Venn", Character: "June"}},
			Crew: []scenario.TMDBCrew{{ID: 91002, Name: "Osei Brandt", Department: "Directing", Job: "Director"}},
		},
		Trailer:     "qOrchard01",
		Collection:  &scenario.TMDBNamed{ID: 92001, Name: "The Orchard Collection"},
		Studio:      &scenario.TMDBNamed{ID: 90001, Name: "Lantern Field Pictures"},
		Images:      []scenario.TMDBImage{{Shape: "poster", Language: "en", Votes: 5.2}, {Shape: "backdrop"}, {Shape: "logo", Language: "fr"}},
		VoteAverage: 7.4,
		VoteCount:   210,
	}
	inEachTier(t, s, func(t *testing.T, c *tmdb.Client, _ faketest.Served) {
		m, err := c.Movie(t.Context(), metadata.NewAsk(metadata.AskOptions{Language: "fr-FR"}), 9000001)
		if err != nil {
			t.Fatal(err)
		}
		files := []*string{&m.PosterPath, &m.BackdropPath, &m.ProductionCompanies[0].LogoPath, &m.Credits.Cast[0].ProfilePath, &m.Credits.Crew[0].ProfilePath}
		for _, list := range [][]tmdb.Image{m.Images.Posters, m.Images.Backdrops, m.Images.Logos} {
			for i := range list {
				files = append(files, &list[i].FilePath)
			}
		}
		for _, p := range files {
			if _, err := c.Images().URL(artwork.ShapePoster, *p, artwork.SizeOriginal); err != nil {
				t.Errorf("file %q: %v", *p, err)
			}
			*p = ""
		}
		m.Genres[0].ID = 0
		m.Videos.Results[0].PublishedAt = ""
		want := tmdb.Movie{
			ID: 9000001, IMDbID: "tt9000001", Title: "Le Verger tranquille", OriginalTitle: "Le Verger tranquille", OriginalLanguage: "fr",
			Overview: "Deux sœurs.", ReleaseDate: "2021-04-16", Runtime: 104, Status: "Released",
			Genres:              []tmdb.Genre{{Name: "Drama"}},
			ProductionCompanies: []tmdb.Company{{ID: 90001, Name: "Lantern Field Pictures", OriginCountry: "US"}},
			BelongsToCollection: &tmdb.Collection{ID: 92001, Name: "The Orchard Collection"},
			VoteAverage:         7.4, VoteCount: 210,
			ExternalIDs: tmdb.ExternalIDs{IMDbID: "tt9000001"},
			ReleaseDates: tmdb.ReleaseDates{Results: []tmdb.CountryReleases{{Country: "US", Releases: []tmdb.Release{
				{Certification: "PG-13", Note: "wide", Date: "2021-04-16T00:00:00.000Z", Type: tmdb.Theatrical},
				{Date: "2021-06-01T00:00:00.000Z", Type: tmdb.Digital},
			}}}},
			AlternativeTitles: tmdb.AlternativeTitles{Titles: []tmdb.AlternativeTitle{{Country: "GB", Title: "Quiet Orchard", Type: "short title"}}},
			Credits: tmdb.Credits{
				Cast: []tmdb.Cast{{ID: 91001, Name: "Mara Venn", Character: "June"}},
				Crew: []tmdb.Crew{{ID: 91002, Name: "Osei Brandt", Department: "Directing", Job: "Director"}},
			},
			Videos: tmdb.Videos{Results: []tmdb.Video{{Name: "Official Trailer", Key: "qOrchard01", Site: "YouTube", Type: "Trailer", Official: true, Language: "en", Country: "US"}}},
			Images: tmdb.Images{
				Backdrops: []tmdb.Image{{Width: 3840, Height: 2160}},
				Logos:     []tmdb.Image{{Language: "fr", Width: 1000, Height: 400, VoteCount: 1}},
				Posters:   []tmdb.Image{{Language: "en", Width: 2000, Height: 3000, VoteAverage: 5.2, VoteCount: 1}},
			},
			Translations: tmdb.Translations{Translations: []tmdb.Translation{
				{Language: "en", Country: "US", Data: tmdb.TranslationData{Title: "The Quiet Orchard", Overview: "Two sisters keep an orchard.", Tagline: "Every tree remembers."}},
				{Language: "fr", Country: "FR", Data: tmdb.TranslationData{Title: "Le Verger tranquille", Overview: "Deux sœurs."}},
			}},
		}
		want.Images.Backdrops[0].VoteCount = 1
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("Movie =\n%+v\nwant\n%+v", m, want)
		}
	})
}

func TestSearchReadsPagesAndThePrimaryReleaseYear(t *testing.T) {
	s := singleMachine(t)
	for i := range 21 {
		s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: int64(9100000 + i), IMDbID: "tt9100000", Title: "Orchard " + strconv.Itoa(i), Released: "2019-01-05", Runtime: time.Hour})
	}
	inEachTier(t, s, func(t *testing.T, c *tmdb.Client, _ faketest.Served) {
		first, err := c.SearchMovies(t.Context(), english, "orchard", 0, 1)
		if err != nil || first.Page != 1 || first.TotalPages != 2 || first.TotalResults != 22 || len(first.Results) != 20 {
			t.Fatalf("page 1 = page %d of %d, %d of %d results, %v", first.Page, first.TotalPages, len(first.Results), first.TotalResults, err)
		}
		second, err := c.SearchMovies(t.Context(), english, "orchard", 0, 2)
		if err != nil || second.Page != 2 || len(second.Results) != 2 {
			t.Fatalf("page 2 = page %d, %d results, %v", second.Page, len(second.Results), err)
		}
		in2021, err := c.SearchMovies(t.Context(), english, "orchard", 2021, 1)
		if want := []tmdb.MovieResult{{ID: 9000001, Title: "The Quiet Orchard", OriginalTitle: "The Quiet Orchard", OriginalLanguage: "en", ReleaseDate: "2021-04-16", PosterPath: in2021.Results[0].PosterPath}}; err != nil || !reflect.DeepEqual(in2021.Results, want) || want[0].PosterPath == "" {
			t.Fatalf("SearchMovies(orchard, 2021) = %+v, %v; want %+v", in2021.Results, err, want)
		}
	})
}

func TestAnUnknownOrDeletedMovieIsNotFound(t *testing.T) {
	inEachTier(t, singleMachine(t), func(t *testing.T, c *tmdb.Client, f faketest.Served) {
		f.Cue(t, faketmdb.Delete.Name, faketmdb.Movie{ID: 9000001})
		for _, id := range []int64{9000002, 9000001} {
			_, err := c.Movie(t.Context(), english, id)
			if want := "GET /movie/" + strconv.FormatInt(id, 10) + ": TMDB: has no such record"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
				t.Errorf("Movie(%d) = %v, want %s", id, err, want)
			}
		}
	})
}

func TestARevokedKeyIsRefusedInTMDBsWords(t *testing.T) {
	keyed := func(key string) metadata.Ask {
		return metadata.NewAsk(metadata.AskOptions{Keys: map[metadata.ProviderID]string{tmdb.ID: key}})
	}
	inEachTier(t, singleMachine(t), func(t *testing.T, c *tmdb.Client, f faketest.Served) {
		revoked := "revoked"
		f.Cue(t, faketmdb.RevokeKey.Name, faketmdb.Key{Key: &revoked})
		err := c.Test(t.Context(), keyed(revoked))
		if want := "GET /configuration: TMDB: refused the key: Invalid API key: You must be granted a valid key."; err == nil || err.Error() != want || !errors.Is(err, works.ErrRefused) {
			t.Errorf("Test with a revoked key = %v, want %s", err, want)
		}
		if err := c.Test(t.Context(), keyed("fresh")); err != nil {
			t.Errorf("Test with another key = %v", err)
		}
		if err := c.Test(t.Context(), metadata.Ask{}); err != nil {
			t.Errorf("Test with no key = %v", err)
		}
	})
}

func TestA404WithoutTMDBsBodyIsUnreachable(t *testing.T) {
	inEachTier(t, singleMachine(t), func(t *testing.T, c *tmdb.Client, _ faketest.Served) {
		_, err := c.Movie(t.Context(), english, 9000001)
		if want := "GET /movie/9000001: TMDB: responded with 404 Not Found but no not-found body"; err == nil || err.Error() != want || !errors.Is(err, works.ErrUnavailable) {
			t.Errorf("Movie behind a stray 404 = %v, want %s", err, want)
		}
	}, faketest.WithFaults(fault.Rule{ID: "stray", Target: "tmdb", Route: "movie", Fault: fault.Status{Code: 404}, Times: 1}))
}

func TestARateLimitIsWaitedOutAndA5xxRetried(t *testing.T) {
	inEachTier(t, singleMachine(t), func(t *testing.T, c *tmdb.Client, _ faketest.Served) {
		start := time.Now()
		m, err := c.Movie(t.Context(), english, 9000001)
		if err != nil || m.Title != "The Quiet Orchard" {
			t.Fatalf("Movie after a 429 and a 503 = %q, %v", m.Title, err)
		}
		if waited := time.Since(start); waited < time.Second {
			t.Errorf("Movie returned after %s, before the 1s Retry-After", waited)
		}
	}, faketest.WithFaults(
		fault.Rule{ID: "limited", Target: "tmdb", Route: "movie", Fault: fault.Status{Code: 429, RetryAfter: time.Second}, Times: 1},
		fault.Rule{ID: "down", Target: "tmdb", Route: "movie", Fault: fault.Status{Code: 503}, Times: 1},
	))
}

func TestMovieChangesListsADayAndRefusesMoreThanTMDBKeeps(t *testing.T) {
	inEachTier(t, singleMachine(t), func(t *testing.T, c *tmdb.Client, f faketest.Served) {
		f.Cue(t, faketmdb.Change.Name, faketmdb.Movie{ID: 9000001})
		now := time.Now()
		got, err := c.MovieChanges(t.Context(), english, now.Add(-24*time.Hour), now, 1)
		if want := (tmdb.ChangesPage{IDs: []int64{9000001}, Page: 1, TotalPages: 1}); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("MovieChanges = %+v, %v; want %+v", got, err, want)
		}
		m, err := c.Movie(t.Context(), english, 9000001)
		if err != nil || m.Overview != "Revision 1." {
			t.Errorf("the changed record's overview = %q, %v", m.Overview, err)
		}
		_, err = c.MovieChanges(t.Context(), english, now.Add(-tmdb.ChangesWindow-time.Second), now, 1)
		if !errors.Is(err, metadata.ErrNoChanges) {
			t.Errorf("MovieChanges past 14 days = %v, want metadata.ErrNoChanges", err)
		}
	})
}

func TestImagesMapSizeClassesPerShape(t *testing.T) {
	lines := metadata.NewLines(nil, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	images := tmdb.New(lines, tmdb.DefaultEndpoints(), tmdb.DefaultTimings()).Images()
	sizes := []artwork.Size{artwork.SizeSmall, artwork.SizeMedium, artwork.SizeLarge, artwork.SizeOriginal}
	for shape, want := range map[artwork.Shape][]string{
		artwork.ShapePoster:   {"w185", "w342", "w780", "original"},
		artwork.ShapeBackdrop: {"w300", "w780", "w1280", "original"},
		artwork.ShapeLogo:     {"w185", "w300", "w500", "original"},
		artwork.ShapePhoto:    {"w45", "w185", "h632", "original"},
	} {
		for i, size := range sizes {
			u, err := images.URL(shape, "/kqjL17yufvn9OVLyXYpvtyrFfak.jpg", size)
			if err != nil || u.String() != tmdb.DefaultEndpoints().Images.String()+"/"+want[i]+"/kqjL17yufvn9OVLyXYpvtyrFfak.jpg" {
				t.Errorf("URL(%s, %s) = %v, %v; want size %s", shape, size, u, err, want[i])
			}
		}
	}
	for _, bad := range []struct {
		shape artwork.Shape
		path  string
	}{{artwork.ShapeCover, "/a.jpg"}, {artwork.ShapePoster, "a.jpg"}, {artwork.ShapePoster, "/../a.jpg"}, {artwork.ShapePoster, "/a.jpg?x=1"}} {
		if u, err := images.URL(bad.shape, bad.path, artwork.SizeSmall); err == nil {
			t.Errorf("URL(%s, %q) = %s, want refused", bad.shape, bad.path, u)
		}
	}
	if perSecond, burst := images.Rate(); perSecond != 20 || burst != 20 || images.ID() != tmdb.ImagesID {
		t.Errorf("Rate = %v/s burst %d, id %s", perSecond, burst, images.ID())
	}
}
