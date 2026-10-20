package movie_test

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketmdb "github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func catalog(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: 9000002, IMDbID: "tt9000002", Title: "Orchard Road", Released: "2019-01-05"},
		scenario.TMDBMovie{ID: 9000004, Title: "Untitled Project"})
	return s
}

func open(f faketest.Served) metadata.Source {
	return movie.Def.OpenProvider(contenttype.ProviderDeps{
		TMDB: tmdb.New(
			metadata.NewLines(f.Client, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))), //nolint:gosec // a fixed draw the test replays
			tmdb.Endpoints{API: endpoint.MustParseURL(f.Address.URL("/3"))}, tmdb.DefaultTimings()),
		Log: slog.New(slog.DiscardHandler),
	}, nil)
}

func library() contenttype.Library {
	return movie.Def.OpenLibrary(contenttype.LibraryDeps{Region: metadata.Region, Log: slog.New(slog.DiscardHandler)})
}

func inEachTier(t *testing.T, test func(t *testing.T, m metadata.Source)) {
	t.Run("loopback", func(t *testing.T) { test(t, open(faketest.Loopback(t, faketmdb.Fake, catalog(t)))) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, open(faketest.InMemory(t, faketmdb.Fake, catalog(t)))) })
	})
}

func TestSearchAnswersTitleAndYear(t *testing.T) {
	inEachTier(t, func(t *testing.T, m metadata.Source) {
		for query, want := range map[string][]works.Candidate{
			"quiet orchard": {{Key: "9000001", Label: "The Quiet Orchard", Fields: []works.FieldValue{works.NumberField("year", 2021)}}},
			"orchard": {
				{Key: "9000001", Label: "The Quiet Orchard", Fields: []works.FieldValue{works.NumberField("year", 2021)}},
				{Key: "9000002", Label: "Orchard Road", Fields: []works.FieldValue{works.NumberField("year", 2019)}},
			},
			"nobody": {},
		} {
			got, err := m.Search(t.Context(), metadata.Ask{}, query)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("Search(%q) = %+v, %v; want %+v", query, got, err, want)
			}
		}
	})
}

func TestFetchReadsTheMovieAsOneItem(t *testing.T) {
	inEachTier(t, func(t *testing.T, m metadata.Source) {
		got, err := m.Fetch(t.Context(), metadata.Ask{}, "9000001")
		want := works.Fetched{
			Label:    "The Quiet Orchard",
			Children: []works.Entry{{Key: "movie", Label: "The Quiet Orchard"}},
			Detail:   []byte(`{"tmdbId":9000001,"imdbId":"tt9000001","title":"The Quiet Orchard","year":2021,"released":"2021-04-16","runtimeMinutes":104}`),
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Fetch = %+v detail %s, %v; want %+v detail %s", got, got.Detail, err, want, want.Detail)
		}
		_, err = m.Fetch(t.Context(), metadata.Ask{}, "9000003")
		if want := "GET /movie/9000003: TMDB: has no such record"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
			t.Fatalf("Fetch(9000003) = %v, want %s", err, want)
		}
	})
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func added(t *testing.T, key works.Key) (contenttype.Library, *storage.DB[storage.Library], works.Tree) {
	t.Helper()
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(movie.Def))...)
	m := library()
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{movie.Def.ID(): m}, Timings: works.DefaultTimings(),
		Source:  metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): open(faketest.Loopback(t, faketmdb.Fake, catalog(t)))}),
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	tree, _, err := w.Add(t.Context(), identity.Admin{}, movie.Def.ID(), key, works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	return m, lib, tree
}

func TestAddedMovieHasItsFieldsAndOneMonitoredItem(t *testing.T) {
	_, _, tree := added(t, "9000001")
	want := []works.FieldValue{
		works.NumberField("year", 2021),
		works.DateField("released", works.Date{Year: 2021, Month: 4, Day: 16}),
		works.DurationField("runtime", 104*time.Minute),
	}
	if tree.Label != "The Quiet Orchard" || tree.Key != "9000001" || !reflect.DeepEqual(tree.Fields, want) {
		t.Fatalf("tree %q key %s fields %+v, want The Quiet Orchard, 9000001, %+v", tree.Label, tree.Key, tree.Fields, want)
	}
	if len(tree.Items) != 1 || tree.Items[0].Key != "movie" || tree.Items[0].Label != "The Quiet Orchard" || !tree.Items[0].Monitored || len(tree.Levels) != 0 {
		t.Fatalf("levels %+v items %+v, want no level and one monitored Item movie", tree.Levels, tree.Items)
	}
}

func TestAMovieWithoutRuntimeHasNoRuntimeField(t *testing.T) {
	_, _, tree := added(t, "9000002")
	want := []works.FieldValue{works.NumberField("year", 2019), works.DateField("released", works.Date{Year: 2019, Month: 1, Day: 5})}
	if !reflect.DeepEqual(tree.Fields, want) {
		t.Fatalf("fields %+v, want %+v", tree.Fields, want)
	}
}

func TestNamingTokensNameTheMovie(t *testing.T) {
	m, lib, tree := added(t, "9000001")
	var tokens map[string]string
	if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		tokens, err = m.NamingTokens(ctx, r, tree.Items[0].ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	templates := m.DefaultTemplates()
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
	m, lib, tree := added(t, "9000001")
	var terms indexers.Terms
	if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		terms, err = m.SearchTerms(ctx, r, tree.Items[0].ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	want := indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"}, Query: "The Quiet Orchard 2021"}
	if !reflect.DeepEqual(terms, want) {
		t.Fatalf("terms %+v, want %+v", terms, want)
	}
}

func timings(t *testing.T, key works.Key) (tree works.Tree, got map[works.ItemID]targets.Timing) {
	t.Helper()
	m, lib, tree := added(t, key)
	if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = m.Timings(ctx, r, []works.WorkID{tree.ID})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return tree, got
}

func TestTimingsAreTheMoviesReleaseDateAndRuntime(t *testing.T) {
	tree, got := timings(t, "9000001")
	want := map[works.ItemID]targets.Timing{tree.Items[0].ID: {
		Released: release.Known(works.Date{Year: 2021, Month: 4, Day: 16}),
		Runtime:  release.Known(104 * time.Minute),
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want %+v", got, want)
	}
}

func TestAMovieWithoutRuntimeHasAnUnknownRuntime(t *testing.T) {
	tree, got := timings(t, "9000002")
	want := map[works.ItemID]targets.Timing{tree.Items[0].ID: {Released: release.Known(works.Date{Year: 2019, Month: 1, Day: 5})}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want %+v", got, want)
	}
}

func TestAnUndatedMovieHasAnUnknownReleaseDate(t *testing.T) {
	tree, got := timings(t, "9000004")
	if want := (map[works.ItemID]targets.Timing{tree.Items[0].ID: {}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want %+v", got, want)
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
