package series

import (
	"cmp"
	"errors"
	"log/slog"
	"maps"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	fakeomdb "github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketmdb "github.com/jatassi/slipstream/fakes/tmdb"
	faketvdb "github.com/jatassi/slipstream/fakes/tvdb"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/works"
)

// providers is fake-tvdb, fake-tmdb and fake-omdb in memory, read through
// one Node's call layer, with every request each was sent.
type providers struct {
	source           metadata.Source
	tvdb, tmdb, omdb faketest.Served
	mu               sync.Mutex
	sent             map[string][]string
}

// sentTo is the path and query of each request the fake named got.
func (p *providers) sentTo(name string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.sent[name])
}

func (p *providers) RoundTrip(r *http.Request) (*http.Response, error) {
	name, to := "omdb", p.omdb
	switch {
	case strings.HasPrefix(r.URL.Path, "/v4/"):
		name, to = "tvdb", p.tvdb
	case strings.HasPrefix(r.URL.Path, "/3/"):
		name, to = "tmdb", p.tmdb
	}
	p.mu.Lock()
	p.sent[name] = append(p.sent[name], r.URL.RequestURI())
	p.mu.Unlock()
	return to.Client.Transport.RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

func scenarioOf(t *testing.T, name string) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load(name)
	if err != nil {
		t.Fatal(err)
	}
	s.TMDB, s.OMDb = cmp.Or(s.TMDB, &scenario.TMDB{}), cmp.Or(s.OMDb, &scenario.OMDb{})
	s.TMDB.Series = append(s.TMDB.Series,
		scenario.TMDBSeries{
			ID: 7001, TVDB: 1001, Name: "The Lantern Keepers", VoteAverage: 8.1, VoteCount: 420,
			Creators: []scenario.TMDBCast{{ID: 1, Name: "Mara Quill"}}, Credits: scenario.TMDBCredits{Cast: []scenario.TMDBCast{{ID: 2, Name: "Ines Varga", Character: "Keeper"}}},
			Certifications: []scenario.TMDBCertification{{Country: "US", Rating: "TV-14"}}, Trailer: "lantern-trailer",
			Images: []scenario.TMDBImage{{Shape: "poster", Language: "en"}, {Shape: "backdrop"}},
		},
		scenario.TMDBSeries{ID: 7999, Name: "The Lantern Road", FirstAired: "2024-05-01"},
	)
	s.OMDb.Titles = append(s.OMDb.Titles, scenario.OMDbTitle{
		IMDbID: "tt1001001", Type: "series", IMDbRating: ptr(8.4), IMDbVotes: ptr(int64(1200)), Awards: "2 wins.",
		Seasons: []scenario.OMDbSeason{
			{Number: 1, Episodes: []scenario.OMDbEpisode{{Episode: 1, IMDbID: "tt1001111", Rating: ptr(8.0)}, {Episode: 2, IMDbID: "tt1001112", Rating: ptr(7.5)}}},
			{Number: 3, Episodes: []scenario.OMDbEpisode{{Episode: 3, IMDbID: "tt1001133", Rating: ptr(9.1)}}},
		},
	})
	return s
}

// serve runs test in a synctest bubble over the fakes, fake-tvdb's faults
// set before it serves.
func serve(t *testing.T, name string, test func(t *testing.T, p *providers), tvdbFaults ...fault.Rule) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		s := scenarioOf(t, name)
		p := &providers{
			tvdb: faketest.InMemory(t, faketvdb.Fake, s, faketest.WithFaults(tvdbFaults...)),
			tmdb: faketest.InMemory(t, faketmdb.Fake, s), omdb: faketest.InMemory(t, fakeomdb.Fake, s),
			sent: map[string][]string{},
		}
		lines := metadata.NewLines(&http.Client{Transport: p}, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
		p.source = Def.OpenProvider(contenttype.ProviderDeps{
			TVDB: tvdb.New(lines, tvdb.Endpoints{API: endpoint.MustParseURL(p.tvdb.Address.URL("/v4")), Artwork: endpoint.MustParseURL("https://artworks.example.test")}, tvdb.Timings{Interval: time.Millisecond, Burst: 100}),
			TMDB: tmdb.New(lines, tmdb.Endpoints{API: endpoint.MustParseURL(p.tmdb.Address.URL("/3"))}, tmdb.DefaultTimings()),
			OMDb: omdb.New(lines, omdb.Endpoints{API: endpoint.MustParseURL(p.omdb.Address.URL(""))}, omdb.DefaultTimings()),
			Log:  slog.New(slog.DiscardHandler),
		}, Def.Endpoints())
		test(t, p)
	})
}

