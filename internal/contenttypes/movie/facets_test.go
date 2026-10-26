package movie_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestListFieldsReadOnlyTheListedFields(t *testing.T) {
	sh, tree := added(t, saltRoad)
	var got map[works.WorkID][]works.FieldValue
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = sh.m.ListFields(ctx, r, []works.WorkID{tree.ID})
		return err
	})
	want := map[works.WorkID][]works.FieldValue{tree.ID: {
		works.NumberField("year", 2022),
		works.DurationField("runtime", 112*time.Minute),
		works.TextField("certification", "PG-13"),
		works.TextsField("genres", []string{"Drama", "Adventure"}),
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListFields = %+v, want %+v", got, want)
	}
}

func TestTitlesAreEveryTitleOnce(t *testing.T) {
	sh, tree := added(t, saltRoad)
	var got []string
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = sh.m.Titles(ctx, r, tree.ID)
		return err
	})
	if want := []string{"The Salt Road", "La Route du Sel", "The Salt Way", "Salt Road"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Titles = %q, want %q", got, want)
	}
}

func TestLinksOpenEachHostTheMovieNames(t *testing.T) {
	_, tree := added(t, saltRoad)
	want := []works.WebLink{
		{Key: "imdb", Label: "IMDb", URL: "https://www.imdb.com/title/tt9000005", Kind: works.LinkPage},
		{Key: "tmdb", Label: "TMDB", URL: "https://www.themoviedb.org/movie/9000005", Kind: works.LinkPage},
		{Key: "trakt", Label: "Trakt", URL: "https://trakt.tv/search/tmdb/9000005?id_type=movie", Kind: works.LinkPage},
		{Key: "trailer", Label: "Trailer", URL: "https://www.youtube.com/watch?v=sa1tR0adTr", Kind: works.LinkTrailer},
	}
	if got := library().Links(tree.Key, tree.Fields); !reflect.DeepEqual(got, want) {
		t.Fatalf("Links = %+v, want %+v", got, want)
	}
	bare := []works.WebLink{want[1], want[2]}
	if got := library().Links(tree.Key, nil); !reflect.DeepEqual(got, bare) {
		t.Fatalf("Links with no fields = %+v, want %+v", got, bare)
	}
}
