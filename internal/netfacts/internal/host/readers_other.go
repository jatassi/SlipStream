//go:build !linux && !windows && !darwin

package host

import (
	"context"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/internal/netmodel"
)

func tunnels() (func(net.Interface) bool, error) {
	return func(i net.Interface) bool { return i.Flags&net.FlagPointToPoint != 0 }, nil
}

// Route names the interface that holds local. Its gateway isn't read
// on this OS, so it is zero.
func Route(ctx context.Context, local netip.Addr) (string, netip.Addr, error) {
	iface, err := ifaceOf(ctx, local)
	return iface, netip.Addr{}, err
}

// Sockets isn't read on this OS, so every port reads Unknown.
func Sockets(context.Context, []uint16) ([]netmodel.Listener, error) { return nil, nil }

// Firewall isn't read on this OS.
func Firewall(_ context.Context, _ string, asks []netmodel.Rule) ([]netmodel.Rule, error) {
	return asks, nil
}
