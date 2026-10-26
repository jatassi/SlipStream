package movie_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"math/rand/v2"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakeomdb "github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketmdb "github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func ptr[T any](v T) *T { return &v }

func id(t *testing.T, k works.Key) int64 {
	t.Helper()
	n, err := strconv.ParseInt(string(k), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

const (
	saltRoad        works.Key = "9000005"
	heat1986        works.Key = "9100001"
	heat1995        works.Key = "9100002"
	heatMaking      works.Key = "9100003"
	bladeRunner2049 works.Key = "9100005"
	war1917         works.Key = "9100006"
	summer1993      works.Key = "9100007"
)

func catalog(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	cast := make([]scenario.TMDBCast, 22)
	for i := range cast {
		cast[i] = scenario.TMDBCast{ID: int64(100 + i), Name: "Player " + string(rune('A'+i)), Character: "Walker " + string(rune('A'+i))}
	}
	s.TMDB.Movies = append(s.TMDB.Movies,
		scenario.TMDBMovie{ID: 9000002, IMDbID: "tt9000002", Title: "Orchard Road", Released: "2019-01-05"},
		scenario.TMDBMovie{ID: 9000004, IMDbID: "tt9000004", Title: "Untitled Project", Runtime: 90 * time.Minute, Status: "Planned"},
		scenario.TMDBMovie{
			ID: 9000005, IMDbID: "tt9000005", Title: "The Salt Road", OriginalTitle: "La Route du Sel", OriginalLanguage: "fr",
			Runtime: 112 * time.Minute, Overview: "Two sisters walk the salt road.", Tagline: "Every grain counts.",
			Genres: []string{"Drama", "Adventure"}, Status: "Released",
			Dates: []scenario.TMDBDate{
				{Country: "FR", Kind: "cinema", Date: "2022-09-14", Certification: "TP"},
				{Country: "US", Kind: "limited", Date: "2023-01-20", Certification: "PG-13"},
				{Country: "US", Kind: "digital", Date: "2023-04-04", Certification: "R", Note: "Director's cut"},
				{Country: "GB", Kind: "digital", Date: "2023-03-01"},
			},
			Titles:       []scenario.TMDBTitle{{Country: "US", Title: "Salt Road"}, {Country: "GB", Title: "The Salt Way", Kind: "working title"}},
			Translations: []scenario.TMDBTranslation{{Language: "fr-FR", Title: "La Route du Sel", Overview: "Deux sœurs marchent."}},
			Credits: scenario.TMDBCredits{
				Cast: cast,
				Crew: []scenario.TMDBCrew{
					{ID: 10, Name: "Ines Varga", Department: "Directing", Job: "Director"},
					{ID: 11, Name: "Ola Brandt", Department: "Writing", Job: "Screenplay"},
					{ID: 12, Name: "Tomas Reyes", Department: "Production", Job: "Producer"},
				},
			},
			Trailer: "sa1tR0adTr", Collection: &scenario.TMDBNamed{ID: 77, Name: "Salt Trilogy"}, Studio: &scenario.TMDBNamed{ID: 88, Name: "Fennel Films"},
			Images: []scenario.TMDBImage{
				{Shape: "poster", Language: "en", Votes: 5}, {Shape: "poster", Language: "fr", Votes: 9},
				{Shape: "backdrop", Language: "en", Votes: 9}, {Shape: "backdrop", Votes: 4},
				{Shape: "logo", Language: "fr", Votes: 5}, {Shape: "logo", Language: "en", Votes: 4}, {Shape: "logo", Language: "en", Votes: 9},
				{Shape: "logo", Votes: 7},
			},
			VoteAverage: 7.2, VoteCount: 410,
		},
		scenario.TMDBMovie{
			ID: 9000006, IMDbID: "tt9000006", Title: "Quiet Harbor", Runtime: 95 * time.Minute,
			Dates:  []scenario.TMDBDate{{Country: "GB", Kind: "digital", Date: "2020-01-01"}, {Country: "US", Kind: "cinema", Date: "2019-11-01"}},
			Images: []scenario.TMDBImage{{Shape: "logo", Language: "fr", Votes: 9}, {Shape: "logo", Votes: 3}},
		},
		scenario.TMDBMovie{ID: 9000007, Title: "Harbor Lights", Dates: []scenario.TMDBDate{
			{Country: "US", Kind: "cinema", Date: "2020-03-01"}, {Country: "FR", Kind: "limited", Date: "2020-01-01"},
		}},
		scenario.TMDBMovie{ID: 9000008, Title: "Paper Moon", Dates: []scenario.TMDBDate{{Country: "FR", Kind: "cinema", Date: "2020-01-10"}}},
		scenario.TMDBMovie{ID: 9000009, Title: "Glass Garden", Dates: []scenario.TMDBDate{{Country: "US", Kind: "premiere", Date: "2021-06-01"}}},
		scenario.TMDBMovie{ID: 9000010, Title: "Early Stream", Dates: []scenario.TMDBDate{
			{Country: "US", Kind: "digital", Date: "2020-01-01"}, {Country: "US", Kind: "cinema", Date: "2020-02-01"},
		}},
	)
	for _, m := range []struct {
		key             works.Key
		title, released string
	}{
		{heat1986, "Heat", "1986-03-14"}, {heat1995, "Heat", "1995-12-15"}, {heatMaking, "Heat 1995: The Making Of", "2005-06-01"},
		{"9100004", "Blade Runner", "1982-06-25"}, {bladeRunner2049, "Blade Runner 2049", "2017-10-06"},
		{war1917, "1917", "2019-12-25"}, {summer1993, "Summer 1993", "1993-09-01"},
	} {
		s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: id(t, m.key), IMDbID: "tt" + string(m.key), Title: m.title, Released: m.released})
	}
	s.OMDb = &scenario.OMDb{Titles: []scenario.OMDbTitle{{
		IMDbID: "tt9000005", IMDbRating: ptr(7.1), IMDbVotes: ptr(int64(12345)), RottenTomatoes: ptr(88), Metacritic: ptr(74),
		Awards: "1 win & 3 nominations",
	}}}
	return s
}

