// Package announce is the announce policy's memory, per torrent and
// tracker: when each private tracker would accept a started, a stopped or a
// regular announce, how long to wait before the next one, and when a
// tracker has called a torrent unregistered for long enough to count.
package announce

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/unit3d"
)

// Event is an announce's event, in engine.AnnounceEvent's order.
type Event uint8

const (
	None Event = iota
	Completed
	Started
	Stopped
)

// Tracker is one tracker of a torrent's list: its index there and its host.
type Tracker struct {
	Index int
	Host  string
}

// Reply is a tracker list entry as posted after a reply: when the reply
// came, and libtorrent's next and min announce times from it.
type Reply struct {
	At, NextAnnounce, MinAnnounce time.Time
}

// Ledger lives in memory only: a restart makes a new peer ID, and so a
// fresh window at every tracker.
type Ledger struct {
	rand     *rand.Rand
	jitter   time.Duration
	floor    time.Duration
	torrents map[bencode.InfoHash]map[int]*line
	hostMin  map[string]time.Duration
}

type line struct {
	host   string
	opened time.Time
	min    time.Duration
	// sent is when each event last went out, or once the tracker answered
	// it, when the answer came.
	sent [Stopped + 1]time.Time
	// awaiting is the event of the announce the tracker has yet to answer.
	awaiting  Event
	answering bool
	// answer is when the tracker last answered.
	answer       time.Time
	unregistered time.Time
}

// Line is the policy's view of one tracker of a torrent.
type Line struct {
	Index int
	// Answered is the tracker's last answer, zero before one.
	Answered time.Time
	// MinInterval is the tracker's min interval, or the policy's stand-in
	// until the tracker replied with one.
	MinInterval time.Duration
	// AcceptsAt is the earliest the tracker would accept an announce.
	AcceptsAt time.Time
}

// New makes a Ledger that draws jitter from r. floor is the min interval of
// a tracker line whose host has never replied.
func New(r *rand.Rand, jitter, floor time.Duration) *Ledger {
	return &Ledger{
		rand: r, jitter: jitter, floor: floor,
		torrents: map[bencode.InfoHash]map[int]*line{}, hostMin: map[string]time.Duration{},
	}
}

// Sent notes an announce that went out at at.
func (l *Ledger) Sent(h bencode.InfoHash, t Tracker, ev Event, at time.Time) {
	ln := l.line(h, t)
	ln.sent[ev], ln.awaiting, ln.answering = at, ev, true
	minInterval := l.minInterval(ln)
	open := ln.open(at, minInterval)
	switch {
	case ev == Stopped:
		if open && at.Sub(ln.opened) > unit3d.ClearAfter {
			ln.opened = time.Time{}
		}
	// Past the shortest window the tracker may have opened a new one, so
	// the ledger assumes it did.
	case !open || !at.Before(ln.opened.Add(minInterval*unit3d.WindowLow/100)):
		ln.opened = at
	}
}

// Replied records the tracker's intervals from r and returns the delay for
// the engine's next announce to it: the base interval plus a uniform draw in
// [0, jitter]. The caller records before it reannounces, because a
// reannounce moves libtorrent's min announce.
func (l *Ledger) Replied(h bencode.InfoHash, t Tracker, r Reply) time.Duration {
	ln := l.line(h, t)
	ln.min = r.MinAnnounce.Sub(r.At)
	ln.unregistered = time.Time{}
	l.hostMin[t.Host] = max(l.hostMin[t.Host], ln.min)
	return r.NextAnnounce.Sub(r.At) + time.Duration(l.rand.Int64N(int64(l.jitter)+1))
}

// Answered notes the tracker's answer at at to the announce it awaited. The
// duplicate rule counts from the answer: the tracker received the announce
// at some time between its send and the answer, and an engine's clock says
// when it sent only roughly.
func (l *Ledger) Answered(h bencode.InfoHash, t Tracker, at time.Time) {
	l.line(h, t).answered(at)
}

