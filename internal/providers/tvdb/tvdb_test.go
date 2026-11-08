package tvdb_test

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketvdb "github.com/jatassi/slipstream/fakes/tvdb"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/works"
)

func client(f faketest.Served) *tvdb.Client {
	lines := metadata.NewLines(f.Client, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	return tvdb.New(lines, tvdb.Endpoints{API: endpoint.MustParseURL(f.Address.URL("/v4")), Artwork: endpoint.MustParseURL("https://artworks.example.test")}, tvdb.DefaultTimings())
}

func inEachTier(t *testing.T, name string, test func(t *testing.T, c *tvdb.Client, f faketest.Served), opts ...faketest.Option) {
	s, err := scenario.Load(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("loopback", func(t *testing.T) {
		f := faketest.Loopback(t, faketvdb.Fake, s, opts...)
		test(t, client(f), f)
	})
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			f := faketest.InMemory(t, faketvdb.Fake, s, opts...)
			test(t, client(f), f)
		})
	})
}

func ask(language string) metadata.Ask {
	return metadata.NewAsk(metadata.AskOptions{Language: language, Keys: map[metadata.ProviderID]string{tvdb.ID: "key"}})
}

var english = ask("en-US")

// around is the last of days on or before today and the first after it,
// by the clock the fake reads: the in-memory tier's starts in 2000.
func around(days ...string) (last, next string) {
	today := time.Now().UTC().Format(time.DateOnly)
	for _, d := range days {
		if d <= today {
			last = d
		} else if next == "" {
			next = d
		}
	}
	return last, next
}

func TestSeriesReadsTheRecord(t *testing.T) {
	inEachTier(t, "series", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		s, err := c.Series(t.Context(), english, 1001)
		if err != nil {
			t.Fatal(err)
		}
		last, next := around("2020-01-10", "2023-03-17")
		if s.Name != "The Lantern Keepers" || s.English != s.Name || s.Status != tvdb.Continuing || s.Network != "Lantern One" || s.IMDb != "tt1001001" || s.TMDB != 7001 ||
			s.FirstAired != "2020-01-10" || s.LastAired != last || s.NextAired != next || s.AverageRuntime != 48*time.Minute || len(s.Genres) != 2 {
			t.Errorf("Series(1001) = %+v", s)
		}
		if want := []tvdb.Order{tvdb.OrderAired, tvdb.OrderDVD, tvdb.OrderAbsolute}; !slices.Equal(s.Orders, want) {
			t.Errorf("Orders = %v, want %v", s.Orders, want)
		}
		var seasons []int
		for _, season := range s.Seasons {
			if season.Order == tvdb.OrderAired {
				seasons = append(seasons, season.Number)
			}
		}
		if !slices.Equal(seasons, []int{0, 1, 2, 3}) {
			t.Errorf("Aired seasons = %v, want 0 to 3", seasons)
		}
		poster, err := c.Artwork().URL(artwork.ShapePoster, s.Poster, artwork.SizeOriginal)
		if err != nil || poster.Host != "artworks.example.test" {
			t.Errorf("poster %q = %v, %v; want it on the artwork endpoint", s.Poster, poster, err)
		}
	})
}

func TestSeriesNamesFallBackToEnglish(t *testing.T) {
	inEachTier(t, "series", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		forest, err := c.Series(t.Context(), ask("fr-FR"), 1004)
		if err != nil || forest.Name != "The Shell Forest" || forest.English != "The Shell Forest" || forest.Original != "Kōkaku no Mori" || forest.OriginalLanguage != "jpn" || forest.TMDB != 7004 {
			t.Errorf("Series(1004) in French = %+v, %v; want English as the fallback name", forest, err)
		}
	})
}

func TestASeriesWithNoTMDBIDReadsZero(t *testing.T) {
	inEachTier(t, "series", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		signal, err := c.Series(t.Context(), english, 1003)
		if _, next := around("2026-10-02", "2027-01-15"); err != nil || signal.TMDB != 0 || signal.NextAired != next {
			t.Errorf("Series(1003) = %+v, %v; want no TMDB id and its next airing", signal, err)
		}
	})
}

func TestSeriesNotOnTVDBIsNotFound(t *testing.T) {
	inEachTier(t, "series", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		if _, err := c.Series(t.Context(), english, 4242); !errors.Is(err, works.ErrNotFound) {
			t.Errorf("Series(4242) = %v, want works.ErrNotFound", err)
		}
		if _, err := c.Episodes(t.Context(), english, 4242, tvdb.OrderAired); !errors.Is(err, works.ErrNotFound) {
			t.Errorf("Episodes(4242) = %v, want works.ErrNotFound", err)
		}
	})
}

