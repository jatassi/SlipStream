//go:build !linux

package host

import (
	"context"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/internal/netmodel"
)

func tunnel(i net.Interface) bool { return i.Flags&net.FlagPointToPoint != 0 }

// Route names the interface that holds local. Its gateway isn't read
// on this OS yet, so it is zero.
func Route(ctx context.Context, local netip.Addr) (string, netip.Addr, error) {
	iface, err := ifaceOf(ctx, local)
	return iface, netip.Addr{}, err
}

// Sockets isn't read on this OS yet, so every port reads Unknown.
func Sockets(context.Context, []uint16) ([]netmodel.Listener, error) { return nil, nil }

// Firewall isn't read on this OS yet.
func Firewall(_ context.Context, port uint16, fam netmodel.Family) netmodel.Rule {
	return netmodel.Rule{Port: port, Family: fam}
}
