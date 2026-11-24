// Package gateway is fake-gateway: each site's router, for S51
// (Reachability diagnostic). Fake answers PCP, NAT-PMP and SSDP on one UDP
// socket per origin, told apart by payload, and IGD answers UPnP's SOAP
// over HTTP. Both play the asking Node's site: its WAN address (public,
// in 100.64/10, private or none), the protocols it speaks, and how it
// refuses a mapping. A mapping granted becomes a forward the outside
// dialer follows. Fake writes each Node's scripted readers' file and
// plays the network cue, which rewrites them.
package gateway

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/netfacts/script"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// Requests is the capture of every request to a router, in
// gateway.request records whose node is the asking origin.
var Requests = fake.CaptureSpec{Name: "gateway-requests", Kind: "gateway.request"}

// Request is one request as its record holds it. Write is a request that
// changes a mapping.
type Request struct {
	Protocol string `json:"protocol"`
	Op       string `json:"op"`
	Write    bool   `json:"write,omitempty"`
}

// Network is the cue that changes the network outside every Node: a
// site's public address or IPv6 prefix, or a Node's firewall or bind, as
// netsite.Change says. Its result names the Nodes whose script changed.
var Network = fake.CueSpec{Name: "network"}

const (
	name    = "gateway"
	igdName = "gateway-igd"
	online  = "no test may send to a real router, so fake-gateway is built from RFC 6886, RFC 6887 and the UPnP IGD specs"
)

var Fake = fake.Spec{
	Name:   name,
	Scheme: "udp",
	Plays:  netsite.Plays,
	NewUDP: newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Network.Gateway = endpoint.Optional{endpoint.MustParseHostPort(at.AddrPort.String())}
	},
	Online:   fake.NoOnlineRun(online),
	Captures: []fake.CaptureSpec{Requests},
	Cues:     []fake.CueSpec{Network},
}

const sharedRequests = "gateway-requests"

type server struct {
	site     *netsite.Model
	requests *fake.Capture
	fronts   fake.Fronts
	dir      string
	start    time.Time
}

func newServer(s scenario.Scenario, env fake.Env) (fake.UDPServer, error) {
	site, err := netsite.From(env, s)
	if err != nil {
		return nil, err
	}
	requests, err := env.Capture(Requests.Name)
	if err != nil {
		return nil, err
	}
	env.Shared.Put(sharedRequests, requests)
	srv := &server{site: site, requests: requests, fronts: env.Fronts, dir: env.Dir, start: time.Now()}
	if err := os.MkdirAll(env.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("fake gateway: %w", err)
	}
	if err := srv.writeScripts(site.Nodes()); err != nil {
		return nil, err
	}
	err = fake.OnCue(env, Network.Name, func(_ context.Context, c netsite.Change) (any, error) {
		nodes, err := site.Apply(c)
		if err != nil {
			return nil, err
		}
		return map[string][]string{"nodes": nodes}, srv.writeScripts(nodes)
	})
	return srv, err
}

// ScriptPath is the scripted readers' file of the Node named name in
// fake-gateway's dir.
func ScriptPath(dir, name string) string { return filepath.Join(dir, name+".json") }

func (s *server) writeScripts(nodes []string) error {
	for _, n := range nodes {
		f, err := s.site.Script(n)
		if err != nil {
			return err
		}
		if err := script.Write(ScriptPath(s.dir, n), f); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) ServeUDP(ctx context.Context, sockets map[string]*net.UDPConn) error {
	var wg sync.WaitGroup
	for origin, conn := range sockets {
		wg.Go(func() { s.serve(fake.WithOrigin(ctx, origin), origin, conn) })
	}
	<-ctx.Done()
	var errs []error
	for _, conn := range sockets {
		errs = append(errs, conn.Close())
	}
	wg.Wait()
	return errors.Join(errs...)
}

func (s *server) serve(ctx context.Context, origin string, conn *net.UDPConn) {
	buf := make([]byte, 2048)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return
		}
		gw, ok := s.site.Gateway(origin)
		if !ok || n == 0 {
			continue
		}
		var reply []byte
		switch p := buf[:n]; {
		case p[0] == natPMPVersion:
			reply = s.natPMP(ctx, origin, gw, p)
		case p[0] == pcpVersion:
			reply = s.pcp(ctx, origin, gw, p)
		case strings.HasPrefix(string(p), "M-SEARCH"):
			reply = s.search(ctx, origin, gw, string(p))
		}
		if reply != nil {
			_, _ = conn.WriteToUDPAddrPort(reply, from)
		}
	}
}

func (s *server) epoch() uint32 { return uint32(time.Since(s.start) / time.Second) } //nolint:gosec // G115: a run lasts far less than 136 years

const (
	natPMPVersion = 0
	pcpVersion    = 2
)

// NAT-PMP's result codes, RFC 6886 section 3.5.
const (
	natPMPSuccess        = 0
	natPMPUnsupported    = 1
	natPMPNotAuthorized  = 2
	natPMPNetworkFailure = 3
	natPMPBadOpcode      = 5
)

