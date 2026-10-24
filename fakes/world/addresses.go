package world

import (
	"context"
	"fmt"
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

// torrentPort is a port the kernel picked that is free on addr for UDP and
// TCP, both of which libtorrent binds. Given port 0 a Node draws its torrent
// port from its seed, and concurrent runs share seeds and loopback
// addresses, so a pinned Node started again could find another run's Node
// on its port. UDP picks first: Windows reserves UDP port ranges its TCP
// ephemeral ports run through, and refuses them with an access error.
func torrentPort(ctx context.Context, addr netip.Addr) (netip.AddrPort, error) {
	const tries = 8
	var lc net.ListenConfig
	var err error
	for range tries {
		var p net.PacketConn
		if p, err = lc.ListenPacket(ctx, "udp", netip.AddrPortFrom(addr, 0).String()); err != nil {
			return netip.AddrPort{}, fmt.Errorf("torrent port on %s: %w", addr, err)
		}
		ap := netip.MustParseAddrPort(p.LocalAddr().String())
		var l net.Listener
		l, err = lc.Listen(ctx, "tcp", ap.String())
		_ = p.Close()
		if err == nil {
			_ = l.Close()
			return ap, nil
		}
	}
	return netip.AddrPort{}, fmt.Errorf("torrent port on %s: %w", addr, err)
}
