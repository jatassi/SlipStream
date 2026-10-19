// Package unit3d holds UNIT3D's announce rules, from checkMinInterval in its
// AnnounceController. They are UNIT3D's rules, not SlipStream timings: the
// announce policy and the fake trackers read them from here.
package unit3d

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
)

const (
	// ClearAfter is how long after the announce that opened a window a
	// stopped must come to clear it.
	ClearAfter = 5 * time.Minute
	// Duplicate is how soon the same event from the same peer is rejected.
	Duplicate = 30 * time.Second
	// WindowLow and WindowHigh bound a window's length, in percent of the
	// min interval.
	WindowLow  = 85
	WindowHigh = 95
)

// Windows plays checkMinInterval for the fake trackers, keyed by passkey,
// info hash and peer ID. It is safe for concurrent use.
type Windows struct {
	mu       sync.Mutex
	interval time.Duration
	rand     *rand.Rand
	windows  map[peer]window
	last     map[peerEvent]time.Time
}

type peer struct {
	passkey string
	hash    bencode.InfoHash
	id      [20]byte
}

type peerEvent struct {
	peer
	event string
}

type window struct{ opened, closes time.Time }

func NewWindows(minInterval time.Duration, r *rand.Rand) *Windows {
	return &Windows{interval: minInterval, rand: r, windows: map[peer]window{}, last: map[peerEvent]time.Time{}}
}

// Rule is the rule a rejected announce broke. UNIT3D answers both with one
// failure reason, so only the fakes tell them apart.
type Rule string

const (
	// Window rejects an announce other than completed or stopped inside an
	// open window.
	Window Rule = "unit3d-window"
	// Repeat rejects the same event from the same peer within Duplicate.
	Repeat Rule = "unit3d-duplicate"
)

// Admit is the rule event at now breaks and the failure reason the tracker
// answers, both empty when it accepts. An event other than started,
// completed or stopped is a regular announce.
func (w *Windows) Admit(passkey string, h bencode.InfoHash, peerID [20]byte, event string, now time.Time) (broke Rule, reason string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch event {
	case "started", "completed", "stopped":
	default:
		event = ""
	}
	p := peer{passkey: passkey, hash: h, id: peerID}
	pe := peerEvent{peer: p, event: event}
	if last, ok := w.last[pe]; ok && now.Sub(last) < Duplicate {
		return Repeat, rejected(now.Sub(last))
	}
	w.last[pe] = now

	win, open := w.windows[p]
	open = open && now.Before(win.closes)
	if !open {
		w.windows[p] = window{opened: now, closes: now.Add(w.draw())}
	}
	switch {
	case event == "stopped" && (!open || now.Sub(win.opened) > ClearAfter):
		delete(w.windows, p)
	case open && event != "completed" && event != "stopped":
		return Window, rejected(now.Sub(win.opened))
	}
	return "", ""
}

func (w *Windows) draw() time.Duration {
	low, high := w.interval*WindowLow/100, w.interval*WindowHigh/100
	return low + time.Duration(w.rand.Int64N(int64(high-low)+1))
}

func rejected(elapsed time.Duration) string {
	return fmt.Sprintf("You last announced %d s ago. Please respect the min interval.", int64(elapsed/time.Second))
}
