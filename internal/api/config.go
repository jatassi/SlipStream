package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
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

func (o *Origin) UnmarshalText(text []byte) error {
	s := string(text)
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.ForceQuery {
		return fmt.Errorf("origin %q is not scheme://host[:port]", s)
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
