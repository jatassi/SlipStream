package nodes

import (
	"net/netip"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/link"
)

// Unknown is a Node not in this library that said Hello: after a restore,
// one paired after the Backup. A Hello carries no name, so it shows by
// fingerprint, address, OS and arch.
type Unknown struct {
	Key     link.Key
	Address netip.AddrPort
	OS      string
	Arch    string
	At      time.Time
}

// unknownList is the Unknown Nodes heard lately, newest last, in memory
// only: it refills on the next dial after a restart.
type unknownList struct {
	max    int
	keep   time.Duration
	byTime []Unknown
}

// Unknown is link.Config.Unknown: it remembers a Node it doesn't know at
// its newest Hello, dropping the oldest past Timings.UnknownNodes.
func (s *Service) Unknown(key link.Key, peer link.NodeInfo, from netip.AddrPort) {
	s.unknownMu.Lock()
	defer s.unknownMu.Unlock()
	l := &s.unknown
	l.byTime = slices.DeleteFunc(l.byTime, func(u Unknown) bool { return u.Key == key })
	l.byTime = append(l.byTime, Unknown{Key: key, Address: from, OS: peer.OS, Arch: peer.Arch, At: time.Now()})
	if over := len(l.byTime) - l.max; over > 0 {
		l.byTime = slices.Delete(l.byTime, 0, over)
	}
}

// Unknowns is every Unknown Node heard within Timings.UnknownFor whose key
// is still unknown, newest first.
func (s *Service) Unknowns() []Unknown {
	s.unknownMu.Lock()
	defer s.unknownMu.Unlock()
	since := time.Now().Add(-s.unknown.keep)
	var out []Unknown
	for _, u := range slices.Backward(s.unknown.byTime) {
		if _, known := s.NodeOf(u.Key); u.At.After(since) && !known {
			out = append(out, u)
		}
	}
	return out
}
