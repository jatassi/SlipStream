package netsite

import (
	"net/netip"
	"slices"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/netfacts/script"
)

// Arrival is where a connection from outside lands: a service on a Node,
// or one of the Admin's proxies, with the address it comes from.
type Arrival struct {
	Service Service
	Proxy   string
	Source  netip.Addr
	// Hairpin is a connection to its own site's public address, which the
	// router turned back in.
	Hairpin bool
}

// Route is the outside dialer's decision for a connection from origin to
// to: through the site's NAT and forwards, its IPv6 firewall, the Node's
// own firewall and bind, or to a proxy. A cgnat site, and a double NAT
// whose outer router forwards nothing, let no connection in.
func (m *Model) Route(origin string, to netip.AddrPort) (Arrival, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	addr, six := to.Addr().Unmap(), to.Addr().Unmap().Is6()
	src, ok := m.source(origin, six)
	if !ok {
		return Arrival{}, ErrUnreachable
	}
	if i := slices.IndexFunc(m.proxies, func(p scenario.Proxy) bool { return netip.MustParseAddr(p.Address) == addr }); i >= 0 {
		return Arrival{Proxy: m.proxies[i].Name, Source: src}, nil
	}
	from := m.siteOf(origin)
	if i := slices.IndexFunc(m.nodes, func(n *nodeState) bool { return global(addr) && n.holds(addr) }); i >= 0 {
		n := m.nodes[i]
		if six && n.site.Firewall6 != "open" && n.site != from {
			return Arrival{}, ErrRefused
		}
		return m.arrive(n, to.Port(), six, src, false)
	}
	if six {
		return Arrival{}, ErrUnreachable
	}
	i := slices.IndexFunc(m.sites, func(s *site) bool { return s.address == addr })
	if i < 0 {
		return Arrival{}, ErrUnreachable
	}
	st := m.sites[i]
	if st.WAN == "cgnat" || st.WAN == "none" || (st.WAN == "private" && st.Upstream != "forwarded") {
		return Arrival{}, ErrUnreachable
	}
	node, port, ok := m.forward(st, to.Port())
	if !ok {
		return Arrival{}, ErrRefused
	}
	return m.arrive(node, port, false, src, st == from)
}

// forward is the Node and port a forward on st sends port to.
func (m *Model) forward(st *site, port uint16) (*nodeState, uint16, bool) {
	for _, g := range st.granted {
		if g.Transport == "TCP" && g.External == port {
			return m.node(g.Node), g.Internal, true
		}
	}
	for _, f := range st.Forwards {
		if at, err := m.locate(Service{Node: f.Node, Kind: f.Service}); err == nil && at.Port() == port {
			return m.node(f.Node), port, true
		}
	}
	return nil, 0, false
}

// arrive lands a connection on n's service on port, through its firewall
// and bind.
func (m *Model) arrive(n *nodeState, port uint16, six bool, src netip.Addr, hairpin bool) (Arrival, error) {
	i := slices.IndexFunc(scenario.Services, func(kind string) bool {
		at, err := m.locate(Service{Node: n.name, Kind: kind})
		return err == nil && at.Port() == port
	})
	if i < 0 {
		return Arrival{}, ErrRefused
	}
	kind := scenario.Services[i]
	family := "ipv4"
	if six {
		family = "ipv6"
	}
	blocked := slices.ContainsFunc(n.firewall, func(r scenario.FirewallRule) bool {
		return r.Service == kind && r.Family == family && r.Policy == "blocks"
	})
	if blocked || (kind == HTTP && n.bind == script.BindLocal) {
		return Arrival{}, ErrRefused
	}
	return Arrival{Service: Service{Node: n.name, Kind: kind}, Source: src, Hairpin: hairpin}, nil
}

// Proxied is the service a proxy routes host to, false when none does.
func (m *Model) Proxied(host string) (Service, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.proxies {
		for _, r := range p.Routes {
			if lower(r.Host) == lower(host) {
				return Service{Node: r.Node, Kind: r.Service}, true
			}
		}
	}
	return Service{}, false
}

// Where is where svc listens as its Node runs now.
func (m *Model) Where(svc Service) (netip.AddrPort, error) { return m.locate(svc) }

// Source is the IPv4 address the internet sees a connection from origin
// come from.
func (m *Model) Source(origin string) netip.Addr {
	m.mu.Lock()
	defer m.mu.Unlock()
	src, _ := m.source(origin, false)
	return src
}
