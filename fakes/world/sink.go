package world

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/miekg/dns"

	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// sink is one UDP socket and one TCP listener on 127.0.0.1 that drop
// whatever arrives, and a resolver that knows no name. A Scenario without
// a network block points a Node's gateway, resolver and outside dialer at
// it, so nothing it sends leaves the machine and no network fake has to
// start. The resolver answers at once, as a home network's does, so a
// lookup of Address doesn't wait out its timeout.
type sink struct {
	udp *net.UDPConn
	dns *net.UDPConn
	tcp net.Listener
}

func (r *Run) bindSink(ctx context.Context, undo *unwind) error {
	if r.scenario.Network != nil {
		return nil
	}
	loopback := netip.MustParseAddrPort("127.0.0.1:0")
	udp, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(loopback))
	if err != nil {
		return fmt.Errorf("bind the sink: %w", err)
	}
	names, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(loopback))
	if err != nil {
		_ = udp.Close()
		return fmt.Errorf("bind the sink: %w", err)
	}
	tcp, err := new(net.ListenConfig).Listen(ctx, "tcp", loopback.String())
	if err != nil {
		_ = errors.Join(udp.Close(), names.Close())
		return fmt.Errorf("bind the sink: %w", err)
	}
	r.sink = &sink{udp: udp, dns: names, tcp: tcp}
	go r.sink.drain()
	go r.sink.knowNoName()
	undo.push(func() { _ = errors.Join(udp.Close(), names.Close(), tcp.Close()) })
	return nil
}

func (s *sink) drain() {
	go func() {
		buf := make([]byte, 2048)
		for {
			if _, _, err := s.udp.ReadFrom(buf); err != nil {
				return
			}
		}
	}()
	for {
		c, err := s.tcp.Accept()
		if err != nil {
			return
		}
		_ = c.Close()
	}
}

// knowNoName answers every query NXDOMAIN.
func (s *sink) knowNoName() {
	buf := make([]byte, 512)
	for {
		n, from, err := s.dns.ReadFromUDPAddrPort(buf)
		if err != nil {
			return
		}
		q := new(dns.Msg)
		if q.Unpack(buf[:n]) != nil {
			continue
		}
		if res, err := new(dns.Msg).SetRcode(q, dns.RcodeNameError).Pack(); err == nil {
			_, _ = s.dns.WriteToUDPAddrPort(res, from)
		}
	}
}

// quiet is the endpoints.network section of a Node in a Scenario without
// a network block: no STUN server, and everything else at the sink.
func (s *sink) quiet() netfacts.Endpoints {
	udp := endpoint.Optional{endpoint.MustParseHostPort(s.udp.LocalAddr().String())}
	return netfacts.Endpoints{
		STUN:          endpoint.Optional{},
		Gateway:       udp,
		Resolver:      endpoint.Optional{endpoint.MustParseHostPort(s.dns.LocalAddr().String())},
		OutsideDialer: endpoint.Optional{endpoint.MustParseHostPort(s.tcp.Addr().String())},
	}
}

// emptyEntry is an endpoints entry whose empty value reaches the real
// network: Census sees no host in it, so admit refuses it by name.
type emptyEntry struct{ key, means, fake string }

var emptyIsReal = []emptyEntry{
	{"endpoints.network.gateway", "the real gateway", "gateway"},
	{"endpoints.network.resolver", "the OS resolver", "dns"},
	{"endpoints.network.outside_dialer", "connecting directly", "outside"},
}
