// Package outside is the outside dialer: the internet as a tester Node
// reaches another Node through it, for S51 (Reachability diagnostic). It
// answers SOCKS5 CONNECT on a front per origin and routes each through
// the site model: the site's NAT and forwards, hairpin for a same-site
// source, the IPv6 firewall, each Node's firewall and bind, a CGNAT
// site's missing inbound path, and the Admin's proxies, which it hands to
// fake-proxy's front for the same origin. On plain HTTP to a Node it adds
// X-Forwarded-For with the address the connection comes from.
package outside

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// Connections is the capture of every CONNECT, in outside.connect
// records whose node is the origin of the front it came to.
var Connections = fake.CaptureSpec{Name: "outside-connections", Kind: "outside.connect"}

// Connect is one CONNECT as its record holds it: the address asked, where
// it arrived, or why it didn't.
type Connect struct {
	To      string `json:"to"`
	Node    string `json:"node,omitempty"`
	Service string `json:"service,omitempty"`
	Proxy   string `json:"proxy,omitempty"`
	Source  string `json:"source,omitempty"`
	Hairpin bool   `json:"hairpin,omitempty"`
	Refused string `json:"refused,omitempty"`
}

// ProxyTarget is the fault target of the fake that plays the Admin's
// proxies, whose front for the same origin a proxy arrival goes to.
const ProxyTarget fault.Target = "proxy"

var Fake = fake.Spec{
	Name:   "outside",
	Scheme: "socks5",
	Plays:  netsite.Plays,
	New: func(s scenario.Scenario, env fake.Env) (fake.Server, error) {
		site, err := netsite.From(env, s)
		if err != nil {
			return nil, err
		}
		connections, err := env.Capture(Connections.Name)
		if err != nil {
			return nil, err
		}
		return &server{site: site, fronts: env.Fronts, connections: connections}, nil
	},
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Network.OutsideDialer = endpoint.Optional{endpoint.MustParseHostPort(at.AddrPort.String())}
	},
	Online:   fake.NoOnlineRun("the internet reaches the Fake world's TEST-NET sites only through the world's own model of them"),
	Captures: []fake.CaptureSpec{Connections},
}

type server struct {
	site        *netsite.Model
	fronts      fake.Fronts
	connections *fake.Capture
}

func (s *server) Serve(ctx context.Context, l net.Listener) error {
	stop := context.AfterFunc(ctx, func() { _ = l.Close() })
	defer stop()
	for {
		c, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept socks5: %w", err)
		}
		go s.serve(fake.ConnContext(ctx, c), c)
	}
}

// SOCKS5's reply codes (RFC 1928).
const (
	replyOK          = 0
	replyFailure     = 1
	replyUnreachable = 4
	replyRefused     = 5
	replyAddrType    = 8
)

func (s *server) serve(ctx context.Context, c net.Conn) {
	defer func() { _ = c.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	r := bufio.NewReader(c)
	to, code, err := greet(r, c)
	if err != nil {
		return
	}
	if code != replyOK {
		reply(c, code)
		return
	}
	rec := Connect{To: to.String()}
	arrival, err := s.site.Route(fake.OriginOf(ctx), to)
	if err != nil {
		rec.Refused = err.Error()
		s.connections.Record(ctx, rec)
		reply(c, map[bool]byte{true: replyRefused, false: replyUnreachable}[errors.Is(err, netsite.ErrRefused)])
		return
	}
	rec.Source, rec.Hairpin = arrival.Source.String(), arrival.Hairpin
	at, err := s.target(ctx, arrival)
	if err != nil {
		rec.Refused = err.Error()
		s.connections.Record(ctx, rec)
		reply(c, replyFailure)
		return
	}
	rec.Node, rec.Service, rec.Proxy = arrival.Service.Node, arrival.Service.Kind, arrival.Proxy
	s.connections.Record(ctx, rec)
	up, err := new(net.Dialer).DialContext(ctx, "tcp", at.String())
	if err != nil {
		reply(c, replyRefused)
		return
	}
	defer func() { _ = up.Close() }()
	reply(c, replyOK)
	go func() {
		_, _ = io.Copy(c, up)
		_ = c.Close()
	}()
	if arrival.Service.Kind == netsite.HTTP {
		forwardHTTP(r, up, arrival.Source)
		return
	}
	_, _ = io.Copy(up, r)
}

// target is where an arrival's bytes go: the service's listener, or
// fake-proxy's front for the same origin.
func (s *server) target(ctx context.Context, a netsite.Arrival) (netip.AddrPort, error) {
	if a.Proxy == "" {
		return s.site.Where(a.Service)
	}
	at, ok := s.fronts.Find(ProxyTarget, fake.OriginOf(ctx))
	if !ok {
		return netip.AddrPort{}, fmt.Errorf("no %s front for %s", ProxyTarget, fake.OriginOf(ctx))
	}
	return at.AddrPort, nil
}

// greet reads the method negotiation and a CONNECT request, answering
// the address asked, or the reply code that refuses it.
func greet(r *bufio.Reader, w io.Writer) (netip.AddrPort, byte, error) {
	var head [2]byte
	if _, err := io.ReadFull(r, head[:]); err != nil || head[0] != 5 {
		return netip.AddrPort{}, 0, errors.New("not socks5")
	}
	if _, err := r.Discard(int(head[1])); err != nil {
		return netip.AddrPort{}, 0, err //nolint:wrapcheck // the caller drops the connection on any error
	}
	if _, err := w.Write([]byte{5, 0}); err != nil {
		return netip.AddrPort{}, 0, err //nolint:wrapcheck // the caller drops the connection on any error
	}
	var req [4]byte
	if _, err := io.ReadFull(r, req[:]); err != nil {
		return netip.AddrPort{}, 0, err //nolint:wrapcheck // the caller drops the connection on any error
	}
	if req[1] != 1 {
		return netip.AddrPort{}, 7, nil
	}
	var addr []byte
	switch req[3] {
	case 1:
		addr = make([]byte, 4)
	case 4:
		addr = make([]byte, 16)
	default:
		return netip.AddrPort{}, replyAddrType, nil
	}
	var port [2]byte
	if _, err := io.ReadFull(r, addr); err != nil {
		return netip.AddrPort{}, 0, err //nolint:wrapcheck // the caller drops the connection on any error
	}
	if _, err := io.ReadFull(r, port[:]); err != nil {
		return netip.AddrPort{}, 0, err //nolint:wrapcheck // the caller drops the connection on any error
	}
	ip, _ := netip.AddrFromSlice(addr)
	return netip.AddrPortFrom(ip, binary.BigEndian.Uint16(port[:])), replyOK, nil
}

func reply(w io.Writer, code byte) {
	_, _ = w.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0})
}

// forwardHTTP writes each request from r to up with X-Forwarded-For set
// to src, or splices r when the stream isn't plain HTTP, such as TLS.
func forwardHTTP(r *bufio.Reader, up io.Writer, src netip.Addr) {
	if first, err := r.Peek(1); err != nil || first[0] == 0x16 {
		_, _ = io.Copy(up, r)
		return
	}
	for {
		req, err := http.ReadRequest(r)
		if err != nil {
			return
		}
		req.Header.Set("X-Forwarded-For", src.String())
		if err := req.Write(up); err != nil {
			return
		}
	}
}
