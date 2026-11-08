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
	}{
		"Blade.Runner.2049.2017.2160p.UHD.BluRay.REMUX.HDR.HEVC-FGT": {[]string{"Blade Runner 2049"}, 2017},
		"Uncut.Gems.2019.REPACK.1080p.BluRay.x264-SPARKS":            {[]string{"Uncut Gems"}, 2019},
		"Mon cousin a.k.a. My Cousin 2020 1080p Blu-ray DD 5.1 x264.mkv": {
			[]string{"Mon cousin AKA My Cousin", "Mon cousin", "My Cousin"}, 2020,
		},
	} {
		n := title(name).Name
		if !slices.Equal(n.Titles, want.titles) {
			t.Errorf("%s reads titles %q; want %q", name, n.Titles, want.titles)
		}
		if y, _ := n.Year.Get(); y != want.year {
			t.Errorf("%s reads year %d; want %d", name, y, want.year)
		}
	}
}

func TestTheTitleNamesNoCut(t *testing.T) {
	if got := title("Uncut.Gems.2019.REPACK.1080p.BluRay.x264-SPARKS").Facts.Cut; got != release.Standard {
		t.Errorf("Uncut Gems reads the %s cut; want standard", got)
	}
	if got := title("Some.Movie.2019.UNCUT.1080p.BluRay.x264-SPARKS").Facts.Cut; got != "uncut" {
		t.Errorf("an UNCUT tag reads the %s cut; want uncut", got)
	}
}

func TestTheTitleNamesNoGroup(t *testing.T) {
	if g, ok := title("Spider-Man (2002)").Facts.Group.Get(); ok {
		t.Errorf("Spider-Man (2002) reads group %q; want none", g)
	}
	if g, _ := title("Spider-Man.2002.1080p.BluRay.x264-SPARKS").Facts.Group.Get(); g != "SPARKS" {
		t.Errorf("the group reads %q; want SPARKS", g)
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
