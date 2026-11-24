package gateway

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// ErrNoAnswer is a router that stayed silent, or answered what isn't the
// protocol: whether it took the request is unknown.
var ErrNoAnswer = errors.New("the router didn't answer")

// Ask is one TCP mapping asked of the router, from External on its
// internet side to Internal on this Node, by Method.
type Ask struct {
	Method   netmodel.MapProtocol
	External uint16
	Internal uint16
	Lifetime time.Duration
	// Nonce is PCP's mapping nonce: a renewal or a delete names the
	// mapping by it.
	Nonce [12]byte
	// Description names a UPnP entry, which is how a Node knows its own
	// entries when it lost its record of them.
	Description string
}

// Toward is the router at its PCP and NAT-PMP address at, with this Node's
// address toward it: reached by unicast, as At, or else as Discovered.
func (c *Client) Toward(ctx context.Context, at netip.AddrPort, unicast bool) (Target, error) {
	conn, err := c.Packets.DialUDP(ctx, at)
	if err != nil {
		return Target{}, fmt.Errorf("open a socket to the router: %w", err)
	}
	local := localOf(conn)
	_ = conn.Close()
	if unicast {
		return At(at, local), nil
	}
	return Discovered(at.Addr(), local), nil
}

func localOf(conn *net.UDPConn) netip.Addr {
	at, _ := netip.ParseAddrPort(conn.LocalAddr().String())
	return at.Addr().Unmap()
}

// Map asks the router for a. A grant on another external port is deleted
// at once and answers OtherPort; a refusal answers its kind. A UPnP router
// that takes only permanent entries (725) is asked again for one, which
// answers a Lifetime of 0. ErrNoAnswer leaves whether the router took it
// unknown.
func (c *Client) Map(ctx context.Context, gw Target, a Ask) (netmodel.Mapping, error) {
	out := netmodel.Mapping{Protocol: a.Method, External: a.External, Internal: a.Internal}
	var granted grant
	var err error
	switch a.Method {
	case netmodel.MapPCP:
		granted, err = c.pcpMap(ctx, gw, a, seconds(a.Lifetime))
	case netmodel.MapNATPMP:
		granted, err = c.natPMPMap(ctx, gw, a, seconds(a.Lifetime))
	case netmodel.MapUPnP:
		granted, err = c.upnpMap(ctx, gw, a)
	}
	if err != nil {
		return out, err
	}
	if granted.refused != netmodel.RefusedNone {
		out.Refused = granted.refused
		return out, nil
	}
	if granted.external != a.External {
		if err := c.Unmap(ctx, gw, a); err != nil {
			return out, err
		}
		out.Refused = netmodel.OtherPort
		return out, nil
	}
	out.Lifetime = granted.lifetime
	if granted.lifetime > 0 {
		out.Expires = time.Now().Add(granted.lifetime)
	}
	return out, nil
}

// Unmap deletes the mapping a asked for. A router with no such mapping
// already holds what Unmap wants.
func (c *Client) Unmap(ctx context.Context, gw Target, a Ask) error {
	var granted grant
	var err error
	switch a.Method {
	case netmodel.MapPCP:
		granted, err = c.pcpMap(ctx, gw, Ask{Method: a.Method, Internal: a.Internal, Nonce: a.Nonce}, 0)
	case netmodel.MapNATPMP:
		granted, err = c.natPMPMap(ctx, gw, Ask{Method: a.Method, Internal: a.Internal}, 0)
	case netmodel.MapUPnP:
		return c.upnpUnmap(ctx, gw, a)
	}
	if err != nil {
		return err
	}
	if granted.refused != netmodel.RefusedNone {
		return fmt.Errorf("the router refused to delete port %d's mapping", a.External)
	}
	return nil
}

// grant is a router's answer to a mapping request.
type grant struct {
	external uint16
	lifetime time.Duration
	refused  netmodel.MapRefusal
}

func seconds(d time.Duration) uint32 { return uint32(d / time.Second) } //nolint:gosec // G115: a lifetime asked is hours, far below 136 years

