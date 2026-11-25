// Package netsite is the Fake world's one model of the network outside
// its Nodes, for S51 (Reachability diagnostic): each site's router and
// public address, each Node's scripted interfaces, firewall and bind, the
// Admin's proxies and the public DNS. fake-stun, fake-gateway, fake-dns,
// the outside dialer and fake-proxy each play one protocol over it, and
// the network cue changes it in one place, so no two of them disagree.
package netsite

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/netfacts/script"
)

// Plays is whether s runs the network fakes: it has a network block.
func Plays(s scenario.Scenario) bool { return s.Network != nil }

// Outside is the address the internet's own clients come from, such as
// the world's own front and plex.tv's connect-back, in either family.
var (
	Outside  = netip.MustParseAddr("198.51.100.254")
	Outside6 = netip.MustParseAddr("2001:db8:ffff::254")
)

const (
	// HTTP is SlipStream's own web listener on a Node.
	HTTP = "http"
	// World is the origin of the world's own socket and front.
	World = events.World
)

// Service is a listener on a Node: HTTP, or a Media Server kind's fake as
// the Node reaches it.
type Service struct {
	Node string
	Kind string
}

// Locate is where a service listens as its Node runs now.
type Locate func(Service) (netip.AddrPort, error)

var (
	// ErrUnreachable is a connection with no route: no site holds the
	// address, or the site's router lets nothing in.
	ErrUnreachable = errors.New("no route to the address")
	// ErrRefused is a connection the address refuses: no forward, a
	// firewall, or nothing listening.
	ErrRefused = errors.New("the address refuses the connection")
)

type Model struct {
	mu      sync.Mutex
	locate  Locate
	sites   []*site
	nodes   []*nodeState
	proxies []scenario.Proxy
	dns     []scenario.Record
}

type site struct {
	scenario.Site
	index   uint8
	address netip.Addr
	gateway netip.Addr
	prefix6 netip.Prefix
	granted []Mapping
}

type nodeState struct {
	name       string
	index      uint8
	site       *site
	controller bool
	addresses  []script.Address
	tunnel     bool
	firewall   []scenario.FirewallRule
	bind       script.Bind
}

const sharedKey = "netsite"

// From is the world's model, which the first network fake built from s
// and its env's fronts and listeners, so every network fake of a run
// plays one model.
func From(env fake.Env, s scenario.Scenario) (*Model, error) {
	if m, ok := env.Shared.Get(sharedKey); ok {
		model, _ := m.(*Model)
		return model, nil
	}
	m, err := New(s, locate(env))
	if err != nil {
		return nil, err
	}
	env.Shared.Put(sharedKey, m)
	return m, nil
}

func locate(env fake.Env) Locate {
	return func(svc Service) (netip.AddrPort, error) {
		if svc.Kind == HTTP {
			if env.Listeners == nil {
				return netip.AddrPort{}, fmt.Errorf("node %s's listeners are unknown outside a world", svc.Node)
			}
			l, err := env.Listeners(svc.Node)
			if err != nil {
				return netip.AddrPort{}, err
			}
			return l.HTTP, nil
		}
		at, ok := env.Fronts.Find(fault.Target(svc.Kind), svc.Node)
		if !ok {
			return netip.AddrPort{}, fmt.Errorf("no %s front for node %s", svc.Kind, svc.Node)
		}
		return at.AddrPort, nil
	}
}

