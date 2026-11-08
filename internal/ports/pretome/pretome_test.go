package pretome

import (
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/newznab"
)

func TestASearchSendsEachCategoryOnceWithItsTagsUnderSeven(t *testing.T) {
	base, _ := url.Parse("https://pretome.example/")
	indexer := &ix{BaseURL: base, Categories: setCapabilities().Categories}
	for _, c := range []struct {
		name       string
		categories []newznab.ID
		want       string
	}{
		{"none", nil, "st=1&search=Harbor+Lights&tags=&tf=all"},
		{"a group with tags", []newznab.ID{newznab.TVUHD.ID}, "st=1&search=Harbor+Lights&cat%5B%5D=7&tags=2160p%2CDolbyVision&tf=any"},
		{"seven tags or more", []newznab.ID{newznab.TV.ID}, "st=1&search=Harbor+Lights&cat%5B%5D=52&cat%5B%5D=7"},
		{"two groups", []newznab.ID{newznab.MoviesUHD.ID, newznab.TVUHD.ID}, "st=1&search=Harbor+Lights&cat%5B%5D=19&cat%5B%5D=7&tags=2160p%2CDolbyVision&tf=any"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := getPagedRequests(indexer, "Harbor Lights", c.categories, "")
			if got.URL.Path != "/browse.php" || got.URL.Query().Encode() != mustQuery(t, c.want) {
				t.Errorf("request URL = %s, want /browse.php?%s", got.URL, c.want)
			}
		})
	}
}

func mustQuery(t *testing.T, raw string) string {
	t.Helper()
	q, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatal(err)
	}
	return q.Encode()
}
