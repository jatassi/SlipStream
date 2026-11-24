package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakegateway "github.com/jatassi/slipstream/fakes/gateway"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/netfacts/gateway"
	"github.com/jatassi/slipstream/internal/netmodel"
)

var nodes = []string{"atlas", "den", "nas", "pi", "box", "seedbox"}

func world() scenario.Scenario {
	site := map[string]string{"atlas": "home", "den": "cell", "nas": "double", "pi": "permanent", "box": "other", "seedbox": "vps"}
	s := scenario.Scenario{Name: "gateway", Network: &scenario.Network{Sites: []scenario.Site{
		{Name: "home", WAN: "public", Address: "203.0.113.9", Mapping: []string{"pcp", "nat-pmp", "upnp"}, Forwards: []scenario.Forward{{Node: "atlas", Service: "plex", Listed: true}}},
		{Name: "cell", WAN: "cgnat", Address: "203.0.113.50", Gateway: "100.64.12.7", Mapping: []string{"pcp", "nat-pmp"}},
		{Name: "double", WAN: "private", Address: "203.0.113.60", Gateway: "192.168.0.5", Mapping: []string{"upnp"}},
		{Name: "permanent", WAN: "public", Address: "203.0.113.70", Mapping: []string{"nat-pmp", "upnp"}, Refuse: "725"},
		{Name: "other", WAN: "empty", Address: "203.0.113.80", Mapping: []string{"pcp", "nat-pmp", "upnp"}},
		{Name: "vps", WAN: "none", Address: "192.0.2.44"},
	}}}
	for _, n := range nodes {
		s.Nodes = append(s.Nodes, scenario.Node{Name: n, Setup: scenario.SetupJoin, Site: site[n]})
	}
	s.Nodes[0].Setup = scenario.SetupController
	return s
}

type loopback struct{}

func (loopback) DialUDP(_ context.Context, raddr netip.AddrPort) (*net.UDPConn, error) {
	c, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(raddr))
	if err != nil {
		return nil, fmt.Errorf("open a UDP socket to %s: %w", raddr, err)
	}
	return c, nil
}

func (loopback) ListenUDP(_ context.Context, local netip.AddrPort) (*net.UDPConn, error) {
	c, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(local))
	if err != nil {
		return nil, fmt.Errorf("open a UDP socket on %s: %w", local, err)
	}
	return c, nil
}

// A read learns each router's protocols, WAN and listed forwards, and asks
// nothing that changes a mapping.
func TestReadLearnsEachRouterAndWritesNothing(t *testing.T) {
	s := world()
	var fronts fake.Fronts
	for i, n := range append(slices.Clone(nodes), events.World) {
		fronts = append(fronts,
			fake.Front{Target: "plex", Node: n, Address: fake.Address{AddrPort: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(32400+i))}},
			fake.Front{Target: "gateway-igd", Node: n, Address: fake.Address{AddrPort: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(41000+i))}})
	}
	shared := fake.NewShared()
	origins := faketest.WithOrigins(append(slices.Clone(nodes), events.World)...)
	udp := faketest.UDP(t, fakegateway.Fake, s, origins, faketest.WithShared(shared), faketest.WithFronts(fronts))
	igd := faketest.InMemory(t, fakegateway.IGD, s, origins, faketest.WithShared(shared), faketest.WithFronts(fronts))
	yes, no := netmodel.Yes, netmodel.No
	for _, c := range []struct {
		origin string
		want   netmodel.Gateway
	}{
		{"atlas", netmodel.Gateway{
			Protocols: netmodel.Protocols{PCP: yes, NATPMP: yes, UPnP: yes}, WAN: netip.MustParseAddr("203.0.113.9"),
			Forwards: []netmodel.Forward{{External: 32400, Client: netip.MustParseAddr("192.168.1.10"), Internal: 32400, Description: "plex"}},
		}},
		{"den", netmodel.Gateway{Protocols: netmodel.Protocols{PCP: yes, NATPMP: yes, UPnP: no}, WAN: netip.MustParseAddr("100.64.12.7")}},
		{"nas", netmodel.Gateway{Protocols: netmodel.Protocols{PCP: no, NATPMP: no, UPnP: yes}, WAN: netip.MustParseAddr("192.168.0.5")}},
		{"pi", netmodel.Gateway{Protocols: netmodel.Protocols{PCP: no, NATPMP: yes, UPnP: yes}, WAN: netip.MustParseAddr("203.0.113.70")}},
		{"box", netmodel.Gateway{Protocols: netmodel.Protocols{PCP: yes, NATPMP: yes, UPnP: yes}}},
		{"seedbox", netmodel.Gateway{Protocols: netmodel.Protocols{PCP: no, NATPMP: no, UPnP: no}}},
	} {
		client := &gateway.Client{Packets: loopback{}, HTTP: igd.From(t, c.origin), Timings: gateway.Timings{Exchange: 200 * time.Millisecond, SSDPWait: 200 * time.Millisecond}}
		got, err := client.Read(t.Context(), gateway.At(udp.Sockets[c.origin], netip.MustParseAddr("192.168.1.20")), []uint16{32400, 8484})
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s's router reads %+v, %v; want %+v", c.origin, got, err, c.want)
		}
	}
	recs := udp.Records(t)
	if len(recs) == 0 {
		t.Fatal("the routers recorded no request")
	}
	for _, r := range recs {
		var req fakegateway.Request
		if err := json.Unmarshal(r.Data, &req); err != nil || req.Write {
			t.Errorf("a read sent %s, which changes a mapping", r.Data)
		}
	}
}

