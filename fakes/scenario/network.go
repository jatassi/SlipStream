package scenario

import (
	"net/netip"
	"slices"
	"strings"
)

// Network is the world outside the Nodes for S51 (Reachability
// diagnostic): the sites they sit in, the Admin's own proxies and the
// public DNS. A Scenario that sets it runs the network fakes, and each of
// its Nodes names a site.
type Network struct {
	Sites   []Site   `jsonschema:"required,minItems=1" yaml:"sites"`
	Proxies []Proxy  `yaml:"proxies,omitempty"`
	DNS     []Record `yaml:"dns,omitempty"`
}

// Site is one place on the internet: a home behind its router, or a
// server with an address of its own.
type Site struct {
	Name string `jsonschema:"required,pattern=^[a-z][a-z0-9-]{0\\,15}$" yaml:"name"`
	// WAN is what the site's router has on its internet side: public is
	// Address, cgnat and private are Gateway behind another NAT, empty is
	// no address at all, and none is no router, so its Nodes hold Address.
	WAN string `jsonschema:"required,enum=public,enum=cgnat,enum=private,enum=empty,enum=none" yaml:"wan"`
	// Address is the IPv4 address the internet sees the site from, a
	// TEST-NET address.
	Address string `jsonschema:"required" yaml:"address"`
	// Gateway is the router's WAN address under cgnat or private.
	Gateway string `yaml:"gateway,omitempty"`
	// Mapping are the protocols the router speaks to grant a forward.
	Mapping []string `jsonschema:"uniqueItems=true,enum=pcp,enum=nat-pmp,enum=upnp" yaml:"mapping,omitempty"`
	// Refuse is how the router refuses every mapping: UPnP error 606 or
	// 725, a refusal in every protocol, or a grant on another port.
	Refuse string `jsonschema:"enum=606,enum=725,enum=refuse,enum=other-port" yaml:"refuse,omitempty"`
	// Prefix6 is the site's IPv6 /64, inside 2001:db8::/32; empty is none.
	Prefix6 string `yaml:"prefix6,omitempty"`
	// Firewall6 is the router's IPv6 firewall: open lets every connection
	// in, closed none.
	Firewall6 string `jsonschema:"enum=open,enum=closed" yaml:"firewall6,omitempty"`
	// Upstream is a double NAT's outer router: forwarded passes every
	// connection to this router, closed none.
	Upstream string `jsonschema:"enum=forwarded,enum=closed" yaml:"upstream,omitempty"`
	// Resolver is what the site's resolver answers: the public records, or
	// split, a record's home answers.
	Resolver string `jsonschema:"enum=outside,enum=split" yaml:"resolver,omitempty"`
	// STUN is how STUN plays from the site: varying answers another port
	// from the second server, and timeout answers nothing.
	STUN string `jsonschema:"enum=varying,enum=timeout" yaml:"stun,omitempty"`
	// Forwards are the forwards the Admin set by hand on the router.
	Forwards []Forward `yaml:"forwards,omitempty"`
}

// Services are the listeners on a Node a network block names, by name
// because the Fake world's ports
// are drawn at launch: http is SlipStream's web listener, and a Media
// Server kind is that server's front as the Node reaches it.
var Services = []string{"http", "plex", "jellyfin", "emby"}

// Forward is a manual forward of a service's port to a Node, on the same
// port. Listed has the router list it over UPnP.
type Forward struct {
	Node    string `jsonschema:"required" yaml:"node"`
	Service string `jsonschema:"required,enum=http,enum=plex,enum=jellyfin,enum=emby" yaml:"service"`
	Listed  bool   `yaml:"listed,omitempty"`
}

// Proxy is the Admin's own reverse proxy on a server of its own, which
// fake-proxy plays.
type Proxy struct {
	Name    string `jsonschema:"required" yaml:"name"`
	Address string `jsonschema:"required" yaml:"address"`
	// Node is a paired Node that is the proxy's machine.
	Node   string       `yaml:"node,omitempty"`
	Routes []ProxyRoute `jsonschema:"required,minItems=1" yaml:"routes"`
}

