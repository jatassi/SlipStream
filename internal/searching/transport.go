package searching

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

// Endpoints is where this Node dials a site's hosts ahead of the network.
// It is empty in production. The Fake world maps fixture-site's domain to
// this Node's own front, so each Node reaches every fake site as itself
// (ADR 0014).
type Endpoints struct {
	Sites []SiteAddress `yaml:"sites"`
}

// SiteAddress sends every host equal to or under Domain to At. Its text
// form is <domain>=<address>, as in fixture-site.fake.localhost=127.0.0.2:4431.
type SiteAddress struct {
	Domain string
	At     netip.AddrPort
}

func (a SiteAddress) MarshalText() ([]byte, error) {
	return []byte(a.Domain + "=" + a.At.String()), nil
}

func (a *SiteAddress) UnmarshalText(b []byte) error {
	domain, at, ok := strings.Cut(string(b), "=")
	if !ok || domain == "" {
		return errors.New("not <domain>=<address>")
	}
	addr, err := netip.ParseAddrPort(at)
	if err != nil {
		return fmt.Errorf("not <domain>=<address>: %w", err)
	}
	*a = SiteAddress{Domain: strings.ToLower(domain), At: addr}
	return nil
}

// Address is where this Node dials host: the address of the first site
// whose domain is host or above it.
func (e Endpoints) Address(host string) (netip.AddrPort, bool) {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, s := range e.Sites {
		if host == s.Domain || strings.HasSuffix(host, "."+s.Domain) {
			return s.At, true
		}
	}
	return netip.AddrPort{}, false
}

type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// Dial is next, dialing a site's host at its address. The request keeps
// the host in its URL, so TLS still names it and the Host header carries
// it.
func (e Endpoints) Dial(next DialFunc) DialFunc {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if host, _, err := net.SplitHostPort(addr); err == nil {
			if at, ok := e.Address(host); ok {
				addr = at.String()
			}
		}
		return next(ctx, network, addr)
	}
}
