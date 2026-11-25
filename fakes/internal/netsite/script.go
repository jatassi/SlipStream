package netsite

import (
	"fmt"
	"net/netip"
	"slices"

	"github.com/jatassi/slipstream/internal/netfacts/script"
)

// Script is the scripted readers' file of the Node named node: its
// interfaces, its egress and its router, its bind and firewall, and in
// the Controller's file where each other Node's link appears to come
// from.
func (m *Model) Script(node string) (script.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.node(node)
	if n == nil {
		return script.File{}, fmt.Errorf("network: no node %s", node)
	}
	f := script.File{
		Interfaces: []script.Interface{{Name: "eth0", Addresses: slices.Clone(n.addresses)}},
		Egress:     "eth0",
		Bind:       n.bind,
	}
	if n.site.WAN != "none" {
		f.Gateway = netip.AddrFrom4([4]byte{192, 168, n.site.index + 1, 1})
	}
	if n.tunnel {
		tun := netip.PrefixFrom(netip.AddrFrom4([4]byte{10, 64, 0, n.index + 2}), 32)
		f.Interfaces = append(f.Interfaces, script.Interface{Name: "wg0", Tunnel: true, Addresses: []script.Address{{Prefix: tun}}})
		f.Egress, f.Gateway = "wg0", netip.Addr{}
	}
	for _, r := range n.firewall {
		rule := script.Rule{Family: script.Family(r.Family), Policy: script.Policy(r.Policy)}
		if r.Service == HTTP {
			rule.Web = true
		} else {
			at, err := m.locate(Service{Node: n.name, Kind: r.Service})
			if err != nil {
				return script.File{}, err
			}
			rule.Port = at.Port()
		}
		f.Firewall = append(f.Firewall, rule)
	}
	if n.controller {
		for _, o := range m.nodes {
			if o == n {
				continue
			}
			from, _ := m.source(o.name, false)
			if o.site == n.site && o.lan().IsValid() {
				from = o.lan()
			}
			f.Links = append(f.Links, script.Link{Node: o.name, Address: from})
		}
	}
	return f, nil
}
