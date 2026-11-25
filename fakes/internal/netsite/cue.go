package netsite

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/netfacts/script"
)

// Change is one play of the network cue, a machine event outside every
// Node: a site's new public address or IPv6 prefix, or a Node's firewall
// rule or bind.
type Change struct {
	Site string `json:"site,omitempty"`
	// WAN is the site's new public address.
	WAN string `json:"wan,omitempty"`
	// Prefix6 is the site's new /64, or none.
	Prefix6 string `json:"prefix6,omitempty"`

	Node     string `json:"node,omitempty"`
	Service  string `json:"service,omitempty"`
	Family   string `json:"family,omitempty"`
	Firewall string `json:"firewall,omitempty"`
	Bind     string `json:"bind,omitempty"`
}

// Apply plays c and returns the Nodes whose script it changed.
func (m *Model) Apply(c Change) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case c.Site != "" && c.Node == "":
		return m.applySite(c)
	case c.Node != "" && c.Site == "":
		return m.applyNode(c)
	}
	return nil, errors.New("network: a change names a site or a node")
}

func (m *Model) applySite(c Change) ([]string, error) {
	st := m.site(c.Site)
	if st == nil {
		return nil, fmt.Errorf("network: no site %s", c.Site)
	}
	switch {
	case c.WAN != "" && c.Prefix6 == "":
		return m.moveWAN(st, c.WAN)
	case c.Prefix6 != "" && c.WAN == "":
		return m.changePrefix(st, c.Prefix6)
	}
	return nil, errors.New("network: a site change sets wan or prefix6")
}

// moveWAN gives st the public address to, and moves it on the Nodes that
// hold it.
func (m *Model) moveWAN(st *site, to string) ([]string, error) {
	wan, err := netip.ParseAddr(to)
	if err != nil || !wan.Is4() {
		return nil, fmt.Errorf("network: wan %q is not an IPv4 address", to)
	}
	var changed []string
	for _, n := range m.nodes {
		if n.site == st && n.holds(st.address) {
			n.replace(func(a netip.Addr) bool { return a == st.address }, func(netip.Addr) netip.Addr { return wan })
			changed = append(changed, n.name)
		}
	}
	st.address = wan
	return changed, nil
}

// changePrefix moves st's Nodes onto the /64 to, or off IPv6 for none.
func (m *Model) changePrefix(st *site, to string) ([]string, error) {
	next := netip.Prefix{}
	if to != "none" {
		p, err := netip.ParsePrefix(to)
		if err != nil || p.Bits() != 64 {
			return nil, fmt.Errorf("network: prefix6 %q is not a /64 or none", to)
		}
		next = p.Masked()
	}
	var changed []string
	for _, n := range m.nodes {
		if n.site != st {
			continue
		}
		before := slices.Clone(n.addresses)
		n.addresses = slices.DeleteFunc(n.addresses, func(a script.Address) bool { return !next.IsValid() && st.prefix6.Contains(a.Prefix.Addr()) })
		n.replace(st.prefix6.Contains, func(a netip.Addr) netip.Addr { return rebase(a, next) })
		if !st.prefix6.IsValid() && next.IsValid() {
			n.addresses = append(n.addresses, script.Address{Prefix: netip.PrefixFrom(hostIn(next, uint64(0x20+n.index)), 64)})
		}
		if !slices.Equal(before, n.addresses) {
			changed = append(changed, n.name)
		}
	}
	st.prefix6 = next
	return changed, nil
}

func (m *Model) applyNode(c Change) ([]string, error) {
	n := m.node(c.Node)
	if n == nil {
		return nil, fmt.Errorf("network: no node %s", c.Node)
	}
	switch {
	case c.Bind != "" && c.Firewall == "":
		if c.Bind != string(script.BindAll) && c.Bind != string(script.BindLocal) {
			return nil, fmt.Errorf("network: bind %q is not all or local", c.Bind)
		}
		n.bind = script.Bind(c.Bind)
	case c.Firewall != "" && c.Bind == "":
		rule := scenario.FirewallRule{Service: c.Service, Family: c.Family, Policy: c.Firewall}
		if !slices.Contains(scenario.Services, rule.Service) || (rule.Family != "ipv4" && rule.Family != "ipv6") || !slices.Contains([]string{"allows", "blocks", "unknown", "docker"}, rule.Policy) {
			return nil, fmt.Errorf("network: firewall rule %+v names no service, family and policy", rule)
		}
		n.firewall = slices.DeleteFunc(n.firewall, func(r scenario.FirewallRule) bool { return r.Service == rule.Service && r.Family == rule.Family })
		n.firewall = append(n.firewall, rule)
	default:
		return nil, errors.New("network: a node change sets firewall or bind")
	}
	return []string{n.name}, nil
}

// replace turns each address of n that match picks out into to's answer,
// keeping its prefix length and flags.
func (n *nodeState) replace(match func(netip.Addr) bool, to func(netip.Addr) netip.Addr) {
	for i, a := range n.addresses {
		if match(a.Prefix.Addr()) {
			n.addresses[i].Prefix = netip.PrefixFrom(to(a.Prefix.Addr()), a.Prefix.Bits())
		}
	}
}

// rebase is a with its first 64 bits from p.
func rebase(a netip.Addr, p netip.Prefix) netip.Addr {
	b, net := a.As16(), p.Addr().As16()
	copy(b[:8], net[:8])
	return netip.AddrFrom16(b)
}
