package gateway_test

import (
	"context"
	"encoding/json"
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
			Forwards: []netmodel.Forward{{External: 32400, Client: netip.MustParseAddr("192.168.1.10"), Internal: 32400}},
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
