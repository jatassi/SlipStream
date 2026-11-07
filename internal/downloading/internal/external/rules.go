package external

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// absence is, from one successful poll, each listed hash absent from two
// polls in a row, with missed the polls in a row each listed hash has been
// absent. listed is that count before this poll, and got the hashes it
// lists. Mass is at least floor of the listed hashes absent at once, and
// more than share percent of them: it marks nothing vanished and leaves
// the counts as they were.
func absence(listed map[bencode.InfoHash]int, got map[bencode.InfoHash]bool, share, floor int) (vanished []bencode.InfoHash, missed map[bencode.InfoHash]int, mass bool) {
	absent := 0
	for h := range listed {
		if !got[h] {
			absent++
		}
	}
	if absent >= floor && absent*100 > share*len(listed) {
		return nil, listed, true
	}
	missed = make(map[bencode.InfoHash]int, len(listed))
	for h, n := range listed {
		switch {
		case got[h]:
			missed[h] = 0
		case n+1 >= 2:
			vanished = append(vanished, h)
		default:
			missed[h] = n + 1
		}
	}
	slices.SortFunc(vanished, func(a, b bencode.InfoHash) int { return slices.Compare(a[:], b[:]) })
	return vanished, missed, false
}

// Resend is what to do for a torrent the Node added that the client
// hasn't listed yet.
type Resend uint8

const (
	ResendWait Resend = iota
	ResendNow
	// ResendNotKept is a second miss: the client didn't keep it.
	ResendNotKept
)

// resend waits wait after each add, then sends once more, then gives up.
func resend(sentAt time.Time, resends int, now time.Time, wait time.Duration) Resend {
	switch {
	case now.Sub(sentAt) < wait:
		return ResendWait
	case resends == 0:
		return ResendNow
	}
	return ResendNotKept
}

// importAt is when a finished torrent imports: delay after the client's
// own finish time, or the Node's first sight of it finished when the
// client gives none, or after the end of the last move the Node saw when
// that is later.
func importAt(clientFinished, firstSight, quietSince time.Time, delay time.Duration) time.Time {
	from := clientFinished
	if from.IsZero() {
		from = firstSight
	}
	if quietSince.After(from) {
		from = quietSince
	}
	return from.Add(delay)
}

// Wanted is what the Node holds of one torrent's wanted state.
type Wanted struct {
	// Applied is the last state the Node sent the client or saw in it. The
	// intent is compared with it, never with the last intent.
	Applied clients.Run
	// Reported is a change made in the client the Node reported, until the
	// intent shows it.
	Reported *WantedChange
}

type WantedChange struct{ From, To clients.Run }

// wantedStep is one poll's wanted sync. An intent that differs from
// applied is sent to the client. Otherwise a client that differs from
// applied was changed there, and the change is reported. A reported
// change holds the intent back until the intent shows it, so a stale
// intent never undoes it. A stop at the client's seeding limit is no
// pause, so it neither reports a change nor sends a resume.
func wantedStep(intent, observed clients.Run, atLimit bool, w Wanted) (send clients.Run, change *WantedChange, next Wanted) {
	next = w
	if r := w.Reported; r != nil {
		if intent == r.To {
			next.Reported = nil
		} else {
			intent = w.Applied
		}
	}
	if atLimit && observed == clients.Paused {
		observed = w.Applied
	}
	switch {
	case intent != w.Applied:
		next.Applied = intent
		return intent, nil, next
	case observed != w.Applied:
		change = &WantedChange{From: w.Applied, To: observed}
		next.Applied, next.Reported = observed, change
		return 0, change, next
	}
	return 0, nil, next
}

// stalled is a running, unfinished torrent whose done hasn't grown for
// after, the built-in client's Stalled after. An after of 0 never stalls.
func stalled(lastGrowth, now time.Time, after time.Duration, running, finished bool) bool {
	return running && !finished && after > 0 && now.Sub(lastGrowth) >= after
}

// unregistered is a tracker message calling the torrent unregistered
// that has held for gap since firstAt, zero while none does.
func unregistered(firstAt, now time.Time, gap time.Duration) bool {
	return !firstAt.IsZero() && now.Sub(firstAt) >= gap
}

var errOutside = errors.New("the file isn't inside the client's download folder")

// deletable is each of a torrent's files as the Node sees them, and the
// folders to remove after them once empty, deepest first, up to within,
// the client's download folder on this Node. A file outside within
// refuses them all.
func deletable(files []string, within string) (paths, dirs []string, err error) {
	within = filepath.Clean(within)
	seen := map[string]bool{}
	for _, f := range files {
		f = filepath.Clean(f)
		rel, err := filepath.Rel(within, f)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, nil, fmt.Errorf("%s: %w", f, errOutside)
		}
		paths = append(paths, f)
		for d := filepath.Dir(f); d != within && !seen[d]; d = filepath.Dir(d) {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	slices.SortFunc(dirs, func(a, b string) int {
		if n := strings.Count(b, string(filepath.Separator)) - strings.Count(a, string(filepath.Separator)); n != 0 {
			return n
		}
		return strings.Compare(a, b)
	})
	return paths, dirs, nil
}

// reach is how a client's polls stand on this Node.
type reach struct {
	state   clients.PollState
	since   time.Time
	problem *clients.Problem
	// firstFailure is the first of the failures in a row, zero after a
	// success.
	firstFailure time.Time
	failures     int
	// stopped is a refused sign-in on a kind with a lockout: no poll until
	// the settings change or an Admin selects Check now.
	stopped bool
}

// failed is r after a poll that failed with p at now. A client that
// doesn't respond is not responding only once it has failed for longer
// than grace, since its first failure. A refused sign-in stops the polls
// of a kind with a lockout.
func failed(r reach, p *clients.Problem, now time.Time, grace time.Duration, lockout bool) reach {
	next := r
	next.failures++
	if next.firstFailure.IsZero() {
		next.firstFailure = now
	}
	state := p.Cause.State()
	if state == clients.NotResponding && now.Sub(next.firstFailure) < grace {
		return next
	}
	if state != r.state {
		next.since = now
		if state == clients.NotResponding {
			next.since = next.firstFailure
		}
	}
	next.state, next.problem = state, p
	next.stopped = state == clients.SignInRefused && lockout
	return next
}

// succeeded is r after a poll that succeeded at now, needing attention
// for attention, nil for none.
func succeeded(r reach, attention *clients.Problem, now time.Time) reach {
	state := clients.Responding
	if attention != nil {
		state = clients.NeedsAttention
	}
	next := reach{state: state, since: r.since, problem: attention}
	if state != r.state {
		next.since = now
	}
	return next
}

// nextPoll is when the pass after one at last runs, zero for none until
// something asks: Check every, the live cadence while an Admin watches,
// the backoff while failing, the refused retry for a kind without a
// lockout.
func nextPoll(last time.Time, r reach, every time.Duration, watched bool, t Timings, rnd *rand.Rand) time.Time {
	switch {
	case r.stopped:
		return time.Time{}
	case r.state == clients.SignInRefused:
		return last.Add(t.RefusedRetry)
	case r.failures > 0:
		return last.Add(backoff.Delay(t.RetryMin, t.RetryMax, r.failures-1, rnd))
	case watched:
		return last.Add(min(every, max(t.LiveSample, t.LiveFloor)))
	}
	return last.Add(every)
}
