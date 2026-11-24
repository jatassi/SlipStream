package netmodel_test

import (
	"net/netip"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/netmodel"
)

var ip = netip.MustParseAddr

func pfx(s string) netip.Prefix { return netip.MustParsePrefix(s) }

func TestAddrClass(t *testing.T) {
	for _, c := range []struct {
		addr    string
		off, on netmodel.AddrClass
	}{
		{"8.8.8.8", netmodel.Global, netmodel.Global},
		{"2606:4700::1111", netmodel.Global, netmodel.Global},
		{"192.168.1.20", netmodel.Private, netmodel.Private},
		{"10.0.0.1", netmodel.Private, netmodel.Private},
		{"100.64.12.7", netmodel.Shared, netmodel.Shared},
		{"fd72::20", netmodel.ULA, netmodel.ULA},
		{"fe80::1", netmodel.LinkLocal, netmodel.LinkLocal},
		{"169.254.1.1", netmodel.LinkLocal, netmodel.LinkLocal},
		{"127.0.0.3", netmodel.Loopback, netmodel.Loopback},
		{"::1", netmodel.Loopback, netmodel.Loopback},
		{"2002:c000:204::1", netmodel.SixToFour, netmodel.SixToFour},
		{"2001:0:4136:e378::1", netmodel.Teredo, netmodel.Teredo},
		{"64:ff9b::808:808", netmodel.NAT64, netmodel.NAT64},
		{"64:ff9b:1::1", netmodel.NAT64, netmodel.NAT64},
		{"::ffff:8.8.8.8", netmodel.Mapped, netmodel.Mapped},
		{"203.0.113.9", netmodel.Documentation, netmodel.Global},
		{"198.51.100.8", netmodel.Documentation, netmodel.Global},
		{"192.0.2.44", netmodel.Documentation, netmodel.Global},
		{"2001:db8:1::20", netmodel.Documentation, netmodel.Global},
		{"0.0.0.0", netmodel.Other, netmodel.Other},
		{"224.0.0.1", netmodel.Other, netmodel.Other},
		{"ff02::1", netmodel.Other, netmodel.Other},
	} {
		a := ip(c.addr)
		if got := (netmodel.Rules{}).Of(a); got != c.off {
			t.Errorf("Of(%s) = %d; want %d", c.addr, got, c.off)
		}
		if got := (netmodel.Rules{DocumentationGlobal: true}).Of(a); got != c.on {
			t.Errorf("Of(%s) under DocumentationGlobal = %d; want %d", c.addr, got, c.on)
		}
	}
	if (netmodel.Rules{}).Of(netip.Addr{}) != netmodel.Other {
		t.Error("the zero address isn't Other")
	}
}

func iface(name string, tunnel bool, addrs ...string) netmodel.Iface {
	i := netmodel.Iface{Name: name, Tunnel: tunnel}
	for _, a := range addrs {
		i.Addrs = append(i.Addrs, netmodel.IfaceAddr{Prefix: pfx(a)})
	}
	return i
}

func stun(addrs ...string) []netmodel.STUNAnswer {
	var out []netmodel.STUNAnswer
	for _, a := range addrs {
		out = append(out, netmodel.STUNAnswer{Server: "stun.fake.localhost:3478", Mapped: netip.MustParseAddrPort(a)})
	}
	return out
}

