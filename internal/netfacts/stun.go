package netfacts

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strconv"
	"syscall"
	"time"

	"github.com/pion/stun/v4"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// askSTUN sends one Binding request to each server, over IPv4 only: STUN
// over IPv6 is S60 (IPv6 direct reach)'s. The first socket's local address
// is the egress.
func (n *Netfacts) askSTUN(ctx context.Context, servers []string) ([]netmodel.STUNAnswer, netip.Addr) {
	var local netip.Addr
	out := make([]netmodel.STUNAnswer, len(servers))
	for i, s := range servers {
		out[i] = netmodel.STUNAnswer{Server: s}
		at, ok := n.stunAddr(ctx, s)
		if !ok {
			out[i].Err = netmodel.STUNNoAnswer
			continue
		}
		var from netip.Addr
		out[i].Mapped, from, out[i].Err = n.bind(ctx, at)
		if !local.IsValid() && from.IsValid() {
			local = from
		}
	}
	return out, local
}

// stunAddr is server's IPv4 address, false when it has none.
func (n *Netfacts) stunAddr(ctx context.Context, server string) (netip.AddrPort, bool) {
	host, p, err := net.SplitHostPort(server)
	if err != nil {
		return netip.AddrPort{}, false
	}
	port, err := strconv.ParseUint(p, 10, 16)
	if err != nil {
		return netip.AddrPort{}, false
	}
	if a, err := netip.ParseAddr(host); err == nil {
		return netip.AddrPortFrom(a, uint16(port)), a.Is4()
	}
	ctx, cancel := context.WithTimeout(ctx, n.cfg.Timings.Resolve)
	defer cancel()
	addrs, err := n.cfg.Env.Lookup(ctx, host)
	if err != nil {
		return netip.AddrPort{}, false
	}
	for _, a := range addrs {
		if a.Unmap().Is4() {
			return netip.AddrPortFrom(a.Unmap(), uint16(port)), true
		}
	}
	return netip.AddrPort{}, false
}

// bind sends one Binding request to at and answers the mapped address,
// the socket's local address, and why there is no answer.
func (n *Netfacts) bind(ctx context.Context, at netip.AddrPort) (netip.AddrPort, netip.Addr, netmodel.STUNError) {
	conn, err := n.cfg.Env.Packets.DialUDP(ctx, at)
	if err != nil {
		return netip.AddrPort{}, netip.Addr{}, netmodel.STUNNoAnswer
	}
	defer func() { _ = conn.Close() }()
	local := netip.MustParseAddrPort(conn.LocalAddr().String()).Addr().Unmap()
	req, err := stun.Build(stun.TransactionID, stun.BindingRequest, stun.Fingerprint)
	if err != nil {
		return netip.AddrPort{}, local, netmodel.STUNNoAnswer
	}
	if _, err := conn.Write(req.Raw); err != nil {
		return netip.AddrPort{}, local, stunError(err)
	}
	deadline := time.Now().Add(n.cfg.Timings.STUN)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetReadDeadline(deadline)
	buf := make([]byte, 1500)
	for {
		size, err := conn.Read(buf)
		if err != nil {
			return netip.AddrPort{}, local, stunError(err)
		}
		res := &stun.Message{Raw: buf[:size]}
		if res.Decode() != nil || res.TransactionID != req.TransactionID || res.Type != stun.BindingSuccess {
			continue
		}
		var mapped stun.XORMappedAddress
		if mapped.GetFrom(res) != nil {
			return netip.AddrPort{}, local, netmodel.STUNNoAnswer
		}
		addr, _ := netip.AddrFromSlice(mapped.IP)
		return netip.AddrPortFrom(addr.Unmap(), uint16(mapped.Port)), local, netmodel.STUNNone //nolint:gosec // G115: a STUN port is 16 bits
	}
}

func stunError(err error) netmodel.STUNError {
	var timeout interface{ Timeout() bool }
	switch {
	case errors.Is(err, syscall.ECONNREFUSED):
		return netmodel.STUNRefused
	case errors.As(err, &timeout) && timeout.Timeout():
		return netmodel.STUNTimeout
	}
	return netmodel.STUNNoAnswer
}
