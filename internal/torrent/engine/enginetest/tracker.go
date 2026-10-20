package enginetest

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// TrackerRules is how a tracker answers announces.
type TrackerRules struct {
	Interval, MinInterval time.Duration
	// Failure, when set, is the failure reason the tracker answers every
	// announce with, and no interval, which libtorrent retries after 1800 s.
	Failure string
	// Warning, when set, is a warning message every successful answer
	// carries.
	Warning string
	// HoldStarted holds each answer to started, HoldRegular each answer to
	// a regular announce, and HoldStopped each answer to stopped, until the
	// tracker is told to answer.
	HoldStarted, HoldRegular, HoldStopped bool
}

// Announce is one announce a tracker received. Event is its event as on
// the wire: started, stopped, completed, or empty for a regular announce.
type Announce struct {
	InfoHash bencode.InfoHash
	Event    string
	At       time.Time
	// PeerID is the peer_id the announce carried, and UserAgent its
	// User-Agent header.
	PeerID, UserAgent string
}

// libtorrent's defaults, which the engine keeps.
const (
	// failRetry is the interval libtorrent reads from a failure without one.
	failRetry = 1800 * time.Second
	// retryFloor and retryCeiling bound the back-off after failures, which
	// tracker_backoff, 250, scales.
	retryFloor, retryCeiling = 5 * time.Second, time.Hour
	backoffRatio             = 250
)

var announceEvents = map[string]engine.AnnounceEvent{
	"": engine.AnnounceNone, "completed": engine.AnnounceCompleted, "started": engine.AnnounceStarted, "stopped": engine.AnnounceStopped,
}

var errTrackerFailure = &engine.Error{Category: engine.CategoryLibtorrent, Code: 173, Text: "tracker sent a failure message"}

// tracker is a tracker the World runs.
type tracker struct {
	rules TrackerRules
	log   []Announce
	// answering is set once the tracker was told to answer: it holds no
	// more answers.
	answering bool
	held      []heldAnnounce
}

type heldAnnounce struct {
	a     *announcing
	event string
}

// Tracker runs a tracker at url that answers by rules. Torrents announce
// only to the trackers their World runs.
func (w *World) Tracker(url string, rules TrackerRules) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.trackers[url] = &tracker{rules: rules}
}

// Announces lists the announces the tracker at url received, oldest first.
func (w *World) Announces(url string) []Announce {
	w.mu.Lock()
	defer w.mu.Unlock()
	if tr, ok := w.trackers[url]; ok {
		return slices.Clone(tr.log)
	}
	return nil
}

// Answer has the tracker at url answer the announces it holds, and every
// later one at once.
func (w *World) Answer(url string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	tr := w.trackers[url]
	tr.answering = true
	held := tr.held
	tr.held = nil
	for _, h := range held {
		h.a.answer(tr, h.event)
	}
}

// announcing is a torrent's state on the tracker its metainfo announces
// to, by libtorrent's rules: an answer sets next to now plus the larger of
// the interval and the floor, and min to now plus the min interval;
// starting and stopping reset both, starting announces only where no
// announce awaits its answer, and stopping sends stopped only where
// started was answered and none awaits its answer;
// completed waits for no min.
type announcing struct {
	t *torrent
	// on is libtorrent's m_announcing: the torrent runs with its files
	// checked.
	on                      bool
	startSent, completeSent bool
	// updating is an announce awaiting its answer.
	updating  bool
	next, min time.Time
	fails     int
	// endpoints is libtorrent's endpoint count, which its first announce
	// makes: one, the World's engines listening on one address.
	endpoints int
	message   string
	lastErr   *engine.Error
	timer     *time.Timer
}

func (a *announcing) tracker() *tracker { return a.t.e.w.trackers[a.t.meta.Announce] }

func (a *announcing) awaitsAnswer() bool { return a.updating }

func (a *announcing) start() {
	if a.on || a.t.state == engine.StateCheckingFiles {
		return
	}
	a.on, a.startSent, a.next, a.min = true, false, time.Time{}, time.Time{}
	if !a.updating {
		a.announce()
	}
}

func (a *announcing) stop() {
	if !a.on {
		return
	}
	a.on = false
	a.cancel()
	now := time.Now()
	a.next, a.min = now, now
	if a.startSent && !a.updating {
		a.send("stopped")
	}
}

// completed announces a torrent that just finished, past its min.
func (a *announcing) completed() {
	if !a.on || a.completeSent {
		return
	}
	now := time.Now()
	a.next, a.min = now, now
	if !a.updating {
		a.announce()
	}
}

