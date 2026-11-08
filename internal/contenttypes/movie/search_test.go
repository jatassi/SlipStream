package movie_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func (sh shelf) expect(t *testing.T, work works.WorkID) indexers.Expect {
	t.Helper()
	var ex indexers.Expect
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		ex, err = sh.m.Expect(ctx, r, work)
		return err
	})
	return ex
}

func matches(sh shelf, c release.Claim, name string) bool {
	m, _ := release.Match(title(name).Name, nil, c, sh.m)
	return m == release.Matched
}

func TestExpectClaimsTheMovieTitlesYearAndOriginalLanguage(t *testing.T) {
	sh, tree := added(t, saltRoad)
	ex := sh.expect(t, tree.ID)
	if want := []string{"The Salt Road", "La Route du Sel", "The Salt Way", "Salt Road"}; !slices.Equal(ex.Claim.Titles, want) {
		t.Fatalf("the claim's titles are %q, want the label, original title and alternative titles %q", ex.Claim.Titles, want)
	}
	if ex.Claim.Year != release.Known(2022) || ex.Claim.Slack != 1 {
		t.Fatalf("the claim's year is %+v with slack %d, want 2022 with slack 1", ex.Claim.Year, ex.Claim.Slack)
	}
	if ex.Original != release.Known(release.Language("fr")) {
		t.Fatalf("the original language is %+v, want fr", ex.Original)
	}
	if got := ex.Covers(title("The.Salt.Road.2022.1080p.WEB-DL").Name); !slices.Equal(got, []works.ItemID{tree.Items[0].ID}) {
		t.Fatalf("a release covers %v, want the movie's one Item", got)
	}
	for name, want := range map[string]bool{
		"The.Salt.Road.2022.1080p.BluRay.x264-GRP":          true,
		"La.Route.du.Sel.2022.FRENCH.1080p.BluRay.x264-GRP": true,
		"Salt.Road.2022.720p.WEB-DL":                        true,
		"Le Chemin a.k.a. The Salt Way 2022 1080p BluRay":   true,
		"Le Chemin (AKA Salt Road) 2022 1080p":              true,
		"The.Pepper.Road.2022.1080p.BluRay.x264-GRP":        false,
		"Le Chemin a.k.a. The Pepper Way 2022 1080p BluRay": false,
	} {
		if got := matches(sh, ex.Claim, name); got != want {
			t.Errorf("%s matches the claim: %t, want %t", name, got, want)
		}
	}
}

func TestAMovieWithNoYearClaimsNone(t *testing.T) {
	sh, tree := added(t, "9000004")
	if y, known := sh.expect(t, tree.ID).Claim.Year.Get(); known {
		t.Fatalf("a planned movie with no release date claims the year %d, want none", y)
	}
}

func TestTMDBsNoLanguageClaimsNoOriginalLanguage(t *testing.T) {
	s := catalog(t)
	s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: 9000011, Title: "Silent Reel", OriginalLanguage: "xx", Released: "1925-01-01"})
	sh := open(t, metadatatest.Serve(map[works.TypeID]metadata.Source{movie.Def.ID(): serve(t, s).open()}))
	tree := sh.add(t, "9000011")
	if l, known := sh.expect(t, tree.ID).Original.Get(); known {
		t.Fatalf("a movie TMDB gives the language xx claims %q, want no original language", l)
	}
}

func TestReleaseYearCanDifferBy(t *testing.T) {
	sh, tree := added(t, saltRoad)
	slack := func(raw string) (setting.CountField, error) {
		t.Helper()
		fields, err := sh.works.SetSettings(t.Context(), identity.Admin{}, movie.Def.ID(), setting.Patch{movie.YearSlack.Key: json.RawMessage(raw)})
		for _, f := range fields {
			if f, ok := f.(setting.CountField); ok && f.Key == "movie.yearSlack" {
				return f, err
			}
		}
		return setting.CountField{}, err
	}
	for _, tc := range []struct {
		raw  string
		want int
	}{{"0", 0}, {"10", 10}, {"null", 1}} {
		f, err := slack(tc.raw)
		if err != nil {
			t.Fatalf("a year slack of %s = %v", tc.raw, err)
		}
		if f.Label != "Release year can differ by" || f.Value != int64(tc.want) || f.Default != 1 || f.IsDefault != (tc.raw == "null") {
			t.Fatalf("after %s the year slack field is %+v", tc.raw, f)
		}
		var terms indexers.Terms
		sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			terms, err = sh.m.SearchTerms(ctx, r, tree.Items[0].ID)
			return err
		})
		if terms.Claim.Slack != tc.want || sh.expect(t, tree.ID).Claim.Slack != tc.want {
			t.Fatalf("after a year slack of %s the search claim's slack is %d, want %d in SearchTerms and Expect", tc.raw, terms.Claim.Slack, tc.want)
		}
	}
	for _, raw := range []string{"-1", "11"} {
		var refused *setting.RangeError
		if _, err := slack(raw); !errors.As(err, &refused) {
			t.Fatalf("a year slack of %s = %v, want a RangeError", raw, err)
		}
	}
}

func TestTheYearSlackJudgesAReleaseYear(t *testing.T) {
	sh, tree := added(t, saltRoad)
	for _, tc := range []struct {
		raw   string
		years map[int]bool
	}{
		{"null", map[int]bool{2021: true, 2023: true, 2024: false}},
		{"0", map[int]bool{2022: true, 2023: false}},
		{"10", map[int]bool{2012: true, 2032: true, 2033: false}},
	} {
		if _, err := sh.works.SetSettings(t.Context(), identity.Admin{}, movie.Def.ID(), setting.Patch{movie.YearSlack.Key: json.RawMessage(tc.raw)}); err != nil {
			t.Fatalf("a year slack of %s = %v", tc.raw, err)
		}
		claim := sh.expect(t, tree.ID).Claim
		for year, want := range tc.years {
			name := fmt.Sprintf("The.Salt.Road.%d.1080p.BluRay.x264-GRP", year)
			if got := matches(sh, claim, name); got != want {
				t.Errorf("at a year slack of %s %s matches the 2022 movie: %t, want %t", tc.raw, name, got, want)
			}
		}
	}
}