// hosts sends each request through the transport of the fake at its host.
type hosts map[string]http.RoundTripper

func (h hosts) RoundTrip(r *http.Request) (*http.Response, error) {
	return h[r.URL.Host].RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

// byPath sends TMDB's requests, all under /3, through tmdb, and every
// other through omdb: in memory both fakes share one address.
type byPath struct{ tmdb, omdb http.RoundTripper }

func (b byPath) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.HasPrefix(r.URL.Path, "/3/") {
		return b.tmdb.RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
	}
	return b.omdb.RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

// counted counts the requests a transport sends.
type counted struct {
	next http.RoundTripper
	sent atomic.Int64
}

func (c *counted) RoundTrip(r *http.Request) (*http.Response, error) {
	c.sent.Add(1)
	return c.next.RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

// sources is fake-tmdb and fake-omdb on loopback, read through one Node's
// call layer.
type sources struct {
	tmdb, omdb         faketest.Served
	tmdbSent, omdbSent *counted
	client             *http.Client
}

func serve(t *testing.T, s scenario.Scenario) sources {
	t.Helper()
	sv := sources{tmdb: faketest.Loopback(t, faketmdb.Fake, s), omdb: faketest.Loopback(t, fakeomdb.Fake, s)}
	sv.tmdbSent, sv.omdbSent = &counted{next: sv.tmdb.Client.Transport}, &counted{next: sv.omdb.Client.Transport}
	sv.client = &http.Client{Transport: hosts{sv.tmdb.Address.AddrPort.String(): sv.tmdbSent, sv.omdb.Address.AddrPort.String(): sv.omdbSent}}
	return sv
}

// serveInMemory is serve inside a synctest bubble.
func serveInMemory(t *testing.T, s scenario.Scenario) sources {
	t.Helper()
	sv := sources{tmdb: faketest.InMemory(t, faketmdb.Fake, s), omdb: faketest.InMemory(t, fakeomdb.Fake, s)}
	sv.tmdbSent, sv.omdbSent = &counted{next: sv.tmdb.Client.Transport}, &counted{next: sv.omdb.Client.Transport}
	sv.client = &http.Client{Transport: byPath{tmdb: sv.tmdbSent, omdb: sv.omdbSent}}
	return sv
}

func (sv sources) open() metadata.Source {
	return provider(sv.client, sv.tmdb.Address.URL("/3"), sv.omdb.Address.URL(""))
}

func provider(c *http.Client, tmdbAPI, omdbAPI string) metadata.Source {
	lines := metadata.NewLines(c, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	return movie.Def.OpenProvider(contenttype.ProviderDeps{
		TMDB: tmdb.New(lines, tmdb.Endpoints{API: endpoint.MustParseURL(tmdbAPI)}, tmdb.DefaultTimings()),
		OMDb: omdb.New(lines, omdb.Endpoints{API: endpoint.MustParseURL(omdbAPI)}, omdb.DefaultTimings()),
		Log:  slog.New(slog.DiscardHandler),
	}, nil)
}

// tmdbOf is TMDB's own answer for id, as the movie reads it.
func (sv sources) tmdbOf(t *testing.T, ask metadata.Ask, id int64) tmdb.Movie {
	t.Helper()
	lines := metadata.NewLines(sv.tmdb.Client, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	m, err := tmdb.New(lines, tmdb.Endpoints{API: endpoint.MustParseURL(sv.tmdb.Address.URL("/3"))}, tmdb.DefaultTimings()).Movie(t.Context(), ask, id)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func library() contenttype.Library {
	return movie.Def.OpenLibrary(contenttype.LibraryDeps{Region: metadata.Region, Log: slog.New(slog.DiscardHandler)}, movie.Def.Endpoints())
}

func ask(language, region string) metadata.Ask {
	return metadata.NewAsk(metadata.AskOptions{Language: language, Region: region, Ratings: true})
}

var inEnglish = ask(metadata.DefaultLanguage, metadata.DefaultRegion)

func inEachTier(t *testing.T, test func(t *testing.T, m metadata.Source)) {
	keyless := func(f faketest.Served) metadata.Source {
		return provider(f.Client, f.Address.URL("/3"), "https://omdb.invalid")
	}
	t.Run("loopback", func(t *testing.T) { test(t, keyless(faketest.Loopback(t, faketmdb.Fake, catalog(t)))) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, keyless(faketest.InMemory(t, faketmdb.Fake, catalog(t)))) })
	})
}

func keys(cs []works.Candidate) []works.Key {
	out := []works.Key{}
	for _, c := range cs {
		out = append(out, c.Key)
	}
	return out
}

func TestTrailingYear(t *testing.T) {
	inEachTier(t, func(t *testing.T, m metadata.Source) {
		for query, want := range map[string][]works.Key{
			"Heat 1995":         {heat1995, heatMaking},
			"heat":              {heat1986, heat1995, heatMaking},
			"Blade Runner 2049": {bladeRunner2049},
			"1917":              {war1917},
			"Summer 1993":       {summer1993},
			"nobody":            {},
		} {
			got, err := m.Search(t.Context(), inEnglish, query)
			if err != nil || !reflect.DeepEqual(keys(got), want) {
				t.Errorf("Search(%q) = %v, %v; want %v", query, keys(got), err, want)
			}
		}
	})
}

func TestOnlyAYearUpToTenYearsAheadIsReadAsAFilter(t *testing.T) {
	sv := serve(t, catalog(t))
	for _, c := range []struct {
		query string
		reads int64
	}{{"Heat 1995", 2}, {"Heat " + strconv.Itoa(time.Now().Year()+10), 2}, {"Heat " + strconv.Itoa(time.Now().Year()+11), 1}, {"Heat 1873", 1}, {"1917", 1}} {
		before := sv.tmdbSent.sent.Load()
		if _, err := sv.open().Search(t.Context(), inEnglish, c.query); err != nil {
			t.Fatal(err)
		}
		if sent := sv.tmdbSent.sent.Load() - before; sent != c.reads {
			t.Errorf("Search(%q) sent %d TMDB requests, want %d", c.query, sent, c.reads)
		}
	}
}

func TestSearchAnswersTitleOriginalOverviewYearAndPoster(t *testing.T) {
	sv := serve(t, catalog(t))
	got, err := sv.open().Search(t.Context(), inEnglish, "salt road")
	want := []works.Candidate{{
		Key: saltRoad, Label: "The Salt Road", Original: "La Route du Sel", Overview: "Two sisters walk the salt road.",
		Poster: artwork.Ref{Source: tmdb.ImagesID, Shape: artwork.ShapePoster, Path: sv.tmdbOf(t, inEnglish, id(t, saltRoad)).PosterPath},
		Fields: []works.FieldValue{works.NumberField("year", 2022)},
	}}
	if err != nil || !reflect.DeepEqual(got, want) || want[0].Poster.Path == "" {
		t.Fatalf("Search = %+v, %v; want %+v", got, err, want)
	}
	if sv.omdbSent.sent.Load() != 0 {
		t.Fatalf("search sent %d OMDb requests, want none", sv.omdbSent.sent.Load())
	}
}

func TestSearchReadsPagesUntil25(t *testing.T) {
	for _, c := range []struct {
		name          string
		movies, pages int
	}{{"more than 25", 30, 2}, {"fewer than 25", 22, 2}, {"one page", 7, 1}} {
		t.Run(c.name, func(t *testing.T) {
			s := catalog(t)
			var want []works.Key
			for i := range c.movies {
				id := int64(9200001 + i)
				s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: id, IMDbID: "tt" + strconv.FormatInt(id, 10), Title: "Lantern " + strconv.Itoa(i+1)})
				if len(want) < 25 {
					want = append(want, works.Key(strconv.FormatInt(id, 10)))
				}
			}
			sv := serve(t, s)
			got, err := sv.open().Search(t.Context(), inEnglish, "lantern")
			if err != nil || !reflect.DeepEqual(keys(got), want) {
				t.Fatalf("Search = %v, %v; want %v", keys(got), err, want)
			}
			if sent := sv.tmdbSent.sent.Load(); sent != int64(c.pages) {
				t.Fatalf("search sent %d TMDB requests, want %d", sent, c.pages)
			}
		})
	}
}

