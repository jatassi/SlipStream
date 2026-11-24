package world

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// sink is one UDP socket and one TCP listener on 127.0.0.1 that drop
// whatever arrives. A Scenario without a network block points a Node's
// gateway, resolver and outside dialer at it, so nothing it sends leaves
// the machine and no network fake has to start.
type sink struct {
	udp *net.UDPConn
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
	tcp, err := new(net.ListenConfig).Listen(ctx, "tcp", loopback.String())
	if err != nil {
		_ = udp.Close()
		return fmt.Errorf("bind the sink: %w", err)
	}
	r.sink = &sink{udp: udp, tcp: tcp}
	go r.sink.drain()
	undo.push(func() { _ = errors.Join(udp.Close(), tcp.Close()) })
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

// quiet is the endpoints.network section of a Node in a Scenario without
// a network block: no STUN server, and everything else at the sink.
func (s *sink) quiet() netfacts.Endpoints {
	udp := endpoint.Optional{endpoint.MustParseHostPort(s.udp.LocalAddr().String())}
	return netfacts.Endpoints{
		STUN:          endpoint.Optional{},
		Gateway:       udp,
		Resolver:      udp,
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