func TestEpisodesReadEveryPage(t *testing.T) {
	inEachTier(t, "series-scale", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		for id, want := range map[tvdb.SeriesID]int{60_000_001: 1_200, 60_000_002: 10_000} {
			episodes, err := c.Episodes(t.Context(), english, id, tvdb.OrderAired)
			if err != nil {
				t.Fatal(err)
			}
			if len(episodes) != want || episodes[0].Season != 1 || episodes[0].Number != 1 || episodes[want-1].Season == 1 {
				t.Errorf("Episodes(%d) = %d episodes from S%02dE%02d, want %d over every page", id, len(episodes), episodes[0].Season, episodes[0].Number, want)
			}
		}
	})
}

func TestEpisodesFailWholeOnAFailedPage(t *testing.T) {
	failed := fault.Rule{ID: "middle", Target: "tvdb", Route: "episodes", Skip: 1, Fault: fault.Status{Code: http.StatusInternalServerError}}
	inEachTier(t, "series-scale", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		episodes, err := c.Episodes(t.Context(), english, 60_000_001, tvdb.OrderAired)
		if !errors.Is(err, works.ErrUnavailable) || episodes != nil {
			t.Errorf("Episodes with page 2 failing = %d episodes, %v; want none and works.ErrUnavailable", len(episodes), err)
		}
	}, faketest.WithFaults(failed))

	once := failed
	once.Times = 1
	inEachTier(t, "series-scale", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		if episodes, err := c.Episodes(t.Context(), english, 60_000_001, tvdb.OrderAired); err != nil || len(episodes) != 1_200 {
			t.Errorf("Episodes with page 2 failing once = %d episodes, %v; want the retry to read all 1,200", len(episodes), err)
		}
	}, faketest.WithFaults(once))
}

func TestARateLimitedCallWaitsItsRetryAfter(t *testing.T) {
	limited := fault.Rule{ID: "limited", Target: "tvdb", Route: "series", Times: 1, Fault: fault.Status{Code: http.StatusTooManyRequests, RetryAfter: time.Second}}
	inEachTier(t, "series", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
		start := time.Now()
		if _, err := c.Series(t.Context(), english, 1001); err != nil || time.Since(start) < time.Second {
			t.Errorf("Series behind a 429 = %v after %s, want the record after a second", err, time.Since(start))
		}
	}, faketest.WithFaults(limited))
}

func TestEpisodesAskInTVDBsLanguageCode(t *testing.T) {
	for tag, code := range map[string]string{"fr-CA": "fra", "de": "deu", "pt-BR": "por", "en-US": "eng", "": "eng", "not a tag": "eng"} {
		t.Run(tag, func(t *testing.T) {
			marked := fault.Rule{ID: "marked", Target: "tvdb", Route: "episodes", Params: map[string]string{"lang": code}, Fault: fault.Status{Code: http.StatusNotFound}}
			inEachTier(t, "series", func(t *testing.T, c *tvdb.Client, _ faketest.Served) {
				if _, err := c.Episodes(t.Context(), ask(tag), 1001, tvdb.OrderAired); !errors.Is(err, works.ErrNotFound) {
					t.Errorf("Episodes in %q = %v, want the request in %s", tag, err, code)
				}
			}, faketest.WithFaults(marked))
		})
	}
}

func TestArtworkServesThumbnailsForSmallSizes(t *testing.T) {
	c := tvdb.New(metadata.NewLines(http.DefaultClient, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))), //nolint:gosec // a fixed draw
		tvdb.Endpoints{Artwork: endpoint.MustParseURL("https://artworks.example.test")}, tvdb.DefaultTimings())
	const path = "/banners/v4/series/1001/posters/abc.jpg"
	for size, want := range map[artwork.Size]string{
		artwork.SizeSmall:    "https://artworks.example.test/banners/v4/series/1001/posters/abc_t.jpg",
		artwork.SizeMedium:   "https://artworks.example.test/banners/v4/series/1001/posters/abc_t.jpg",
		artwork.SizeLarge:    "https://artworks.example.test/banners/v4/series/1001/posters/abc.jpg",
		artwork.SizeOriginal: "https://artworks.example.test/banners/v4/series/1001/posters/abc.jpg",
	} {
		if u, err := c.Artwork().URL(artwork.ShapePoster, path, size); err != nil || u.String() != want {
			t.Errorf("URL(%s) = %v, %v; want %s", size, u, err, want)
		}
	}
	for _, bad := range []string{"/t/p/abc.jpg", "/banners/../secret.jpg", "/banners/v4/x.exe", ""} {
		if _, err := c.Artwork().URL(artwork.ShapePoster, bad, artwork.SizeSmall); err == nil {
			t.Errorf("URL(%q) took a path that is not TVDB artwork", bad)
		}
	}
}
