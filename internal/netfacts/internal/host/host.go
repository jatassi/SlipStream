// Package host is this machine's network readers for netfacts: its
// interfaces, the route a local address leaves by, what listens on a port
// and what the host firewall says about it. A reader that can't tell
// leaves a fact Unknown, never Yes or Allows.
package host

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// Interfaces is every interface that is up, with its addresses. Their
// class is the caller's, under its Rules.
func Interfaces(context.Context) ([]netmodel.Iface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("read the network interfaces: %w", err)
	}
	var out []netmodel.Iface
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			return nil, fmt.Errorf("read %s's addresses: %w", i.Name, err)
		}
		iface := netmodel.Iface{Name: i.Name, Index: i.Index, Tunnel: tunnel(i)}
		for _, a := range addrs {
			n, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			addr, _ := netip.AddrFromSlice(n.IP)
			bits, _ := n.Mask.Size()
			iface.Addrs = append(iface.Addrs, netmodel.IfaceAddr{Prefix: netip.PrefixFrom(addr.Unmap(), bits)})
		}
		out = append(out, iface)
	}
	return out, nil
}

// ifaceOf is the name of the interface that holds local, empty when none
// does.
func ifaceOf(ctx context.Context, local netip.Addr) (string, error) {
	ifs, err := Interfaces(ctx)
	if err != nil {
		return "", err
	}
	for _, i := range ifs {
		for _, a := range i.Addrs {
			if a.Prefix.Addr() == local.Unmap() {
				return i.Name, nil
			}
		}
	}
	return "", nil
}
