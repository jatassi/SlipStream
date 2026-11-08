package myanonamouse

import (
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/ports/port"
)

func TestATermWithNoWordCharactersSendsNoRequest(t *testing.T) {
	base, _ := url.Parse("https://myanonamouse.fixture-site.fake.localhost/")
	c := &ix{BaseURL: base, Categories: Port.Entry().Capabilities.Categories}
	for term, want := range map[string]int{"!! --": 0, "": 1, "The Quiet Orchard": 1} {
		if got := len(getPagedRequests(c, port.SearchCriteria{SearchTerm: term})); got != want {
			t.Errorf("getPagedRequests(%q) sends %d requests, want %d", term, got, want)
		}
	}
}