func TestFetchReadsTheMovieAsOneItem(t *testing.T) {
	inEachTier(t, func(t *testing.T, m metadata.Source) {
		preview := metadata.NewAsk(metadata.AskOptions{Language: metadata.DefaultLanguage, Region: metadata.DefaultRegion})
		got, err := m.Fetch(t.Context(), preview, "9000001")
		if err != nil || got.Label != "The Quiet Orchard" || !reflect.DeepEqual(got.Children, []works.Entry{{Key: "movie", Label: "The Quiet Orchard"}}) || got.RatingsRead {
			t.Fatalf("Fetch = %+v, %v; want The Quiet Orchard with one Item movie and no ratings read", got, err)
		}
		_, err = m.Fetch(t.Context(), preview, "9000003")
		if want := "GET /movie/9000003: TMDB: has no such record"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
			t.Fatalf("Fetch(9000003) = %v, want %s", err, want)
		}
	})
}

type discard[T any] struct{}

type noPrefetch struct{}

func (noPrefetch) Prefetch([]artwork.Want) {}

func (discard[T]) Publish(context.Context, T) {}

type published struct{ n atomic.Int64 }

func (p *published) Publish(context.Context, works.Changed) { p.n.Add(1) }

// retyped records each works.type-changed by its Content Type.
type retyped struct {
	mu    sync.Mutex
	types []string
}

func (p *retyped) Publish(_ context.Context, e works.TypeChanged) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.types = append(p.types, e.ContentType)
}

func (p *retyped) sent() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.types)
}

type shelf struct {
	works   *works.Works
	m       contenttype.Library
	lib     *storage.DB[storage.Library]
	sent    *published
	retyped *retyped
}

func open(t *testing.T, src works.Source) shelf {
	t.Helper()
	sh := shelf{m: library(), lib: storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(movie.Def))...), sent: &published{}, retyped: &retyped{}}
	var err error
	sh.works, err = works.Open(t.Context(), works.Config{
		Library: sh.lib, Catalogs: map[works.TypeID]works.Catalog{movie.Def.ID(): sh.m}, Timings: works.DefaultTimings(),
		Source: src, Artwork: noPrefetch{}, Changed: sh.sent, TypesChanged: discard[works.ContentTypesChanged]{}, TypeChanged: sh.retyped,
		Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	return sh
}

func (sh shelf) add(t *testing.T, key works.Key) works.Tree {
	t.Helper()
	tree, _, err := sh.works.Add(t.Context(), identity.Admin{}, movie.Def.ID(), key, works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func (sh shelf) refresh(t *testing.T, id works.WorkID) works.Tree {
	t.Helper()
	tree, err := sh.works.Refresh(t.Context(), identity.Admin{}, id)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func (sh shelf) read(t *testing.T, f func(ctx context.Context, r storage.Reader[storage.Library]) error) {
	t.Helper()
	if err := sh.lib.Read(t.Context(), f); err != nil {
		t.Fatal(err)
	}
}

// asked is the Controller's metadata reading in whichever language and
// region a test sets, from whichever provider half it points at.
type asked struct {
	src metadata.Source
	ask metadata.Ask
}

func (a *asked) Search(ctx context.Context, _ works.TypeID, query string) ([]works.Candidate, error) {
	return a.src.Search(ctx, a.ask, query)
}

func (a *asked) Fetch(ctx context.Context, _ works.TypeID, key works.Key, _ works.Fetching) (works.Fetched, error) {
	return a.src.Fetch(ctx, a.ask, key)
}

func added(t *testing.T, key works.Key) (shelf, works.Tree) {
	t.Helper()
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): serve(t, catalog(t)).open()}))
	return sh, sh.add(t, key)
}

