package clients

import (
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
)

// Countdowns derives when a torrent last announced for a kind that
// reports only the countdown to its next announce: the read at which the
// countdown rose since the read before, which errs late by up to one
// read and never early. Its owner calls it from one goroutine.
type Countdowns struct {
	last map[bencode.InfoHash]countdown
}

type countdown struct {
	left time.Duration
	// at is the read the countdown last rose at, zero before one.
	at time.Time
}

// Read is the TrackerLine of each of left's torrents, read at now, with
// left the countdown each reports. A torrent left doesn't name is
// forgotten.
func (c *Countdowns) Read(left map[bencode.InfoHash]time.Duration, now time.Time) map[bencode.InfoHash][]TrackerLine {
	next := make(map[bencode.InfoHash]countdown, len(left))
	out := make(map[bencode.InfoHash][]TrackerLine, len(left))
	for h, l := range left {
		was, seen := c.last[h]
		cd := countdown{left: l, at: was.at}
		if seen && l > was.left {
			cd.at = now
		}
		next[h] = cd
		a := TrackerLine{Last: cd.at}
		if l > 0 {
			a.Next = now.Add(l)
		}
		out[h] = []TrackerLine{a}
	}
	c.last = next
	return out
}
