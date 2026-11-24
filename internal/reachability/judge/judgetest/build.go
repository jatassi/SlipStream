// Package judgetest builds judge.Facts in the shapes of the remote-*
// Scenarios, holds the verdict table TestJudge and words_test.go share, and
// generates the Facts FuzzJudge reads.
package judgetest

import (
	"net/netip"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func id(b byte) role.NodeID {
	u, err := storage.UUIDFromBytes([]byte{0x01, 0x9a, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, b})
	if err != nil {
		panic(err)
	}
	return role.NodeID{UUID: u}
}

// The Nodes of every shape: atlas is the Controller at home, den a Worker
// at home, seedbox a Worker in another site, the-vps the Admin's proxy
// paired as a Worker.
var (
	Atlas   = id(1)
	Den     = id(2)
	Seedbox = id(3)
	VPS     = id(4)
	Names   = map[role.NodeID]string{Atlas: "atlas", Den: "den", Seedbox: "seedbox", VPS: "the-vps"}
)

var (
	Now        = time.Date(2026, 11, 24, 21, 38, 0, 0, time.UTC)
	HomeWAN    = ip("203.0.113.9")
	AtlasLAN   = ip("192.168.1.20")
	SeedboxIP  = ip("192.0.2.44")
	ProxyIP    = ip("198.51.100.8")
	AtlasV6    = ip("2001:db8:1::20")
	SeedboxV6  = ip("2001:db8:9::44")
	Expires    = time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	PlexServer = link.ServerID{UUID: id(9).UUID}
)

func ip(s string) netip.Addr { return netip.MustParseAddr(s) }

// B builds one judge.Facts. Each method changes one thing and returns B.
type B struct{ f judge.Facts }

func (b *B) Facts() judge.Facts { return b.f }

func base(public string) *B {
	b := &B{f: judge.Facts{
		Rules:  netmodel.Rules{DocumentationGlobal: true},
		Origin: judge.Origin{Node: Atlas, Label: judge.RunsOn},
		Nodes:  map[role.NodeID]netmodel.NetworkFacts{},
		Paired: map[netip.Addr]role.NodeID{},
		Pins:   map[role.NodeID]netip.Addr{},
		Churn:  map[role.NodeID]bool{},
	}}
	b.Plex(public)
	b.node(Atlas, true, homeLAN(netmodel.Mappable))
	b.ServerListens(netmodel.Yes, netmodel.Yes).Firewall(netmodel.Allows)
	return b
}

// Plex makes the Exposure a Plex Media Server on atlas at public.
func (b *B) Plex(public string) *B {
	u, err := url.Parse(public)
	if err != nil {
		panic(err)
	}
	b.f.Exposure = judge.Exposure{ID: judge.ExposureID{UUID: id(8).UUID}, Kind: judge.MediaServer, Server: PlexServer, Public: u, Local: 32400}
	b.f.Origin = judge.Origin{Node: Atlas, Label: judge.RunsOn}
	b.f.Server = &judge.ServerFacts{Kind: judge.Plex, Name: "Plex", Custom: []string{public}}
	return b
}

// UI makes the Exposure the UI and portal at public, served by atlas on
// port 8484.
func (b *B) UI(public string) *B {
	u, err := url.Parse(public)
	if err != nil {
		panic(err)
	}
	b.f.Exposure = judge.Exposure{ID: judge.ExposureID{UUID: id(7).UUID}, Kind: judge.UIPortal, Public: u, Local: 8484}
	b.f.Origin = judge.Origin{Node: Atlas, Label: judge.Controller}
	b.f.Server = nil
	nf := b.f.Nodes[Atlas]
	nf.Listeners = []netmodel.Listener{{Port: 8484, Program: netmodel.Program{Name: "slipstream", SlipStream: true}, V4: netmodel.Yes, V6: netmodel.Yes}}
	nf.Firewall = rules(8484, netmodel.Allows)
	b.f.Nodes[Atlas] = nf
	return b
}

func rules(port uint16, p netmodel.Policy) []netmodel.Rule {
	return []netmodel.Rule{
		{Tool: netmodel.WindowsDefender, Port: port, Family: netmodel.IPv4, Policy: p},
		{Tool: netmodel.WindowsDefender, Port: port, Family: netmodel.IPv6, Policy: p},
	}
}

func iface(name string, tunnel bool, prefixes ...string) netmodel.Iface {
	i := netmodel.Iface{Name: name, Tunnel: tunnel}
	for _, p := range prefixes {
		i.Addrs = append(i.Addrs, netmodel.IfaceAddr{Prefix: netip.MustParsePrefix(p)})
	}
	return i
}

func stun(a netip.Addr) []netmodel.STUNAnswer {
	return []netmodel.STUNAnswer{
		{Server: "stun.fake.localhost:3478", Mapped: netip.AddrPortFrom(a, 40000)},
		{Server: "stun-2.fake.localhost:3478", Mapped: netip.AddrPortFrom(a, 40000)},
	}
}

// homeLAN is atlas's network observed so that Classify gives c.
func homeLAN(c netmodel.Class4) netmodel.NetworkFacts {
	nf := netmodel.NetworkFacts{
		Node: Atlas, At: Now, Watched: true,
		Interfaces: []netmodel.Iface{iface("eth0", false, "192.168.1.20/24"), iface("wg0", true, "10.8.0.2/24")},
		Egress:     netmodel.Egress{Iface: "eth0", Local: AtlasLAN},
		Gateway:    netmodel.Gateway{Addr: ip("192.168.1.1"), WAN: HomeWAN, Protocols: netmodel.Protocols{UPnP: netmodel.Yes, NATPMP: netmodel.No, PCP: netmodel.No}},
		STUN:       stun(HomeWAN),
	}
	switch c {
	case netmodel.Mappable:
	case netmodel.Open:
		nf.Interfaces = []netmodel.Iface{iface("eth0", false, "203.0.113.9/24")}
		nf.Egress.Local = HomeWAN
		nf.Gateway = netmodel.Gateway{}
	case netmodel.ThroughVPN:
		nf.Egress = netmodel.Egress{Iface: "wg0", Local: ip("10.8.0.2")}
		nf.STUN = stun(ip("198.51.100.77"))
	case netmodel.CGNAT:
		nf.Gateway.WAN = ip("100.64.12.7")
		nf.STUN = stun(ip("203.0.113.50"))
	case netmodel.DoubleNAT:
		nf.Gateway.WAN = ip("192.168.0.5")
	case netmodel.BehindRouter:
		nf.Gateway = netmodel.Gateway{Addr: ip("192.168.1.1"), Protocols: netmodel.Protocols{UPnP: netmodel.No, NATPMP: netmodel.No, PCP: netmodel.No}}
	case netmodel.Class4Unknown:
		nf.STUN = nil
		nf.Egress = netmodel.Egress{}
		nf.Gateway = netmodel.Gateway{}
	}
	return nf
}

func (b *B) node(n role.NodeID, self bool, nf netmodel.NetworkFacts) {
	nf.Node = n
	old := b.f.Nodes[n]
	nf.Listeners, nf.Firewall, nf.Mappings, nf.Previous4, nf.Previous6 = old.Listeners, old.Firewall, old.Mappings, old.Previous4, old.Previous6
	b.f.Nodes[n] = nf
	b.f.Candidates = slices.DeleteFunc(b.f.Candidates, func(c judge.Candidate) bool { return c.Node == n })
	c := judge.Candidate{Node: n, Name: Names[n], InStep: true, Self: self}
	if !self {
		c.Link = judge.LinkSeen{Addr: b.f.Rules.Public4(nf.Observed())}
	}
	b.f.Candidates = append(b.f.Candidates, c)
	for _, i := range nf.Interfaces {
		for _, a := range i.Addrs {
			if b.f.Rules.Global(a.Prefix.Addr()) {
				b.f.Paired[a.Prefix.Addr()] = n
			}
		}
	}
}

// Class makes atlas's network classify as c.
func (b *B) Class(c netmodel.Class4) *B {
	b.node(Atlas, true, homeLAN(c))
	return b
}

// Outside pairs seedbox, an Open Worker in another site, with a Global
// IPv6 address of its own.
func (b *B) Outside() *B {
	b.node(Seedbox, false, netmodel.NetworkFacts{
		Watched:    true,
		Interfaces: []netmodel.Iface{iface("eth0", false, "192.0.2.44/24", "2001:db8:9::44/64")},
		Egress:     netmodel.Egress{Iface: "eth0", Local: SeedboxIP},
		Gateway:    netmodel.Gateway{Addr: ip("192.0.2.1")},
		STUN:       stun(SeedboxIP),
	})
	return b
}

// Inside pairs den, a Worker on atlas's home network.
func (b *B) Inside() *B {
	nf := homeLAN(netmodel.Mappable)
	nf.Interfaces = []netmodel.Iface{iface("eth0", false, "192.168.1.30/24")}
	nf.Egress.Local = ip("192.168.1.30")
	b.node(Den, false, nf)
	return b
}

// Proxy pairs the-vps, the Admin's own proxy, as a Worker.
func (b *B) Proxy() *B {
	b.node(VPS, false, netmodel.NetworkFacts{
		Watched:    true,
		Interfaces: []netmodel.Iface{iface("eth0", false, "198.51.100.8/24")},
		Egress:     netmodel.Egress{Iface: "eth0", Local: ProxyIP},
		Gateway:    netmodel.Gateway{Addr: ip("198.51.100.1")},
		STUN:       stun(ProxyIP),
	})
	return b
}

// V6 gives atlas the Global address prefix, such as 2001:db8:1::20/64.
func (b *B) V6(prefix string) *B {
	nf := b.f.Nodes[Atlas]
	nf.Interfaces = append(slices.Clone(nf.Interfaces), iface("eth1", false, prefix))
	b.f.Nodes[Atlas] = nf
	b.f.Paired[netip.MustParsePrefix(prefix).Addr()] = Atlas
	return b
}

// Resolves sets what every paired Node resolves the public host to.
func (b *B) Resolves(as ...string) *B {
	b.f.DNS = nil
	for _, c := range b.f.Candidates {
		d := judge.DNSAnswer{By: c.Node}
		for _, s := range as {
			if a := ip(s); a.Is4() {
				d.A = append(d.A, a)
			} else {
				d.AAAA = append(d.AAAA, a)
			}
		}
		b.f.DNS = append(b.f.DNS, d)
	}
	return b
}

// ResolvesAt sets one Node's answer apart from the others.
func (b *B) ResolvesAt(n role.NodeID, as ...string) *B {
	b.f.DNS = slices.DeleteFunc(b.f.DNS, func(d judge.DNSAnswer) bool { return d.By == n })
	d := judge.DNSAnswer{By: n}
	for _, s := range as {
		if a := ip(s); a.Is4() {
			d.A = append(d.A, a)
		} else {
			d.AAAA = append(d.AAAA, a)
		}
	}
	b.f.DNS = append(b.f.DNS, d)
	return b
}

// DNSError makes every resolver fail the A or the AAAA lookup.
func (b *B) DNSError(fam netmodel.Family, e judge.DNSError) *B {
	for i := range b.f.DNS {
		if fam == netmodel.IPv4 {
			b.f.DNS[i].A, b.f.DNS[i].ErrA = nil, e
		} else {
			b.f.DNS[i].AAAA, b.f.DNS[i].ErrAAAA = nil, e
		}
	}
	return b
}

func (b *B) atlas(fn func(*netmodel.NetworkFacts)) *B {
	nf := b.f.Nodes[Atlas]
	fn(&nf)
	b.f.Nodes[Atlas] = nf
	return b
}

// ServerListens sets the service's own socket's families.
func (b *B) ServerListens(v4, v6 netmodel.Tri) *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		port := b.f.Exposure.Local
		nf.Listeners = slices.DeleteFunc(slices.Clone(nf.Listeners), func(l netmodel.Listener) bool { return l.Port == port })
		nf.Listeners = append(nf.Listeners, netmodel.Listener{Port: port, Program: program(b.f.Exposure), V4: v4, V6: v6})
	})
}

