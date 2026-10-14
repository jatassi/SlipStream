package fault

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"slices"
	"sync"
)

const backlog = 128

var (
	errUnreachable = errors.New("network is unreachable")
	errNotLocal    = errors.New("cannot assign requested address")
	errInUse       = errors.New("address already in use")
)

// Network is an in-memory network of Hosts. Each dial is a pipe pair whose
// accepting end plays the Plan of the path between the two addresses.
type Network struct {
	mu        sync.Mutex
	paths     map[[2]netip.Addr]*Plan
	listeners map[endpoint]*listener
	issued    uint16
}

// endpoint names a listening socket: a TCP address, or a unix path on
// one Host.
type endpoint struct {
	tcp  netip.AddrPort
	host *Host
	path string
}

func NewNetwork() *Network {
	return &Network{paths: map[[2]netip.Addr]*Plan{}, listeners: map[endpoint]*listener{}}
}

// path is the one front of a path's Plan. A rule with no Target covers it.
var path = Front{Layers: Conn}

// Path returns the Plan of the path between a and b, either way round.
// It plays connection faults only, and Refuse refuses dials.
func (n *Network) Path(a, b netip.Addr) *Plan {
	if b.Less(a) {
		a, b = b, a
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	key := [2]netip.Addr{a, b}
	p := n.paths[key]
	if p == nil {
		p = NewPlan([]Front{path}, nil)
		n.paths[key] = p
	}
	return p
}

// Host is one machine on the Network, holding addrs. A dial leaves from
// the Host's first address of the target's family.
type Host struct {
	net   *Network
	addrs []netip.Addr
}

func (n *Network) Host(addrs ...netip.Addr) *Host {
	return &Host{net: n, addrs: addrs}
}

// Listen takes "tcp" with an address of either family, the Host's own or
// unspecified, and "unix" with a path. Port 0 picks a free port.
func (h *Host) Listen(network, address string) (net.Listener, error) {
	switch network {
	case "tcp":
		ap, err := netip.ParseAddrPort(address)
		if err != nil {
			return nil, &net.OpError{Op: "listen", Net: network, Err: err}
		}
		return h.listenTCP(ap)
	case "unix":
		addr := &net.UnixAddr{Name: address, Net: network}
		return h.net.bind(addr, []endpoint{{host: h, path: address}})
	default:
		return nil, &net.OpError{Op: "listen", Net: network, Err: net.UnknownNetworkError(network)}
	}
}

func (h *Host) listenTCP(ap netip.AddrPort) (net.Listener, error) {
	addrs := h.bound(ap.Addr())
	if len(addrs) == 0 {
		return nil, &net.OpError{Op: "listen", Net: "tcp", Addr: net.TCPAddrFromAddrPort(ap), Err: errNotLocal}
	}
	n := h.net
	n.mu.Lock()
	defer n.mu.Unlock()
	port := ap.Port()
	for port == 0 {
		if p := n.ephemeral(); !n.taken(addrs, p) {
			port = p
		}
	}
	if n.taken(addrs, port) {
		return nil, &net.OpError{Op: "listen", Net: "tcp", Addr: net.TCPAddrFromAddrPort(ap), Err: errInUse}
	}
	keys := make([]endpoint, len(addrs))
	for i, a := range addrs {
		keys[i] = endpoint{tcp: netip.AddrPortFrom(a, port)}
	}
	return n.register(net.TCPAddrFromAddrPort(netip.AddrPortFrom(ap.Addr(), port)), keys), nil
}

// bound is the Host addresses a listener on a takes: a itself, or for an
// unspecified address every address it covers, both families for "::".
func (h *Host) bound(a netip.Addr) []netip.Addr {
	if !a.IsUnspecified() {
		if slices.Contains(h.addrs, a) {
			return []netip.Addr{a}
		}
		return nil
	}
	if a.Is6() {
		return h.addrs
	}
	return slices.DeleteFunc(slices.Clone(h.addrs), func(b netip.Addr) bool { return !b.Is4() })
}

func (h *Host) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	switch network {
	case "tcp":
		ap, err := netip.ParseAddrPort(address)
		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: network, Err: err}
		}
		return h.dialTCP(ctx, ap)
	case "unix":
		addr := &net.UnixAddr{Name: address, Net: network}
		c, s := Pipe(&net.UnixAddr{Net: network}, addr)
		return h.net.connect(endpoint{host: h, path: address}, addr, c, s)
	default:
		return nil, &net.OpError{Op: "dial", Net: network, Err: net.UnknownNetworkError(network)}
	}
}

