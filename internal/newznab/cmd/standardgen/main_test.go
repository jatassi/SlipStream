package main

import (
	"strings"
	"testing"
)

const cs = `public static class NewznabStandardCategory
{
    public static readonly IndexerCategory ZedOther = new(0000, "Other");
    public static readonly IndexerCategory Movies = new(2000, "Movies");
    public static readonly IndexerCategory MoviesHD = new(2040, "Movies/HD");

    public static readonly IndexerCategory[] AllCats =
    {
        Movies,
        MoviesHD,
        ZedOther
    };

    static NewznabStandardCategory()
    {
        Movies.SubCategories.AddRange(
            new List<IndexerCategory>
            {
                MoviesHD
            });
    }
}`

func TestEveryMemberIsNamedWithItsParentAndAllCatsKeepsItsOrder(t *testing.T) {
	src, err := generate([]byte(cs), "ef9c1824")
	if err != nil {
		t.Fatal(err)
	}
	got := string(src)
	for _, want := range []string{
		"from Prowlarr's src/NzbDrone.Core/Indexers/NewznabStandardCategory.cs at ef9c1824. DO NOT EDIT.",
		`ZedOther = Category{ID: 0, Name: "Other", Parent: 0}`,
		`MoviesHD = Category{ID: 2040, Name: "Movies/HD", Parent: 2000}`,
		"var allCats = []Category{\n\tMovies,\n\tMoviesHD,\n\tZedOther,\n}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("standard.go lacks %q:\n%s", want, got)
		}
	}
	if _, err := generate([]byte(strings.Replace(cs, "MoviesHD\n            });", "MoviesUHD\n            });", 1)), "x"); err == nil {
		t.Error("a subcategory the file never declares generated")
	}
}
