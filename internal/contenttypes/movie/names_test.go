package movie_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
)

func TestTitleAndYear(t *testing.T) {
	for name, want := range map[string]struct {
		titles []string
		year   int
		cut    release.CutKey
		group  string
	}{
		"Blade.Runner.2049.2017.2160p.UHD.BluRay.REMUX.HDR.HEVC-FGT": {[]string{"Blade Runner 2049"}, 2017, release.Standard, "FGT"},
		"Uncut.Gems.2019.REPACK.1080p.BluRay.x264-SPARKS":            {[]string{"Uncut Gems"}, 2019, release.Standard, "SPARKS"},
		"Some.Movie.2019.UNCUT.1080p.BluRay.x264-SPARKS":             {[]string{"Some Movie"}, 2019, "uncut", "SPARKS"},
		"Spider-Man (2002)": {[]string{"Spider-Man"}, 2002, release.Standard, ""},
		"Mon cousin a.k.a. My Cousin 2020 1080p Blu-ray DD 5.1 x264.mkv": {
			[]string{"Mon cousin AKA My Cousin", "Mon cousin", "My Cousin"}, 2020, release.Standard, "",
		},
	} {
		r := title(name)
		if !slices.Equal(r.Name.Titles, want.titles) {
			t.Errorf("%s reads titles %q; want %q", name, r.Name.Titles, want.titles)
		}
		if y, _ := r.Name.Year.Get(); y != want.year {
			t.Errorf("%s reads year %d; want %d", name, y, want.year)
		}
		if r.Facts.Cut != want.cut {
			t.Errorf("%s reads the %s cut; want %s, as a title word names no cut", name, r.Facts.Cut, want.cut)
		}
		if g, _ := r.Facts.Group.Get(); g != want.group {
			t.Errorf("%s reads group %q; want %q", name, g, want.group)
		}
	}
}

func TestAReleaseTitleConditionSkipsTheMoviesTitle(t *testing.T) {
	remux := formats.Format{ID: formats.ID{1}, Name: "Remux", Conditions: []formats.Condition{
		{Name: "Remux", Kind: formats.ReleaseTitle, Fields: json.RawMessage(`{"value":"\\bremux\\b"}`)},
	}}
	c := formats.Compile([]formats.Format{remux}, formats.DefaultTimings())
	for name, want := range map[string]bool{
		"The.Remux.Heist.2021.1080p.WEB-DL.x264-GRP": false,
		"The.Heist.2021.1080p.BluRay.REMUX.AVC-GRP":  true,
	} {
		if got := len(c.Match(title(name)).Matched) == 1; got != want {
			t.Errorf("a Release title condition on remux matches %s: %t; want %t, as Radarr tests SimpleReleaseTitle with the movie's title masked", name, got, want)
		}
	}
}
