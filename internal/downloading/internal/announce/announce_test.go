package announce_test

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
)

var (
	t0     = time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	h      = bencode.InfoHash{1}
	other  = bencode.InfoHash{2}
	beacon = announce.Tracker{Index: 0, Host: "beacon.example"}
	second = announce.Tracker{Index: 1, Host: "aither.example"}
)

const (
	jitter = 3 * time.Minute
	floor  = 5 * time.Minute
)

func ledger() *announce.Ledger {
	return announce.New(rand.New(rand.NewChaCha8([32]byte{'l', 'e', 'd', 'g', 'e', 'r'})), jitter, floor) //nolint:gosec // seeded on purpose so tests replay
}

func at(d time.Duration) time.Time { return t0.Add(d) }

type step func(*announce.Ledger)

func sent(h bencode.InfoHash, t announce.Tracker, ev announce.Event, d time.Duration) step {
	return func(l *announce.Ledger) { l.Sent(h, t, ev, at(d)) }
}

func replied(h bencode.InfoHash, t announce.Tracker, d, minInterval time.Duration) step {
	return func(l *announce.Ledger) {
		l.Replied(h, t, announce.Reply{At: at(d), NextAnnounce: at(d + 45*time.Minute), MinAnnounce: at(d + minInterval)})
	}
}

type query struct {
	name  string
	steps []step
	ask   func(*announce.Ledger) time.Time
	want  time.Time
}

func startAt(private bool, d time.Duration) func(*announce.Ledger) time.Time {
	return func(l *announce.Ledger) time.Time { return l.StartAt(h, private, at(d)) }
}

func check(t *testing.T, qs []query) {
	t.Helper()
	for _, q := range qs {
		t.Run(q.name, func(t *testing.T) {
			l := ledger()
			for _, s := range q.steps {
				s(l)
			}
			if got := q.ask(l); !got.Equal(q.want) {
				t.Fatalf("got +%v, want +%v", got.Sub(t0), q.want.Sub(t0))
			}
		})
	}
}

func TestAQuickPauseAndResumeWaitsOutTheWindow(t *testing.T) {
	check(t, []query{
		{
			name:  "a pause two minutes in",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), sent(h, beacon, announce.Stopped, 2*time.Minute)},
			ask:   startAt(true, 2*time.Minute),
			want:  at(30 * time.Minute),
		},
		{
			name:  "a pause exactly five minutes in",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), sent(h, beacon, announce.Stopped, 5*time.Minute)},
			ask:   startAt(true, 6*time.Minute),
			want:  at(30 * time.Minute),
		},
		{
			name:  "a completed opens a window",
			steps: []step{sent(h, beacon, announce.Completed, 0), replied(h, beacon, time.Second, 30*time.Minute)},
			ask:   startAt(true, time.Minute),
			want:  at(30 * time.Minute),
		},
		{
			name:  "a completed inside the window leaves it as it was",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), sent(h, beacon, announce.Completed, 20*time.Minute)},
			ask:   startAt(true, 21*time.Minute),
			want:  at(30 * time.Minute),
		},
		{
			name: "an announce past the shortest window may have opened a new one",
			steps: []step{
				sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute),
				sent(h, beacon, announce.None, 26*time.Minute), sent(h, beacon, announce.Stopped, 28*time.Minute),
			},
			ask:  startAt(true, 28*time.Minute),
			want: at(56 * time.Minute),
		},
		{
			name:  "the window has passed",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), sent(h, beacon, announce.Stopped, 2*time.Minute)},
			ask:   startAt(true, 30*time.Minute),
			want:  at(30 * time.Minute),
		},
	})
}

func TestAStoppedAfterFiveMinutesClearsTheWindow(t *testing.T) {
	check(t, []query{
		{
			name:  "just over five minutes",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), sent(h, beacon, announce.Stopped, 5*time.Minute+time.Second)},
			ask:   startAt(true, 5*time.Minute+time.Second),
			want:  at(5*time.Minute + time.Second),
		},
		{
			name: "a regular announce inside the window does not reopen it",
			steps: []step{
				sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute),
				sent(h, beacon, announce.None, 20*time.Minute), sent(h, beacon, announce.Stopped, 21*time.Minute),
			},
			ask:  startAt(true, 21*time.Minute),
			want: at(21 * time.Minute),
		},
	})
}

