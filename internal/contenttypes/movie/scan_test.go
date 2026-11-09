package movie_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestHintNamesTheKeyATMDBTagGives(t *testing.T) {
	tests := []struct {
		name string
		want scanning.Hint
	}{
		{"The Salt Road (2022) {tmdb-9000005}", scanning.Hint{Title: "The Salt Road", Year: 2022, IDs: []scanning.ID{{Provider: scanning.TMDB, Value: "9000005"}}, Key: saltRoad}},
		{"The Salt Road (2022) [imdbid-tt9000005]", scanning.Hint{Title: "The Salt Road", Year: 2022, IDs: []scanning.ID{{Provider: scanning.IMDb, Value: "tt9000005"}}}},
		{"Trailer Park Boys The Movie (2006)", scanning.Hint{Title: "Trailer Park Boys The Movie", Year: 2006}},
	}
	for _, tt := range tests {
		if got := library().Hint(tt.name); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Hint(%q) = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestClaimsAreTheYearAndBothIDs(t *testing.T) {
	sh, tree := added(t, saltRoad)
	want := map[works.WorkID]scanning.Claim{tree.ID: {Year: 2022, IDs: []scanning.ID{
		{Provider: scanning.TMDB, Value: "9000005"}, {Provider: scanning.IMDb, Value: "tt9000005"},
	}}}
	ids := []works.WorkID{tree.ID}
	for _, ws := range []works.Among[works.WorkID]{works.Only(ids), works.Every(ids)} {
		var got map[works.WorkID]scanning.Claim
		sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			got, err = sh.m.Claims(ctx, r, ws)
			return err
		})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Claims of every movie (%t) = %+v, want %+v", ws.Every(), got, want)
		}
	}
}

func TestYearOfASearchResult(t *testing.T) {
	got, err := serve(t, catalog(t)).open().Search(t.Context(), inEnglish, "salt road")
	if err != nil || len(got) != 1 {
		t.Fatalf("Search = %+v, %v; want The Salt Road", got, err)
	}
	if y := library().YearOf(got[0]); y != 2022 {
		t.Fatalf("YearOf(%+v) = %d, want 2022", got[0], y)
	}
}

func TestPlaceAnswersTheMoviesOneItem(t *testing.T) {
	sh, tree := added(t, saltRoad)
	var place scanning.Placer
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		place, err = sh.m.Place(ctx, r, tree.ID)
		return err
	})
	for _, rel := range []string{"The Salt Road (2022).mkv", "Extras Folder/The Salt Road.mkv"} {
		if item, ok := place(rel).Item(); !ok || item != tree.Items[0].ID {
			t.Errorf("Place(%q) = %v, %t; want the movie's Item %v", rel, item, ok, tree.Items[0].ID)
		}
	}
}
