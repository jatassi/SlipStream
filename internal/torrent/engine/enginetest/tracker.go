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
	// FailWithoutInterval answers every announce with a failure reason and
	// no interval, which libtorrent retries after 1800 s.
	FailWithoutInterval bool
	// HoldStarted holds each answer to started until the tracker is told to
	// answer.
	HoldStarted bool
}

// Announce is one announce a tracker received. Event is its event as on
// the wire: started, stopped, completed, or empty for a regular announce.
type Announce struct {
	InfoHash bencode.InfoHash
	Event    string
	At       time.Time
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

var errTrackerFailure = &engine.Error{Category: engine.CategoryLibtorrent, Code: 173, Text: "tracker sent a failure message"}

// tracker is a tracker the World runs.
type tracker struct {
	rules TrackerRules
	log   []Announce
	// answering is set once the tracker was told to answer: it holds no
	// more answers.
	answering bool
	held      []*announcing
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

// Answer has the tracker at url answer the started announces it holds,
// and every later one at once.
func (w *World) Answer(url string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	tr := w.trackers[url]
	tr.answering = true
	held := tr.held
	tr.held = nil
	for _, a := range held {
		a.answer(tr, "started")
	}
}

// announcing is a torrent's state on the tracker its metainfo announces
// to, by libtorrent's rules: an answer sets next to now plus the larger of
// the interval and the floor, and min to now plus the min interval;
// stopping resets both and sends stopped only where started was answered;
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
	timer     *time.Timer
}

func (a *announcing) tracker() *tracker { return a.t.e.w.trackers[a.t.meta.Announce] }

func (a *announcing) awaitsStarted() bool { return a.updating && !a.startSent }

func (a *announcing) start() {
	if a.on || a.t.state == engine.StateCheckingFiles {
		return
	}
	a.on, a.startSent, a.next, a.min = true, false, time.Time{}, time.Time{}
	a.announce()
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

func (a *announcing) send(event string) {
	tr := a.tracker()
	if tr == nil {
		return
	}
	now := time.Now()
	tr.log = append(tr.log, Announce{InfoHash: a.t.meta.InfoHash, Event: event, At: now})
	a.updating, a.next, a.min = true, now, now
	if event == "started" && tr.rules.HoldStarted && !tr.answering {
		tr.held = append(tr.held, a)
		return
	}
	a.answer(tr, event)
}

func (a *announcing) answer(tr *tracker, event string) {
	now := time.Now()
	a.updating = false
	if tr.rules.FailWithoutInterval {
		a.fails++
		a.next = now.Add(max(failRetry, min(retryCeiling, retryFloor+time.Duration(a.fails*a.fails)*retryFloor*backoffRatio/100)))
		if a.t.e.holds(a.t) {
			a.t.e.emit(engine.TrackerFailed{Torrent: a.t.handle, Err: errTrackerFailure})
		}
	} else {
		a.startSent = a.startSent || event == "started"
		a.completeSent = a.completeSent || event == "completed"
		a.fails = 0
		a.next, a.min = now.Add(max(tr.rules.Interval, a.t.e.floor)), now.Add(tr.rules.MinInterval)
		if a.t.e.holds(a.t) {
			a.t.e.emit(engine.TrackerReplied{Torrent: a.t.handle})
		}
	}
	a.schedule()
	a.t.settle()
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
