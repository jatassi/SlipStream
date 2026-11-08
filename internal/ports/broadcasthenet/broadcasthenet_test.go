package broadcasthenet

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/ports/port"
)

func TestAnAbsoluteEpisodeNumberWithAnIDSendsNoSearch(t *testing.T) {
	chain, err := getTvSearchRequests(&indexer{}, port.SearchCriteria{SearchTerm: "112", TvdbID: 248841, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.ValueOf(chain).IsZero() {
		t.Errorf("chain = %v, want none", chain)
	}
}
