//go:build online

package ports_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	conformance "github.com/jatassi/slipstream/conformance/ports"
	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/searching"
)

// searches are a search each public site has had Releases for, for
// years.
var searches = map[port.ID]searching.Query{
	"knaben":      {Mode: "movie", Text: "Big Buck Bunny"},
	"subsplease":  {Mode: "search", Text: "One Piece"},
	"torrentscsv": {Mode: "search", Text: "Ubuntu"},
}

// TestThePublicPortsConform runs every public port against its live site,
// at its first address.
func TestThePublicPortsConform(t *testing.T) {
	var bodies []shape.Body
	for _, e := range all.Registry().Entries() {
		if e.Privacy != port.Public {
			continue
		}
		t.Run(string(e.ID), func(t *testing.T) {
			q, ok := searches[e.ID]
			if !ok {
				t.Fatalf("the online suite has no search for %s: add one to searches", e.ID)
			}
			tgt := conformance.Target{Port: e.ID, Address: e.IndexerURLs[0], Client: &http.Client{Timeout: time.Minute}, Timings: searching.DefaultTimings(), Search: q}
			if *shape.Update {
				tgt.Record = func(key string, body []byte) { bodies = append(bodies, shape.Body{Key: key, JSON: body}) }
			}
			conformance.Run(t, tgt)
		})
	}
	if *shape.Update && !t.Failed() {
		if err := shape.Write("shape.json", bodies); err != nil {
			t.Fatal(err)
		}
	}
}