// ProxyRoute sends a host name to a service on a Node.
type ProxyRoute struct {
	Host    string `jsonschema:"required" yaml:"host"`
	Node    string `jsonschema:"required" yaml:"node"`
	Service string `jsonschema:"required,enum=http,enum=plex,enum=jellyfin,enum=emby" yaml:"service"`
}

// Record is a public DNS name, under .localhost so the run CA can name it.
type Record struct {
	Name string   `jsonschema:"required" yaml:"name"`
	A    []string `yaml:"a,omitempty"`
	AAAA []string `yaml:"aaaa,omitempty"`
	// Home is what a split resolver answers inside a site, for A and AAAA
	// by family.
	Home []string `yaml:"home,omitempty"`
	// Answer is an error the resolver answers in place of the records:
	// NXDOMAIN, SERVFAIL, or SERVFAIL for AAAA only. UDP has no front, so
	// this is no fault rule.
	Answer string `jsonschema:"enum=nxdomain,enum=servfail,enum=aaaa-servfail" yaml:"answer,omitempty"`
}

// NodeAddress is a scripted interface address with its flags.
type NodeAddress struct {
	Prefix     string `jsonschema:"required" yaml:"prefix"`
	Temporary  bool   `yaml:"temporary,omitempty"`
	Deprecated bool   `yaml:"deprecated,omitempty"`
}

// FirewallRule is the Node's host firewall for one service and family.
// unknown and docker allow the connection, though the reader can't tell.
type FirewallRule struct {
	Service string `jsonschema:"required,enum=http,enum=plex,enum=jellyfin,enum=emby" yaml:"service"`
	Family  string `jsonschema:"required,enum=ipv4,enum=ipv6" yaml:"family"`
	Policy  string `jsonschema:"required,enum=allows,enum=blocks,enum=unknown,enum=docker" yaml:"policy"`
}

var documentation6 = netip.MustParsePrefix("2001:db8::/32")

func (s Scenario) checkNetwork(add func(format string, args ...any)) {
	if s.Network == nil {
		for _, n := range s.Nodes {
			if n.Site != "" || len(n.Addresses) > 0 || n.Tunnel || len(n.Firewall) > 0 || n.Bind != "" {
				add("node %s sets network fields, and the Scenario has no network", n.Name)
			}
		}
		return
	}
	sites := map[string]Site{}
	for _, site := range s.Network.Sites {
		if _, ok := sites[site.Name]; ok {
			add("network site %s is named twice", site.Name)
		}
		sites[site.Name] = site
		site.check(add, s)
	}
	for _, n := range s.Nodes {
		site, ok := sites[n.Site]
		if !ok {
			add("node %s's site %q is not a network site", n.Name, n.Site)
			continue
		}
		n.checkNetwork(add, site)
	}
	for _, p := range s.Network.Proxies {
		p.check(add, s)
	}
	for _, r := range s.Network.DNS {
		r.check(add)
	}
}

func (n Node) checkNetwork(add func(format string, args ...any), site Site) {
	oneOf(add, "node "+n.Name+" bind", n.Bind, "", "all", "local")
	for _, r := range n.Firewall {
		oneOf(add, "node "+n.Name+" firewall service", r.Service, Services...)
		oneOf(add, "node "+n.Name+" firewall family", r.Family, "ipv4", "ipv6")
		oneOf(add, "node "+n.Name+" firewall policy", r.Policy, "allows", "blocks", "unknown", "docker")
	}
	for _, a := range n.Addresses {
		if p, err := netip.ParsePrefix(a.Prefix); err != nil || !site.holds(p.Addr()) {
			add("node %s's address %s is not inside site %s", n.Name, a.Prefix, site.Name)
		}
	}
}

func (p Proxy) check(add func(format string, args ...any), s Scenario) {
	if _, err := netip.ParseAddr(p.Address); err != nil {
		add("network proxy %s's address %q is not an IP address", p.Name, p.Address)
	}
	if p.Node != "" && !s.hasNode(p.Node) {
		add("network proxy %s's node %s is not a node", p.Name, p.Node)
	}
	for _, r := range p.Routes {
		oneOf(add, "network proxy "+p.Name+" service", r.Service, Services...)
		if !s.hasNode(r.Node) {
			add("network proxy %s routes %s to %s, which is not a node", p.Name, r.Host, r.Node)
		}
		if !strings.HasSuffix(r.Host, ".localhost") {
			add("network proxy %s routes %s, which is not under .localhost", p.Name, r.Host)
		}
	}
}