func (a *announcing) announce() {
	switch {
	case !a.startSent:
		a.send("started")
	case a.t.state == engine.StateSeeding && !a.completeSent:
		a.send("completed")
	default:
		a.send("")
	}
}

// send announces event, as libtorrent does from each listen address: an
// engine that does not listen has none.
func (a *announcing) send(event string) {
	tr := a.tracker()
	if tr == nil || !a.t.e.listen.IsValid() {
		return
	}
	now := time.Now()
	tr.log = append(tr.log, Announce{InfoHash: a.t.meta.InfoHash, Event: event, At: now, PeerID: a.t.peerID, UserAgent: version.UserAgent})
	a.updating, a.next, a.min, a.endpoints = true, now, now, 1
	if a.t.e.holds(a.t) {
		a.t.e.emit(engine.TrackerAnnounced{Torrent: a.t.handle, Event: announceEvents[event], At: now})
	}
	if tr.holds(event) {
		tr.held = append(tr.held, heldAnnounce{a, event})
		return
	}
	a.answer(tr, event)
}

func (tr *tracker) holds(event string) bool {
	return !tr.answering && tr.rules.Holds(event)
}

// Holds is whether the rules hold the answer to an announce of event.
func (r TrackerRules) Holds(event string) bool {
	return event == "started" && r.HoldStarted || event == "" && r.HoldRegular || event == "stopped" && r.HoldStopped
}

func (a *announcing) answer(tr *tracker, event string) {
	now := time.Now()
	a.updating = false
	if tr.rules.Failure != "" {
		a.fails++
		a.next = now.Add(max(failRetry, min(retryCeiling, retryFloor+time.Duration(a.fails*a.fails)*retryFloor*backoffRatio/100)))
		a.message, a.lastErr = tr.rules.Failure, errTrackerFailure
		if a.t.e.holds(a.t) {
			a.t.e.emit(engine.TrackerFailed{Torrent: a.t.handle, Times: a.fails, Reason: tr.rules.Failure, Err: errTrackerFailure, At: now})
		}
	} else {
		a.startSent = a.startSent || event == "started"
		a.completeSent = a.completeSent || event == "completed"
		a.fails = 0
		a.next, a.min = now.Add(max(tr.rules.Interval, a.t.e.floor)), now.Add(tr.rules.MinInterval)
		a.message, a.lastErr = tr.rules.Warning, nil
		if a.t.e.holds(a.t) {
			if tr.rules.Warning != "" {
				a.t.e.emit(engine.TrackerWarned{Torrent: a.t.handle, Message: tr.rules.Warning})
			}
			a.t.e.emit(engine.TrackerReplied{Torrent: a.t.handle, At: now})
		}
	}
	a.schedule()
	a.t.settle()
}

// reannounce moves the next announce as force_reannounce does: to delay
// from now, held to min unless ignoreMin, a second later, and min with it.
func (a *announcing) reannounce(index int, delay time.Duration, ignoreMin bool) {
	if a.t.paused || index != 0 || a.endpoints == 0 {
		return
	}
	at := time.Now().Add(delay)
	if !ignoreMin {
		at = later(at, a.min)
	}
	a.next = at.Add(time.Second)
	a.min = a.next
	a.schedule()
}

// list is the torrent's tracker list: the one tracker its metainfo names.
func (a *announcing) list() []engine.Tracker {
	if a.t.meta.Announce == "" {
		return nil
	}
	return []engine.Tracker{{
		URL: a.t.meta.Announce, Endpoints: a.endpoints, NextAnnounce: a.next, MinAnnounce: a.min,
		Message: a.message, LastError: a.lastErr, Fails: a.fails, Updating: a.updating, StartSent: a.startSent,
	}}
}

// schedule sets the timer for the next regular announce, which waits for
// min unless it is the completed one.
func (a *announcing) schedule() {
	a.cancel()
	if !a.on {
		return
	}
	at := a.next
	if a.startSent && (a.completeSent || a.t.state != engine.StateSeeding) {
		at = later(at, a.min)
	}
	a.timer = time.AfterFunc(time.Until(at), func() {
		w := a.t.e.w
		w.mu.Lock()
		defer w.mu.Unlock()
		if !a.t.e.closed && a.on && !a.updating {
			a.announce()
		}
	})
}

func (a *announcing) cancel() {
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}
}

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
