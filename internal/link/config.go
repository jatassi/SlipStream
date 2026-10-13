package link

import (
	"net/netip"
	"time"
)

// Settings is the link config section. Its defaults and Timings' join
// node.Config with the composition root that binds link.listen.
type Settings struct {
	// Listen is bound by every Node, in internal/node.
	Listen netip.AddrPort `yaml:"listen"`
	// StreamWindow is the largest receive window one mux or HTTP/2 stream
	// grows to, in bytes.
	StreamWindow uint32 `yaml:"stream_window"`
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