func TestTheSameEventWithin30sWaits(t *testing.T) {
	check(t, []query{
		{
			name:  "a second started",
			steps: []step{sent(h, beacon, announce.Started, 0), sent(h, beacon, announce.Stopped, 10*time.Second)},
			ask:   startAt(false, 20*time.Second),
			want:  at(30 * time.Second),
		},
		{
			name:  "a second stopped",
			steps: []step{sent(h, beacon, announce.Stopped, 0)},
			ask:   func(l *announce.Ledger) time.Time { return l.StopAt(h, at(20*time.Second)) },
			want:  at(30 * time.Second),
		},
		{
			name:  "a stopped after a started",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute)},
			ask:   func(l *announce.Ledger) time.Time { return l.StopAt(h, at(time.Second)) },
			want:  at(time.Second),
		},
		{
			name:  "counted from the tracker's answer",
			steps: []step{
				sent(h, beacon, announce.Started, 0),
				func(l *announce.Ledger) { l.Answered(h, beacon, at(2*time.Second)) },
				sent(h, beacon, announce.Stopped, 10*time.Second),
			},
			ask:   startAt(false, 20*time.Second),
			want:  at(32 * time.Second),
		},
		{
			name: "counted from a failure's answer",
			steps: []step{
				sent(h, beacon, announce.Stopped, 0),
				func(l *announce.Ledger) { l.Failed(h, beacon, "Too many requests", time.Hour, at(3*time.Second)) },
			},
			ask:  func(l *announce.Ledger) time.Time { return l.StopAt(h, at(20*time.Second)) },
			want: at(33 * time.Second),
		},
		{
			name:  "another torrent",
			steps: []step{sent(other, beacon, announce.Started, 0)},
			ask:   startAt(true, 10*time.Second),
			want:  at(10 * time.Second),
		},
	})
}

func TestTheWindowIsTheRepliedMinInterval(t *testing.T) {
	check(t, []query{
		{
			name:  "the latest reply's",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), replied(h, beacon, 2*time.Second, 45*time.Minute)},
			ask:   startAt(true, time.Minute),
			want:  at(45 * time.Minute),
		},
		{
			name:  "measured from the reply, not from libtorrent's moved min announce",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, 10*time.Second, 28*time.Minute)},
			ask:   startAt(true, time.Minute),
			want:  at(28 * time.Minute),
		},
	})
}

func TestAnUnknownMinIntervalFallsBackToTheHostsLargest(t *testing.T) {
	check(t, []query{
		{
			name: "the largest this host replied with",
			steps: []step{
				sent(other, beacon, announce.Started, 0), replied(other, beacon, time.Second, 40*time.Minute),
				sent(bencode.InfoHash{3}, beacon, announce.Started, 0), replied(bencode.InfoHash{3}, beacon, time.Second, 30*time.Minute),
				sent(h, beacon, announce.Started, time.Minute),
			},
			ask:  startAt(true, 2*time.Minute),
			want: at(41 * time.Minute),
		},
		{
			name: "another host's does not count",
			steps: []step{
				sent(other, second, announce.Started, 0), replied(other, second, time.Second, 40*time.Minute),
				sent(h, beacon, announce.Started, time.Minute),
			},
			ask:  startAt(true, 2*time.Minute),
			want: at(time.Minute + floor),
		},
	})
}

func TestAnnounceNowWaitsForEveryTracker(t *testing.T) {
	check(t, []query{
		{
			name: "the later window",
			steps: []step{
				sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute),
				sent(h, second, announce.Started, 10*time.Minute), replied(h, second, 10*time.Minute, 45*time.Minute),
			},
			ask:  func(l *announce.Ledger) time.Time { return l.AnnounceAt(h, true, at(31*time.Minute)) },
			want: at(55 * time.Minute),
		},
		{
			name: "a regular announce opens a window too",
			steps: []step{
				sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute),
				sent(h, beacon, announce.None, 40*time.Minute),
			},
			ask:  func(l *announce.Ledger) time.Time { return l.AnnounceAt(h, true, at(40*time.Minute+5*time.Second)) },
			want: at(70 * time.Minute),
		},
		{
			name: "a public torrent waits out the duplicate rule only",
			steps: []step{
				sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute),
				sent(h, beacon, announce.None, 40*time.Minute),
			},
			ask:  func(l *announce.Ledger) time.Time { return l.AnnounceAt(h, false, at(40*time.Minute+5*time.Second)) },
			want: at(40*time.Minute + 30*time.Second),
		},
	})
}

