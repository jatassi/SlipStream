package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Settings is the http config section. Every key is machine-specific.
type Settings struct {
	Listen netip.AddrPort `yaml:"listen"`
	// TrustedOrigins are origins the CSRF check accepts beyond the request's
	// own, for a reverse proxy that rewrites Host.
	TrustedOrigins []Origin `yaml:"trusted_origins"`
	// PortsToTry is how many ports from Listen's the Node tries, in order,
	// keeping the first that binds. The install kind sets the default.
	PortsToTry int `yaml:"ports_to_try"`
	// TrustedProxies are the reverse proxies whose forwarded client address
	// and scheme are believed. Nothing else reads forwarded headers.
	TrustedProxies []Proxy `yaml:"trusted_proxies"`
}

func DefaultSettings() Settings {
	return Settings{Listen: netip.AddrPortFrom(netip.IPv6Unspecified(), 8484)}
}

// Timings is the timings.http section.
type Timings struct {
	ReadHeader time.Duration `yaml:"read_header"`
	Idle       time.Duration `yaml:"idle"`
}

func DefaultTimings() Timings {
	return Timings{ReadHeader: 10 * time.Second, Idle: 2 * time.Minute}
}

// Origin is scheme://host[:port] and nothing else. It names no host the
// Node dials, so it is not an endpoint.
type Origin struct {
	origin string
}

var defaultPorts = map[string]string{"http": "80", "https": "443"}

// UnmarshalText accepts only the form a browser sends in Origin: lower-case
// scheme and host, no default port and no path. CrossOriginProtection
// compares origins byte for byte, so any other spelling would never match.
func (o *Origin) UnmarshalText(text []byte) error {
	s := string(text)
	u, err := url.Parse(s)
	if err != nil || defaultPorts[u.Scheme] == "" || u.Hostname() == "" || u.User != nil ||
		u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.ForceQuery {
		return errors.New("not scheme://host[:port]")
	}
	host := strings.ToLower(u.Hostname())
	if port := u.Port(); port != "" && port != defaultPorts[u.Scheme] {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	canonical := u.Scheme + "://" + host
	if s != canonical {
		return fmt.Errorf("not the canonical origin %s", canonical)
	}
	o.origin = s
	return nil
}

func (o Origin) MarshalText() ([]byte, error) { return []byte(o.origin), nil }

func (o Origin) String() string { return o.origin }

// Server is the Node's one http.Server. Its Shutdown also closes each
// connection that has not begun a request, which net/http would otherwise
// wait five seconds on: a browser opens such spare connections and may never
// use them.
type Server struct {
	*http.Server

	mu       sync.Mutex
	fresh    map[net.Conn]struct{}
	stopping bool
}

// NewServer builds the Server. WriteTimeout stays zero because streams are
// long-lived; Huma bounds each operation's body read.
func NewServer(h http.Handler, t Timings, log *slog.Logger) *Server {
	s := &Server{fresh: make(map[net.Conn]struct{})}
	s.Server = &http.Server{
		Handler:           h,
		ReadHeaderTimeout: t.ReadHeader,
		IdleTimeout:       t.Idle,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
		ConnState:         s.track,
	}
	return s
}

func (s *Server) track(c net.Conn, state http.ConnState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case state != http.StateNew:
		delete(s.fresh, c)
	case s.stopping:
		_ = c.Close()
	default:
		s.fresh[c] = struct{}{}
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.stopping = true
	for c := range s.fresh {
		_ = c.Close()
	}
	clear(s.fresh)
	s.mu.Unlock()
	if err := s.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shut down HTTP: %w", err)
	}
	return nil
}
