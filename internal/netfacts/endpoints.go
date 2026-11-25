// Package netfacts gathers a Node's network facts: its interfaces, the
// address the internet sees it from, its gateway, and its listeners and
// firewall, for S51 (Reachability diagnostic).
package netfacts

import "github.com/jatassi/slipstream/internal/node/endpoint"

// Endpoints is a Node's endpoints.network section.
type Endpoints struct {
	// STUN are the servers asked for the Node's public address. Empty asks
	// none.
	STUN endpoint.Optional `yaml:"stun"`
	// Gateway is where PCP, NAT-PMP and SSDP go, by unicast. Empty
	// discovers the egress interface's gateway.
	Gateway endpoint.Optional `yaml:"gateway"`
	// Resolver answers the Node's lookups of public names. Empty is the
	// OS resolver.
	Resolver endpoint.Optional `yaml:"resolver"`
	// OutsideDialer is a SOCKS5 proxy a tester connects through. Empty
	// connects directly.
	OutsideDialer endpoint.Optional `yaml:"outside_dialer"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{STUN: endpoint.Optional{
		endpoint.MustParseHostPort("stun.cloudflare.com:3478"),
		endpoint.MustParseHostPort("stun.l.google.com:19302"),
	}}
}
