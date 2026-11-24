package netfacts_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"testing"

	fakedns "github.com/jatassi/slipstream/fakes/dns"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// A lookup keeps each record type's own answer and rcode, through the
// resolver entry, or as the OS resolver tells them with the entry empty.
func TestResolveKeepsEachRecordsAnswer(t *testing.T) {
	s := scenario.Scenario{
		Name:  "dns",
		Nodes: []scenario.Node{{Name: "atlas", Setup: scenario.SetupController, Site: "home"}, {Name: "seedbox", Setup: scenario.SetupJoin, Site: "ams"}},
		Network: &scenario.Network{
			Sites: []scenario.Site{{Name: "home", WAN: "public", Address: "203.0.113.9", Resolver: "split"}, {Name: "ams", WAN: "none", Address: "192.0.2.44"}},
			DNS: []scenario.Record{
				{Name: "plex.home.localhost", A: []string{"203.0.113.9"}, AAAA: []string{"2001:db8:1::20"}, Home: []string{"192.168.1.10"}},
				{Name: "gone.home.localhost", A: []string{"203.0.113.9"}, Answer: "nxdomain"},
				{Name: "broken.home.localhost", A: []string{"203.0.113.9"}, Answer: "servfail"},
				{Name: "v4.home.localhost", A: []string{"203.0.113.9"}, AAAA: []string{"2001:db8:1::20"}, Answer: "aaaa-servfail"},
			},
		},
	}
	srv := faketest.UDP(t, fakedns.Fake, s, faketest.WithOrigins("atlas", "seedbox"))
	osLookup := func(_ context.Context, host string) ([]netip.Addr, error) {
		switch host {
		case "gone.example":
			return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
		case "slow.example":
			return nil, &net.DNSError{Err: "i/o timeout", Name: host, IsTimeout: true}
		case "broken.example":
			return nil, &net.DNSError{Err: "server misbehaving", Name: host}
		}
		return []netip.Addr{netip.MustParseAddr("203.0.113.9"), netip.MustParseAddr("2001:db8:1::20")}, nil
	}
	for _, c := range []struct {
		origin, host string
		want         string
	}{
		{"seedbox", "plex.home.localhost", "[203.0.113.9] - [2001:db8:1::20] -"},
		{"atlas", "plex.home.localhost", "[192.168.1.10] - [] -"},
		{"seedbox", "gone.home.localhost", "[] nxdomain [] nxdomain"},
		{"seedbox", "broken.home.localhost", "[] servfail [] servfail"},
		{"seedbox", "v4.home.localhost", "[203.0.113.9] - [] servfail"},
		{"", "plex.example", "[203.0.113.9] - [2001:db8:1::20] - os"},
		{"", "gone.example", "[] nxdomain [] nxdomain os"},
		{"", "slow.example", "[] timeout [] timeout os"},
		{"", "broken.example", "[] servfail [] servfail os"},
	} {
		e := env()
		e.Lookup = osLookup
		var endpoints netfacts.Endpoints
		if c.origin != "" {
			endpoints.Resolver = endpoint.Optional{endpoint.MustParseHostPort(srv.Sockets[c.origin].String())}
		}
		svc := netfacts.NewService(netfacts.New(netfacts.Config{Local: storagetest.Local(t, netfacts.Schema()), Env: e, Endpoints: endpoints, Timings: timings(), Log: slog.New(slog.DiscardHandler)}))
		res, err := svc.Resolve(t.Context(), &netv1.ResolveRequest{Host: c.host})
		if err != nil {
			t.Fatal(err)
		}
		got := fmt.Sprintf("%v %s %v %s", addrs(res.GetA()), rcode(res.GetErrorA()), addrs(res.GetAaaa()), rcode(res.GetErrorAaaa()))
		if res.GetOsResolver() {
			got += " os"
		}
		if got != c.want {
			t.Errorf("%s looking up %s = %s, want %s", c.origin, c.host, got, c.want)
		}
	}
}

func addrs(raw [][]byte) []netip.Addr {
	out := []netip.Addr{}
	for _, b := range raw {
		a, _ := netip.AddrFromSlice(b)
		out = append(out, a)
	}
	return out
}

func rcode(e netv1.DNSError) string {
	if e == netv1.DNSError_DNS_ERROR_UNSPECIFIED {
		return "-"
	}
	return strings.ToLower(strings.TrimPrefix(e.String(), "DNS_ERROR_"))
}
