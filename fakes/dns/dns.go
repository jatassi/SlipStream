// Package dns is fake-dns: the public DNS a Node resolves an Exposure's
// name through, for S51 (Reachability diagnostic). It serves miekg/dns on
// one UDP socket per origin, so it knows the asking Node's site and plays
// a split resolver's home answers there. It answers the Scenario's A and
// AAAA records, under .localhost, with NXDOMAIN, SERVFAIL or a SERVFAIL
// for AAAA alone where a record says so, and NXDOMAIN for any other name.
// It holds no default answer, such as a ::1 AAAA.
package dns

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/miekg/dns"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var Fake = fake.Spec{
	Name:   "dns",
	Scheme: "udp",
	Plays:  netsite.Plays,
	NewUDP: func(s scenario.Scenario, env fake.Env) (fake.UDPServer, error) {
		site, err := netsite.From(env, s)
		if err != nil {
			return nil, err
		}
		return &server{site: site}, nil
	},
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.Network.Resolver = endpoint.Optional{endpoint.MustParseHostPort(at.AddrPort.String())}
	},
	Online: fake.NoOnlineRun("the Fake world's public names are .localhost names on TEST-NET addresses, which no public resolver answers"),
}

// ttl keeps a Node from holding an answer past the network cue.
const ttl = 1

type server struct {
	site *netsite.Model
}

func (s *server) ServeUDP(ctx context.Context, sockets map[string]*net.UDPConn) error {
	var wg, started sync.WaitGroup
	servers := make([]*dns.Server, 0, len(sockets))
	for origin, conn := range sockets {
		started.Add(1)
		srv := &dns.Server{
			PacketConn:        conn,
			Handler:           dns.HandlerFunc(func(w dns.ResponseWriter, q *dns.Msg) { s.answer(origin, w, q) }),
			NotifyStartedFunc: started.Done,
		}
		servers = append(servers, srv)
		wg.Go(func() { _ = srv.ActivateAndServe() })
	}
	started.Wait()
	<-ctx.Done()
	var errs []error
	for _, srv := range servers {
		errs = append(errs, srv.Shutdown())
	}
	wg.Wait()
	return errors.Join(errs...)
}

var rcodes = map[netsite.Answer]int{netsite.NoError: dns.RcodeSuccess, netsite.NXDomain: dns.RcodeNameError, netsite.ServFail: dns.RcodeServerFailure}

func (s *server) answer(origin string, w dns.ResponseWriter, q *dns.Msg) {
	res := new(dns.Msg).SetReply(q)
	res.Authoritative = true
	for _, question := range q.Question {
		if question.Qtype != dns.TypeA && question.Qtype != dns.TypeAAAA {
			continue
		}
		addrs, answer := s.site.Lookup(origin, question.Name, question.Qtype == dns.TypeAAAA)
		res.Rcode = rcodes[answer]
		hdr := dns.RR_Header{Name: question.Name, Rrtype: question.Qtype, Class: dns.ClassINET, Ttl: ttl}
		for _, a := range addrs {
			if question.Qtype == dns.TypeA {
				res.Answer = append(res.Answer, &dns.A{Hdr: hdr, A: a.AsSlice()})
			} else {
				res.Answer = append(res.Answer, &dns.AAAA{Hdr: hdr, AAAA: a.AsSlice()})
			}
		}
	}
	_ = w.WriteMsg(res)
}
