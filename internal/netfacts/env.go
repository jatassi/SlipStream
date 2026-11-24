package netfacts

import (
	"context"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/internal/netfacts/gateway"
	"github.com/jatassi/slipstream/internal/netfacts/internal/host"
	"github.com/jatassi/slipstream/internal/netmodel"
)

// Env is every way netfacts reads or reaches the network. internal/node
// builds it: OS's real readers, or Scripted's under the Fake world's
// scripted-network divergence. Nothing in netfacts binds a socket itself.
type Env struct {
	Interfaces func(ctx context.Context) ([]netmodel.Iface, error)
	// Route is the interface and gateway a local address leaves by: the
	// egress learned from the STUN socket's local address.
	Route func(ctx context.Context, local netip.Addr) (Route, error)
	// Sockets is what listens on each port. A port it can't read is left
	// out, so it reads Unknown.
	Sockets func(ctx context.Context, ports []uint16) ([]netmodel.Listener, error)
	// Firewall answers each query. A reader that can't tell answers
	// PolicyUnknown, never Allows.
	Firewall func(ctx context.Context, qs []Query) ([]netmodel.Rule, error)
	Packets  PacketNet
	// Lookup is the OS resolver, used when the resolver entry is empty.
	Lookup func(ctx context.Context, host string) ([]netip.Addr, error)
	// Self is SlipStream's own listeners, their families read from the
	// sockets themselves, since [::] shows neither.
	Self func() []netmodel.Listener
}

// PacketNet opens UDP sockets: connected ones for STUN, PCP, NAT-PMP and
// DNS, and unconnected ones for SSDP's multicast search.
type PacketNet = gateway.Packets

type Route struct {
	Iface string
	// Gateway is zero when the route has none, as on a VPS's /32 or a
	// tunnel.
	Gateway netip.Addr
}

// Query asks the firewall about one TCP port and family.
type Query struct {
	Port   uint16
	Family netmodel.Family
}

// OS is this machine's readers, with packets for its sockets. The Node
// sets Self once its listeners are bound.
func OS(packets PacketNet) Env {
	return Env{
		Interfaces: host.Interfaces,
		Route: func(ctx context.Context, local netip.Addr) (Route, error) {
			iface, gw, err := host.Route(ctx, local)
			return Route{Iface: iface, Gateway: gw}, err
		},
		Sockets: host.Sockets,
		Firewall: func(ctx context.Context, qs []Query) ([]netmodel.Rule, error) {
			out := make([]netmodel.Rule, len(qs))
			for i, q := range qs {
				out[i] = host.Firewall(ctx, q.Port, q.Family)
			}
			return out, nil
		},
		Packets: packets,
		Lookup: func(ctx context.Context, name string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", name)
		},
	}
}