func field(tree works.Tree, key string) (works.FieldValue, bool) {
	for _, f := range tree.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return works.FieldValue{}, false
}

func TestAddStoresTheFullRecord(t *testing.T) {
	sv := serve(t, catalog(t))
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): sv.open()}))
	tree := sh.add(t, saltRoad)
	if tmdbSent, omdbSent := sv.tmdbSent.sent.Load(), sv.omdbSent.sent.Load(); tmdbSent != 1 || omdbSent != 1 {
		t.Fatalf("the add sent TMDB %d requests and OMDb %d, want one each", tmdbSent, omdbSent)
	}
	tm := sv.tmdbOf(t, inEnglish, 9000005)

	photo := func(id int64) artwork.Ref { return photoOf(t, tm, id) }
	people := []works.Person{
		{Name: "Ines Varga", Role: "Director", Department: "Directing", Photo: photo(10)},
		{Name: "Ola Brandt", Role: "Screenplay", Department: "Writing", Photo: photo(11)},
	}
	for i := range 20 {
		people = append(people, works.Person{
			Name: "Player " + string(rune('A'+i)), Role: "Walker " + string(rune('A'+i)), Department: "Acting", Photo: photo(int64(100 + i)),
		})
	}
	want := []works.FieldValue{
		works.TextField("english_title", "The Salt Road"),
		works.TextField("original_title", "La Route du Sel"),
		works.TextField("original_language", "fr"),
		works.NumberField("year", 2022),
		works.DateField("release_date", works.Date{Year: 2022, Month: 9, Day: 14}),
		works.DurationField("runtime", 112*time.Minute),
		works.TextField("status", "Released"),
		works.TextField("certification", "PG-13"),
		works.TextField("certification_country", "US"),
		works.TextsField("genres", []string{"Drama", "Adventure"}),
		works.TextField("tagline", "Every grain counts."),
		works.TextField("overview", "Two sisters walk the salt road."),
		works.TextField("studio", "Fennel Films"),
		works.TextField("collection", "Salt Trilogy"),
		works.PeopleField("credits", people),
		works.TextsField("alternative_titles", []string{"Salt Road"}),
		works.TextField("imdb_id", "tt9000005"),
		works.TextField("trailer_site", "YouTube"),
		works.TextField("trailer_key", "sa1tR0adTr"),
		works.TextField("awards", "1 win & 3 nominations"),
	}
	if tree.Label != "The Salt Road" || !reflect.DeepEqual(tree.Fields, want) {
		t.Fatalf("label %q fields\n%+v\nwant The Salt Road\n%+v", tree.Label, tree.Fields, want)
	}

	type rating struct {
		Source works.RatingSource
		Value  float64
		Votes  *int64
	}
	var ratings []rating
	for _, r := range tree.Ratings {
		ratings = append(ratings, rating{r.Source, r.Value, r.Votes})
	}
	wantRatings := []rating{{"imdb", 7.1, ptr(int64(12345))}, {"metacritic", 74, nil}, {"rottentomatoes", 88, nil}, {"tmdb", 7.2, ptr(int64(410))}}
	if !reflect.DeepEqual(ratings, wantRatings) || tree.RatingsReadAt.IsZero() {
		t.Fatalf("ratings %+v read at %v, want %+v read now", ratings, tree.RatingsReadAt, wantRatings)
	}

	logo := func(ims []tmdb.Image, language string, votes float64) string { return imageOf(t, ims, language, votes) }
	wantArt := map[string]artwork.Ref{
		"poster":          {Source: tmdb.ImagesID, Shape: artwork.ShapePoster, Path: logo(tm.Images.Posters, "en", 5)},
		"backdrop":        {Source: tmdb.ImagesID, Shape: artwork.ShapeBackdrop, Path: logo(tm.Images.Backdrops, "", 4)},
		"title_treatment": {Source: tmdb.ImagesID, Shape: artwork.ShapeLogo, Path: logo(tm.Images.Logos, "en", 9)},
		"studio_logo":     {Source: tmdb.ImagesID, Shape: artwork.ShapeLogo, Path: tm.ProductionCompanies[0].LogoPath},
	}
	if !reflect.DeepEqual(tree.Artwork, wantArt) {
		t.Fatalf("artwork %+v, want %+v", tree.Artwork, wantArt)
	}
}

// photoOf is the photo TMDB lists for person id.
func photoOf(t *testing.T, m tmdb.Movie, id int64) artwork.Ref {
	t.Helper()
	paths := map[int64]string{}
	for _, c := range m.Credits.Cast {
		paths[c.ID] = c.ProfilePath
	}
	for _, c := range m.Credits.Crew {
		paths[c.ID] = c.ProfilePath
	}
	if paths[id] == "" {
		t.Fatalf("TMDB lists no photo of person %d", id)
	}
	return artwork.Ref{Source: tmdb.ImagesID, Shape: artwork.ShapePhoto, Path: paths[id]}
}

