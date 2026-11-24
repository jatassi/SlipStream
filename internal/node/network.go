package node

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/node/internal/host"
)

// osPackets opens this process's UDP sockets for netfacts.
type osPackets struct{}

func (osPackets) DialUDP(_ context.Context, raddr netip.AddrPort) (*net.UDPConn, error) {
	c, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(raddr))
	if err != nil {
		return nil, fmt.Errorf("open a UDP socket to %s: %w", raddr, err)
	}
	return c, nil
}

func (osPackets) ListenUDP(_ context.Context, local netip.AddrPort) (*net.UDPConn, error) {
	c, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(local))
	if err != nil {
		return nil, fmt.Errorf("open a UDP socket on %s: %w", local, err)
	}
	return c, nil
}

// newNetwork is the half's network facts over the Env's readers, or over
// the scripted readers' file under the Fake world's scripted-network
// divergence.
func (n *Node) newNetwork() *netfacts.Netfacts {
	env := n.env.Network
	env.Self = n.webListeners
	if n.cfg.Network.Script != "" {
		env = netfacts.Scripted(n.cfg.Network.Script, env)
	}
	return netfacts.New(netfacts.Config{
		Env:       env,
		Endpoints: n.cfg.Endpoints.Network,
		Timings:   n.cfg.Timings.Netfacts,
		Rules:     netmodel.RulesOf(n.cfg.Network),
		HTTP:      n.httpClient(n.env.Net.DialContext),
		Log:       n.rootLog.With("subsystem", "netfacts"),
	})
}

// webListeners is SlipStream's own web listener as netfacts reports it,
// none before HTTP binds.
func (n *Node) webListeners() []netmodel.Listener {
	if l := n.web.Load(); l != nil {
		return []netmodel.Listener{*l}
	}
	return nil
}

// webListener is the one port lns bound, with the families they accept
// beyond loopback, read from each socket since [::] shows neither. A test
// host's in-memory listener has no socket, so its families read Unknown.
func webListener(lns []net.Listener) netmodel.Listener {
	l := netmodel.Listener{Port: boundTCP(lns[0]).Port(), Program: netmodel.Program{Name: "SlipStream", SlipStream: true}, V4: netmodel.No, V6: netmodel.No, Loopback: true}
	for _, ln := range lns {
		if boundTCP(ln).Addr().IsLoopback() {
			continue
		}
		l.Loopback = false
		tcp, ok := ln.(*net.TCPListener)
		if !ok {
			l.V4, l.V6 = netmodel.Unknown, netmodel.Unknown
			continue
		}
		f, err := host.FamiliesOf(tcp)
		if err != nil {
			l.V4, l.V6 = netmodel.Unknown, netmodel.Unknown
			continue
		}
		if f.IPv4 {
			l.V4 = netmodel.Yes
		}
		if f.IPv6 {
			l.V6 = netmodel.Yes
		}
	}
	return l
}

// hasLoopback6 is whether this host has ::1, which a host with IPv6
// turned off lacks.
func hasLoopback6(ctx context.Context, interfaces func(context.Context) ([]netmodel.Iface, error)) (bool, error) {
	ifaces, err := interfaces(ctx)
	if err != nil {
		return false, fmt.Errorf("read the interfaces: %w", err)
	}
	for _, i := range ifaces {
		for _, a := range i.Addrs {
			if a.Prefix.Addr() == netip.IPv6Loopback() {
				return true, nil
			}
		}
	}
	return false, nil
}
