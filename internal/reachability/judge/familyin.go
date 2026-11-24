package judge

import (
	"net/netip"
	"net/url"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// familyIn is one family's facts, built once before any row runs, so a row
// never reads another row's result.
type familyIn struct {
	f          Facts
	pl         Placement
	fam        netmodel.Family
	route      FamilyRoute
	exposure   Exposure
	host       string
	publicPort uint16
	dns        dnsView
	o          origin
	class      netmodel.Class4
	notes      []netmodel.Note
	// port is the port the Origin hop reads: the public URL's port when a
	// program on the origin listens on it, else the Exposure's local port.
	port     uint16
	listener netmodel.Listener
	// own is the service's own listener when another program owns port.
	own                           netmodel.Listener
	firewall                      netmodel.Rule
	player, connectBack, proxyLeg *Dial
	server                        *ServerFacts
}

func newFamilyIn(f Facts, pl Placement, fam netmodel.Family) *familyIn {
	o := originOf(f, pl.Origin)
	in := &familyIn{
		f: f, pl: pl, fam: fam, route: pl.Routes[fam], exposure: f.Exposure,
		host: f.Exposure.Public.Hostname(), publicPort: f.Exposure.PublicPort(),
		dns: answers(f, f.Origin.Node), o: o, server: f.Server,
	}
	in.class, in.notes = f.Rules.Classify(o.facts.Observed())
	in.port = f.Exposure.Local
	if _, ok := o.facts.Listener(in.publicPort); ok {
		in.port = in.publicPort
	}
	in.listener, _ = o.facts.Listener(in.port)
	in.own, _ = o.facts.Listener(f.Exposure.Local)
	in.firewall = o.facts.Rule(in.port, fam)
	for i := range f.Dials {
		d := &f.Dials[i]
		if d.Family != fam {
			continue
		}
		switch d.Kind {
		case PlayerDial:
			in.player = first(in.player, d)
		case ConnectBackDial:
			in.connectBack = first(in.connectBack, d)
		case ProxyLegDial:
			in.proxyLeg = first(in.proxyLeg, d)
		}
	}
	return in
}

func first(a, b *Dial) *Dial {
	if a != nil {
		return a
	}
	return b
}

// ours is whether the program on the Origin hop's port is the service
// itself rather than a proxy in front of it.
func (in *familyIn) ours() bool { return in.port == in.exposure.Local }

func (in *familyIn) direct() bool { return in.route.Kind == Direct }

func (in *familyIn) plex() bool { return in.server != nil && in.server.Kind == Plex }

// subject is the service's name for a fix: the Media Server's name, empty
// for SlipStream itself.
func (in *familyIn) subject() string {
	if in.exposure.Kind == MediaServer && in.server != nil {
		return in.server.Name
	}
	return ""
}

func (in *familyIn) https() bool { return in.exposure.Public.Scheme == "https" }

// localAddr is the origin's address inside the home, where a forward
// points.
func (in *familyIn) localAddr() netip.Addr {
	for _, i := range in.o.facts.Interfaces {
		for _, a := range i.Addrs {
			if ad := a.Prefix.Addr(); !i.Tunnel && ad.Is4() && in.f.Rules.Of(ad) == netmodel.Private {
				return ad
			}
		}
	}
	return in.o.facts.Egress.Local
}

// liveMapping is SlipStream's granted mapping, a forward the router lists
// to the origin, or Plex's own mapping at the address this route uses.
func (in *familyIn) liveMapping() bool {
	nf := in.o.facts
	if slices.ContainsFunc(nf.Mappings, func(m netmodel.Mapping) bool {
		return m.Refused == netmodel.RefusedNone && m.External == in.publicPort
	}) {
		return true
	}
	if slices.ContainsFunc(nf.Gateway.Forwards, func(fw netmodel.Forward) bool {
		return fw.External == in.publicPort && nf.Holds(fw.Client)
	}) {
		return true
	}
	return in.plexCounts()
}

// plexCounts is Plex's own check counting for this route: mapped, at an
// address the route connects to, on the public port.
func (in *familyIn) plexCounts() bool {
	if in.fam != netmodel.IPv4 || !in.plex() || !in.server.Mapping.Counts() {
		return false
	}
	pub := in.server.Mapping.Public
	return pub.Port() == in.publicPort && slices.Contains(in.route.Dialed, pub.Addr())
}

func (in *familyIn) refusedOtherPort() bool {
	return slices.ContainsFunc(in.o.facts.Mappings, func(m netmodel.Mapping) bool { return m.Refused == netmodel.OtherPort })
}

// untestedPoints is what an untested family points to: Reachable on a
// Direct IPv4 route, where origin.class, origin.bind and origin.firewall
// point on their own, and Fixable everywhere else, which nothing local can
// prove.
func (in *familyIn) untestedPoints() Verdict {
	if in.direct() && in.fam == netmodel.IPv4 {
		return Reachable
	}
	return Fixable
}

func (in *familyIn) forwardParams() Params {
	return Params{Node: in.o.node, Port: in.publicPort, LocalAddr: in.localAddr(), LocalPort: in.port}
}

// cause is the first local reason a connection would fail, the same Fix
// the row naming it returns, so two rows that name one cause give one
// step.
func (in *familyIn) cause() *Fix {
	if !in.direct() {
		switch {
		case in.proxyLeg != nil && in.proxyLeg.Stage >= Connected:
			return fix(FixProxyPort, Params{Port: in.publicPort, Addr: in.route.Proxy})
		case in.listener.Loopback:
			return in.bindFix()
		}
		return fix(FixProxyForward, Params{Addr: in.route.Proxy, Node: in.o.node, Port: in.port})
	}
	dns := rowDNSA
	if in.fam == netmodel.IPv6 {
		dns = rowDNSAAAA
	}
	for _, r := range []row{dns, rowBind, rowFirewall, rowClass} {
		if r.Applies(in) {
			if o := r.Eval(in); o.Result != Passes && o.Fix != nil {
				return o.Fix
			}
		}
	}
	switch {
	case in.fam == netmodel.IPv6:
		return fix(FixBrokenAAAA, Params{Host: in.host, Addr: in.route.Dialed[0], Port: in.publicPort, Node: in.o.node})
	case in.o.facts.Gateway.Addr.IsValid():
		return fix(FixManualForward, in.forwardParams())
	}
	return fix(FixNoGateway, Params{Node: in.o.node, Port: in.publicPort})
}

func (in *familyIn) bindFix() *Fix {
	if in.listener.Program.SlipStream {
		return fix(FixBind, Params{Node: in.o.node})
	}
	return fix(FixBindProgram, Params{Node: in.o.node, Program: in.listener.Program.Name, Port: in.listener.Port})
}

// hopFacts are the family's neutral facts that no row owns.
func (in *familyIn) hopFacts() []placedFact {
	fam := in.fam
	var out []placedFact
	add := func(h HopKind, id FactID, p Params) {
		out = append(out, placedFact{h, Fact{ID: id, Family: &fam, Params: p}})
	}
	_, skipped := Testers(in.f, in.pl, fam)
	skipped = append(skipped, in.f.Skipped...)
	for _, s := range skipped {
		if s.Family != nil && *s.Family != fam {
			continue
		}
		if in.player == nil || s.Params.Tester == in.player.Tester {
			out = append(out, placedFact{PlayerHop, s})
		}
	}
	if d := in.proxyLeg; d != nil {
		add(ProxyHop, FactProxyLeg, Params{Tester: d.Tester, Node: in.o.node})
	}
	if fam != netmodel.IPv4 {
		return out
	}
	if in.f.Churn[in.o.node] {
		add(OriginHop, FactAddressChurn, Params{Node: in.o.node})
	}
	if in.pl.OriginLabel == Reaching {
		add(OriginHop, FactClassFromReaching, Params{Node: in.o.node, Program: in.subject()})
	}
	if !in.direct() {
		add(OriginHop, FactClass, Params{Node: in.o.node, Class: in.class})
	}
	for _, n := range in.notes {
		id := FactSTUNVariesPort
		if n == netmodel.STUNVariesAddress {
			id = FactSTUNVariesAddress
		}
		add(OriginHop, id, Params{Node: in.o.node})
	}
	if slices.ContainsFunc(in.o.facts.Mappings, func(m netmodel.Mapping) bool {
		return m.Refused == netmodel.RefusedNone && m.Lifetime == 0
	}) {
		add(OriginHop, FactMappingPermanent, Params{Node: in.o.node})
	}
	return out
}

type placedFact struct {
	hop  HopKind
	fact Fact
}

// listsPublic is whether the Media Server lists the public URL among its
// custom URLs, by host and port with default ports normalized.
func (in *familyIn) listsPublic() bool {
	want := in.exposure.Public
	for _, c := range in.server.Custom {
		for _, s := range strings.Split(c, ",") {
			u, err := url.Parse(strings.TrimSpace(s))
			if err == nil && strings.EqualFold(u.Hostname(), want.Hostname()) &&
				(Exposure{Public: u}).PublicPort() == in.publicPort {
				return true
			}
		}
	}
	return false
}
