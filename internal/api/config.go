package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// Settings is the http config section. Both keys are machine-specific.
type Settings struct {
	Listen netip.AddrPort `yaml:"listen"`
	// TrustedOrigins are origins the CSRF check accepts beyond the request's
	// own, for a reverse proxy that rewrites Host.
	TrustedOrigins []Origin `yaml:"trusted_origins"`
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
		return fmt.Errorf("not a canonical origin; write %s", canonical)
	}
	o.origin = s
	return nil
}

func (o Origin) MarshalText() ([]byte, error) { return []byte(o.origin), nil }

func (o Origin) String() string { return o.origin }

// NewServer is the Node's one http.Server. WriteTimeout stays zero because
// streams are long-lived; Huma bounds each operation's body read.
func NewServer(h http.Handler, t Timings, log *slog.Logger) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: t.ReadHeader,
		IdleTimeout:       t.Idle,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}