func TestAPublicTorrentIsNotGated(t *testing.T) {
	check(t, []query{
		{
			name:  "started",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), sent(h, beacon, announce.Stopped, 2*time.Minute)},
			ask:   startAt(false, 2*time.Minute),
			want:  at(2 * time.Minute),
		},
		{
			name:  "announce now",
			steps: []step{sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute)},
			ask:   func(l *announce.Ledger) time.Time { return l.AnnounceAt(h, false, at(time.Minute)) },
			want:  at(time.Minute),
		},
	})
}

func TestAPartedTorrentStartsFresh(t *testing.T) {
	check(t, []query{{
		name: "forgotten",
		steps: []step{
			sent(h, beacon, announce.Started, 0), replied(h, beacon, time.Second, 30*time.Minute), sent(h, beacon, announce.Stopped, time.Minute),
			func(l *announce.Ledger) { l.Forget(h) },
		},
		ask:  startAt(true, 2*time.Minute),
		want: at(2 * time.Minute),
	}})
}

func TestJitterStaysWithinItsBound(t *testing.T) {
	l := ledger()
	const base = 45 * time.Minute
	lowest, highest := time.Duration(1<<63-1), time.Duration(0)
	for i := range 10_000 {
		d := l.Replied(h, beacon, announce.Reply{At: at(time.Duration(i) * time.Hour), NextAnnounce: at(time.Duration(i)*time.Hour + base), MinAnnounce: at(time.Duration(i)*time.Hour + 30*time.Minute)})
		if d < base || d > base+jitter {
			t.Fatalf("draw %d: delay %v outside [%v, %v]", i, d, base, base+jitter)
		}
		lowest, highest = min(lowest, d), max(highest, d)
	}
	if lowest > base+time.Second || highest < base+jitter-time.Second {
		t.Fatalf("delays spanned [%v, %v], want the whole of [%v, %v]", lowest, highest, base, base+jitter)
	}
}

func TestAnUnregisteredTrackerFaultsAfterTheGap(t *testing.T) {
	const gap = 30 * time.Minute
	type event struct {
		success step
		tracker announce.Tracker
		reason  string
		at      time.Duration
		fault   bool
	}
	const window = "You last announced 12 s ago. Please respect the min interval."
	twice := func(reason string) []event {
		return []event{{tracker: beacon, reason: reason}, {tracker: beacon, reason: reason, at: 29 * time.Minute}, {tracker: beacon, reason: reason, at: gap, fault: true}}
	}
	for name, events := range map[string][]event{
		"fake-tracker's wording": twice("torrent not registered with this tracker"),
		"UNIT3D's wording":       twice("Torrent not registered with this tracker."),
		"Gazelle's wording":      twice("Unregistered torrent"),
		"another failure": {
			{tracker: beacon, reason: window},
			{tracker: beacon, reason: window, at: time.Hour},
		},
		"a success between starts it over": {
			{tracker: beacon, reason: "Unregistered torrent"},
			{success: replied(h, beacon, 10*time.Minute, 30*time.Minute)},
			{tracker: beacon, reason: "Unregistered torrent", at: 20 * time.Minute},
			{tracker: beacon, reason: "Unregistered torrent", at: 45 * time.Minute},
			{tracker: beacon, reason: "Unregistered torrent", at: 50 * time.Minute, fault: true},
		},
		"another tracker's success does not": {
			{tracker: beacon, reason: "Unregistered torrent"},
			{success: replied(h, second, 10*time.Minute, 30*time.Minute)},
			{tracker: beacon, reason: "Unregistered torrent", at: gap, fault: true},
		},
	} {
		t.Run(name, func(t *testing.T) {
			l := ledger()
			for _, e := range events {
				if e.success != nil {
					e.success(l)
					continue
				}
				if got := l.Failed(h, e.tracker, e.reason, gap, at(e.at)); got != e.fault {
					t.Fatalf("%q at +%v: fault %v, want %v", e.reason, e.at, got, e.fault)
				}
			}
		})
	}
}
