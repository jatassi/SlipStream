package torrentscsv

import (
	"net/url"
	"testing"
)

func TestATermUnderThreeCharactersSendsNoRequest(t *testing.T) {
	base, _ := url.Parse("https://torrentscsv.fixture-site.fake.localhost/")
	for term, want := range map[string]int{"": 0, "  ": 0, "Oz": 0, "Ozu": 1} {
		if got := len(getPagedRequests(&ix{BaseURL: base}, term)); got != want {
			t.Errorf("getPagedRequests(%q) sends %d requests, want %d", term, got, want)
		}
	}
}
