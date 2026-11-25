// Package netmodel is one model of a Node's network that the Node and the
// Controller share: what an address is for outsiders, the facts a Node
// reads about its interfaces, gateway, STUN answers, sockets and firewall,
// the class rule over them, and the summary the Node publishes. It is pure:
// it reads nothing and holds no state.
package netmodel

import "net/netip"

// AddrClass is what an address is for outsiders. Its zero value is Other,
// never Global, so an unclassified address is never a target.
type AddrClass uint8

const (
	Other AddrClass = iota // unspecified, multicast, anything unlisted
	Global
	Private // RFC 1918
	Shared  // 100.64.0.0/10, RFC 6598
	ULA     // fc00::/7
	LinkLocal
	Loopback
	SixToFour     // 2002::/16
	Teredo        // 2001::/32
	NAT64         // 64:ff9b::/96 and 64:ff9b:1::/48
	Documentation // 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24, 2001:db8::/32
	Mapped        // ::ffff:0:0/96
)

var (
	shared        = netip.MustParsePrefix("100.64.0.0/10")
	sixToFour     = netip.MustParsePrefix("2002::/16")
	teredo        = netip.MustParsePrefix("2001::/32")
	nat64         = []netip.Prefix{netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48")}
	documentation = []netip.Prefix{
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("2001:db8::/32"),
	}
)

// Rules are the classification rules. DocumentationGlobal is the
// scripted-network divergence: the Fake world's sites live on the IPv4
// TEST-NETs and 2001:db8::/32.
type Rules struct{ DocumentationGlobal bool }

// Of checks the special prefixes before netip's IsGlobalUnicast, which is
// true for 2001:db8::/32, 6to4, Teredo and NAT64.
func (r Rules) Of(a netip.Addr) AddrClass {
	switch {
	case !a.IsValid() || a.IsUnspecified() || a.IsMulticast():
		return Other
	case a.Is4In6():
		return Mapped
	case a.IsLoopback():
		return Loopback
	case a.IsLinkLocalUnicast():
		return LinkLocal
	case a.Is6() && a.IsPrivate():
		return ULA
	case a.IsPrivate():
		return Private
	case shared.Contains(a):
		return Shared
	case in(documentation, a):
		if r.DocumentationGlobal {
			return Global
		}
		return Documentation
	case sixToFour.Contains(a):
		return SixToFour
	case teredo.Contains(a):
		return Teredo
	case in(nat64, a):
		return NAT64
	case a.IsGlobalUnicast():
		return Global
	}
	return Other
}

func (r Rules) Global(a netip.Addr) bool { return r.Of(a) == Global }

func in(ps []netip.Prefix, a netip.Addr) bool {
	for _, p := range ps {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// Slash64 is the /64 that holds a.
func Slash64(a netip.Addr) netip.Prefix {
	p, _ := a.Prefix(64)
	return p
}