func (h *Host) dialTCP(ctx context.Context, ap netip.AddrPort) (net.Conn, error) {
	remote := net.TCPAddrFromAddrPort(ap)
	i := slices.IndexFunc(h.addrs, func(a netip.Addr) bool { return a.Is4() == ap.Addr().Is4() })
	if i < 0 {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Addr: remote, Err: errUnreachable}
	}
	plan := h.net.Path(h.addrs[i], ap.Addr())
	switch plan.dialing() {
	case Refuse{}:
		return nil, &net.OpError{Op: "dial", Net: "tcp", Addr: remote, Err: ErrRefused}
	case Stall{}:
		<-ctx.Done()
		return nil, &net.OpError{Op: "dial", Net: "tcp", Addr: remote, Err: ctx.Err()}
	}
	h.net.mu.Lock()
	local := net.TCPAddrFromAddrPort(netip.AddrPortFrom(h.addrs[i], h.net.ephemeral()))
	h.net.mu.Unlock()
	client, server := Pipe(local, remote)
	return h.net.connect(endpoint{tcp: ap}, remote, client, plan.Conn(server, path))
}

// connect queues s on the listener at e and returns c, or refuses at once
// when nothing listens there or its backlog is full.
func (n *Network) connect(e endpoint, remote net.Addr, c, s net.Conn) (net.Conn, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if l := n.listeners[e]; l != nil {
		select {
		case l.conns <- s:
			return c, nil
		default:
		}
	}
	_ = c.Close()
	_ = s.Close()
	return nil, &net.OpError{Op: "dial", Net: remote.Network(), Addr: remote, Err: ErrRefused}
}

func (n *Network) bind(addr net.Addr, keys []endpoint) (net.Listener, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, k := range keys {
		if n.listeners[k] != nil {
			return nil, &net.OpError{Op: "listen", Net: addr.Network(), Addr: addr, Err: errInUse}
		}
	}
	return n.register(addr, keys), nil
}

func (n *Network) register(addr net.Addr, keys []endpoint) *listener {
	l := &listener{net: n, addr: addr, keys: keys, conns: make(chan net.Conn, backlog), done: make(chan struct{})}
	for _, k := range keys {
		n.listeners[k] = l
	}
	return l
}

func (n *Network) taken(addrs []netip.Addr, port uint16) bool {
	return slices.ContainsFunc(addrs, func(a netip.Addr) bool {
		return n.listeners[endpoint{tcp: netip.AddrPortFrom(a, port)}] != nil
	})
}

func (n *Network) ephemeral() uint16 {
	p := 49152 + n.issued%16384
	n.issued++
	return p
}

type listener struct {
	net   *Network
	addr  net.Addr
	keys  []endpoint
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func (l *listener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, &net.OpError{Op: "accept", Net: l.addr.Network(), Addr: l.addr, Err: net.ErrClosed}
	}
}

// Close refuses later dials and drops the connections not yet accepted.
func (l *listener) Close() error {
	l.once.Do(func() {
		l.net.mu.Lock()
		defer l.net.mu.Unlock()
		for _, k := range l.keys {
			delete(l.net.listeners, k)
		}
		close(l.done)
		for len(l.conns) > 0 {
			_ = (<-l.conns).Close()
		}
	})
	return nil
}

func (l *listener) Addr() net.Addr { return l.addr }