func program(e judge.Exposure) netmodel.Program {
	if e.Kind == judge.UIPortal {
		return netmodel.Program{Name: "slipstream", SlipStream: true}
	}
	return netmodel.Program{Name: "Plex Media Server"}
}

// Loopback binds the service to loopback only.
func (b *B) Loopback() *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		for i := range nf.Listeners {
			if nf.Listeners[i].Port == b.f.Exposure.Local {
				nf.Listeners[i].Loopback, nf.Listeners[i].V4, nf.Listeners[i].V6 = true, netmodel.No, netmodel.No
			}
		}
	})
}

// Caddy puts another program on the public port, in front of the service.
func (b *B) Caddy() *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		port := b.f.Exposure.PublicPort()
		nf.Listeners = append(slices.Clone(nf.Listeners), netmodel.Listener{Port: port, Program: netmodel.Program{Name: "caddy"}, V4: netmodel.Yes, V6: netmodel.Yes})
		nf.Firewall = append(slices.Clone(nf.Firewall), rules(port, netmodel.Allows)...)
	})
}

// Firewall sets atlas's answer for the service's port on both families.
func (b *B) Firewall(p netmodel.Policy) *B {
	return b.FirewallFor(netmodel.IPv4, p).FirewallFor(netmodel.IPv6, p)
}

