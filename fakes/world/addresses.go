package world

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
)

// CheckAddresses binds and closes the address of each of n Nodes, and
// returns them. Past the 31 addresses setup-loopback aliases, it refuses
// without trying.
func CheckAddresses(ctx context.Context, n int) ([]netip.Addr, error) {
	addrs := make([]netip.Addr, n)
	for i := range n {
		addr := setup.Address(i)
		refuse := fmt.Errorf(`%s is not usable on this machine; on macOS run "sudo fakeworld setup-loopback" once`, addr)
		if i >= scenario.MaxNodes {
			return nil, refuse
		}
		l, err := new(net.ListenConfig).Listen(ctx, "tcp", netip.AddrPortFrom(addr, 0).String())
		if err != nil {
			return nil, refuse
		}
		_ = l.Close()
		addrs[i] = addr
	}
	return addrs, nil
}

// torrentPort is a port free on addr for both TCP and UDP, both of which
// libtorrent binds. Given port 0 a Node draws its torrent port from its
// seed, and concurrent runs share seeds and loopback addresses, so a pinned
// Node started again could find another run's Node on its port. It draws
// below the ephemeral range: Windows reserves runs of ephemeral ports for
// one protocol, and the kernel's next pick sits beside its last one.
func torrentPort(ctx context.Context, addr netip.Addr) (netip.AddrPort, error) {
	const (
		tries = 32
		low   = 20000
		high  = 40000
	)
	var lc net.ListenConfig
	var err error
	for range tries {
		ap := netip.AddrPortFrom(addr, uint16(low+rand.IntN(high-low))) //nolint:gosec // a port to try, not a secret
		var l net.Listener
		if l, err = lc.Listen(ctx, "tcp", ap.String()); err != nil {
			continue
		}
		var p net.PacketConn
		p, err = lc.ListenPacket(ctx, "udp", ap.String())
		_ = l.Close()
		if err == nil {
			_ = p.Close()
			return ap, nil
		}
	}
	return netip.AddrPort{}, fmt.Errorf("torrent port on %s: %w", addr, err)
}