// natPMP answers the NAT-PMP request p, nil on a router that doesn't
// speak it.
func (s *server) natPMP(ctx context.Context, origin string, gw netsite.Gateway, p []byte) []byte {
	if !gw.Speaks("nat-pmp") || len(p) < 2 {
		return nil
	}
	op := p[1]
	head := binary.BigEndian.AppendUint32([]byte{natPMPVersion, 128 + op, 0, 0}, s.epoch())
	switch {
	case op == 0:
		s.requests.Record(ctx, Request{Protocol: "nat-pmp", Op: "address"})
		if !gw.WAN.IsValid() {
			head[3] = natPMPNetworkFailure
			return append(head, 0, 0, 0, 0)
		}
		return append(head, gw.WAN.AsSlice()...)
	case (op == 1 || op == 2) && len(p) >= 12:
		s.requests.Record(ctx, Request{Protocol: "nat-pmp", Op: "map", Write: true})
		transport := map[byte]string{1: "UDP", 2: "TCP"}[op]
		internal, external, lifetime := binary.BigEndian.Uint16(p[4:]), binary.BigEndian.Uint16(p[6:]), binary.BigEndian.Uint32(p[8:])
		if lifetime == 0 {
			s.site.Release(origin, func(m netsite.Mapping) bool {
				return m.Node == origin && m.Transport == transport && m.Internal == internal
			})
			external = 0
		} else {
			granted, ok := s.grant(origin, gw, netsite.Mapping{Protocol: "nat-pmp", Transport: transport, Internal: internal, External: external, Lifetime: time.Duration(lifetime) * time.Second})
			if !ok {
				head[3] = natPMPNotAuthorized
			}
			external = granted
		}
		out := binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16(head, internal), external)
		return binary.BigEndian.AppendUint32(out, lifetime)
	}
	head[3] = natPMPBadOpcode
	return head
}

// grant grants m on origin's site as gw refuses or grants it, and
// returns the external port granted.
func (s *server) grant(origin string, gw netsite.Gateway, m netsite.Mapping) (uint16, bool) {
	if gw.Refuse == "refuse" {
		return 0, false
	}
	if m.External == 0 {
		m.External = m.Internal
	}
	if gw.Refuse == "other-port" {
		m.External++
	}
	m.Node = origin
	s.site.Grant(origin, m)
	return m.External, true
}

// PCP's result codes, RFC 6887 section 7.4.
const (
	pcpSuccess       = 0
	pcpNotAuthorized = 2
	pcpBadOpcode     = 4
)

const (
	pcpAnnounce = 0
	pcpMap      = 1
	pcpHeader   = 24
	pcpMapData  = 36
)

// pcp answers the PCP request p. A router that speaks NAT-PMP and not PCP
// answers NAT-PMP's unsupported version, as RFC 6887 section 9 has it,
// and one that speaks neither, nothing.
func (s *server) pcp(ctx context.Context, origin string, gw netsite.Gateway, p []byte) []byte {
	if !gw.Speaks("pcp") {
		if gw.Speaks("nat-pmp") && len(p) >= 2 {
			return binary.BigEndian.AppendUint32([]byte{natPMPVersion, 128 + p[1]&0x7f, 0, natPMPUnsupported}, s.epoch())
		}
		return nil
	}
	if len(p) < pcpHeader {
		return nil
	}
	op, lifetime := p[1]&0x7f, binary.BigEndian.Uint32(p[4:])
	reply := func(result byte, lifetime uint32, data []byte) []byte {
		out := binary.BigEndian.AppendUint32([]byte{pcpVersion, 0x80 | op, 0, result}, lifetime)
		out = binary.BigEndian.AppendUint32(out, s.epoch())
		return append(append(out, make([]byte, 12)...), data...)
	}
	switch {
	case op == pcpAnnounce:
		s.requests.Record(ctx, Request{Protocol: "pcp", Op: "announce"})
		return reply(pcpSuccess, 0, nil)
	case op == pcpMap && len(p) >= pcpHeader+pcpMapData:
		s.requests.Record(ctx, Request{Protocol: "pcp", Op: "map", Write: true})
		data := append([]byte(nil), p[pcpHeader:pcpHeader+pcpMapData]...)
		transport := map[byte]string{6: "TCP", 17: "UDP"}[data[12]]
		internal, external := binary.BigEndian.Uint16(data[16:]), binary.BigEndian.Uint16(data[18:])
		if lifetime == 0 {
			s.site.Release(origin, func(m netsite.Mapping) bool {
				return m.Node == origin && m.Transport == transport && m.Internal == internal
			})
			return reply(pcpSuccess, 0, data)
		}
		granted, ok := s.grant(origin, gw, netsite.Mapping{Protocol: "pcp", Transport: transport, Internal: internal, External: external, Lifetime: time.Duration(lifetime) * time.Second})
		if !ok {
			return reply(pcpNotAuthorized, 0, data)
		}
		binary.BigEndian.PutUint16(data[18:], granted)
		wan := netip.IPv4Unspecified()
		if gw.WAN.IsValid() {
			wan = gw.WAN
		}
		copy(data[20:], netip.AddrFrom16(wan.As16()).AsSlice())
		return reply(pcpSuccess, lifetime, data)
	}
	return reply(pcpBadOpcode, 0, nil)
}

// igdTypes are the search targets an IGD answers.
var igdTypes = []string{"ssdp:all", "upnp:rootdevice", "urn:schemas-upnp-org:device:InternetGatewayDevice:1", ipService, pppService}

// search answers an SSDP M-SEARCH, sent by unicast in the Fake world, with
// the IGD front the origin reaches.
func (s *server) search(ctx context.Context, origin string, gw netsite.Gateway, msg string) []byte {
	if !gw.Speaks("upnp") {
		return nil
	}
	s.requests.Record(ctx, Request{Protocol: "ssdp", Op: "search"})
	st := ""
	for line := range strings.SplitSeq(msg, "\r\n") {
		if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "ST") {
			st = strings.TrimSpace(v)
		}
	}
	at, ok := s.fronts.Find(igdName, origin)
	if !ok || !slices.Contains(igdTypes, st) {
		return nil
	}
	return []byte("HTTP/1.1 200 OK\r\nCACHE-CONTROL: max-age=120\r\nST: " + st + "\r\nUSN: uuid:fake-gateway::" + st +
		"\r\nLOCATION: " + at.URL(descriptionPath) + "\r\nSERVER: fake-gateway UPnP/1.1\r\n\r\n")
}
