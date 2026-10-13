package link

import (
	"net/netip"
	"time"
)

// Settings is the link config section.
type Settings struct {
	// Listen is bound by every Node, in internal/node.
	Listen netip.AddrPort `yaml:"listen"`
	// StreamWindow is the largest receive window one mux or HTTP/2 stream
	// grows to, in bytes.
	StreamWindow uint32 `yaml:"stream_window"`
}

func DefaultSettings() Settings {
	return Settings{Listen: netip.AddrPortFrom(netip.IPv6Unspecified(), 8485), StreamWindow: 16 << 20}
}

// Timings is the timings.link section.
type Timings struct {
	// Handshake bounds TCP, TLS and Hello for one attempt.
	Handshake time.Duration `yaml:"handshake"`
	// Keepalive is how often a session pings, under the 35 s median CGN
	// timeout. A ping unanswered within WriteTimeout ends the session.
	Keepalive    time.Duration `yaml:"keepalive"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	RedialMin    time.Duration `yaml:"redial_min"`
	RedialMax    time.Duration `yaml:"redial_max"`
	// FallbackDelay is how long IPv6 dials run alone before IPv4 joins.
	FallbackDelay time.Duration `yaml:"fallback_delay"`
}

func DefaultTimings() Timings {
	return Timings{
		Handshake:     10 * time.Second,
		Keepalive:     25 * time.Second,
		WriteTimeout:  10 * time.Second,
		RedialMin:     time.Second,
		RedialMax:     time.Minute,
		FallbackDelay: 300 * time.Millisecond,
	}
}