func (b *B) FirewallFor(fam netmodel.Family, p netmodel.Policy) *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		port := b.f.Exposure.Local
		nf.Firewall = slices.DeleteFunc(slices.Clone(nf.Firewall), func(r netmodel.Rule) bool { return r.Port == port && r.Family == fam })
		nf.Firewall = append(nf.Firewall, netmodel.Rule{Tool: netmodel.WindowsDefender, Port: port, Family: fam, Policy: p})
	})
}

// FirewalldPartial makes firewalld allow the service's port while its
// other rules are unread.
func (b *B) FirewalldPartial() *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		port := b.f.Exposure.Local
		nf.Firewall = []netmodel.Rule{
			{Tool: netmodel.Firewalld, Port: port, Family: netmodel.IPv4, Policy: netmodel.Allows, Partial: true},
			{Tool: netmodel.Firewalld, Port: port, Family: netmodel.IPv6, Policy: netmodel.Allows, Partial: true},
		}
	})
}

// STUNVaries makes the second STUN server see atlas on another port, or
// at another address.
func (b *B) STUNVaries(address bool) *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		if len(nf.STUN) < 2 {
			return
		}
		nf.STUN = slices.Clone(nf.STUN)
		m := nf.STUN[0].Mapped
		nf.STUN[1].Mapped = netip.AddrPortFrom(m.Addr(), m.Port()+1)
		if address {
			nf.STUN[1].Mapped = netip.AddrPortFrom(ip("203.0.113.10"), m.Port())
		}
	})
}

