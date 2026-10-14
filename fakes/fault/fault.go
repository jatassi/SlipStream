// Package fault is the one fault vocabulary for both offline tiers. A
// Fault is one connection, TLS or HTTP failure with one canonical text,
// and a Rule says where and when it plays. A Plan owns every rule's state:
// it wraps connections and HTTP handlers, and network.go's in-memory
// Network gives each path between two addresses its own Plan. Waits are
// channels and timers, so all of it runs inside a testing/synctest bubble
// on fake time.
package fault

import (
	"errors"
	"strconv"
	"time"
)

// Layer is the layer a Fault plays at. A front's Layers are the set it
// carries.
type Layer uint8

const (
	Conn Layer = 1 << iota
	TLS
	HTTP
)

func (l Layer) String() string {
	switch l {
	case Conn:
		return "connection"
	case TLS:
		return "TLS"
	case HTTP:
		return "HTTP"
	default:
		return "layers(" + strconv.Itoa(int(l)) + ")"
	}
}

func (l Layer) article() string {
	if l == HTTP {
		return "an"
	}
	return "a"
}

type Fault interface {
	Layer() Layer
	// String is the canonical text: Parse(f.String()) equals f.
	String() string
}

// Rate is bytes per second.
type Rate int64

type (
	// Refuse stops a front listening; it binds the same port again when
	// the rule clears. On a Network path, dials are refused.
	Refuse struct{}
	// Reset resets live and new connections.
	Reset struct{}
	// Stall stops bytes both ways and keeps connections open: writes are
	// dropped and reads held until the rule clears.
	Stall struct{}
	// Latency delays the bytes each way.
	Latency time.Duration
	// Throttle passes bytes at Rate each way.
	Throttle Rate
	// Cut passes n more bytes each way, then closes the connection.
	Cut int64
	// BadTLS has a front present a leaf that fails the client's check.
	BadTLS TLSFailure
	// Status answers Code without reaching the handler, with Retry-After
	// and the ResetHeader both carrying RetryAfter in seconds.
	Status struct {
		Code        int
		RetryAfter  time.Duration
		ResetHeader string
	}
	// Hang reads the request and never answers it.
	Hang struct{}
	// Malformed answers the real status and content type with a body that
	// doesn't parse.
	Malformed struct{}
	// Truncate declares the real body's Content-Length, sends half, then
	// closes.
	Truncate struct{}
	// SlowBody sends the headers at once and the body at Rate.
	SlowBody Rate
)

type TLSFailure uint8

const (
	Untrusted TLSFailure = iota + 1
	Expired
	WrongName
)

var tlsFailures = map[TLSFailure]string{Untrusted: "untrusted", Expired: "expired", WrongName: "wrong-name"}

func (f TLSFailure) String() string { return tlsFailures[f] }

func (Refuse) Layer() Layer    { return Conn }
func (Reset) Layer() Layer     { return Conn }
func (Stall) Layer() Layer     { return Conn }
func (Latency) Layer() Layer   { return Conn }
func (Throttle) Layer() Layer  { return Conn }
func (Cut) Layer() Layer       { return Conn }
func (BadTLS) Layer() Layer    { return TLS }
func (Status) Layer() Layer    { return HTTP }
func (Hang) Layer() Layer      { return HTTP }
func (Malformed) Layer() Layer { return HTTP }
func (Truncate) Layer() Layer  { return HTTP }
func (SlowBody) Layer() Layer  { return HTTP }

func (Refuse) String() string     { return "refuse" }
func (Reset) String() string      { return "reset" }
func (Stall) String() string      { return "stall" }
func (f Latency) String() string  { return "latency=" + time.Duration(f).String() }
func (f Throttle) String() string { return "throttle=" + Rate(f).String() }
func (f Cut) String() string      { return "cut=" + size(int64(f)) }
func (f BadTLS) String() string   { return "tls=" + TLSFailure(f).String() }
func (Hang) String() string       { return "hang" }
func (Malformed) String() string  { return "malformed" }
func (Truncate) String() string   { return "truncate" }
func (f SlowBody) String() string { return "slow-body=" + Rate(f).String() }
func (r Rate) String() string     { return size(int64(r)) + "/s" }

func (f Status) String() string {
	s := "status=" + strconv.Itoa(f.Code)
	if f.RetryAfter > 0 {
		s += ",retry-after=" + f.RetryAfter.String()
	}
	if f.ResetHeader != "" {
		s += ",reset-header=" + f.ResetHeader
	}
	return s
}

var (
	ErrReset   = errors.New("connection reset")
	ErrCut     = errors.New("connection cut")
	ErrRefused = errors.New("connection refused")
)
