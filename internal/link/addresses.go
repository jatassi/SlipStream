package link

import (
	"fmt"
	"net"
	"net/netip"
	"slices"

	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

// MaxAddresses bounds a peer address record, as the Join Code does.
const MaxAddresses = 8

// Addresses is a peer address record: 1 to 8 distinct literal addresses of
// either family, IPv6 first, never a zone or a host name. Only
// ParseAddresses makes one; the zero value is no record. It is comparable,
// so two records compare with ==.
type Addresses struct {
	n    int
	list [MaxAddresses]netip.AddrPort
}

// ParseAddresses orders in IPv6 first, keeping each family's order, and
// drops repeats.
func ParseAddresses(in []netip.AddrPort) (Addresses, error) {
	list := order(in)
	if len(list) < 1 || len(list) > MaxAddresses {
		return Addresses{}, fmt.Errorf("a peer address record holds 1 to %d addresses, not %d", MaxAddresses, len(list))
	}
	for _, a := range list {
		if !a.IsValid() || a.Port() == 0 || a.Addr().Zone() != "" {
			return Addresses{}, fmt.Errorf("address %s is not a literal IP and port", a)
		}
	}
	a := Addresses{n: len(list)}
	copy(a.list[:], list)
	return a, nil
}

func order(in []netip.AddrPort) []netip.AddrPort {
	var v6, v4 []netip.AddrPort
	for _, a := range in {
		a = netip.AddrPortFrom(a.Addr().Unmap(), a.Port())
		if slices.Contains(v6, a) || slices.Contains(v4, a) {
			continue
		}
		if a.Addr().Is4() {
			v4 = append(v4, a)
		} else {
			v6 = append(v6, a)
		}
	}
	return append(v6, v4...)
}

// All is the record in dial order.
func (a Addresses) All() []netip.AddrPort { return slices.Clone(a.list[:a.n]) }

func (a Addresses) IsZero() bool { return a.n == 0 }

func (a Addresses) families() (v6, v4 []netip.AddrPort) {
	list := a.list[:a.n]
	i := slices.IndexFunc(list, func(ap netip.AddrPort) bool { return ap.Addr().Is4() })
	if i < 0 {
		return list, nil
	}
	return list[:i], list[i:]
}

// AddressesFromProto reads the frozen PeerAddresses. An absent or empty
// record is the zero Addresses.
func AddressesFromProto(p *linkv1.PeerAddresses) (Addresses, error) {
	if len(p.GetAddresses()) == 0 {
		return Addresses{}, nil
	}
	list := make([]netip.AddrPort, 0, len(p.GetAddresses()))
	for _, a := range p.GetAddresses() {
		ip, ok := netip.AddrFromSlice(a.GetIp())
		if !ok || a.GetPort() > 0xffff {
			return Addresses{}, fmt.Errorf("address of %d bytes and port %d is not an IP and port", len(a.GetIp()), a.GetPort())
		}
		list = append(list, netip.AddrPortFrom(ip, uint16(a.GetPort()))) //nolint:gosec // checked against 0xffff above
	}
	return ParseAddresses(list)
}

func (a Addresses) Proto() *linkv1.PeerAddresses {
	if a.IsZero() {
		return nil
	}
	p := &linkv1.PeerAddresses{}
	for _, ap := range a.list[:a.n] {
		p.Addresses = append(p.Addresses, &linkv1.Address{Ip: ap.Addr().AsSlice(), Port: uint32(ap.Port())})
	}
	return p
}

// advertise is the record a Node bound to bound gives out: bound itself,
// or for an unspecified address every usable host address it covers with
// the bound port, IPv6 first, at most MaxAddresses.
func advertise(bound netip.AddrPort, host []netip.Addr) (Addresses, bool) {
	if !bound.Addr().IsUnspecified() {
		a, err := ParseAddresses([]netip.AddrPort{bound})
		return a, err == nil
	}
	var list []netip.AddrPort
	for _, ip := range host {
		ip = ip.Unmap()
		if !ip.IsGlobalUnicast() || ip.Zone() != "" || (bound.Addr().Is4() && !ip.Is4()) {
			continue
		}
		list = append(list, netip.AddrPortFrom(ip, bound.Port()))
	}
	list = order(list)
	a, err := ParseAddresses(list[:min(len(list), MaxAddresses)])
	return a, err == nil
}

// Advertising is the record Advertised answers once a link with s serves
// on s.Listen, for a command that runs no link.
func Advertising(s Settings) (Addresses, bool) {
	if len(s.Advertise) > 0 {
		a, err := ParseAddresses(s.Advertise)
		return a, err == nil
	}
	return advertiseBound(s.Listen)
}

func advertiseBound(bound netip.AddrPort) (Addresses, bool) {
	var host []netip.Addr
	if bound.Addr().IsUnspecified() {
		host = hostAddrs()
	}
	return advertise(bound, host)
}

// hostAddrs is every address on an up interface of this machine.
func hostAddrs() []netip.Addr {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []netip.Addr
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok {
				if ip, ok := netip.AddrFromSlice(n.IP); ok {
					out = append(out, ip)
				}
			}
		}
	}
	return out
}