// New builds the model of s's network, which Validate passed.
func New(s scenario.Scenario, locate Locate) (*Model, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("scenario %s has no network", s.Name)
	}
	m := &Model{locate: locate, proxies: s.Network.Proxies, dns: s.Network.DNS}
	for i, in := range s.Network.Sites {
		st := &site{Site: in, index: uint8(i), address: netip.MustParseAddr(in.Address)}
		st.gateway, _ = netip.ParseAddr(in.Gateway)
		st.prefix6, _ = netip.ParsePrefix(in.Prefix6)
		m.sites = append(m.sites, st)
	}
	for i, n := range s.Nodes {
		st := m.site(n.Site)
		ns := &nodeState{name: n.Name, index: uint8(i), site: st, controller: n.Setup == scenario.SetupController, tunnel: n.Tunnel, firewall: slices.Clone(n.Firewall), bind: script.BindAll}
		if n.Bind != "" {
			ns.bind = script.Bind(n.Bind)
		}
		for _, a := range n.Addresses {
			ns.addresses = append(ns.addresses, script.Address{Prefix: netip.MustParsePrefix(a.Prefix), Temporary: a.Temporary, Deprecated: a.Deprecated})
		}
		if len(n.Addresses) == 0 {
			ns.addresses = st.defaults(ns.index)
		}
		m.nodes = append(m.nodes, ns)
	}
	return m, nil
}

// defaults are the addresses of Node i with none scripted: its site's
// address on a site with no router, else a LAN address, and an address in
// the site's IPv6 prefix.
func (st *site) defaults(i uint8) []script.Address {
	v4 := netip.PrefixFrom(netip.AddrFrom4([4]byte{192, 168, st.index + 1, i + 10}), 24)
	if st.WAN == "none" {
		v4 = netip.PrefixFrom(st.address, 24)
	}
	out := []script.Address{{Prefix: v4}}
	if st.prefix6.IsValid() {
		out = append(out, script.Address{Prefix: netip.PrefixFrom(hostIn(st.prefix6, uint64(0x20+i)), 64)})
	}
	return out
}

func hostIn(p netip.Prefix, host uint64) netip.Addr {
	b := p.Masked().Addr().As16()
	for i := range 8 {
		b[15-i] = byte(host >> (8 * i)) //nolint:gosec // G115: each byte of host in turn
	}
	return netip.AddrFrom16(b)
}

func (m *Model) site(name string) *site {
	i := slices.IndexFunc(m.sites, func(s *site) bool { return s.Name == name })
	if i < 0 {
		return nil
	}
	return m.sites[i]
}

func (m *Model) node(name string) *nodeState {
	i := slices.IndexFunc(m.nodes, func(n *nodeState) bool { return n.name == name })
	if i < 0 {
		return nil
	}
	return m.nodes[i]
}

// Nodes are the names of the model's Nodes.
func (m *Model) Nodes() []string {
	out := make([]string, len(m.nodes))
	for i, n := range m.nodes {
		out[i] = n.name
	}
	return out
}

// global is whether a is an address on the internet in the Fake world,
// where the documentation prefixes stand for it.
func global(a netip.Addr) bool {
	return a.IsGlobalUnicast() && !a.IsPrivate() && !shared.Contains(a)
}

var shared = netip.MustParsePrefix("100.64.0.0/10")

func (n *nodeState) holds(a netip.Addr) bool {
	return slices.ContainsFunc(n.addresses, func(x script.Address) bool { return x.Prefix.Addr() == a })
}

// lan is the Node's first private IPv4 address, invalid when it has none.
func (n *nodeState) lan() netip.Addr {
	for _, a := range n.addresses {
		if a.Prefix.Addr().Is4() && a.Prefix.Addr().IsPrivate() {
			return a.Prefix.Addr()
		}
	}
	return netip.Addr{}
}

// source is the address the internet sees a connection of family six or
// not from origin come from.
func (m *Model) source(origin string, six bool) (netip.Addr, bool) {
	n := m.node(origin)
	if n == nil {
		if six {
			return Outside6, true
		}
		return Outside, true
	}
	if six {
		var out netip.Addr
		for _, a := range n.addresses {
			if p := a.Prefix.Addr(); p.Is6() && global(p) && (!out.IsValid() || a.Temporary) {
				out = p
			}
		}
		return out, out.IsValid()
	}
	for _, a := range n.addresses {
		if p := a.Prefix.Addr(); p.Is4() && global(p) {
			return p, true
		}
	}
	return n.site.address, true
}

func (m *Model) siteOf(origin string) *site {
	if n := m.node(origin); n != nil {
		return n.site
	}
	return nil
}

func lower(s string) string { return strings.ToLower(strings.TrimSuffix(s, ".")) }
