package netfacts

import "time"

// Timings is timings.netfacts: the bounds of each request a Node sends to
// learn its network or test a route, and how much of an answer it reads.
type Timings struct {
	// STUN bounds one Binding request to one server.
	STUN time.Duration `yaml:"stun"`
	// Gateway bounds one request to the router and its answer.
	Gateway time.Duration `yaml:"gateway"`
	// SSDPWait is how long a multicast search waits for the router.
	SSDPWait time.Duration `yaml:"ssdp_wait"`
	Resolve  time.Duration `yaml:"resolve"`
	Connect  time.Duration `yaml:"connect"`
	TLS      time.Duration `yaml:"tls"`
	// Read bounds a tested address's HTTP answer.
	Read time.Duration `yaml:"read"`
	// ReadBytes is the most of an answer's body a test keeps.
	ReadBytes int64 `yaml:"read_bytes"`
	// Pass bounds one full pass: interfaces, STUN and the router.
	Pass time.Duration `yaml:"pass"`
	// ConfirmFloor is the shortest wait before a change is observed again,
	// whatever the intent asks.
	ConfirmFloor time.Duration `yaml:"confirm_floor"`
	// Mapping is the lifetime a router mapping is asked for.
	Mapping time.Duration `yaml:"mapping"`
	// RenewFraction is how far into a mapping's lifetime it is asked
	// again, and how long a refusal waits before it is.
	RenewFraction float64 `yaml:"renew_fraction"`
}

func DefaultTimings() Timings {
	return Timings{
		STUN:          2 * time.Second,
		Gateway:       time.Second,
		SSDPWait:      2 * time.Second,
		Resolve:       5 * time.Second,
		Connect:       10 * time.Second,
		TLS:           10 * time.Second,
		Read:          10 * time.Second,
		ReadBytes:     64 << 10,
		Pass:          15 * time.Second,
		ConfirmFloor:  5 * time.Second,
		Mapping:       2 * time.Hour,
		RenewFraction: 0.5,
	}
}