func (r Record) check(add func(format string, args ...any)) {
	if !strings.HasSuffix(r.Name, ".localhost") {
		add("network dns name %s is not under .localhost", r.Name)
	}
	oneOf(add, "network dns name "+r.Name+" answer", r.Answer, "", "nxdomain", "servfail", "aaaa-servfail")
	for _, a := range slices.Concat(r.A, r.AAAA, r.Home) {
		if _, err := netip.ParseAddr(a); err != nil {
			add("network dns name %s's address %q is not an IP address", r.Name, a)
		}
	}
}

func (site Site) check(add func(format string, args ...any), s Scenario) {
	what := "network site " + site.Name
	oneOf(add, what+" wan", site.WAN, "public", "cgnat", "private", "empty", "none")
	for _, m := range site.Mapping {
		oneOf(add, what+" mapping", m, "pcp", "nat-pmp", "upnp")
	}
	oneOf(add, what+" refuse", site.Refuse, "", "606", "725", "refuse", "other-port")
	oneOf(add, what+" firewall6", site.Firewall6, "", "open", "closed")
	oneOf(add, what+" upstream", site.Upstream, "", "forwarded", "closed")
	oneOf(add, what+" resolver", site.Resolver, "", "outside", "split")
	oneOf(add, what+" stun", site.STUN, "", "varying", "timeout")
	address, err := netip.ParseAddr(site.Address)
	if err != nil || !address.Is4() {
		add("network site %s's address %q is not an IPv4 address", site.Name, site.Address)
	}
	gateway, err := netip.ParseAddr(site.Gateway)
	switch {
	case site.WAN == "cgnat" && (err != nil || !netip.MustParsePrefix("100.64.0.0/10").Contains(gateway)):
		add("network site %s is cgnat, so its gateway is inside 100.64.0.0/10", site.Name)
	case site.WAN == "private" && (err != nil || !gateway.IsPrivate()):
		add("network site %s is private, so its gateway is a private address", site.Name)
	case site.WAN != "cgnat" && site.WAN != "private" && site.Gateway != "":
		add("network site %s is %s and sets a gateway, which only cgnat and private take", site.Name, site.WAN)
	}
	if site.Upstream != "" && site.WAN != "private" {
		add("network site %s sets upstream, which only a private site's outer router takes", site.Name)
	}
	if site.Prefix6 != "" {
		p, err := netip.ParsePrefix(site.Prefix6)
		if err != nil || p.Bits() != 64 || !documentation6.Contains(p.Addr()) {
			add("network site %s's prefix6 %q is not a /64 inside %s", site.Name, site.Prefix6, documentation6)
		}
	}
	for _, f := range site.Forwards {
		oneOf(add, what+" forward service", f.Service, Services...)
		if !s.hasNode(f.Node) {
			add("network site %s forwards %s to %s, which is not a node", site.Name, f.Service, f.Node)
		}
	}
}

// holds is whether a is one of the site's own addresses: private, its
// public address or inside its IPv6 prefix.
func (site Site) holds(a netip.Addr) bool {
	if a.Is4() {
		return a.IsPrivate() || a.String() == site.Address
	}
	p, err := netip.ParsePrefix(site.Prefix6)
	return a.IsPrivate() || a.IsLinkLocalUnicast() || (err == nil && p.Contains(a))
}

func (s Scenario) hasNode(name string) bool {
	return slices.ContainsFunc(s.Nodes, func(n Node) bool { return n.Name == name })
}

func oneOf(add func(format string, args ...any), what, v string, allowed ...string) {
	if !slices.Contains(allowed, v) {
		add("%s %q is not one of %s", what, v, strings.Join(slices.DeleteFunc(slices.Clone(allowed), func(a string) bool { return a == "" }), ", "))
	}
}