// exchangeFrom is exchange with a request that names this Node's address
// toward the router, as PCP's does.
func (c *Client) exchangeFrom(ctx context.Context, at netip.AddrPort, build func(local netip.Addr) []byte) ([]byte, error) {
	conn, err := c.Packets.DialUDP(ctx, at)
	if err != nil {
		return nil, fmt.Errorf("open a socket to the router: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Write(build(localOf(conn))); err != nil {
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

// PCP's MAP, RFC 6887 section 11.
const (
	pcpMapOp        = 1
	pcpTCP          = 6
	pcpNotAuthorize = 2
)

func (c *Client) pcpMap(ctx context.Context, gw Target, a Ask, lifetime uint32) (grant, error) {
	reply, err := c.exchangeFrom(ctx, gw.PCP, func(local netip.Addr) []byte {
		req := make([]byte, pcpHeaderLen+pcpMapLen)
		req[0], req[1] = pcpVersion, pcpMapOp
		binary.BigEndian.PutUint32(req[4:], lifetime)
		client := local.As16()
		copy(req[8:], client[:])
		data := req[pcpHeaderLen:]
		copy(data, a.Nonce[:])
		data[12] = pcpTCP
		binary.BigEndian.PutUint16(data[16:], a.Internal)
		binary.BigEndian.PutUint16(data[18:], a.External)
		suggested := netip.IPv4Unspecified().As16()
		copy(data[20:], suggested[:])
		return req
	})
	if err != nil {
		return grant{}, err
	}
	if len(reply) < pcpHeaderLen+pcpMapLen || reply[0] != pcpVersion || reply[1] != 0x80|pcpMapOp || [12]byte(reply[pcpHeaderLen:]) != a.Nonce {
		return grant{}, ErrNoAnswer
	}
	switch reply[3] {
	case 0:
		data := reply[pcpHeaderLen:]
		return grant{external: binary.BigEndian.Uint16(data[18:]), lifetime: time.Duration(binary.BigEndian.Uint32(reply[4:])) * time.Second}, nil
	case pcpNotAuthorize:
		return grant{refused: netmodel.NotAuthorized}, nil
	}
	return grant{refused: netmodel.NoAnswer}, nil
}

const (
	pcpHeaderLen = 24
	pcpMapLen    = 36
)

// NAT-PMP's TCP mapping, RFC 6886 section 3.3.
const (
	natPMPMapTCP        = 2
	natPMPNotAuthorized = 2
)

func (c *Client) natPMPMap(ctx context.Context, gw Target, a Ask, lifetime uint32) (grant, error) {
	req := make([]byte, 12)
	req[1] = natPMPMapTCP
	binary.BigEndian.PutUint16(req[4:], a.Internal)
	binary.BigEndian.PutUint16(req[6:], a.External)
	binary.BigEndian.PutUint32(req[8:], lifetime)
	reply, err := c.exchange(ctx, gw.PCP, req)
	if err != nil {
		return grant{}, err
	}
	if len(reply) < 16 || reply[0] != natPMPVersion || reply[1] != 128+natPMPMapTCP || binary.BigEndian.Uint16(reply[8:]) != a.Internal {
		return grant{}, ErrNoAnswer
	}
	switch binary.BigEndian.Uint16(reply[2:]) {
	case 0:
		return grant{external: binary.BigEndian.Uint16(reply[10:]), lifetime: time.Duration(binary.BigEndian.Uint32(reply[12:])) * time.Second}, nil
	case natPMPNotAuthorized:
		return grant{refused: netmodel.NotAuthorized}, nil
	}
	return grant{refused: netmodel.NoAnswer}, nil
}

// UPnP's error codes, WANIPConnection:1 section 2.4.
const (
	upnpNotAuthorized = 606
	upnpNoSuchEntry   = 714
	upnpConflict      = 718
	upnpPermanentOnly = 725
)

// control is the router's WAN connection service a mapping goes to: the
// first that answers an external address, as a router listing both
// WANIPConnection and WANPPPConnection keeps one set of entries.
func (c *Client) control(ctx context.Context, gw Target) (service, error) {
	location, err := c.search(ctx, gw)
	if err != nil {
		return service{}, err
	}
	if location == nil {
		return service{}, ErrNoAnswer
	}
	services, err := c.services(ctx, location)
	if err != nil {
		return service{}, err
	}
	if len(services) == 0 {
		return service{}, ErrNoAnswer
	}
	for _, s := range services {
		if c.wan(ctx, s).IsValid() {
			return s, nil
		}
	}
	return services[0], nil
}

func (c *Client) upnpMap(ctx context.Context, gw Target, a Ask) (grant, error) {
	s, err := c.control(ctx, gw)
	if err != nil {
		return grant{}, err
	}
	lease := seconds(a.Lifetime)
	for {
		_, code := c.soap(ctx, s, "AddPortMapping", [][2]string{
			{"NewRemoteHost", ""}, {"NewExternalPort", strconv.Itoa(int(a.External))}, {"NewProtocol", "TCP"},
			{"NewInternalPort", strconv.Itoa(int(a.Internal))}, {"NewInternalClient", gw.Local.String()}, {"NewEnabled", "1"},
			{"NewPortMappingDescription", a.Description}, {"NewLeaseDuration", strconv.FormatUint(uint64(lease), 10)},
		})
		switch {
		case code == 0:
			return grant{external: a.External, lifetime: time.Duration(lease) * time.Second}, nil
		case code == upnpPermanentOnly && lease != 0:
			lease = 0
		case code == upnpNotAuthorized:
			return grant{refused: netmodel.NotAuthorized}, nil
		case code == upnpConflict:
			return grant{refused: netmodel.OtherPort}, nil
		case code < 0:
			return grant{}, ErrNoAnswer
		default:
			return grant{refused: netmodel.NoAnswer}, nil
		}
	}
}

func (c *Client) upnpUnmap(ctx context.Context, gw Target, a Ask) error {
	s, err := c.control(ctx, gw)
	if err != nil {
		return err
	}
	_, code := c.soap(ctx, s, "DeletePortMapping", [][2]string{
		{"NewRemoteHost", ""}, {"NewExternalPort", strconv.Itoa(int(a.External))}, {"NewProtocol", "TCP"},
	})
	switch code {
	case 0, upnpNoSuchEntry:
		return nil
	case -1:
		return ErrNoAnswer
	}
	return fmt.Errorf("the router refused to delete port %d's mapping with UPnP error %d", a.External, code)
}
