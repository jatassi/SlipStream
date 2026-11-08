package movie_test

import (
	"slices"
	"testing"

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