func TestClassify(t *testing.T) {
	lan := []netmodel.Iface{iface("eth0", false, "192.168.1.20/24"), iface("wg0", true, "10.8.0.2/24")}
	egress := netmodel.Egress{Iface: "eth0", Local: ip("192.168.1.20")}
	upnp := netmodel.Protocols{UPnP: netmodel.Yes}
	gw := func(wan string, p netmodel.Protocols) netmodel.Gateway {
		g := netmodel.Gateway{Addr: ip("192.168.1.1"), Protocols: p}
		if wan != "" {
			g.WAN = ip(wan)
		}
		return g
	}
	for _, c := range []struct {
		name    string
		o       netmodel.Observed
		want    netmodel.Class4
		notes   []netmodel.Note
		public4 string
	}{
		{"an interface holds the STUN answer", netmodel.Observed{
			Interfaces: []netmodel.Iface{iface("eth0", false, "203.0.113.9/24")}, Egress: netmodel.Egress{Iface: "eth0"},
			STUN: stun("203.0.113.9:40000")}, netmodel.Open, nil, "203.0.113.9"},
		{"the egress is a tunnel, before CGNAT is tried", netmodel.Observed{
			Interfaces: lan, Egress: netmodel.Egress{Iface: "wg0"}, Gateway: gw("100.64.12.7", upnp),
			STUN: stun("198.51.100.77:40000")}, netmodel.ThroughVPN, nil, ""},
		{"a tunnel that doesn't carry the egress", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("203.0.113.9", upnp),
			STUN: stun("203.0.113.9:40000")}, netmodel.Mappable, nil, "203.0.113.9"},
		{"a gateway WAN in 100.64.0.0/10", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("100.64.12.7", upnp),
			STUN: stun("203.0.113.50:40000")}, netmodel.CGNAT, nil, "203.0.113.50"},
		{"a private gateway WAN", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("192.168.0.5", upnp),
			STUN: stun("203.0.113.9:40000")}, netmodel.DoubleNAT, nil, "203.0.113.9"},
		{"a public gateway WAN that differs from STUN", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("203.0.113.70", netmodel.Protocols{}),
			STUN: stun("203.0.113.9:40000")}, netmodel.DoubleNAT, nil, "203.0.113.9"},
		{"WAN equals STUN and NAT-PMP answered", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("203.0.113.9", netmodel.Protocols{NATPMP: netmodel.Yes}),
			STUN: stun("203.0.113.9:40000")}, netmodel.Mappable, nil, "203.0.113.9"},
		{"WAN equals STUN and no protocol answered", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("203.0.113.9", netmodel.Protocols{UPnP: netmodel.No}),
			STUN: stun("203.0.113.9:40000")}, netmodel.BehindRouter, nil, "203.0.113.9"},
		{"the gateway hid its WAN", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("", upnp),
			STUN: stun("203.0.113.9:40000")}, netmodel.BehindRouter, nil, "203.0.113.9"},
		{"no gateway", netmodel.Observed{
			Interfaces: lan, Egress: egress, STUN: stun("203.0.113.9:40000")}, netmodel.BehindRouter, nil, "203.0.113.9"},
		{"STUN answers differ by port", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("203.0.113.9", upnp),
			STUN: stun("203.0.113.9:40000", "203.0.113.9:40001")}, netmodel.Mappable, []netmodel.Note{netmodel.STUNVariesPort}, "203.0.113.9"},
		{"STUN answers differ by address", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("203.0.113.9", upnp),
			STUN: stun("203.0.113.9:40000", "203.0.113.10:40000")}, netmodel.BehindRouter, []netmodel.Note{netmodel.STUNVariesAddress}, "203.0.113.9"},
		{"no STUN answer", netmodel.Observed{
			Interfaces: lan, Egress: egress, Gateway: gw("203.0.113.9", upnp),
			STUN: []netmodel.STUNAnswer{{Server: "stun.fake.localhost:3478", Err: netmodel.STUNTimeout}}}, netmodel.Class4Unknown, nil, ""},
	} {
		r := netmodel.Rules{DocumentationGlobal: true}
		got, notes := r.Classify(c.o)
		if got != c.want || !slices.Equal(notes, c.notes) {
			t.Errorf("%s: Classify = %d %v; want %d %v", c.name, got, notes, c.want, c.notes)
		}
		var want4 netip.Addr
		if c.public4 != "" {
			want4 = ip(c.public4)
		}
		if got := r.Public4(c.o); got != want4 {
			t.Errorf("%s: Public4 = %v; want %v", c.name, got, want4)
		}
	}
}

func TestClass6(t *testing.T) {
	r := netmodel.Rules{DocumentationGlobal: true}
	for _, c := range []struct {
		addrs    []string
		want     netmodel.Class6
		prefixes []netip.Prefix
	}{
		{[]string{"192.168.1.20/24"}, netmodel.Class6None, nil},
		{[]string{"fe80::1/64", "fd72::20/64"}, netmodel.LocalOnly, nil},
		{[]string{"fe80::1/64", "2001:db8:1::20/64", "2001:db8:1::abcd/64", "2001:db8:2::1/64"}, netmodel.Global6,
			[]netip.Prefix{pfx("2001:db8:1::/64"), pfx("2001:db8:2::/64")}},
	} {
		ifs := []netmodel.Iface{iface("eth0", false, c.addrs...)}
		if got := r.Class6(ifs); got != c.want {
			t.Errorf("Class6(%v) = %d; want %d", c.addrs, got, c.want)
		}
		if got := r.Prefixes6(ifs); !slices.Equal(got, c.prefixes) {
			t.Errorf("Prefixes6(%v) = %v; want %v", c.addrs, got, c.prefixes)
		}
	}
}

