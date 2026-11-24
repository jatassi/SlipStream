package judge

import (
	"net/netip"
	"slices"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/role"
)

type Placement struct {
	Origin      role.NodeID
	OriginLabel OriginLabel
	Offered     []netmodel.Family // IPv4 always first
	Routes      map[netmodel.Family]FamilyRoute
	Notes       []Fact // split-horizon, ipv6-skips-proxy
}

type FamilyRoute struct {
	Kind RouteKind
	// Dialed is the addresses a player would connect to.
	Dialed []netip.Addr
	// Proxy is the own proxy's or Relay Node's address, and ProxyNode the
	// paired Node that holds it.
	Proxy     netip.Addr
	ProxyNode role.NodeID
	Hops      []HopKind
}

var hopsOf = map[RouteKind][]HopKind{
	Direct:       {PlayerHop, DNSHop, TLSHop, OriginHop, MediaServerHop},
	Unplaced:     {PlayerHop, DNSHop, TLSHop, OriginHop, MediaServerHop},
	OwnProxy:     {PlayerHop, DNSHop, ProxyHop, TLSHop, OriginHop, MediaServerHop},
	ThroughRelay: {PlayerHop, DNSHop, RelayHop, TLSHop, NodeLinkHop, OriginHop, MediaServerHop},
}

// Place decides each family's route and hops from the DNS answers, the
// origin and the paired Nodes. The run calls it before testing to pick
// testers, and Judge calls it again on the full facts.
func Place(f Facts) Placement {
	v := answers(f, f.Origin.Node)
	pl := Placement{Origin: f.Origin.Node, OriginLabel: f.Origin.Label, Routes: map[netmodel.Family]FamilyRoute{}}
	if f.Exposure.Kind == UIPortal {
		for _, a := range v.a {
			if n, ok := f.Paired[a]; ok && slices.Contains(f.Origin.Serving, n) {
				pl.Origin, pl.OriginLabel = n, Serving
				break
			}
		}
	}
	o := originOf(f, pl.Origin)
	r4 := o.place(v.a, v.outside)
	pl.Routes[netmodel.IPv4] = r4
	pl.Offered = []netmodel.Family{netmodel.IPv4}
	if r6, ok := o.place6(v.aaaa, published6(f)); ok {
		pl.Routes[netmodel.IPv6] = r6
		pl.Offered = append(pl.Offered, netmodel.IPv6)
		if r4.Kind == OwnProxy && r6.Kind == Direct {
			pl.Notes = append(pl.Notes, Fact{ID: FactIPv6SkipsProxy, Params: Params{Node: pl.Origin, Addr: r4.Proxy}})
		}
	}
	if r4.Kind != Unplaced {
		if home := v.homeLocal(f.Rules); home.IsValid() {
			pl.Notes = append(pl.Notes, Fact{ID: FactSplitHorizon, Params: Params{Host: f.Exposure.Public.Hostname(), Addr: home}})
		}
	}
	return pl
}

// origin is the origin's network as placement and the rows read it.
type origin struct {
	f       *Facts
	node    role.NodeID
	facts   netmodel.NetworkFacts
	public4 netip.Addr
	addrs   []netip.Addr   // Public4, the gateway's Global WAN, Global interface addresses
	slash64 []netip.Prefix // the Global /64s it holds
}

func originOf(f Facts, node role.NodeID) origin {
	nf := f.Nodes[node]
	o := origin{f: &f, node: node, facts: nf, public4: f.Rules.Public4(nf.Observed()), slash64: f.Rules.Prefixes6(nf.Interfaces)}
	if o.public4.IsValid() {
		o.addrs = append(o.addrs, o.public4)
	}
	if w := nf.Gateway.WAN; f.Rules.Global(w) {
		o.addrs = append(o.addrs, w)
	}
	for _, i := range nf.Interfaces {
		for _, a := range i.Addrs {
			if f.Rules.Global(a.Prefix.Addr()) {
				o.addrs = append(o.addrs, a.Prefix.Addr())
			}
		}
	}
	return o
}

func (o origin) holds(a netip.Addr) bool {
	if a.Is6() && o.f.Rules.Global(a) {
		return slices.Contains(o.slash64, netmodel.Slash64(a))
	}
	return slices.Contains(o.addrs, a)
}

func (o origin) previous(a netip.Addr) bool {
	if a.Is6() {
		return slices.Contains(o.facts.Previous6, netmodel.Slash64(a))
	}
	return o.facts.Previous4.IsValid() && a == o.facts.Previous4
}

