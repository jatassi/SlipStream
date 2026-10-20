package downloading

import "net/netip"

// cgnat is the shared address space carriers put behind their NAT (RFC
// 6598): a peer there may be a neighbour on the same carrier.
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// PublicPeer is the listen-port check's rule: whether an incoming peer at
// an address proves the port reachable from the internet. Loopback,
// private, link-local and CGNAT addresses prove nothing, unless
// loopbackProves, for a world whose every peer is on loopback.
func PublicPeer(loopbackProves bool) func(netip.Addr) bool {
	return func(a netip.Addr) bool {
		a = a.Unmap()
		if a.IsLoopback() {
			return loopbackProves
		}
		return a.IsGlobalUnicast() && !a.IsPrivate() && !cgnat.Contains(a)
	}
}