// imageOf is the file of the one image in ims in language with votes.
func imageOf(t *testing.T, ims []tmdb.Image, language string, votes float64) string {
	t.Helper()
	for _, im := range ims {
		if im.Language == language && im.VoteAverage == votes {
			return im.FilePath
		}
	}
	t.Fatalf("TMDB lists no image in %q with %v votes", language, votes)
	return ""
}

func TestPreviewSkipsOMDbAndCommitsNothing(t *testing.T) {
	sv := serve(t, catalog(t))
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): sv.open()}))
	preview, stored, err := sh.works.Preview(t.Context(), identity.Admin{}, movie.Def.ID(), saltRoad, noneWithin)
	if err != nil || stored {
		t.Fatalf("Preview = stored %v, %v; want a record not in the library", stored, err)
	}
	if sv.omdbSent.sent.Load() != 0 || sv.tmdbSent.sent.Load() != 1 {
		t.Fatalf("preview sent %d TMDB and %d OMDb requests, want 1 and none", sv.tmdbSent.sent.Load(), sv.omdbSent.sent.Load())
	}
	var listed []works.Row
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		listed, err = sh.works.RowsIn(ctx, r, works.Scope{})
		return err
	})
	if len(listed) != 0 || sh.sent.n.Load() != 0 {
		t.Fatalf("after a preview the library lists %v and published %d works.changed, want nothing", listed, sh.sent.n.Load())
	}
	if len(preview.Ratings) != 1 || preview.Ratings[0].Source != "tmdb" || !preview.RatingsReadAt.IsZero() {
		t.Fatalf("preview ratings %+v read at %v, want only TMDB's and no ratings read", preview.Ratings, preview.RatingsReadAt)
	}

	added := sh.add(t, saltRoad)
	without := func(fs []works.FieldValue) []works.FieldValue {
		return slices.DeleteFunc(slices.Clone(fs), func(f works.FieldValue) bool { return f.Key == "awards" })
	}
	if !reflect.DeepEqual(without(added.Fields), preview.Fields) || !reflect.DeepEqual(added.Artwork, preview.Artwork) ||
		added.Label != preview.Label || len(added.Items) != 1 || len(preview.Items) != 1 || added.Items[0].Key != preview.Items[0].Key {
		t.Fatalf("added %+v\nwant the preview but OMDb's awards %+v", added, preview)
	}
	again, stored, err := sh.works.Preview(t.Context(), identity.Admin{}, movie.Def.ID(), saltRoad, noneWithin)
	if err != nil || !stored || again.ID != added.ID || !reflect.DeepEqual(again.Fields, added.Fields) {
		t.Fatalf("Preview once added = %v %v, %v; want the stored Work %v", again.ID, stored, err, added.ID)
	}
}

func TestAddAfterPreviewMakesNoSecondTMDBCall(t *testing.T) {
	sv := serve(t, catalog(t))
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): sv.open()}))
	if _, _, err := sh.works.Preview(t.Context(), identity.Admin{}, movie.Def.ID(), saltRoad, noneWithin); err != nil {
		t.Fatal(err)
	}
	sh.add(t, saltRoad)
	if got := sv.tmdbSent.sent.Load(); got != 1 {
		t.Fatalf("a preview then an add sent %d TMDB requests, want 1: the add reuses the preview's answer", got)
	}
	if got := sv.omdbSent.sent.Load(); got != 1 {
		t.Fatalf("the add sent %d OMDb requests, want 1", got)
	}
}

func TestAddedMovieHasOneMonitoredItem(t *testing.T) {
	_, tree := added(t, "9000001")
	if tree.Label != "The Quiet Orchard" || tree.Key != "9000001" {
		t.Fatalf("tree %q key %s, want The Quiet Orchard, 9000001", tree.Label, tree.Key)
	}
	if len(tree.Items) != 1 || tree.Items[0].Key != "movie" || tree.Items[0].Label != "The Quiet Orchard" || !tree.Items[0].Monitored || len(tree.Levels) != 0 {
		t.Fatalf("levels %+v items %+v, want no level and one monitored Item movie", tree.Levels, tree.Items)
	}
}

func TestAMovieOMDbHasNoRecordOfIsReadWithNoRatingsFromIt(t *testing.T) {
	_, tree := added(t, "9000002")
	if _, ok := field(tree, "runtime"); ok {
		t.Errorf("fields %+v carry a runtime TMDB doesn't list", tree.Fields)
	}
	if len(tree.Ratings) != 0 || tree.RatingsReadAt.IsZero() {
		t.Errorf("ratings %+v read at %v, want none, read now", tree.Ratings, tree.RatingsReadAt)
	}
}

