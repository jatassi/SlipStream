package announce_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
)

// exchange is one announce of a torrent to a tracker: sent at sent, while
// the torrent was complete or not, and answered at answered, by a failure
// when failed.
type exchange struct {
	tr             announce.Tracker
	ev             announce.Event
	complete       bool
	sent, answered time.Duration
	failed         bool
}

func seeding(ev announce.Event, sent time.Duration) exchange {
	return exchange{tr: beacon, ev: ev, complete: true, sent: sent, answered: sent}
}

func leeching(ev announce.Event, sent time.Duration) exchange {
	return exchange{tr: beacon, ev: ev, sent: sent, answered: sent}
}

func (e exchange) fails() exchange {
	e.failed = true
	return e
}

func (e exchange) to(tr announce.Tracker) exchange {
	e.tr = tr
	return e
}

func (e exchange) answeredAt(d time.Duration) exchange {
	e.answered = d
	return e
}

func TestTheSeedTimeATrackerCredits(t *testing.T) {
	const m = time.Minute
	for _, c := range []struct {
		name      string
		exchanges []exchange
		want      time.Duration
	}{
		{
			name:      "a leecher's announces credit nothing",
			exchanges: []exchange{leeching(announce.Started, 0), leeching(announce.None, 30*m)},
			want:      0,
		},
		{
			name: "the interval it completes in credits nothing, and the next counts from the completed",
			exchanges: []exchange{
				leeching(announce.Started, 0), leeching(announce.None, 30*m), leeching(announce.Completed, 50*m), seeding(announce.None, 80*m),
			},
			want: 30 * m,
		},
		{
			name:      "steady seeding credits every interval",
			exchanges: []exchange{seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.None, 60*m), seeding(announce.None, 90*m)},
			want:      90 * m,
		},
		{
			name:      "a gap of exactly 90 minutes credits nothing, and the next counts from it",
			exchanges: []exchange{seeding(announce.Started, 0), seeding(announce.None, 90*m), seeding(announce.None, 120*m)},
			want:      30 * m,
		},
		{
			name:      "a gap just under 90 minutes credits",
			exchanges: []exchange{seeding(announce.Started, 0), seeding(announce.None, 90*m-time.Second)},
			want:      90*m - time.Second,
		},
		{
			name: "a pause credits its stopped, and the started after it nothing",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.Stopped, 45*m), seeding(announce.Started, 50*m), seeding(announce.None, 80*m),
			},
			want: 75 * m,
		},
		{
			name: "a pause whose stopped failed credits nothing until after the started",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.Stopped, 45*m).fails(), seeding(announce.Started, 50*m), seeding(announce.None, 80*m),
			},
			want: 60 * m,
		},
		{
			name: "a restart's started credits nothing though no stopped came before it",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.Started, 40*m), seeding(announce.None, 70*m),
			},
			want: 60 * m,
		},
		{
			name: "a failed announce between two successes neither credits nor resets",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.None, 60*m).fails(), seeding(announce.None, 70*m),
			},
			want: 70 * m,
		},
		{
			name: "an outage past 90 minutes credits nothing across it",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.None, 60*m).fails(), seeding(announce.None, 100*m).fails(), seeding(announce.None, 125*m),
			},
			want: 30 * m,
		},
		{
			name: "two trackers count apart and sum",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.None, 60*m).fails(),
				seeding(announce.Started, 61*m).to(second), seeding(announce.None, 91*m).to(second),
			},
			want: 60 * m,
		},
		{
			name: "a tracker taken over and back credits no time twice",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 30*m), seeding(announce.None, 40*m).fails(),
				seeding(announce.Started, 41*m).to(second), seeding(announce.None, 71*m).to(second), seeding(announce.None, 76*m),
			},
			want: 65 * m,
		},
		{
			name: "a credit runs from the last answer to the send, in whole seconds",
			exchanges: []exchange{
				seeding(announce.Started, 0).answeredAt(time.Second), seeding(announce.None, 30*m+1500*time.Millisecond).answeredAt(30*m + 2*time.Second),
			},
			want: 30 * m,
		},
		{
			name: "the gap runs from the last send to the answer",
			exchanges: []exchange{
				seeding(announce.Started, 0), seeding(announce.None, 90*m-time.Second).answeredAt(90 * m),
			},
			want: 0,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := ledger()
			var got time.Duration
			for _, e := range c.exchanges {
				l.Sent(h, e.tr, e.ev, at(e.sent), e.complete)
				if e.failed {
					l.Failed(h, e.tr, "connection refused", time.Hour, at(e.answered))
					continue
				}
				credit := l.Answered(h, e.tr, at(e.answered))
				if credit < 0 {
					t.Fatalf("the answer at %v credited %v", e.answered, credit)
				}
				got += credit
			}
			if got != c.want {
				t.Errorf("credited %v, want %v", got, c.want)
			}
		})
	}
}

func TestAForgottenTorrentCreditsNothingFromBefore(t *testing.T) {
	l := ledger()
	l.Sent(h, beacon, announce.Started, at(0), true)
	l.Answered(h, beacon, at(0))
	l.Forget(h)
	l.Sent(h, beacon, announce.None, at(30*time.Minute), true)
	if got := l.Answered(h, beacon, at(30*time.Minute)); got != 0 {
		t.Errorf("the first answer after Forget credited %v, want nothing", got)
	}
}
