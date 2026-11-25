package netsite

import (
	"net/netip"
	"slices"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// stunPort is the first port a site's NAT maps a Node's STUN socket to,
// one per Node so two Nodes behind one address differ.
const stunPort = 50000

// Mapped is the address and port fake-stun answers origin, from its first
// server or its second, and false when the origin's site plays a STUN
// timeout. A site that plays a varying mapping answers the second server
// another port.
func (m *Model) Mapped(origin string, second bool) (netip.AddrPort, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.node(origin)
	if n == nil {
		return netip.AddrPortFrom(Outside, stunPort), true
	}
	if n.site.STUN == "timeout" {
		return netip.AddrPort{}, false
	}
	port := stunPort + uint16(n.index)
	if second && n.site.STUN == "varying" {
		port += 1000
	}
	src, _ := m.source(origin, false)
	return netip.AddrPortFrom(src, port), true
}

// Answers is whether origin's site answers STUN at all, and whether its
// router answers NAT-PMP, so doctor knows a silence it pings is a play.
func (m *Model) Answers(origin string) (stun, natPMP bool) {
	gw, ok := m.Gateway(origin)
	_, stun = m.Mapped(origin, false)
	return stun, ok && gw.Speaks("nat-pmp")
}

// Gateway is a site's router as it answers a Node behind it.
type Gateway struct {
	// WAN is the address it reports on its internet side, invalid when it
	// has none.
	WAN       netip.Addr
	Protocols []string
	Refuse    string
}

func (g Gateway) Speaks(protocol string) bool { return slices.Contains(g.Protocols, protocol) }

// Gateway is the router origin's gateway entry reaches, false when its
// site has none. The world's own socket answers as a public router that
// speaks every protocol.
func (m *Model) Gateway(origin string) (Gateway, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.siteOf(origin)
	if st == nil {
		return Gateway{WAN: Outside, Protocols: []string{"pcp", "nat-pmp", "upnp"}}, true
	}
	if st.WAN == "none" {
		return Gateway{}, false
	}
	g := Gateway{Protocols: st.Mapping, Refuse: st.Refuse}
	switch st.WAN {
	case "public":
		g.WAN = st.address
	case "cgnat", "private":
		g.WAN = st.gateway
	}
	return g, true
}

// Mapping is a forward on a site's router: one a protocol granted, or one
// the Admin set by hand, which Protocol leaves empty.
type Mapping struct {
	Protocol    string
	Transport   string
	External    uint16
	Node        string
	Internal    uint16
	Lifetime    time.Duration
	Description string
}

// Grant adds g to origin's site, replacing any forward of its external
// port and transport.
func (m *Model) Grant(origin string, g Mapping) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.siteOf(origin)
	if st == nil {
		return
	}
	st.granted = slices.DeleteFunc(st.granted, func(x Mapping) bool { return x.External == g.External && x.Transport == g.Transport })
	st.granted = append(st.granted, g)
}

// Release removes origin's site's granted forwards that match picks out,
// false when there is none.
func (m *Model) Release(origin string, match func(Mapping) bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.siteOf(origin)
	if st == nil {
		return false
	}
	n := len(st.granted)
	st.granted = slices.DeleteFunc(st.granted, match)
	return len(st.granted) < n
}

// Forwards are the forwards origin's router lists over UPnP: the manual
// ones the Scenario lists, at their services' ports as they run now, and
// every one granted.
func (m *Model) Forwards(origin string) []Mapping {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.siteOf(origin)
	if st == nil {
		return nil
	}
	var out []Mapping
	for _, f := range st.Forwards {
		if !f.Listed {
			continue
		}
		at, err := m.locate(Service{Node: f.Node, Kind: f.Service})
		if err != nil {
			continue
		}
		out = append(out, Mapping{Transport: "TCP", External: at.Port(), Node: f.Node, Internal: at.Port(), Description: f.Service})
	}
	return append(out, st.granted...)
}

// LAN is a Node's address on its site's LAN, its internal client in a
// forward.
func (m *Model) LAN(node string) netip.Addr {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n := m.node(node); n != nil {
		return n.lan()
	}
	return netip.Addr{}
}

// Answer is what a resolver answers in place of records.
type Answer uint8

const (
	NoError Answer = iota
	NXDomain
	ServFail
)

// Lookup is fake-dns's answer to origin's query for name's A records, or
// AAAA with six: the record's own addresses, or under a split resolver
// its home answers.
func (m *Model) Lookup(origin, name string, six bool) ([]netip.Addr, Answer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := slices.IndexFunc(m.dns, func(r scenario.Record) bool { return lower(r.Name) == lower(name) })
	if i < 0 {
		return nil, NXDomain
	}
	r := m.dns[i]
	switch {
	case r.Answer == "nxdomain":
		return nil, NXDomain
	case r.Answer == "servfail", r.Answer == "aaaa-servfail" && six:
		return nil, ServFail
	}
	addrs := r.A
	if six {
		addrs = r.AAAA
	}
	if st := m.siteOf(origin); st != nil && st.Resolver == "split" && len(r.Home) > 0 {
		addrs = r.Home
	}
	var out []netip.Addr
	for _, s := range addrs {
		if a := netip.MustParseAddr(s); a.Is6() == six {
			out = append(out, a)
		}
	}
	return out, NoError
}