func TestCarry(t *testing.T) {
	r := netmodel.Rules{DocumentationGlobal: true}
	facts := func(public4 string, v6 ...string) netmodel.NetworkFacts {
		f := netmodel.NetworkFacts{
			Interfaces: []netmodel.Iface{iface("eth0", false, append([]string{"192.168.1.20/24"}, v6...)...)},
			Egress:     netmodel.Egress{Iface: "eth0"},
		}
		if public4 != "" {
			f.STUN = stun(public4 + ":40000")
		}
		return f
	}
	p1, p2 := pfx("2001:db8:1::/64"), pfx("2001:db8:2::/64")
	for _, c := range []struct {
		name   string
		stored netmodel.Stored
		fresh  netmodel.NetworkFacts
		want   netmodel.Stored
	}{
		{"a first read", netmodel.Stored{}, facts("203.0.113.9", "2001:db8:1::20/64"),
			netmodel.Stored{Public4: ip("203.0.113.9"), Prefixes6: []netip.Prefix{p1}}},
		{"no change keeps the previous addresses",
			netmodel.Stored{Public4: ip("203.0.113.9"), Previous4: ip("203.0.113.7"), Prefixes6: []netip.Prefix{p1}, Previous6: []netip.Prefix{p2}},
			facts("203.0.113.9", "2001:db8:1::abcd/64"),
			netmodel.Stored{Public4: ip("203.0.113.9"), Previous4: ip("203.0.113.7"), Prefixes6: []netip.Prefix{p1}, Previous6: []netip.Prefix{p2}}},
		{"a new Public4 and a new /64",
			netmodel.Stored{Public4: ip("203.0.113.9"), Prefixes6: []netip.Prefix{p1}},
			facts("203.0.113.10", "2001:db8:2::20/64"),
			netmodel.Stored{Public4: ip("203.0.113.10"), Previous4: ip("203.0.113.9"), Prefixes6: []netip.Prefix{p2}, Previous6: []netip.Prefix{p1}}},
		{"an unknown Public4 keeps the last known one",
			netmodel.Stored{Public4: ip("203.0.113.9"), Previous4: ip("203.0.113.7")},
			facts(""),
			netmodel.Stored{Public4: ip("203.0.113.9"), Previous4: ip("203.0.113.7")}},
		{"a /64 added keeps nothing previous",
			netmodel.Stored{Public4: ip("203.0.113.9"), Prefixes6: []netip.Prefix{p1}},
			facts("203.0.113.9", "2001:db8:1::20/64", "2001:db8:2::20/64"),
			netmodel.Stored{Public4: ip("203.0.113.9"), Prefixes6: []netip.Prefix{p1, p2}}},
	} {
		got, row := r.Carry(c.stored, c.fresh)
		if row.Public4 != c.want.Public4 || row.Previous4 != c.want.Previous4 ||
			!slices.Equal(row.Prefixes6, c.want.Prefixes6) || !slices.Equal(row.Previous6, c.want.Previous6) {
			t.Errorf("%s: row = %+v; want %+v", c.name, row, c.want)
		}
		if got.Previous4 != row.Previous4 || !slices.Equal(got.Previous6, row.Previous6) {
			t.Errorf("%s: facts carry %v %v; the row has %v %v", c.name, got.Previous4, got.Previous6, row.Previous4, row.Previous6)
		}
	}
}

func TestSummaryRevision(t *testing.T) {
	r := netmodel.Rules{DocumentationGlobal: true}
	facts := func(public4 string, addrs ...string) netmodel.NetworkFacts {
		return netmodel.NetworkFacts{
			Watched:    true,
			Interfaces: []netmodel.Iface{iface("eth0", false, addrs...)},
			Egress:     netmodel.Egress{Iface: "eth0"},
			STUN:       stun(public4 + ":40000"),
		}
	}
	base := r.Summary(facts("203.0.113.9", "192.168.1.20/24", "2001:db8:1::20/64"))
	for _, c := range []struct {
		name string
		f    netmodel.NetworkFacts
		same bool
	}{
		{"a rotated address inside the /64", facts("203.0.113.9", "192.168.1.21/24", "2001:db8:1::99/64"), true},
		{"a new Public4", facts("203.0.113.10", "192.168.1.20/24", "2001:db8:1::20/64"), false},
		{"a new /64", facts("203.0.113.9", "192.168.1.20/24", "2001:db8:2::20/64"), false},
		{"a new class", facts("203.0.113.9", "203.0.113.9/24", "2001:db8:1::20/64"), false},
	} {
		if got := r.Summary(c.f).Revision == base.Revision; got != c.same {
			t.Errorf("%s: same revision = %v; want %v", c.name, got, c.same)
		}
	}
}
