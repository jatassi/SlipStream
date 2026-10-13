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
