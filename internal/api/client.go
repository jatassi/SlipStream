package api

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

// Proxy is a peer whose X-Forwarded-For and X-Forwarded-Proto SlipStream
// believes: an address, or a prefix of them.
type Proxy struct{ prefix netip.Prefix }

func (p *Proxy) UnmarshalText(text []byte) error {
	s := string(text)
	if a, err := netip.ParseAddr(s); err == nil {
		a = a.Unmap()
		p.prefix = netip.PrefixFrom(a, a.BitLen())
		return nil
	}
	prefix, err := netip.ParsePrefix(s)
	if err != nil {
		return err //nolint:wrapcheck // the config loader names the key
	}
	p.prefix = prefix.Masked()
	return nil
}

func (p Proxy) MarshalText() ([]byte, error) {
	if p.prefix.IsSingleIP() {
		return []byte(p.prefix.Addr().String()), nil
	}
	return []byte(p.prefix.String()), nil
}

type proxies []Proxy

func (ps proxies) trust(a netip.Addr) bool {
	for _, p := range ps {
		if p.prefix.Contains(a) {
			return true
		}
	}
	return false
}

type overLink struct{}

// OverLink is http.Server's ConnContext for the connections a Serving
// Worker opens over the link: their requests' client is the address the
// Worker's Forwarded header names, over plain HTTP.
func OverLink(ctx context.Context, _ net.Conn) context.Context {
	return context.WithValue(ctx, overLink{}, true)
}

// Forwarded is the Forwarded header a Serving Worker sends for client.
func Forwarded(client netip.Addr) string {
	if client.Is6() {
		return `for="[` + client.String() + `]"`
	}
	return "for=" + client.String()
}

// forwardedFor is the for= address of the last element of a Forwarded
// header, or the zero address.
func forwardedFor(header string) netip.Addr {
	elements := strings.Split(header, ",")
	for pair := range strings.SplitSeq(elements[len(elements)-1], ";") {
		name, value, _ := strings.Cut(strings.TrimSpace(pair), "=")
		if !strings.EqualFold(name, "for") {
			continue
		}
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		if a, ok := parseHop(value); ok {
			return a
		}
		a, _ := parseHop(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
		return a
	}
	return netip.Addr{}
}

// clientOf is req's client address and whether it used HTTPS. Over the
// link they are the Serving Worker's Forwarded address and plain HTTP.
// Otherwise they are the TCP peer's and this connection's, unless the peer
// is a trusted proxy: then the address is the rightmost X-Forwarded-For
// entry that isn't one, and HTTPS is the last X-Forwarded-Proto value. A
// malformed entry ends the walk at the last proxy seen, never a guess past
// it.
func (ps proxies) clientOf(req *http.Request) (netip.Addr, bool) {
	if req.Context().Value(overLink{}) != nil {
		return forwardedFor(req.Header.Get("Forwarded")), false
	}
	peer, _ := netip.ParseAddrPort(req.RemoteAddr)
	addr, https := peer.Addr().Unmap(), req.TLS != nil
	if !ps.trust(addr) {
		return addr, https
	}
	hops := listValues(req.Header.Values("X-Forwarded-For"))
	for i := len(hops) - 1; i >= 0; i-- {
		hop, ok := parseHop(hops[i])
		if !ok {
			break
		}
		addr = hop
		if !ps.trust(hop) {
			break
		}
	}
	if protos := listValues(req.Header.Values("X-Forwarded-Proto")); len(protos) > 0 {
		switch strings.ToLower(protos[len(protos)-1]) {
		case "https":
			https = true
		case "http":
			https = false
		}
	}
	return addr, https
}

func listValues(headers []string) []string {
	var out []string
	for _, h := range headers {
		for v := range strings.SplitSeq(h, ",") {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

func parseHop(s string) (netip.Addr, bool) {
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap(), true
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap(), true
	}
	return netip.Addr{}, false
}