// Failed notes the tracker's answer at at with a failure reason, and
// reports whether the tracker has now called the torrent unregistered on
// answers at least gap apart with no success between them.
func (l *Ledger) Failed(h bencode.InfoHash, t Tracker, reason string, gap time.Duration, at time.Time) bool {
	ln := l.line(h, t)
	ln.answered(at)
	if !Unregistered(reason) {
		return false
	}
	if ln.unregistered.IsZero() {
		ln.unregistered = at
	}
	return at.Sub(ln.unregistered) >= gap
}

// StartAt is the earliest a started would be accepted by every tracker of
// h: unit3d.Duplicate after its last started and, for a private torrent,
// once each open window has passed.
func (l *Ledger) StartAt(h bencode.InfoHash, private bool, now time.Time) time.Time {
	return l.acceptsAt(h, Started, private, now)
}

// StopAt is the earliest a stopped may go out: unit3d.Duplicate after the
// last one. A window never holds a stopped.
func (l *Ledger) StopAt(h bencode.InfoHash, now time.Time) time.Time {
	return l.acceptsAt(h, Stopped, false, now)
}

// AnnounceAt is the earliest a regular announce would be accepted by every
// tracker of h, as StartAt is for a started.
func (l *Ledger) AnnounceAt(h bencode.InfoHash, private bool, now time.Time) time.Time {
	return l.acceptsAt(h, None, private, now)
}

// Detail is the policy's view of each tracker of h it has heard of, in
// tracker order.
func (l *Ledger) Detail(h bencode.InfoHash, private bool, now time.Time) []Line {
	lines := l.torrents[h]
	out := make([]Line, 0, len(lines))
	for _, index := range slices.Sorted(maps.Keys(lines)) {
		ln := lines[index]
		out = append(out, Line{Index: index, Answered: ln.answer, MinInterval: l.minInterval(ln), AcceptsAt: l.acceptsLine(ln, None, private, now)})
	}
	return out
}

// Forget drops h once it parts: the next time it is added it has a new peer
// ID.
func (l *Ledger) Forget(h bencode.InfoHash) {
	delete(l.torrents, h)
}

func (l *Ledger) acceptsAt(h bencode.InfoHash, ev Event, windowed bool, now time.Time) time.Time {
	at := now
	for _, ln := range l.torrents[h] {
		at = later(at, l.acceptsLine(ln, ev, windowed, now))
	}
	return at
}

func (l *Ledger) acceptsLine(ln *line, ev Event, windowed bool, now time.Time) time.Time {
	at := later(now, ln.sent[ev].Add(unit3d.Duplicate))
	if minInterval := l.minInterval(ln); windowed && ln.open(now, minInterval) {
		at = later(at, ln.opened.Add(minInterval))
	}
	return at
}

func (l *Ledger) line(h bencode.InfoHash, t Tracker) *line {
	lines := l.torrents[h]
	if lines == nil {
		lines = map[int]*line{}
		l.torrents[h] = lines
	}
	ln := lines[t.Index]
	if ln == nil {
		ln = &line{}
		lines[t.Index] = ln
	}
	ln.host = t.Host
	return ln
}

// minInterval is the line's min interval, until it is known the largest any
// tracker on its host replied with, else the floor.
func (l *Ledger) minInterval(ln *line) time.Duration {
	switch {
	case ln.min > 0:
		return ln.min
	case l.hostMin[ln.host] > 0:
		return l.hostMin[ln.host]
	}
	return l.floor
}

func (ln *line) answered(at time.Time) {
	ln.answer = later(ln.answer, at)
	if ln.answering {
		ln.sent[ln.awaiting] = later(ln.sent[ln.awaiting], at)
		ln.answering = false
	}
}

func (ln *line) open(now time.Time, minInterval time.Duration) bool {
	return !ln.opened.IsZero() && now.Before(ln.opened.Add(minInterval))
}

// Unregistered reports whether a tracker's failure reason says the torrent
// is unregistered there.
func Unregistered(reason string) bool {
	r := strings.ToLower(reason)
	return strings.Contains(r, "unregistered") || strings.Contains(r, "not registered")
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
