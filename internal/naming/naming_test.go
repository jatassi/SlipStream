package naming_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
)

func TestRender(t *testing.T) {
	orchard := map[string]string{"Movie Title": "The Quiet Orchard", "Year": "2021"}
	for _, c := range []struct {
		template string
		tokens   map[string]string
		core     naming.Core
		want     string
	}{
		{"{Movie Title} ({Year})", orchard, naming.Core{}, "The Quiet Orchard (2021)"},
		{"{Movie Title} ({Year}) - {Edition Tag} {Target}", orchard, naming.Core{Target: "1080p"}, "The Quiet Orchard (2021) - 1080p"},
		{"{Movie Title} ({Year}) - {Edition Tag} {Target}", orchard, naming.Core{Target: "2160p", EditionTag: "{edition-Director's Cut}", Edition: "Director's Cut"}, "The Quiet Orchard (2021) - {edition-Director's Cut} 2160p"},
		{"{Movie Title} ({Year}) - {Edition} {Target}", orchard, naming.Core{Target: "2160p", EditionTag: "{edition-Director's Cut}", Edition: "Director's Cut"}, "The Quiet Orchard (2021) - Director's Cut 2160p"},
		{"{Movie Title} ({Year}) - {Edition} {Target}", orchard, naming.Core{Target: "1080p"}, "The Quiet Orchard (2021) - 1080p"},
		{"{Movie Title} ({Year})", map[string]string{"Movie Title": "AC/DC: Live\\Loud", "Year": "1991"}, naming.Core{}, "AC DC Live Loud (1991)"},
		{"  {Movie Title}  ", map[string]string{"Movie Title": "Wait..."}, naming.Core{}, "Wait"},
		{"{Movie Title}", map[string]string{"Movie Title": "con"}, naming.Core{}, "_con"},
	} {
		got, err := naming.Render(c.template, c.tokens, c.core)
		if err != nil || got != c.want {
			t.Errorf("Render(%q, %v, %+v) = %q, %v; want %q", c.template, c.tokens, c.core, got, err, c.want)
		}
	}
}

func TestRenderRefusesAnUnfilledTokenAndAnEmptyName(t *testing.T) {
	for _, c := range []struct {
		template string
		tokens   map[string]string
		want     string
	}{
		{"{Movie Title} ({Year}) {Quality}", map[string]string{"Movie Title": "X"}, `template "{Movie Title} ({Year}) {Quality}" names {Year}, {Quality}, which no token fills`},
		{"{Movie Title}", map[string]string{"Movie Title": " ?* . "}, `template "{Movie Title}" renders to an empty name`},
		{"", nil, `template "" renders to an empty name`},
	} {
		if _, err := naming.Render(c.template, c.tokens, naming.Core{}); err == nil || err.Error() != c.want {
			t.Errorf("Render(%q) error = %v, want %s", c.template, err, c.want)
		}
	}
}

func TestRenderPadsWholeNumbers(t *testing.T) {
	tokens := map[string]string{"season": "3", "episode": "12", "absolute": "1042", "title": "Pilot"}
	for template, want := range map[string]string{
		"Season {season:00}":                "Season 03",
		"S{season:00}E{episode:00}":         "S03E12",
		"{absolute:000}":                    "1042",
		"{episode:0000} {title}":            "0012 Pilot",
		"Season {season}":                   "Season 3",
		"{title} S{season:00}E{episode:00}": "Pilot S03E12",
	} {
		got, err := naming.Render(template, tokens, naming.Core{})
		if err != nil || got != want {
			t.Errorf("Render(%q) = %q, %v; want %q", template, got, err, want)
		}
	}
	for template, want := range map[string]string{
		"{title:00}":  `template "{title:00}" pads {title:00}, which needs a whole number and a pattern of zeros`,
		"{season:0x}": `template "{season:0x}" pads {season:0x}, which needs a whole number and a pattern of zeros`,
		"{season:}":   `template "{season:}" pads {season:}, which needs a whole number and a pattern of zeros`,
		"{year:00}":   `template "{year:00}" names {year:00}, which no token fills`,
	} {
		if _, err := naming.Render(template, tokens, naming.Core{}); err == nil || err.Error() != want {
			t.Errorf("Render(%q) error = %v, want %s", template, err, want)
		}
	}
}

func TestFoldersRenderEachLevelAndTheAsideInPlaceOfTheOutermost(t *testing.T) {
	tv := naming.Templates{Folder: "{title}", Levels: []string{"Season {season:00}"}, Aside: "Specials", File: "{title}"}
	tokens := map[string]string{"title": "Pilot", "season": "2", "disc": "1"}
	for _, c := range []struct {
		name      string
		templates naming.Templates
		aside, on bool
		want      []string
	}{
		{"a level folder per level", tv, false, true, []string{"Season 02"}},
		{"an aside Item takes the aside folder", tv, true, true, []string{"Specials"}},
		{"level folders off", tv, false, false, nil},
		{"a type naming no level", naming.Templates{Folder: "{title}", File: "{title}"}, false, true, nil},
		{"the aside replaces only the outermost level", naming.Templates{Levels: []string{"Season {season}", "Disc {disc:00}"}, Aside: "Specials"}, true, true, []string{"Specials", "Disc 01"}},
		{"no aside template keeps the level's", naming.Templates{Levels: []string{"Season {season}"}}, true, true, []string{"Season 2"}},
	} {
		got, err := naming.Folders(c.templates, tokens, c.aside, c.on)
		if err != nil || !slices.Equal(got, c.want) {
			t.Errorf("%s: Folders = %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

func TestSanitize(t *testing.T) {
	for in, want := range map[string]string{
		"The Quiet Orchard (2021)":  "The Quiet Orchard (2021)",
		`a<b>c:d"e/f\g|h?i*j`:       "a b c d e f g h i j",
		"Tab\there\x00and\x7fthere": "Tab here and there",
		"Line\nbreak\u0085next":     "Line break next",
		"Trailing dots...":          "Trailing dots",
		"Trailing . . . ":           "Trailing",
		"  Leading":                 "Leading",
		"Half / Half":               "Half Half",
		"CON":                       "_CON",
		"nul":                       "_nul",
		"Aux.txt":                   "_Aux.txt",
		"NUL .tar.gz":               "_NUL .tar.gz",
		"COM1":                      "_COM1",
		"lpt9.log":                  "_lpt9.log",
		"COM0":                      "COM0",
		"LPT10":                     "LPT10",
		"CONSOLE":                   "CONSOLE",
		"Icon":                      "Icon",
		"…and Justice for All":      "…and Justice for All",
		"東京物語":                      "東京物語",
		"???":                       "",
	} {
		if got := naming.Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
