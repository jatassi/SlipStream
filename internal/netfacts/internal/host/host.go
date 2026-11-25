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
	tunnel, err := tunnels()
	if err != nil {
		return nil, err
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

// socket is one listening TCP socket.
type socket struct {
	local netip.AddrPort
	owner string // the owning program, empty when unread
	// v4 is whether a wildcard IPv6 socket takes IPv4 too.
	v4 netmodel.Tri
}

// listenerOn is what socks say listens on port p. A port nothing listens
// on reads No for both families.
func listenerOn(p uint16, socks []socket) netmodel.Listener {
	l := netmodel.Listener{Port: p, V4: netmodel.No, V6: netmodel.No, Loopback: true}
	found := false
	for _, s := range socks {
		if s.local.Port() != p {
			continue
		}
		found = true
		if s.owner != "" {
			l.Program.Name = s.owner
			l.Docker = l.Docker || s.owner == "docker-proxy"
		}
		a := s.local.Addr()
		if a.IsLoopback() || a.Unmap().IsLoopback() {
			continue
		}
		l.Loopback = false
		switch {
		case a.Is4() || a.Is4In6():
			l.V4 = netmodel.Yes
		case a.IsUnspecified():
			l.V6 = netmodel.Yes
			if l.V4 != netmodel.Yes {
				l.V4 = s.v4
			}
		default:
			l.V6 = netmodel.Yes
		}
	}
	l.Loopback = l.Loopback && found
	return l
}

func listenersOn(ports []uint16, socks []socket) []netmodel.Listener {
	out := make([]netmodel.Listener, 0, len(ports))
	for _, p := range ports {
		out = append(out, listenerOn(p, socks))
	}
	return out
}
