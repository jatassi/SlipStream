package torrentday

import (
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/newznab"
)

func indexer(s torrentDaySettings) *ix {
	base, _ := url.Parse("https://torrentday.fixture-site.fake.localhost/")
	return &ix{Settings: s, BaseURL: base, Categories: Port.Entry().Capabilities.Categories}
}

// The query is the site's own: categories joined by ";", then free and
// the term, which url.Values can't read, so the fixtures match only the
// path and this test reads the raw query.
func TestTheSearchQueryJoinsCategoriesFreeleechAndTerm(t *testing.T) {
	for _, c := range []struct {
		name     string
		settings torrentDaySettings
		term     string
		cats     []newznab.ID
		imdbID   string
		want     string
	}{
		{"a movie by IMDb id", torrentDaySettings{}, "The Quiet Orchard", []newznab.ID{newznab.MoviesBluRay.ID}, "tt0000001", "11;5;q=tt0000001+The+Quiet+Orchard"},
		{"freeleech only", torrentDaySettings{FreeLeechOnly: true}, "Harbor Lights S02E03", []newznab.ID{newznab.TVHD.ID}, "", "32;7;free;q=Harbor+Lights+S02E03"},
		{"no category", torrentDaySettings{}, "", nil, "", "25;96;11;5;103;3;21;22;13;44;48;1;24;104;32;31;33;46;82;14;26;7;34;2;10;4;18;8;9;17;27;23;41;16;29;42;20;102;30;95;47;43;45;28;12;19;6;15;q="},
	} {
		if got := getPagedRequests(indexer(c.settings), c.term, c.cats, c.imdbID).URL.RawQuery; got != c.want {
			t.Errorf("%s: query = %q, want %q", c.name, got, c.want)
		}
	}
}
