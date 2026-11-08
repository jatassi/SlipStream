package xspeeds

import (
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/newznab"
)

func TestASearchNamesACategoryOnlyWhenOneMaps(t *testing.T) {
	base, _ := url.Parse("https://xspeeds.example/")
	indexer := &ix{BaseURL: base, Categories: setCapabilities().Categories}
	for _, c := range []struct {
		name       string
		categories []newznab.ID
		want       string
	}{
		{"none", nil, "0"},
		{"one", []newznab.ID{newznab.AudioAudiobook.ID}, "154"},
		{"several", []newznab.ID{newznab.MoviesUHD.ID}, "0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := getPagedRequests(indexer, "", c.categories).URL.Query().Get("category"); got != c.want {
				t.Errorf("category = %q, want %q", got, c.want)
			}
		})
	}
}