// Docker marks the service's port as published by docker-proxy, past ufw.
func (b *B) Docker() *B {
	b.Firewall(netmodel.PolicyUnknown)
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		for i := range nf.Listeners {
			nf.Listeners[i].Docker = true
		}
	})
}

// Mapped gives atlas SlipStream's own granted mapping of the public port.
func (b *B) Mapped(lifetime time.Duration) *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		p := b.f.Exposure.PublicPort()
		nf.Mappings = []netmodel.Mapping{{Protocol: netmodel.MapUPnP, External: p, Internal: p, Lifetime: lifetime}}
	})
}

// Refused makes the gateway refuse SlipStream's mapping the way r says.
func (b *B) Refused(r netmodel.MapRefusal) *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) {
		p := b.f.Exposure.PublicPort()
		nf.Mappings = []netmodel.Mapping{{Protocol: netmodel.MapPCP, External: p, Internal: p, Refused: r}}
	})
}

// PlexMapping sets Plex's own mapping state at the home's address.
func (b *B) PlexMapping(state string, pending bool) *B {
	b.f.Server.Mapping = &judge.PlexMapping{State: state, Pending: pending, Public: netip.AddrPortFrom(HomeWAN, b.f.Exposure.PublicPort())}
	return b
}

// Custom sets Plex's custom server access URLs.
func (b *B) Custom(urls ...string) *B {
	b.f.Server.Custom = urls
	return b
}

func (b *B) Previous4(a string) *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) { nf.Previous4 = ip(a) })
}

func (b *B) Previous6(p string) *B {
	return b.atlas(func(nf *netmodel.NetworkFacts) { nf.Previous6 = []netip.Prefix{netip.MustParsePrefix(p)} })
}

func (b *B) Pin(a string) *B {
	b.f.Pins[Atlas] = ip(a)
	return b
}

func (b *B) Churn() *B {
	b.f.Churn[Atlas] = true
	return b
}

// Away takes n out of step with the Controller.
func (b *B) Away(n role.NodeID) *B {
	for i := range b.f.Candidates {
		if b.f.Candidates[i].Node == n {
			b.f.Candidates[i].InStep = false
		}
	}
	return b
}

// LinkFrom sets the address n's Node link arrives from.
func (b *B) LinkFrom(n role.NodeID, a string) *B {
	for i := range b.f.Candidates {
		if b.f.Candidates[i].Node == n {
			b.f.Candidates[i].Link = judge.LinkSeen{Addr: ip(a)}
		}
	}
	return b
}

// Serves marks n as a Worker that serves the Exposure.
func (b *B) Serves(n role.NodeID) *B {
	for i := range b.f.Candidates {
		if b.f.Candidates[i].Node == n {
			b.f.Candidates[i].Serves = true
		}
	}
	return b
}

// Reaching makes the Media Server run on no Node, read through atlas.
func (b *B) Reaching() *B {
	b.f.Origin.Label = judge.Reaching
	return b
}

func pfx(s string) netip.Prefix { return netip.MustParsePrefix(s) }
