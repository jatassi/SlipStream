// Package endpoint holds the leaf types of a Node's endpoints block: the
// fixed external hosts a boundary talks to, which Scenarios point at fakes.
package endpoint

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// URL is an absolute URL with a scheme and a host.
type URL struct{ u url.URL }

func ParseURL(s string) (URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return URL{}, fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return URL{}, errNotAbsolute
	}
	return URL{*u}, nil
}

// MustParseURL is for the literals in a package's DefaultEndpoints.
func MustParseURL(s string) URL {
	u, err := ParseURL(s)
	if err != nil {
		panic(fmt.Sprintf("endpoint: %q: %v", s, err))
	}
	return u
}

func (u URL) Hostname() string { return u.u.Hostname() }

func (u URL) String() string { return u.u.String() }

// JoinPath is a copy of u with elem joined onto its path.
func (u URL) JoinPath(elem ...string) *url.URL { return u.u.JoinPath(elem...) }

func (u URL) MarshalText() ([]byte, error) { return []byte(u.String()), nil }

func (u *URL) UnmarshalText(b []byte) error {
	p, err := ParseURL(string(b))
	if err != nil {
		return err
	}
	*u = p
	return nil
}

// HostPort is a DNS name or IP literal with a port, for boundaries that are
// not URLs, such as STUN servers and DHT bootstrap routers.
type HostPort struct {
	host string
	port uint16
}

func ParseHostPort(s string) (HostPort, error) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return HostPort{}, fmt.Errorf("parse host and port: %w", err)
	}
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return HostPort{}, fmt.Errorf("parse port: %w", err)
	}
	if _, err := netip.ParseAddr(host); err != nil && !isDNSName(host) {
		return HostPort{}, errNotHost
	}
	return HostPort{host: host, port: uint16(n)}, nil
}

// MustParseHostPort is for the literals in a package's DefaultEndpoints.
func MustParseHostPort(s string) HostPort {
	h, err := ParseHostPort(s)
	if err != nil {
		panic(fmt.Sprintf("endpoint: %q: %v", s, err))
	}
	return h
}

var (
	errNotAbsolute = errors.New("not an absolute URL with a scheme and a host")
	errNotHost     = errors.New("host is neither an IP literal nor a DNS name")
)

func isDNSName(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	for label := range strings.SplitSeq(strings.TrimSuffix(s, "."), ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	return true
}

func (h HostPort) Hostname() string { return h.host }

func (h HostPort) Port() uint16 { return h.port }

func (h HostPort) String() string { return net.JoinHostPort(h.host, strconv.Itoa(int(h.port))) }

func (h HostPort) MarshalText() ([]byte, error) { return []byte(h.String()), nil }

func (h *HostPort) UnmarshalText(b []byte) error {
	p, err := ParseHostPort(string(b))
	if err != nil {
		return err
	}
	*h = p
	return nil
}
