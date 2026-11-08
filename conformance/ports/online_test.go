//go:build online

package ports_test

import (
	"net"
	"net/http"
	"net/url"
	"slices"
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
	"anidex":      {Mode: "search", Text: "One Piece"},
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
			if down(t, e.IndexerURLs[0]) {
				t.Skipf("%s resolves only to loopback while the site is down: https://github.com/jatassi/SlipStream/issues/477", e.IndexerURLs[0])
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

// down reports whether address's host resolves only to loopback, as a
// site's name can once its owner points it away from a dead site.
func down(t *testing.T, address string) bool {
	t.Helper()
	u, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return false
	}
	return !slices.ContainsFunc(ips, func(ip net.IP) bool { return !ip.IsLoopback() })
}
