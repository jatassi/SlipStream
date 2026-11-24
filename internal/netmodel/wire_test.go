package netmodel_test

import (
	"net/netip"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Facts cross the wire unchanged, under the id of the Node the Controller
// asked whatever the Node is, and a summary keeps its revision.
func TestFactsCrossTheWire(t *testing.T) {
	r := netmodel.Rules{DocumentationGlobal: true}
	asked := role.NodeID{UUID: storage.NewUUID()}
	sent := netmodel.NetworkFacts{
		Node:       role.NodeID{UUID: storage.NewUUID()},
		At:         time.UnixMilli(1_700_000_000_000),
		Watched:    true,
		Interfaces: []netmodel.Iface{{Name: "eth0", Index: 2, Addrs: []netmodel.IfaceAddr{{Prefix: pfx("192.168.1.20/24"), Class: netmodel.Private}, {Prefix: pfx("2001:db8:1::20/64"), Class: netmodel.Global, Temporary: netmodel.No}}}, {Name: "wg0", Tunnel: true}},
		Egress:     netmodel.Egress{Iface: "eth0", Local: ip("192.168.1.20")},
		Gateway: netmodel.Gateway{
			Addr: ip("192.168.1.1"), WAN: ip("203.0.113.9"), Protocols: netmodel.Protocols{PCP: netmodel.No, NATPMP: netmodel.Yes, UPnP: netmodel.Yes},
			Forwards: []netmodel.Forward{{External: 32400, Client: ip("192.168.1.10"), Internal: 32400}},
		},
		STUN:      []netmodel.STUNAnswer{{Server: "stun.example:3478", Mapped: netip.MustParseAddrPort("203.0.113.9:50000")}, {Server: "stun2.example:3478", Err: netmodel.STUNTimeout}},
		Previous4: ip("203.0.113.8"),
		Listeners: []netmodel.Listener{{Port: 8484, Program: netmodel.Program{Name: "SlipStream", SlipStream: true}, V4: netmodel.Yes, V6: netmodel.Yes}, {Port: 32400, Docker: true, Loopback: true}},
		Firewall:  []netmodel.Rule{{Tool: netmodel.Firewalld, Port: 32400, Family: netmodel.IPv6, Policy: netmodel.Allows, Partial: true, Because: "public zone"}},
		Mappings:  []netmodel.Mapping{{Protocol: netmodel.MapNATPMP, External: 8484, Internal: 8484, Lifetime: time.Hour, Expires: time.UnixMilli(1_700_003_600_000)}, {Protocol: netmodel.MapUPnP, External: 32400, Refused: netmodel.NotAuthorized}},
	}
	got, err := netmodel.ParseFacts(asked, netmodel.FactsOnWire(sent), r)
	want := sent
	want.Node, want.Previous4 = asked, netmodel.NetworkFacts{}.Previous4
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("parsed %+v, %v; want %+v", got, err, want)
	}

	s := r.Summary(sent)
	back, err := netmodel.ParseSummary(netmodel.SummaryOnWire(s, sent.At))
	if err != nil || !reflect.DeepEqual(back, s) {
		t.Errorf("summary %+v, %v; want %+v", back, err, s)
	}

	for name, m := range map[string]*netv1.NetworkFacts{
		"a short address":     {Egress: &netv1.Egress{Local: []byte{1, 2, 3}}},
		"a port past 16 bits": {Listeners: []*netv1.Listener{{Port: 70000}}},
		"an unknown tri":      {Listeners: []*netv1.Listener{{Port: 1, V4: 7}}},
		"no family":           {Firewall: []*netv1.Rule{{Port: 1}}},
	} {
		if _, err := netmodel.ParseFacts(asked, m, r); err == nil {
			t.Errorf("%s parsed", name)
		}
	}
}
