package netfacts

import (
	"context"
	"errors"
	"net"
	"net/netip"

	"github.com/miekg/dns"

	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
)

// resolve looks up host's A and AAAA records through the resolver entry,
// keeping each answer's rcode, or through the OS resolver with the entry
// empty, which tells only NXDOMAIN and a timeout apart from a failure.
func (n *Netfacts) resolve(ctx context.Context, host string) *netv1.ResolveResponse {
	ctx, cancel := context.WithTimeout(ctx, n.cfg.Timings.Resolve)
	defer cancel()
	if len(n.cfg.Endpoints.Resolver) == 0 {
		return n.resolveOS(ctx, host)
	}
	at, err := netip.ParseAddrPort(n.cfg.Endpoints.Resolver[0].String())
	if err != nil {
		return &netv1.ResolveResponse{ErrorA: netv1.DNSError_DNS_ERROR_NO_RESOLVER, ErrorAaaa: netv1.DNSError_DNS_ERROR_NO_RESOLVER}
	}
	out := &netv1.ResolveResponse{}
	out.A, out.ErrorA = n.query(ctx, at, host, dns.TypeA)
	out.Aaaa, out.ErrorAaaa = n.query(ctx, at, host, dns.TypeAAAA)
	return out
}

func (n *Netfacts) query(ctx context.Context, at netip.AddrPort, host string, qtype uint16) ([][]byte, netv1.DNSError) {
	conn, err := n.cfg.Env.Packets.DialUDP(ctx, at)
	if err != nil {
		return nil, netv1.DNSError_DNS_ERROR_NO_RESOLVER
	}
	defer func() { _ = conn.Close() }()
	q := new(dns.Msg).SetQuestion(dns.Fqdn(host), qtype)
	res, _, err := (&dns.Client{Timeout: n.cfg.Timings.Resolve}).ExchangeWithConnContext(ctx, q, &dns.Conn{Conn: conn})
	switch {
	case err != nil:
		return nil, netv1.DNSError_DNS_ERROR_TIMEOUT
	case res.Rcode == dns.RcodeNameError:
		return nil, netv1.DNSError_DNS_ERROR_NXDOMAIN
	case res.Rcode != dns.RcodeSuccess:
		return nil, netv1.DNSError_DNS_ERROR_SERVFAIL
	}
	var out [][]byte
	for _, rr := range res.Answer {
		switch rr := rr.(type) {
		case *dns.A:
			if qtype == dns.TypeA {
				out = append(out, rr.A.To4())
			}
		case *dns.AAAA:
			if qtype == dns.TypeAAAA {
				out = append(out, rr.AAAA.To16())
			}
		}
	}
	return out, netv1.DNSError_DNS_ERROR_UNSPECIFIED
}

func (n *Netfacts) resolveOS(ctx context.Context, host string) *netv1.ResolveResponse {
	out := &netv1.ResolveResponse{OsResolver: true}
	addrs, err := n.cfg.Env.Lookup(ctx, host)
	if err != nil {
		e := netv1.DNSError_DNS_ERROR_SERVFAIL
		var dnsErr *net.DNSError
		switch {
		case errors.As(err, &dnsErr) && dnsErr.IsNotFound:
			e = netv1.DNSError_DNS_ERROR_NXDOMAIN
		case errors.As(err, &dnsErr) && dnsErr.IsTimeout, errors.Is(err, context.DeadlineExceeded):
			e = netv1.DNSError_DNS_ERROR_TIMEOUT
		}
		out.ErrorA, out.ErrorAaaa = e, e
		return out
	}
	for _, a := range addrs {
		if a = a.Unmap(); a.Is4() {
			out.A = append(out.A, a.AsSlice())
		} else {
			out.Aaaa = append(out.Aaaa, a.AsSlice())
		}
	}
	return out
}
