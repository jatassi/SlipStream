package host

import (
	"fmt"
	"net"
	"net/netip"
)

// Families is which IP families a listener accepts.
type Families struct{ IPv4, IPv6 bool }

// FamiliesOf reads which families ln accepts. An IPv6 wildcard socket
// accepts IPv4 too unless IPV6_V6ONLY is set, which ln.Addr() doesn't
// show: it reads [::] either way.
func FamiliesOf(ln *net.TCPListener) (Families, error) {
	bound, err := netip.ParseAddrPort(ln.Addr().String())
	if err != nil {
		return Families{}, fmt.Errorf("read the web listener's address: %w", err)
	}
	addr := bound.Addr()
	if addr.Is4() || addr.Is4In6() {
		return Families{IPv4: true}, nil
	}
	if !addr.IsUnspecified() {
		return Families{IPv6: true}, nil
	}
	rc, err := ln.SyscallConn()
	if err != nil {
		return Families{}, fmt.Errorf("read the web listener's socket: %w", err)
	}
	only, err := v6Only(rc)
	if err != nil {
		return Families{}, fmt.Errorf("read IPV6_V6ONLY on the web listener: %w", err)
	}
	return Families{IPv4: !only, IPv6: true}, nil
}