func TestTheEnglishTitleFeedsSearchTermsAndNaming(t *testing.T) {
	src := &asked{src: serve(t, catalog(t)).open(), ask: ask("fr-FR", "FR")}
	sh := open(t, src)
	tree := sh.add(t, saltRoad)
	if tree.Label != "La Route du Sel" {
		t.Fatalf("label %q, want the title in French, La Route du Sel", tree.Label)
	}
	var terms indexers.Terms
	var tokens map[string]string
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if terms, err = sh.m.SearchTerms(ctx, r, tree.Items[0].ID); err != nil {
			return err
		}
		tokens, err = sh.m.NamingTokens(ctx, r, tree.Items[0].ID)
		return err
	})
	want := indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": "9000005", "imdbid": "tt9000005"}, Query: "The Salt Road 2022"}
	if !reflect.DeepEqual(terms, want) {
		t.Fatalf("terms %+v, want %+v", terms, want)
	}
	templates := sh.m.DefaultTemplates()
	folder, err := naming.Render(templates.Folder, tokens, naming.Core{Target: "1080p"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := naming.Render(templates.File, tokens, naming.Core{Target: "1080p"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := folder+"/"+file, "The Salt Road (2022)/The Salt Road (2022) - 1080p"; got != want {
		t.Fatalf("named %q, want %q", got, want)
	}
}

func TestLanguageChangesOnlyTheLabel(t *testing.T) {
	src := &asked{src: serve(t, catalog(t)).open(), ask: inEnglish}
	sh := open(t, src)
	before := sh.add(t, saltRoad)
	src.ask = ask("fr-FR", metadata.DefaultRegion)
	after := sh.refresh(t, before.ID)
	if before.Label != "The Salt Road" || after.Label != "La Route du Sel" || after.Items[0].Label != "La Route du Sel" {
		t.Fatalf("labels %q then %q (Item %q), want The Salt Road then La Route du Sel", before.Label, after.Label, after.Items[0].Label)
	}
	english, _ := field(after, "english_title")
	overview, _ := field(after, "overview")
	if english.Text != "The Salt Road" || overview.Text != "Deux sœurs marchent." {
		t.Fatalf("English title %q overview %q, want The Salt Road and the French overview", english.Text, overview.Text)
	}
	var terms indexers.Terms
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		terms, err = sh.m.SearchTerms(ctx, r, after.Items[0].ID)
		return err
	})
	if terms.Query != "The Salt Road 2022" {
		t.Fatalf("query %q after the language changed, want The Salt Road 2022", terms.Query)
	}
}

func TestCertificationNamesItsCountry(t *testing.T) {
	sv := serve(t, catalog(t))
	for region, want := range map[string]string{"US": "PG-13 in US", "FR": "TP in FR", "DE": "none"} {
		t.Run(region, func(t *testing.T) {
			tree := open(t, &asked{src: sv.open(), ask: ask(metadata.DefaultLanguage, region)}).add(t, saltRoad)
			got := "none"
			if code, ok := field(tree, "certification"); ok {
				country, _ := field(tree, "certification_country")
				got = code.Text + " in " + country.Text
			}
			if got != want {
				t.Fatalf("certification %s, want %s", got, want)
			}
		})
	}
}

func TestARefreshReadsOMDbNoMoreOftenThanTheRatingsSetting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sv := serveInMemory(t, catalog(t))
		sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): sv.open()}))
		tree := sh.add(t, saltRoad)
		time.Sleep(metadata.RatingsEvery.Default - time.Hour)
		sh.refresh(t, tree.ID)
		if sent := sv.omdbSent.sent.Load(); sent != 1 {
			t.Fatalf("OMDb was sent %d requests by an add and a refresh inside the ratings setting, want the add's", sent)
		}
		time.Sleep(time.Hour)
		if after := sh.refresh(t, tree.ID); after.RatingsReadAt.Compare(tree.RatingsReadAt) <= 0 {
			t.Fatalf("ratings read at %v after the setting's time passed, want later than the add's %v", after.RatingsReadAt, tree.RatingsReadAt)
		}
		if sent := sv.omdbSent.sent.Load(); sent != 2 {
			t.Fatalf("OMDb was sent %d requests, want the add's and the refresh's once the setting's time passed", sent)
		}
	})
}

func TestAnOMDbCapKeepsLastRatingsAndAwards(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sv := serveInMemory(t, catalog(t))
		sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): sv.open()}))
		before := sh.add(t, saltRoad)
		time.Sleep(metadata.RatingsEvery.Default)
		sv.omdb.Cue(t, fakeomdb.Cap.Name, fakeomdb.Key{})
		after := sh.refresh(t, before.ID)
		if sent := sv.omdbSent.sent.Load(); sent != 2 {
			t.Fatalf("OMDb was sent %d requests, want the add's and the refresh's", sent)
		}
		omdbRatings := func(tree works.Tree) []works.StoredRating {
			var out []works.StoredRating
			for _, r := range tree.Ratings {
				if r.Source != "tmdb" {
					out = append(out, r)
				}
			}
			return out
		}
		awards, _ := field(after, "awards")
		if len(omdbRatings(before)) != 3 || !reflect.DeepEqual(omdbRatings(after), omdbRatings(before)) || after.RatingsReadAt != before.RatingsReadAt {
			t.Fatalf("OMDb ratings %+v read at %v after the cap, want %+v read at %v", omdbRatings(after), after.RatingsReadAt, omdbRatings(before), before.RatingsReadAt)
		}
		if awards.Text != "1 win & 3 nominations" {
			t.Fatalf("awards %q after the cap, want the stored line", awards.Text)
		}
	})
}

func TestARefreshPublishesOnlyARealChange(t *testing.T) {
	sv := serve(t, catalog(t))
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): sv.open()}))
	tree := sh.add(t, saltRoad)
	sh.refresh(t, tree.ID)
	if n := sh.sent.n.Load(); n != 1 {
		t.Fatalf("works.changed sent %d times by an add and a refresh that changed nothing, want 1", n)
	}
	sv.tmdb.Cue(t, faketmdb.Change.Name, faketmdb.Movie{ID: 9000005})
	after := sh.refresh(t, tree.ID)
	overview, _ := field(after, "overview")
	if n := sh.sent.n.Load(); n != 2 || overview.Text != "Two sisters walk the salt road. Revision 1." {
		t.Fatalf("works.changed sent %d times, overview %q; want 2 and the revised overview", n, overview.Text)
	}
}

