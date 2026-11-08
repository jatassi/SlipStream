package speedcd

import (
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/ports/port"
)

func TestASearchTermKeepsTheEscapesUrlEncodeWrote(t *testing.T) {
	base, _ := url.Parse("https://speedcd.example/")
	indexer := &ix{BaseURL: base, Categories: setCapabilities().Categories}
	got := getPagedRequests(indexer, "Harbor Lights' Café (2019)", port.SearchCriteria{}, false)
	if want := "https://speedcd.example/browse/q/Harbor+Lights%27+Caf%C3%A9+(2019)"; got.URL.String() != want {
		t.Errorf("request URL = %s, want %s", got.URL, want)
	}
}