// Each protocol maps a port on a router that speaks it, a renewal keeps
// one mapping, and a delete leaves none, twice over; each refusal answers
// its kind and leaves nothing behind.
func TestMapAndUnmapOnEachRouter(t *testing.T) {
	origins := []string{"atlas", "den", "nas", "pi", "box", "seedbox", "pc"}
	site := map[string]string{"atlas": "all", "den": "natpmp", "pc": "natpmp", "nas": "igd", "pi": "permanent", "box": "locked", "seedbox": "other"}
	s := scenario.Scenario{Name: "mapping", Network: &scenario.Network{Sites: []scenario.Site{
		{Name: "all", WAN: "public", Address: "203.0.113.9", Mapping: []string{"pcp", "nat-pmp", "upnp"}},
		{Name: "natpmp", WAN: "public", Address: "203.0.113.20", Mapping: []string{"nat-pmp", "upnp"}},
		{Name: "igd", WAN: "public", Address: "203.0.113.30", Mapping: []string{"upnp"}},
		{Name: "permanent", WAN: "public", Address: "203.0.113.40", Mapping: []string{"upnp"}, Refuse: "725"},
		{Name: "locked", WAN: "public", Address: "203.0.113.50", Mapping: []string{"upnp"}, Refuse: "606"},
		{Name: "other", WAN: "public", Address: "203.0.113.60", Mapping: []string{"pcp", "upnp"}, Refuse: "other-port"},
	}}}
	for _, n := range origins {
		s.Nodes = append(s.Nodes, scenario.Node{Name: n, Setup: scenario.SetupJoin, Site: site[n]})
	}
	s.Nodes[0].Setup = scenario.SetupController
	var fronts fake.Fronts
	for i, n := range append(slices.Clone(origins), events.World) {
		fronts = append(fronts, fake.Front{Target: "gateway-igd", Node: n, Address: fake.Address{AddrPort: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(42000+i))}})
	}
	shared := fake.NewShared()
	opts := faketest.WithOrigins(append(slices.Clone(origins), events.World)...)
	udp := faketest.UDP(t, fakegateway.Fake, s, opts, faketest.WithShared(shared), faketest.WithFronts(fronts))
	igd := faketest.InMemory(t, fakegateway.IGD, s, opts, faketest.WithShared(shared), faketest.WithFronts(fronts))
	for _, c := range []mapCase{
		{origin: "atlas", method: netmodel.MapPCP, lifetime: time.Hour},
		{origin: "den", method: netmodel.MapNATPMP, lifetime: time.Hour},
		{origin: "nas", method: netmodel.MapUPnP, lifetime: time.Hour},
		{origin: "pi", method: netmodel.MapUPnP},
		{origin: "box", method: netmodel.MapUPnP, refused: netmodel.NotAuthorized},
		{origin: "seedbox", method: netmodel.MapPCP, refused: netmodel.OtherPort},
		{origin: "pc", method: netmodel.MapPCP, silent: true},
	} {
		client := &gateway.Client{Packets: loopback{}, HTTP: igd.From(t, c.origin), Timings: gateway.Timings{Exchange: 200 * time.Millisecond, SSDPWait: 200 * time.Millisecond}}
		gw, err := client.Toward(t.Context(), udp.Sockets[c.origin], true)
		if err != nil {
			t.Fatal(err)
		}
		checkMapping(t, client, gw, c)
	}
}

type mapCase struct {
	origin   string
	method   netmodel.MapProtocol
	lifetime time.Duration
	refused  netmodel.MapRefusal
	silent   bool
}

// checkMapping maps port 32400 on gw as c asks, renews it and deletes it
// twice, and reports each way the router's answer or its list misses c.
func checkMapping(t *testing.T, client *gateway.Client, gw gateway.Target, c mapCase) {
	t.Helper()
	ask := gateway.Ask{Method: c.method, External: 32400, Internal: 32400, Lifetime: time.Hour, Nonce: [12]byte{7}, Description: "SlipStream test"}
	m, err := client.Map(t.Context(), gw, ask)
	switch {
	case c.silent:
		if !errors.Is(err, gateway.ErrNoAnswer) {
			t.Errorf("%s: a mapping by a protocol its router doesn't speak answers %v, want no answer", c.origin, err)
		}
		return
	case err != nil || m.Refused != c.refused || m.Refused == netmodel.RefusedNone && m.Lifetime != c.lifetime:
		t.Errorf("%s: the mapping is %+v, %v; want refused %v, lifetime %v", c.origin, m, err, c.refused, c.lifetime)
	case c.refused != netmodel.RefusedNone:
		if n := forwardsTo(t, client, gw); n != 0 {
			t.Errorf("%s: after a refusal the router lists %d forwards to the port, want none", c.origin, n)
		}
		return
	}
	if _, err := client.Map(t.Context(), gw, ask); err != nil {
		t.Errorf("%s: renew: %v", c.origin, err)
	}
	if n := forwardsTo(t, client, gw); n != 1 {
		t.Errorf("%s: mapped and renewed, the router lists %d forwards to the port, want one", c.origin, n)
	}
	for range 2 {
		if err := client.Unmap(t.Context(), gw, ask); err != nil {
			t.Errorf("%s: unmap: %v", c.origin, err)
		}
	}
	if n := forwardsTo(t, client, gw); n != 0 {
		t.Errorf("%s: unmapped, the router lists %d forwards to the port, want none", c.origin, n)
	}
}

// forwardsTo is how many forwards to port 32400 the router lists.
func forwardsTo(t *testing.T, client *gateway.Client, gw gateway.Target) int {
	t.Helper()
	g, err := client.Read(t.Context(), gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range g.Forwards {
		if f.Internal == 32400 {
			n++
		}
	}
	return n
}
