// Package system holds a running Node's runtime facts as plain values, so
// the system endpoint reports them without importing the Node.
package system

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/buildinfo"
)

type Facts struct {
	// Build is nil for an unstamped build.
	Build     *buildinfo.Stamp
	DataDir   string
	LogsDir   string
	PID       int
	Seed      string
	StartedAt time.Time
	// HTTPConfigured is http.listen at start. Listeners.HTTP is what bound,
	// a later port after a fallback.
	HTTPConfigured netip.AddrPort
	Listeners      Listeners
}

type Listeners struct {
	HTTP    netip.AddrPort
	Torrent netip.AddrPort
	Link    netip.AddrPort
	Control string
}

// LogRecord is one record of a Node's log as its viewer reads it.
type LogRecord struct {
	// Seq counts up from the serving process's start in Unix microseconds.
	Seq       uint64
	Time      time.Time
	Level     string
	Subsystem string
	Message   string
	// Attrs is every other attribute as one JSON object.
	Attrs json.RawMessage
}

// ResetReason is why a log read could not go on from its reader's last
// record.
type ResetReason uint8

const (
	// Lapped: the ring dropped records the reader had not read.
	Lapped ResetReason = iota + 1
	// Restarted: the reader's last record came from an earlier process.
	Restarted
)

var resetReasons = [...]string{Lapped: "lapped", Restarted: "restarted"}

func (r ResetReason) String() string { return resetReasons[r] }

func (r ResetReason) MarshalText() ([]byte, error) { return []byte(r.String()), nil }

func (r *ResetReason) UnmarshalText(b []byte) error {
	for i, name := range resetReasons {
		if name != "" && name == string(b) {
			*r = ResetReason(i)
			return nil
		}
	}
	return fmt.Errorf("reset reason %q is not lapped or restarted", b)
}

// LogReset starts a read over: its records follow After, the ring's
// oldest less one.
type LogReset struct {
	Reason ResetReason
	// StartedAt is when the serving process opened its log.
	StartedAt time.Time
	After     uint64
}

// LogBatch is one read of a Node's log, empty when nothing came in time.
type LogBatch struct {
	Reset   *LogReset
	Records []LogRecord
}

// Claim is what a Node is, computed on every read and never stored: a
// Worker while it holds a pairing record, else a Controller while
// library.db holds an Admin, else Unclaimed.
type Claim uint8

const (
	Unclaimed Claim = iota
	Controller
	Worker
)

var claimNames = [...]string{Unclaimed: "unclaimed", Controller: "controller", Worker: "worker"}

func (c Claim) String() string { return claimNames[c] }

func (c Claim) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

func (c *Claim) UnmarshalText(b []byte) error {
	for i, name := range claimNames {
		if name == string(b) {
			*c = Claim(i)
			return nil
		}
	}
	return fmt.Errorf("claim %q is not unclaimed, controller or worker", b)
}
