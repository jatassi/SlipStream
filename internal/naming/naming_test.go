package naming_test

import (
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
		{"{Movie Title} ({Year}) - {Edition Tag} {Target}", orchard, naming.Core{Target: "2160p", EditionTag: "Director's Cut"}, "The Quiet Orchard (2021) - Director's Cut 2160p"},
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
