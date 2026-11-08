package announce

import (
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/unit3d"
)

// exchange is an announce the tracker accepted: when it went out and when
// the answer came. The tracker received it at some time between the two.
type exchange struct {
	sent, answered time.Time
}

// credit is the seed time the tracker credited torrent h for announce a on
// line ln, answered at at: UNIT3D's AutoUpsertHistories seen from the
// client, one tracker line at a time. A seeder announce credits the time
// since the line's last accepted seeder announce when that came less than
// unit3d.SeedGap before, in whole seconds. A started credits nothing, and
// nothing counts from a leecher's announce, so neither the interval in
// which the torrent completes nor the first one after a pause or a restart
// credits. A failed announce never reaches credit, so it neither credits
// nor resets.
//
// The credit runs from the last announce's answer to a's send, and the gap
// is judged from the last announce's send to a's answer, the narrowest and
// the widest the tracker's own times could be apart, so the Node never
// credits more than the tracker does. A line that takes over from another
// and back credits no time the other already did.
func (l *Ledger) credit(h bencode.InfoHash, ln *line, a announcement, at time.Time) time.Duration {
	last := ln.seeded
	if !a.seeder {
		ln.seeded = nil
		return 0
	}
	ln.seeded = &exchange{sent: a.at, answered: at}
	if last == nil || a.event == Started || at.Sub(last.sent) >= unit3d.SeedGap {
		return 0
	}
	from := later(last.answered, l.credited[h])
	l.credited[h] = later(l.credited[h], a.at)
	return max(a.at.Sub(from), 0).Truncate(time.Second)
}