func TestARefreshReplacesTheCredits(t *testing.T) {
	recast := catalog(t)
	for i, m := range recast.TMDB.Movies {
		if m.ID == 9000005 {
			recast.TMDB.Movies[i].Credits = scenario.TMDBCredits{Cast: []scenario.TMDBCast{{ID: 500, Name: "Nell Ash", Character: "Walker"}}}
			recast.TMDB.Movies[i].Genres = []string{"Mystery"}
		}
	}
	src := &asked{src: serve(t, catalog(t)).open(), ask: inEnglish}
	sh := open(t, src)
	tree := sh.add(t, saltRoad)
	src.src = serve(t, recast).open()
	after := sh.refresh(t, tree.ID)
	credits, _ := field(after, "credits")
	genres, _ := field(after, "genres")
	if len(credits.People) != 1 || credits.People[0].Name != "Nell Ash" || !reflect.DeepEqual(genres.Texts, []string{"Mystery"}) {
		t.Fatalf("credits %+v genres %v, want only Nell Ash and Mystery", credits.People, genres.Texts)
	}
}

func TestATitleTreatmentPrefersLanguageThenEnglishThenAny(t *testing.T) {
	sv := serve(t, catalog(t))
	m := sv.open()
	for _, c := range []struct {
		key      works.Key
		language string
		want     func(tmdb.Image) bool
	}{
		{saltRoad, "fr-FR", func(im tmdb.Image) bool { return im.Language == "fr" }},
		{saltRoad, "en-US", func(im tmdb.Image) bool { return im.Language == "en" && im.VoteAverage == 9 }},
		{saltRoad, "de-DE", func(im tmdb.Image) bool { return im.Language == "en" && im.VoteAverage == 9 }},
		{"9000006", "de-DE", func(im tmdb.Image) bool { return im.Language == "" }},
	} {
		a := metadata.NewAsk(metadata.AskOptions{Language: c.language, Region: metadata.DefaultRegion})
		f, err := m.Fetch(t.Context(), a, c.key)
		if err != nil {
			t.Fatal(err)
		}
		var got artwork.Ref
		for _, art := range f.Artwork {
			if art.Kind == "title_treatment" {
				got = art.Ref
			}
		}
		var want string
		for _, im := range sv.tmdbOf(t, a, id(t, c.key)).Images.Logos {
			if c.want(im) {
				want = im.FilePath
			}
		}
		if got != (artwork.Ref{Source: tmdb.ImagesID, Shape: artwork.ShapeLogo, Path: want}) || want == "" {
			t.Errorf("movie %s in %s: title treatment %+v, want %s", c.key, c.language, got, want)
		}
	}
}

func TestNamingTokensNameTheMovie(t *testing.T) {
	sh, tree := added(t, "9000001")
	var tokens map[string]string
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		tokens, err = sh.m.NamingTokens(ctx, r, tree.Items[0].ID)
		return err
	})
	templates := sh.m.DefaultTemplates()
	folder, err := naming.Render(templates.Folder, tokens, naming.Core{Target: "1080p"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := naming.Render(templates.File, tokens, naming.Core{Target: "1080p"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := folder+"/"+file, "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p"; got != want {
		t.Fatalf("named %q, want %q", got, want)
	}
}

func TestSearchTermsCarryTheIDsAndTitleYear(t *testing.T) {
	sh, tree := added(t, "9000001")
	var terms indexers.Terms
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		terms, err = sh.m.SearchTerms(ctx, r, tree.Items[0].ID)
		return err
	})
	want := indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"}, Query: "The Quiet Orchard 2021"}
	if !reflect.DeepEqual(terms, want) {
		t.Fatalf("terms %+v, want %+v", terms, want)
	}
}

func timings(t *testing.T, key works.Key) (tree works.Tree, got map[works.ItemID]targets.Timing) {
	t.Helper()
	sh, tree := added(t, key)
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = sh.m.Timings(ctx, r, works.Only([]works.WorkID{tree.ID}))
		return err
	})
	return tree, got
}

func TestTimingsCarryTheRuntime(t *testing.T) {
	tree, got := timings(t, saltRoad)
	if r, _ := got[tree.Items[0].ID].Runtime.Get(); r != 112*time.Minute {
		t.Fatalf("Timings = %+v, want a runtime of 112 min", got)
	}
}

func TestAMovieWithoutRuntimeHasAnUnknownRuntime(t *testing.T) {
	tree, got := timings(t, "9000002")
	if _, known := got[tree.Items[0].ID].Runtime.Get(); known {
		t.Fatalf("Timings = %+v, want an unknown runtime", got)
	}
}

type availability struct {
	date  string
	basis string
}

