package release_test

import (
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

// folding compares titles in lower case with dots read as spaces.
type folding struct{ release.TitleOnly }

func (folding) CleanTitle(title string) string {
	return strings.ToLower(strings.ReplaceAll(title, ".", " "))
}

func TestMatch(t *testing.T) {
	heat := release.Claim{
		IDs:    map[string]string{"tmdbid": "949", "imdbid": "tt0113277"},
		Titles: []string{"Heat", "Fièvre"},
		Year:   release.Known(1995),
		Slack:  1,
	}
	name := func(year int, titles ...string) release.Name {
		n := release.Name{Titles: titles}
		if year != 0 {
			n.Year = release.Known(year)
		}
		return n
	}
	for _, c := range []struct {
		why   string
		name  release.Name
		ids   map[string]string
		want  release.Matching
		given string
	}{
		{"a shared id matches whatever the title", name(1986, "Heat"), map[string]string{"tmdbid": "949"}, release.Matched, ""},
		{"a conflicting id is another Work", name(1995, "Heat"), map[string]string{"tmdbid": "11"}, release.WrongWork, ""},
		{"a conflicting id outweighs a shared one", name(1995, "Heat"), map[string]string{"tmdbid": "949", "imdbid": "tt0000001"}, release.WrongWork, ""},
		{"an id the Work lacks is no evidence", name(1995, "Heat"), map[string]string{"tvdbid": "5"}, release.Matched, ""},
		{"the title under the clean rule", name(1995, "HEAT"), nil, release.Matched, ""},
		{"any other title", name(1995, "The.Heat", "fièvre"), nil, release.Matched, ""},
		{"a year within the slack", name(1996, "Heat"), nil, release.Matched, ""},
		{"no year passes", name(0, "Heat"), nil, release.Matched, ""},
		{"a year past the slack", name(1986, "Heat"), nil, release.WrongTitle, "Heat (1986)"},
		{"another title", name(1995, "Heat Wave"), nil, release.WrongTitle, "Heat Wave (1995)"},
		{"no title", name(0), nil, release.WrongTitle, ""},
	} {
		got, given := release.Match(c.name, c.ids, heat, folding{})
		if got != c.want || given != c.given {
			t.Errorf("%s: Match = %d, %q, want %d, %q", c.why, got, given, c.want, c.given)
		}
	}
	if got, _ := release.Match(name(1986, "Heat"), nil, release.Claim{Titles: []string{"Heat"}}, folding{}); got != release.Matched {
		t.Errorf("a Work with no year matches any year, got %d", got)
	}
}