func askFor(r metadata.Ratings) metadata.Ask {
	return metadata.NewAsk(metadata.AskOptions{
		Language: "en-US", Region: "US", Ratings: r,
		Keys: map[metadata.ProviderID]string{tvdb.ID: "tvdb-key", tmdb.ID: "tmdb-key", omdb.ID: "omdb-key"},
	})
}

func itemsOf(f works.Fetched) []works.Key {
	var out []works.Key
	for _, level := range f.Children {
		for _, it := range level.Children {
			out = append(out, it.Key)
		}
	}
	return out
}

func TestFetchReadsEveryPage(t *testing.T) {
	serve(t, "series-scale", func(t *testing.T, p *providers) {
		f, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsNone), "60000001", nil)
		if err != nil {
			t.Fatal(err)
		}
		items := itemsOf(f)
		if len(f.Children) != 12 || len(items) != 1_200 || len(slices.Compact(slices.Sorted(slices.Values(items)))) != 1_200 {
			t.Errorf("Fetch = %d seasons of %d episodes in all, want 12 seasons of 1,200 distinct episodes over every page", len(f.Children), len(items))
		}
		if pages := len(p.sentTo("tvdb")) - 2; pages != 3 {
			t.Errorf("TVDB got %d episode pages, want 3", pages)
		}
	})
}

func TestFailedMiddlePageFailsWhole(t *testing.T) {
	middle := fault.Rule{ID: "middle", Target: "tvdb", Route: "episodes", Skip: 1, Fault: fault.Status{Code: http.StatusInternalServerError}}
	serve(t, "series-scale", func(t *testing.T, p *providers) {
		f, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsNone), "60000001", nil)
		if !errors.Is(err, works.ErrUnavailable) || f.Detail != nil || f.Children != nil {
			t.Errorf("Fetch with page 2 failing = %d levels, %v; want nothing and works.ErrUnavailable", len(f.Children), err)
		}
	}, middle)
}

func TestFetchByStoredIDNeverSearches(t *testing.T) {
	serve(t, "series", func(t *testing.T, p *providers) {
		f, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsWork), "1001", nil)
		if err != nil {
			t.Fatal(err)
		}
		if f.Label != "The Lantern Keepers" || f.SortLabel != "Lantern Keepers, The" || len(f.Children) != 4 || f.Children[3].Label != "Specials" || !f.Children[3].Aside {
			t.Errorf("Fetch(1001) = %q with %d levels, want The Lantern Keepers in three seasons and the specials last", f.Label, len(f.Children))
		}
		for _, name := range []string{"tvdb", "tmdb", "omdb"} {
			if i := slices.IndexFunc(p.sentTo(name), func(u string) bool { return strings.Contains(u, "/search") || strings.Contains(u, "s=") }); i >= 0 {
				t.Errorf("Fetch sent %s %s, want no search", name, p.sentTo(name)[i])
			}
		}

		before := len(p.sentTo("tvdb")) + len(p.sentTo("tmdb"))
		if _, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsWork), "tmdb-7999", nil); !errors.Is(err, works.ErrNotOnSource) {
			t.Errorf("Fetch(tmdb-7999) = %v, want works.ErrNotOnSource", err)
		}
		if after := len(p.sentTo("tvdb")) + len(p.sentTo("tmdb")); after != before {
			t.Errorf("Fetch(tmdb-7999) sent %d requests, want none", after-before)
		}
		if _, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsNone), "1002", []byte(`{"order":"dvd"}`)); !errors.Is(err, works.ErrOrderUnavailable) {
			t.Errorf("Fetch(1002) in DVD order = %v, want works.ErrOrderUnavailable", err)
		}
	})
}

func TestSearchFallsBackToTMDBAndMarksUnlisted(t *testing.T) {
	serve(t, "series", func(t *testing.T, p *providers) {
		found, err := p.source.Search(t.Context(), askFor(metadata.RatingsNone), "lantern")
		if err != nil || len(found) != 1 || found[0].Key != "1001" || found[0].Unlisted || len(p.sentTo("tmdb")) != 0 {
			t.Errorf("Search(lantern) = %+v, %v after %d TMDB calls; want TVDB's The Lantern Keepers alone", found, err, len(p.sentTo("tmdb")))
		}
	})
	down := fault.Rule{ID: "down", Target: "tvdb", Route: "search", Fault: fault.Status{Code: http.StatusServiceUnavailable}}
	serve(t, "series", func(t *testing.T, p *providers) {
		found, err := p.source.Search(t.Context(), askFor(metadata.RatingsNone), "lantern")
		if err != nil {
			t.Fatal(err)
		}
		got := map[works.Key]bool{}
		for _, c := range found {
			got[c.Key] = c.Unlisted
		}
		if want := map[works.Key]bool{"1001": false, "tmdb-7999": true}; len(found) != 2 || !maps.Equal(got, want) {
			t.Errorf("Search(lantern) with TVDB down = %v by key and Unlisted, want %v", got, want)
		}
		if lookups := len(p.sentTo("tmdb")) - 1; lookups != 2 {
			t.Errorf("Search read %d external ids, want one a result", lookups)
		}
	}, down)
}

