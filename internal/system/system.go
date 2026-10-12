// Package system holds a running Node's runtime facts as plain values, so
// the system endpoint reports them without importing the Node.
package system

import (
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
	Control string
}
