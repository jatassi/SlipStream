package dns_test

import (
	"fmt"
	"testing"

	"github.com/miekg/dns"

	fakedns "github.com/jatassi/slipstream/fakes/dns"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func TestEachSiteResolvesTheScenarioRecords(t *testing.T) {
	s := scenario.Scenario{
		Name: "dns",
		Nodes: []scenario.Node{
			{Name: "atlas", Setup: scenario.SetupController, Site: "home"},
			{Name: "seedbox", Setup: scenario.SetupJoin, Site: "ams"},
		},
		Network: &scenario.Network{
			Sites: []scenario.Site{
				{Name: "home", WAN: "public", Address: "203.0.113.9", Resolver: "split"},
				{Name: "ams", WAN: "none", Address: "192.0.2.44"},
			},
			DNS: []scenario.Record{
				{Name: "plex.home.localhost", A: []string{"203.0.113.9"}, AAAA: []string{"2001:db8:1::20"}, Home: []string{"192.168.1.10"}},
				{Name: "stale.home.localhost", A: []string{"203.0.113.200"}, AAAA: []string{"2001:db8:9::20"}},
				{Name: "private.home.localhost", A: []string{"192.168.1.20"}, AAAA: []string{"fd72::20"}},
				{Name: "gone.home.localhost", A: []string{"203.0.113.9"}, Answer: "nxdomain"},
				{Name: "broken.home.localhost", A: []string{"203.0.113.9"}, Answer: "servfail"},
				{Name: "v4.home.localhost", A: []string{"203.0.113.9"}, AAAA: []string{"2001:db8:1::20"}, Answer: "aaaa-servfail"},
			},
		},
	}
	srv := faketest.UDP(t, fakedns.Fake, s, faketest.WithOrigins("atlas", "seedbox"))
	for _, c := range []struct {
		origin, name string
		qtype        uint16
		want         string
	}{
		{"seedbox", "plex.home.localhost.", dns.TypeA, "NOERROR [203.0.113.9]"},
		{"seedbox", "plex.home.localhost.", dns.TypeAAAA, "NOERROR [2001:db8:1::20]"},
		{"atlas", "plex.home.localhost.", dns.TypeA, "NOERROR [192.168.1.10]"},
		{"atlas", "plex.home.localhost.", dns.TypeAAAA, "NOERROR []"},
		{"seedbox", "stale.home.localhost.", dns.TypeA, "NOERROR [203.0.113.200]"},
		{"seedbox", "stale.home.localhost.", dns.TypeAAAA, "NOERROR [2001:db8:9::20]"},
		{"seedbox", "private.home.localhost.", dns.TypeA, "NOERROR [192.168.1.20]"},
		{"seedbox", "private.home.localhost.", dns.TypeAAAA, "NOERROR [fd72::20]"},
		{"seedbox", "gone.home.localhost.", dns.TypeA, "NXDOMAIN []"},
		{"seedbox", "broken.home.localhost.", dns.TypeA, "SERVFAIL []"},
		{"seedbox", "v4.home.localhost.", dns.TypeA, "NOERROR [203.0.113.9]"},
		{"seedbox", "v4.home.localhost.", dns.TypeAAAA, "SERVFAIL []"},
		{"seedbox", "localhost.", dns.TypeAAAA, "NXDOMAIN []"},
	} {
		q := new(dns.Msg).SetQuestion(c.name, c.qtype)
		res, err := dns.Exchange(q, srv.Sockets[c.origin].String())
		if err != nil {
			t.Fatal(err)
		}
		var addrs []string
		for _, rr := range res.Answer {
			switch rr := rr.(type) {
			case *dns.A:
				addrs = append(addrs, rr.A.String())
			case *dns.AAAA:
				addrs = append(addrs, rr.AAAA.String())
			}
		}
		if got := fmt.Sprintf("%s %v", dns.RcodeToString[res.Rcode], addrs); got != c.want {
			t.Errorf("%s asking %s %s = %s, want %s", c.origin, dns.TypeToString[c.qtype], c.name, got, c.want)
		}
	}
}
