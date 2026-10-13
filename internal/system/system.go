// Package system holds a running Node's runtime facts as plain values, so
// the system endpoint reports them without importing the Node.
package system

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/buildinfo"
)

type Facts struct {
	// Build is nil for an unstamped build.
	Build     *buildinfo.Stamp
	DataDir   string
	PID       int
	Seed      string
	StartedAt time.Time
	Listeners Listeners
}

type Listeners struct {
	HTTP    netip.AddrPort
	Torrent netip.AddrPort
	Link    netip.AddrPort
	Control string
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
