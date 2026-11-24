// Package gateway is a Node's client of its router, for S51 (Reachability
// diagnostic): NAT-PMP (RFC 6886), PCP (RFC 6887) and UPnP IGD. Read
// learns what the router speaks, its WAN address and the forwards it
// lists, and writes nothing.
package gateway

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// Packets opens the UDP sockets the client speaks over.
type Packets interface {
	// DialUDP is a socket connected to raddr.
	DialUDP(ctx context.Context, raddr netip.AddrPort) (*net.UDPConn, error)
	// ListenUDP is an unconnected socket, for SSDP's multicast search.
	ListenUDP(ctx context.Context, local netip.AddrPort) (*net.UDPConn, error)
}

type Timings struct {
	// Exchange bounds one request and its answer.
	Exchange time.Duration
	// SSDPWait is how long a multicast search waits for answers.
	SSDPWait time.Duration
}

type Client struct {
	Packets Packets
	HTTP    *http.Client
	Timings Timings
}

// Target is where the router is: PCP and NAT-PMP share one address, and
// SSDP goes to the multicast group or, Unicast, to that address. An
// answer counts only from the router's own address.
type Target struct {
	PCP, SSDP netip.AddrPort
	Unicast   bool
	// Local is this Node's address toward the router, which PCP names.
	Local netip.Addr
}

// Discovered is the router at gw, found from the egress route.
func Discovered(gw, local netip.Addr) Target {
	return Target{PCP: netip.AddrPortFrom(gw, 5351), SSDP: netip.MustParseAddrPort("239.255.255.250:1900"), Local: local}
}

// At is the router at an address the Node's config names, which every
// protocol shares and SSDP reaches by unicast.
func At(at netip.AddrPort, local netip.Addr) Target {
	return Target{PCP: at, SSDP: at, Unicast: true, Local: local}
}

// Read asks the router at gw what it speaks, its WAN address and its
// forwards, the wanted ports' among them. Where the router is, Addr, is
// the caller's to say. A protocol that stays silent
// reads No; only a failure to send is an error.
func (c *Client) Read(ctx context.Context, gw Target, wanted []uint16) (netmodel.Gateway, error) {
	var out netmodel.Gateway
	wan, speaks, err := c.natPMPAddress(ctx, gw)
	if err != nil {
		return out, err
	}
	out.Protocols.NATPMP, out.WAN = speaks, wan
	if out.Protocols.PCP, err = c.pcpAnnounce(ctx, gw); err != nil {
		return out, err
	}
	igd, err := c.upnp(ctx, gw, wanted)
	if err != nil {
		return out, err
	}
	out.Protocols.UPnP = igd.speaks
	if !out.WAN.IsValid() {
		out.WAN = igd.wan
	}
	out.Forwards = igd.forwards
	return out, nil
}

// exchange sends req to at and answers the first reply, nil when none
// comes inside the exchange bound.
func (c *Client) exchange(ctx context.Context, at netip.AddrPort, req []byte) ([]byte, error) {
	conn, err := c.Packets.DialUDP(ctx, at)
	if err != nil {
		return nil, fmt.Errorf("open a socket to the router: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Write(req); err != nil {
		return nil, nil
	}
	deadline := time.Now().Add(c.Timings.Exchange)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetReadDeadline(deadline)
	buf := make([]byte, 1100)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, nil
	}
	return buf[:n], nil
}

const (
	natPMPVersion        = 0
	natPMPNetworkFailure = 3
	pcpVersion           = 2
)

// natPMPAddress asks NAT-PMP's opcode 0. A network failure answer is a
// router that speaks it with no WAN address.
func (c *Client) natPMPAddress(ctx context.Context, gw Target) (netip.Addr, netmodel.Tri, error) {
	reply, err := c.exchange(ctx, gw.PCP, []byte{natPMPVersion, 0})
	if err != nil || len(reply) < 8 || reply[0] != natPMPVersion || reply[1] != 128 {
		return netip.Addr{}, netmodel.No, err
	}
	switch binary.BigEndian.Uint16(reply[2:]) {
	case 0:
		if len(reply) < 12 {
			return netip.Addr{}, netmodel.No, nil
		}
		wan := netip.AddrFrom4([4]byte(reply[8:12]))
		if wan.IsUnspecified() {
			wan = netip.Addr{}
		}
		return wan, netmodel.Yes, nil
	case natPMPNetworkFailure:
		return netip.Addr{}, netmodel.Yes, nil
	}
	return netip.Addr{}, netmodel.No, nil
}

// pcpAnnounce sends PCP's ANNOUNCE, which writes nothing. A NAT-PMP
// router answers it with its own unsupported version.
func (c *Client) pcpAnnounce(ctx context.Context, gw Target) (netmodel.Tri, error) {
	req := make([]byte, 24)
	req[0] = pcpVersion
	local := gw.Local.As16()
	copy(req[8:], local[:])
	reply, err := c.exchange(ctx, gw.PCP, req)
	if err != nil || len(reply) < 4 || reply[0] != pcpVersion || reply[1] != 0x80 {
		return netmodel.No, err
	}
	return netmodel.Yes, nil
}

// readFrom answers the first datagram on conn from the router, nil when
// none comes by deadline.
func readFrom(conn *net.UDPConn, router netip.Addr, deadline time.Time, keep func([]byte) bool) []byte {
	_ = conn.SetReadDeadline(deadline)
	buf := make([]byte, 2048)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return nil
		}
		if from.Addr().Unmap() == router.Unmap() && keep(buf[:n]) {
			return bytes.Clone(buf[:n])
		}
	}
}