func TestAvailabilityRule(t *testing.T) {
	keys := []works.Key{"9000002", "9000004", saltRoad, "9000006", "9000007", "9000008", "9000009", "9000010"}
	home := map[works.Key]availability{
		"9000002": {"2019-01-05", "Digital release in United States"},
		saltRoad:  {"2023-04-04", "Digital release in United States"},
		"9000006": {"2020-01-01", "Digital release in United Kingdom"},
		"9000007": {"2020-05-30", "Cinema release in United States plus 90 days"},
		"9000008": {"2020-04-09", "Cinema release in France plus 90 days"},
		"9000009": {"2021-08-30", "Release date plus 90 days"},
		"9000010": {"2020-01-01", "Digital release in United States"},
	}
	cases := []struct {
		name  string
		patch setting.Patch
		want  map[works.Key]availability
	}{
		{"home release by default", nil, home},
		{"cinema release, the earlier of cinema and home", setting.Patch{movie.CountsAs.Key: json.RawMessage(`"cinema"`)}, map[works.Key]availability{
			"9000002": home["9000002"],
			saltRoad:  {"2023-01-20", "Limited cinema release in United States"},
			"9000006": {"2019-11-01", "Cinema release in United States"},
			"9000007": {"2020-03-01", "Cinema release in United States"},
			"9000008": {"2020-01-10", "Cinema release in France"},
			"9000009": home["9000009"],
			"9000010": home["9000010"],
		}},
		{"a 30-day cinema delay", setting.Patch{movie.CinemaDelay.Key: json.RawMessage(`2592000`)}, map[works.Key]availability{
			"9000002": home["9000002"],
			saltRoad:  home[saltRoad],
			"9000006": home["9000006"],
			"9000007": {"2020-03-31", "Cinema release in United States plus 30 days"},
			"9000008": {"2020-02-09", "Cinema release in France plus 30 days"},
			"9000009": {"2021-07-01", "Release date plus 30 days"},
			"9000010": home["9000010"],
		}},
		{"no cinema delay", setting.Patch{movie.CinemaDelay.Key: json.RawMessage(`0`)}, map[works.Key]availability{
			"9000002": home["9000002"],
			saltRoad:  home[saltRoad],
			"9000006": home["9000006"],
			"9000007": {"2020-03-01", "Cinema release in United States"},
			"9000008": {"2020-01-10", "Cinema release in France"},
			"9000009": {"2021-06-01", "Release date"},
			"9000010": home["9000010"],
		}},
	}
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): serve(t, catalog(t)).open()}))
	added := map[works.WorkID]works.Key{}
	for _, k := range keys {
		added[sh.add(t, k).ID] = k
	}
	reset := setting.Patch{movie.CountsAs.Key: json.RawMessage(`null`), movie.CinemaDelay.Key: json.RawMessage(`null`)}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sh.set(t, reset)
			sh.set(t, c.patch)
			ids := slices.Collect(maps.Keys(added))
			for _, ws := range []works.Among[works.WorkID]{works.Only(ids), works.Every(ids)} {
				got := sh.availabilities(t, ws, added)
				if !reflect.DeepEqual(got, c.want) {
					t.Fatalf("availability of every movie (%t) = %v, want %v (9000004 has no date)", ws.Every(), got, c.want)
				}
			}
		})
	}
}

func (sh shelf) availabilities(t *testing.T, ws works.Among[works.WorkID], added map[works.WorkID]works.Key) map[works.Key]availability {
	t.Helper()
	got := map[works.Key]availability{}
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		items, err := works.ItemsOf(ctx, r, ws)
		if err != nil {
			return err
		}
		timings, err := sh.m.Timings(ctx, r, ws)
		if err != nil {
			return err
		}
		for w, its := range items {
			if d, known := timings[its[0].ID].Released.Get(); known {
				got[added[w]] = availability{d.String(), timings[its[0].ID].Basis}
			}
		}
		return nil
	})
	return got
}

func (sh shelf) set(t *testing.T, p setting.Patch) []setting.Field {
	t.Helper()
	fields, err := sh.works.SetSettings(t.Context(), identity.Admin{}, movie.Def.ID(), p)
	if err != nil {
		t.Fatal(err)
	}
	return fields
}

func TestATypeSettingChangePublishesOneTypeChanged(t *testing.T) {
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): serve(t, catalog(t)).open()}))
	cinema := setting.Patch{movie.CountsAs.Key: json.RawMessage(`"cinema"`), movie.CinemaDelay.Key: json.RawMessage(`604800`)}
	fields := sh.set(t, cinema)
	want := []setting.Field{
		setting.Switch{Key: "movie.add.monitor", Label: "New movies are monitored", Default: true}.Field(nil),
		movie.CountsAs.Field(ptr(movie.CountsAs.Options[1].Value)),
		movie.CinemaDelay.Field(ptr(int64(604800))),
	}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("fields = %+v, want %+v", fields, want)
	}
	sh.set(t, cinema)
	if got := sh.retyped.sent(); !reflect.DeepEqual(got, []string{"movie"}) {
		t.Fatalf("works.type-changed sent for %v by a change and a repeat, want once for movie", got)
	}
	_, err := sh.works.SetSettings(t.Context(), identity.Admin{}, movie.Def.ID(), setting.Patch{
		movie.CountsAs.Key: json.RawMessage(`null`), movie.CinemaDelay.Key: json.RawMessage(`86400000`),
	})
	if r, ok := errors.AsType[*setting.RangeError](err); !ok || r.Sentence != "Between 0 s and 365 days." {
		t.Fatalf("a delay past a year = %v, want Between 0 s and 365 days.", err)
	}
	_, err = sh.works.SetSettings(t.Context(), identity.Admin{}, movie.Def.ID(), setting.Patch{"movie.region": json.RawMessage(`"GB"`)})
	if u, ok := errors.AsType[*setting.UnknownKeyError](err); !ok || u.Key != "movie.region" {
		t.Fatalf("an unknown key = %v, want an UnknownKeyError for movie.region", err)
	}
	read, err := sh.works.Settings(t.Context(), identity.Admin{}, movie.Def.ID())
	if err != nil || !reflect.DeepEqual(read, want) || len(sh.retyped.sent()) != 1 {
		t.Fatalf("after two refused changes: fields %+v, %v, events %v; want %+v and still one event", read, err, sh.retyped.sent(), want)
	}
}

func TestTheMovieSettingsAreWellFormed(t *testing.T) {
	for _, d := range []interface{ Check() error }{movie.CountsAs, movie.CinemaDelay} {
		if err := d.Check(); err != nil {
			t.Error(err)
		}
	}
}

func TestAMovieHasNoCutsOfItsOwn(t *testing.T) {
	if cuts := library().Cuts(); len(cuts) != 0 {
		t.Fatalf("Cuts = %+v, want none", cuts)
	}
}

func TestTheLibraryAuditPassesWithMovie(t *testing.T) {
	storagetest.AuditLibrary(t, node.LibrarySchemas(contenttype.NewSet(movie.Def)))
}

func noneWithin(context.Context, storage.Reader[storage.Library], works.Tree) error { return nil }
