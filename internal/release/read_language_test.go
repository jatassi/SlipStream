package release_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

// masking reads a name as Radarr does: the title replaced, so a title
// word never reads as a language.
type masking struct{ masked string }

func (m masking) ReadName(release.Prepared) release.Name {
	return release.Name{Titles: []string{"A Movie"}, Masked: m.masked, Tail: m.masked}
}

func (masking) CleanTitle(title string) string { return title }

func TestLanguages(t *testing.T) {
	cases := []struct {
		name  string
		names release.Names
		want  []release.Language
	}{
		{"The.Quiet.Orchard.S01E01.1080p.WEB-DL-GRP", release.TitleOnly{}, nil},
		{"The.Quiet.Orchard.S01E01.MULTi.1080p.WEB-DL-GRP", release.TitleOnly{}, nil},
		{"The.Quiet.Orchard.S01E01.VOSTFR.1080p.WEB-DL-GRP", release.TitleOnly{}, nil},
		{"The.Quiet.Orchard.S01E01.MULTi.VFF.1080p.WEB-DL-GRP", release.TitleOnly{}, []release.Language{"fr"}},
		{"The.Quiet.Orchard.S01E01.German.DL.1080p.WEB-DL-GRP", release.TitleOnly{}, []release.Language{"de", release.Original}},
		{"The.Quiet.Orchard.S01E01.German.ML.1080p.WEB-DL-GRP", release.TitleOnly{}, []release.Language{"de", release.Original, "en"}},
		{"The.Quiet.Orchard.S01E01.Ger.Eng.Sub.1080p.WEB-DL-GRP", release.TitleOnly{}, nil},
		{"The.Quiet.Orchard.S01E01.ITA.ENG.1080p.WEB-DL-GRP", release.TitleOnly{}, []release.Language{"it", "en"}},
		{"The.Quiet.Orchard.S01E01.Brazilian.1080p.WEB-DL-GRP", release.TitleOnly{}, []release.Language{"pt"}},
		{"The.Quiet.Orchard.2021.Persian.1080p.WEBRip-GRP", release.TitleOnly{}, []release.Language{"fa"}},
		{"The.French.Connection.1971.1080p.BluRay-GRP", masking{"A.Movie.1971.1080p.BluRay-GRP"}, nil},
		{"The.Quiet.Orchard.2021.1080p.BluRay-RUS", masking{"A.Movie.2021.1080p.BluRay-RUS"}, nil},
		{"The.Quiet.Orchard.2021.ITA.1080p.BluRay-GRP", masking{"A.Movie.2021.ITA.1080p.BluRay-GRP"}, []release.Language{"it"}},
	}
	for _, c := range cases {
		got, known := release.Title(c.name, c.names, nil).Facts.Languages.Get()
		if known != (c.want != nil) || !slices.Equal(got, c.want) {
			t.Errorf("%s read languages %v (known %v), want %v", c.name, got, known, c.want)
		}
	}
}

func TestUpstreamLanguage(t *testing.T) {
	for name, want := range map[string]release.Language{
		"PortugueseBrazil": "pt", "PortugueseBR": "pt", "SpanishLatino": "es", "Flemish": "nl", "Original": release.Original,
	} {
		if got, ok := release.UpstreamLanguage(name); !ok || got != want {
			t.Errorf("%s reads as %q, want %q", name, got, want)
		}
	}
	if _, ok := release.UpstreamLanguage("Unknown"); ok {
		t.Error("Unknown reads as a language")
	}
}