func (o origin) place(as []netip.Addr, outside bool) FamilyRoute {
	r := o.f.Rules
	switch {
	case len(as) == 0:
		return route(Direct, nil)
	case !outside && !slices.ContainsFunc(as, r.Global):
		return route(Unplaced, as)
	case slices.ContainsFunc(as, o.holds) || slices.ContainsFunc(as, o.previous) ||
		slices.ContainsFunc(as, func(a netip.Addr) bool { return !r.Global(a) }):
		return route(Direct, as)
	}
	fr := route(OwnProxy, as)
	fr.Proxy = as[0]
	if n, ok := o.f.Paired[as[0]]; ok {
		fr.ProxyNode = n
		if o.f.relaying(n) {
			fr.Kind, fr.Hops = ThroughRelay, hopsOf[ThroughRelay]
		}
	}
	return fr
}

// place6 offers IPv6 for a Global AAAA or a Global published IPv6
// connection, never for any other class.
func (o origin) place6(aaaa, published []netip.Addr) (FamilyRoute, bool) {
	global := slices.DeleteFunc(slices.Clone(aaaa), func(a netip.Addr) bool { return !o.f.Rules.Global(a) })
	switch {
	case len(global) > 0:
		return o.place(global, true), true
	case len(published) > 0:
		return route(Direct, published), true
	}
	return FamilyRoute{}, false
}

func route(k RouteKind, as []netip.Addr) FamilyRoute {
	return FamilyRoute{Kind: k, Dialed: as, Hops: hopsOf[k]}
}

func published6(f Facts) []netip.Addr {
	if f.Server == nil {
		return nil
	}
	var out []netip.Addr
	for _, c := range f.Server.Published {
		if c.Family == netmodel.IPv6 && !c.Relay && f.Rules.Global(c.Addr) {
			out = append(out, c.Addr)
		}
	}
	return out
}

func (f *Facts) relaying(n role.NodeID) bool {
	return slices.ContainsFunc(f.Candidates, func(c Candidate) bool { return c.Node == n && c.Relaying })
}

// dnsView is the answers that decide the route: outside resolvers' when any
// outside Node resolved, else the home Nodes'.
type dnsView struct {
	a, aaaa       []netip.Addr
	errA, errAAAA DNSError
	outside       bool
	home          []netip.Addr // home resolvers' A answers, beside outside ones
}

func answers(f Facts, originNode role.NodeID) dnsView {
	if a, err := netip.ParseAddr(f.Exposure.Public.Hostname()); err == nil {
		if a.Is4() {
			return dnsView{a: []netip.Addr{a}, outside: true}
		}
		return dnsView{aaaa: []netip.Addr{a}, outside: true}
	}
	var out, home []DNSAnswer
	for _, d := range f.DNS {
		if outsideOf(f, originNode, d.By) {
			out = append(out, d)
		} else {
			home = append(home, d)
		}
	}
	v := dnsView{outside: len(out) > 0}
	chosen := home
	if v.outside {
		chosen = out
		for _, d := range home {
			v.home = append(v.home, d.A...)
		}
	}
	for _, d := range chosen {
		v.a = union(v.a, d.A)
		v.aaaa = union(v.aaaa, d.AAAA)
		if v.errA == DNSNone {
			v.errA = d.ErrA
		}
		if v.errAAAA == DNSNone {
			v.errAAAA = d.ErrAAAA
		}
	}
	return v
}

// homeLocal is a non-Global answer a home resolver got while outside ones
// answered: split-horizon DNS, a neutral fact.
func (v dnsView) homeLocal(r netmodel.Rules) netip.Addr {
	for _, a := range v.home {
		if !r.Global(a) {
			return a
		}
	}
	return netip.Addr{}
}

func union(a, b []netip.Addr) []netip.Addr {
	for _, x := range b {
		if !slices.Contains(a, x) {
			a = append(a, x)
		}
	}
	return a
}

// outsideOf is whether n is outside the origin's network: its Public4 is
// known and differs, and it holds none of the origin's Global /64s.
func outsideOf(f Facts, originNode, n role.NodeID) bool {
	return n != originNode && !sameNetwork(f, originNode, n) && f.Rules.Public4(f.Nodes[n].Observed()).IsValid()
}

func sameNetwork(f Facts, a, b role.NodeID) bool {
	fa, fb := f.Nodes[a], f.Nodes[b]
	pa, pb := f.Rules.Public4(fa.Observed()), f.Rules.Public4(fb.Observed())
	if pa.IsValid() && pa == pb {
		return true
	}
	sa := f.Rules.Prefixes6(fa.Interfaces)
	return slices.ContainsFunc(f.Rules.Prefixes6(fb.Interfaces), func(p netip.Prefix) bool { return slices.Contains(sa, p) })
}