func TestAddReadsNoSeasonRatings(t *testing.T) {
	serve(t, "series", func(t *testing.T, p *providers) {
		f, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsWork), "1001", nil)
		if err != nil {
			t.Fatal(err)
		}
		if i := slices.IndexFunc(p.sentTo("omdb"), func(u string) bool { return strings.Contains(u, "Season=") }); i >= 0 {
			t.Errorf("an add's Fetch sent OMDb %s, want no season call", p.sentTo("omdb")[i])
		}
		if !f.RatingsRead || slices.ContainsFunc(f.Ratings, func(r works.Rating) bool { return r.Item != "" }) || !slices.ContainsFunc(f.Ratings, func(r works.Rating) bool { return r.Source == "imdb" }) {
			t.Errorf("an add's ratings = %+v, want the series' IMDb rating and no episode's", f.Ratings)
		}
	})
}

func TestARatingsDueRefreshReadsSeasonsLatestFirst(t *testing.T) {
	serve(t, "series", func(t *testing.T, p *providers) {
		f, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsAll), "1001", nil)
		if err != nil {
			t.Fatal(err)
		}
		rated := map[works.Key]float64{}
		for _, r := range f.Ratings {
			if r.Item != "" {
				rated[r.Item] = r.Value
			}
		}
		if want := map[works.Key]float64{"100111": 8.0, "100112": 7.5, "100133": 9.1}; !maps.Equal(rated, want) {
			t.Errorf("a ratings-due refresh rated episodes %v, want %v by TVDB episode", rated, want)
		}
		var seasons []string
		for _, u := range p.sentTo("omdb") {
			if _, n, ok := strings.Cut(u, "Season="); ok {
				n, _, _ = strings.Cut(n, "&")
				seasons = append(seasons, n)
			}
		}
		if !slices.Equal(seasons, []string{"3", "2", "1"}) {
			t.Errorf("OMDb was asked for seasons %v, want the latest first and no specials", seasons)
		}
	})
}

func TestChangesSumsBothListsAndAliasesTMDB(t *testing.T) {
	serve(t, "series", func(t *testing.T, p *providers) {
		since := time.Now().Add(-time.Hour)
		p.tvdb.Cue(t, faketvdb.Change.Name, faketvdb.ID{ID: 1001})
		p.tvdb.Cue(t, faketvdb.Change.Name, faketvdb.ID{ID: 1002})
		p.tmdb.Cue(t, faketmdb.SeriesChange.Name, faketmdb.Series{ID: 7001})
		read := func(since time.Time) (pages int, keys []works.Key) {
			for page := 1; ; page++ {
				c, err := p.source.Changes(t.Context(), askFor(metadata.RatingsNone), since, time.Now(), page)
				if err != nil {
					t.Fatal(err)
				}
				keys = append(keys, c.Keys...)
				if page >= c.Pages {
					return c.Pages, keys
				}
			}
		}
		if pages, keys := read(since); pages != 2 || !slices.Equal(keys, []works.Key{"1001", "1002", "tmdb-7001"}) {
			t.Errorf("Changes = %v over %d pages, want TVDB's two then TMDB's one as tmdb-7001 over 2", keys, pages)
		}
		if pages, keys := read(time.Now().Add(-tmdb.ChangesWindow - time.Hour)); pages != 1 || !slices.Equal(keys, []works.Key{"1001", "1002"}) {
			t.Errorf("Changes from past TMDB's window = %v over %d pages, want TVDB's alone", keys, pages)
		}
	})
}

func TestDetailOf10000EpisodesFitsTheCap(t *testing.T) {
	serve(t, "series-scale", func(t *testing.T, p *providers) {
		f, err := p.source.Fetch(t.Context(), askFor(metadata.RatingsNone), "60000002", nil)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := works.EncodeFetched(f)
		if err != nil {
			t.Fatal(err)
		}
		if items := len(itemsOf(f)); items != 10_000 || len(wire) >= link.MaxMessage/4 {
			t.Errorf("Fetch = %d episodes in %d bytes, want 10,000 well inside the %d-byte link message", items, len(wire), link.MaxMessage)
		}
	})
}
