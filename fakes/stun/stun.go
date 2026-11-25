// Package stun is fake-stun: the STUN servers a Node asks for its public
// address, for S51 (Reachability diagnostic). Fake and Second are two
// servers, each with one UDP socket per origin, so a Node has two STUN
// addresses and a site can play a mapping that varies by destination.
// Each answers a Binding request with the asking Node's address as its
// site's NAT shows it, or nothing on a site that plays a timeout, and
// records it in the stun-requests capture.
package stun

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/pion/stun/v4"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// Requests is the capture of every Binding request, in stun.binding
// records whose node is the origin of the socket it came to.
var Requests = fake.CaptureSpec{Name: "stun-requests", Kind: "stun.binding"}

// Binding is one request as its record holds it: the server asked.
type Binding struct {
	Server string `json:"server"`
}

const (
	name       = "stun"
	secondName = "stun-2"
	online     = "a public STUN server answers the address it sees, which the Fake world's TEST-NET sites can't be"
)

var Fake = fake.Spec{
	Name:   name,
	Scheme: "udp",
	Plays:  netsite.Plays,
	NewUDP: func(s scenario.Scenario, env fake.Env) (fake.UDPServer, error) {
		requests, err := env.Capture(Requests.Name)
		if err != nil {
			return nil, err
		}
		env.Shared.Put(sharedRequests, requests)
		return newServer(s, env, requests, false)
	},
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Network.STUN = endpoint.Optional{endpoint.MustParseHostPort(at.AddrPort.String())}
	},
	Online:   fake.NoOnlineRun(online),
	Captures: []fake.CaptureSpec{Requests},
}

// Second is the second STUN server. It records into Fake's capture, so the
// world serves it after Fake.
var Second = fake.Spec{
	Name:   secondName,
	Scheme: "udp",
	Plays:  netsite.Plays,
	NewUDP: func(s scenario.Scenario, env fake.Env) (fake.UDPServer, error) {
		requests, ok := env.Shared.Get(sharedRequests)
		if !ok {
			return nil, errors.New("stun-2 records into fake-stun's capture, and the world built no fake-stun before it")
		}
		capture, _ := requests.(*fake.Capture)
		return newServer(s, env, capture, true)
	},
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Network.STUN = append(e.Network.STUN, endpoint.MustParseHostPort(at.AddrPort.String()))
	},
	Online: fake.NoOnlineRun(online),
}

const sharedRequests = "stun-requests"

type server struct {
	site     *netsite.Model
	second   bool
	requests *fake.Capture
	name     string
}

func newServer(s scenario.Scenario, env fake.Env, requests *fake.Capture, second bool) (fake.UDPServer, error) {
	site, err := netsite.From(env, s)
	if err != nil {
		return nil, err
	}
	return &server{site: site, second: second, requests: requests, name: env.Name}, nil
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
	buf := make([]byte, 1500)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return
		}
		req := &stun.Message{Raw: append([]byte(nil), buf[:n]...)}
		if req.Decode() != nil || req.Type != stun.BindingRequest {
			continue
		}
		s.requests.Record(ctx, Binding{Server: s.name})
		mapped, ok := s.site.Mapped(origin, s.second)
		if !ok {
			continue
		}
		res, err := stun.Build(stun.NewTransactionIDSetter(req.TransactionID), stun.BindingSuccess,
			&stun.XORMappedAddress{IP: mapped.Addr().AsSlice(), Port: int(mapped.Port())}, stun.Fingerprint)
		if err != nil {
			continue
		}
		_, _ = conn.WriteToUDPAddrPort(res.Raw, from)
	}
}
