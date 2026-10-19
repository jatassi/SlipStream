package plan_test

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
	"github.com/jatassi/slipstream/internal/downloading/internal/plan"
)

var (
	t0      = time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	a, b, c = bencode.InfoHash{'a'}, bencode.InfoHash{'b'}, bencode.InfoHash{'c'}
	tracker = announce.Tracker{Index: 0, Host: "beacon.example"}
)

func at(d time.Duration) time.Time { return t0.Add(d) }

type step func(*announce.Ledger)

// quickPause leaves h inside a 30-minute window that opened at t0.
func quickPause(h bencode.InfoHash) step {
	return func(l *announce.Ledger) {
		l.Sent(h, tracker, announce.Started, t0)
		l.Replied(h, tracker, announce.Reply{At: t0, NextAnnounce: at(30 * time.Minute), MinAnnounce: at(30 * time.Minute)})
		l.Sent(h, tracker, announce.Stopped, at(time.Minute))
	}
}

func stoppedAt(h bencode.InfoHash, d time.Duration) step {
	return func(l *announce.Ledger) { l.Sent(h, tracker, announce.Stopped, at(d)) }
}

var node = plan.Node{ActiveDownloads: 1, ActiveChecks: 1, StalledAfter: 10 * time.Minute, Live: true}

type decision struct {
	name   string
	node   plan.Node
	ledger []step
	ts     []plan.Torrent
	now    time.Duration
	want   plan.Plan
}

func check(t *testing.T, ds []decision) {
	t.Helper()
	for _, d := range ds {
		t.Run(d.name, func(t *testing.T) {
			l := announce.New(rand.New(rand.NewChaCha8([32]byte{'p', 'l', 'a', 'n'})), 3*time.Minute, 5*time.Minute) //nolint:gosec // seeded on purpose so tests replay
			for _, s := range d.ledger {
				s(l)
			}
			got := plan.Decide(d.ts, d.node, l, at(d.now))
			if !slices.Equal(got.Commands, d.want.Commands) {
				t.Errorf("commands %v, want %v", names(got.Commands), names(d.want.Commands))
			}
			if !maps.Equal(got.States, d.want.States) && (len(got.States) > 0 || len(d.want.States) > 0) {
				t.Errorf("states %v, want %v", got.States, d.want.States)
			}
			if !got.Wake.Equal(d.want.Wake) {
				t.Errorf("wake %v, want %v", got.Wake, d.want.Wake)
			}
		})
	}
}

func names(cs []plan.Command) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = fmt.Sprintf("%T%v", c, c)
	}
	return out
}

func TestTheQueueStartsInAddedOrderUpToTheLimit(t *testing.T) {
	two := node
	two.ActiveDownloads = 2
	unlimited := node
	unlimited.ActiveDownloads = 0
	queued := []plan.Torrent{
		{Hash: a, Order: 3, Phase: plan.Added},
		{Hash: b, Order: 1, Phase: plan.Added},
		{Hash: c, Order: 2, Phase: plan.Added},
	}
	check(t, []decision{
		{
			name: "the first two in added order",
			node: two, ts: queued,
			want: plan.Plan{
				Commands: []plan.Command{plan.Resume{Hash: b}, plan.Resume{Hash: c}},
				States:   map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldSlot}},
			},
		},
		{
			name: "a running download keeps its slot",
			node: two,
			ts: []plan.Torrent{
				{Hash: a, Order: 3, Phase: plan.Running, Resumed: t0},
				{Hash: b, Order: 1, Phase: plan.Added},
				{Hash: c, Order: 2, Phase: plan.Added},
			},
			now: time.Minute,
			want: plan.Plan{
				Commands: []plan.Command{plan.Resume{Hash: b}},
				States:   map[bencode.InfoHash]plan.State{c: {Hold: plan.HoldSlot}},
				Wake:     at(10 * time.Minute),
			},
		},
		{
			name: "a paused download frees its slot",
			node: node,
			ts: []plan.Torrent{
				{Hash: a, Order: 1, Phase: plan.Running, Paused: true, Resumed: t0},
				{Hash: b, Order: 2, Phase: plan.Added},
			},
			now: time.Minute,
			want: plan.Plan{
				Commands: []plan.Command{plan.Pause{Hash: a}, plan.Resume{Hash: b}},
				States:   map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldWanted}},
			},
		},
		{
			name: "no limit",
			node: unlimited, ts: queued,
			want: plan.Plan{Commands: []plan.Command{plan.Resume{Hash: b}, plan.Resume{Hash: c}, plan.Resume{Hash: a}}},
		},
	})
}

func TestAStalledDownloadFreesItsSlotAndKeepsRunning(t *testing.T) {
	ts := []plan.Torrent{
		{Hash: a, Order: 1, Phase: plan.Running, Resumed: t0, LastPayload: at(5 * time.Minute)},
		{Hash: b, Order: 2, Phase: plan.Added},
	}
	check(t, []decision{
		{
			name: "stalled",
			node: node, ts: ts, now: 15 * time.Minute,
			want: plan.Plan{
				Commands: []plan.Command{plan.Resume{Hash: b}},
				States:   map[bencode.InfoHash]plan.State{a: {Stalled: true}},
			},
		},
		{
			name: "not yet",
			node: node, ts: ts, now: 15*time.Minute - time.Second,
			want: plan.Plan{
				States: map[bencode.InfoHash]plan.State{b: {Hold: plan.HoldSlot}},
				Wake:   at(15 * time.Minute),
			},
		},
		{
			name: "the clock starts at the resume, not at a payload before the queue",
			node: node,
			ts: []plan.Torrent{
				{Hash: a, Order: 1, Phase: plan.Running, Resumed: at(time.Hour), LastPayload: t0},
				{Hash: b, Order: 2, Phase: plan.Added},
			},
			now: time.Hour + time.Minute,
			want: plan.Plan{
				States: map[bencode.InfoHash]plan.State{b: {Hold: plan.HoldSlot}},
				Wake:   at(time.Hour + 10*time.Minute),
			},
		},
	})
}

func TestARecoveredDownloadOverTheLimitKeepsRunning(t *testing.T) {
	check(t, []decision{{
		name: "two running on a limit of one",
		node: node,
		ts: []plan.Torrent{
			{Hash: a, Order: 1, Phase: plan.Running, Resumed: t0, LastPayload: at(29 * time.Minute)},
			{Hash: b, Order: 2, Phase: plan.Running, Resumed: at(20 * time.Minute), LastPayload: at(29 * time.Minute)},
			{Hash: c, Order: 3, Phase: plan.Added},
		},
		now: 30 * time.Minute,
		want: plan.Plan{
			States: map[bencode.InfoHash]plan.State{c: {Hold: plan.HoldSlot}},
			Wake:   at(39 * time.Minute),
		},
	}})
}

func TestSeedsNeverQueue(t *testing.T) {
	guarded := node
	guarded.HoldDownloads = true
	guarded.Draining = true
	check(t, []decision{
		{
			name: "past a full queue",
			node: node,
			ts: []plan.Torrent{
				{Hash: a, Order: 1, Phase: plan.Running, Resumed: t0},
				{Hash: b, Order: 2, Phase: plan.Added, Complete: true},
				{Hash: c, Order: 3, Phase: plan.Added, Complete: true},
			},
			want: plan.Plan{Commands: []plan.Command{plan.Resume{Hash: b}, plan.Resume{Hash: c}}, Wake: at(10 * time.Minute)},
		},
		{
			name: "under the disk guard while draining",
			node: guarded,
			ts:   []plan.Torrent{{Hash: b, Order: 2, Phase: plan.Added, Complete: true}},
			want: plan.Plan{Commands: []plan.Command{plan.Resume{Hash: b}}},
		},
		{
			name: "a running seed never stalls",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Running, Complete: true, Resumed: t0}},
			now:  time.Hour,
		},
	})
}

func TestChecksWaitForACheckSlot(t *testing.T) {
	away := node
	away.Live = false
	away.Draining = true
	check(t, []decision{
		{
			name: "one at a time, in added order",
			node: node,
			ts: []plan.Torrent{
				{Hash: c, Order: 3, Phase: plan.Added, NeedsCheck: true},
				{Hash: b, Order: 2, Phase: plan.Added, NeedsCheck: true, Complete: true},
			},
			want: plan.Plan{
				Commands: []plan.Command{plan.Recheck{Hash: b}},
				States:   map[bencode.InfoHash]plan.State{c: {Hold: plan.HoldCheckSlot}},
			},
		},
		{
			name: "behind a check that is running",
			node: node,
			ts: []plan.Torrent{
				{Hash: a, Order: 1, Phase: plan.Checking},
				{Hash: b, Order: 2, Phase: plan.Added, NeedsCheck: true},
			},
			want: plan.Plan{States: map[bencode.InfoHash]plan.State{b: {Hold: plan.HoldCheckSlot}}},
		},
		{
			name: "a paused torrent checks and stays paused",
			node: node,
			ts:   []plan.Torrent{{Hash: b, Order: 2, Phase: plan.Added, NeedsCheck: true, Paused: true}},
			want: plan.Plan{Commands: []plan.Command{plan.Recheck{Hash: b}}},
		},
		{
			name: "local data checks while draining and out of step",
			node: away,
			ts:   []plan.Torrent{{Hash: b, Order: 2, Phase: plan.Added, NeedsCheck: true}},
			want: plan.Plan{Commands: []plan.Command{plan.Recheck{Hash: b}}},
		},
	})
}

func TestDrainingAndOutOfStepStartOnlyWhatTheNodeHolds(t *testing.T) {
	draining := node
	draining.Draining = true
	away := node
	away.Live = false
	ts := []plan.Torrent{
		{Hash: a, Order: 1, Phase: plan.Fetching},
		{Hash: b, Order: 2, Phase: plan.Added},
		{Hash: c, Order: 3, Phase: plan.Added, Complete: true},
	}
	check(t, []decision{
		{
			name: "draining",
			node: draining, ts: ts,
			want: plan.Plan{
				Commands: []plan.Command{plan.Resume{Hash: b}, plan.Resume{Hash: c}},
				States:   map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldDraining}},
			},
		},
		{
			name: "out of step",
			node: away, ts: ts,
			want: plan.Plan{
				Commands: []plan.Command{plan.Resume{Hash: b}, plan.Resume{Hash: c}},
				States:   map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldController}},
			},
		},
		{
			name: "in step, a fetch is due",
			node: node, ts: ts,
			want: plan.Plan{Commands: []plan.Command{plan.Fetch{Hash: a}, plan.Resume{Hash: b}, plan.Resume{Hash: c}}},
		},
		{
			name: "a paused torrent is fetched too",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Fetching, Paused: true}},
			want: plan.Plan{
				Commands: []plan.Command{plan.Fetch{Hash: a}},
				States:   map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldWanted}},
			},
		},
	})
}

func TestTheDiskGuardHoldsQueuedDownloadsOnly(t *testing.T) {
	guarded := node
	guarded.ActiveDownloads = 0
	guarded.HoldDownloads = true
	check(t, []decision{{
		name: "running downloads and seeds carry on",
		node: guarded,
		ts: []plan.Torrent{
			{Hash: a, Order: 1, Phase: plan.Running, Resumed: t0},
			{Hash: b, Order: 2, Phase: plan.Added},
			{Hash: c, Order: 3, Phase: plan.Added, Complete: true},
		},
		want: plan.Plan{
			Commands: []plan.Command{plan.Resume{Hash: c}},
			States:   map[bencode.InfoHash]plan.State{b: {Hold: plan.HoldDiskGuard}},
			Wake:     at(10 * time.Minute),
		},
	}})
}

func TestADiskFullPauseResumesThroughThePolicy(t *testing.T) {
	recovered := node
	recovered.DiskRecovered = true
	recoveredButGuarded := recovered
	recoveredButGuarded.HoldDownloads = true
	full := []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added, DiskFull: true, Private: true}}
	check(t, []decision{
		{
			name: "no more free space",
			node: node, ts: full,
			want: plan.Plan{States: map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldDiskFull}}},
		},
		{
			name: "more space but the guard holds",
			node: recoveredButGuarded, ts: full,
			want: plan.Plan{States: map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldDiskFull}}},
		},
		{
			name: "more space",
			node: recovered, ts: full,
			want: plan.Plan{Commands: []plan.Command{plan.Resume{Hash: a}}},
		},
		{
			name:   "more space inside the window",
			node:   recovered,
			ledger: []step{quickPause(a)},
			ts:     full,
			now:    2 * time.Minute,
			want: plan.Plan{
				States: map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldWindow, Until: at(30 * time.Minute)}},
				Wake:   at(30 * time.Minute),
			},
		},
	})
}

func TestEveryStartWaitsForTheWindow(t *testing.T) {
	window := plan.State{Hold: plan.HoldWindow, Until: at(30 * time.Minute)}
	check(t, []decision{
		{
			name:   "a resume keeps its slot while it waits",
			node:   node,
			ledger: []step{quickPause(a)},
			ts: []plan.Torrent{
				{Hash: a, Order: 1, Phase: plan.Added, Private: true},
				{Hash: b, Order: 2, Phase: plan.Added},
			},
			now: 2 * time.Minute,
			want: plan.Plan{
				States: map[bencode.InfoHash]plan.State{a: window, b: {Hold: plan.HoldSlot}},
				Wake:   at(30 * time.Minute),
			},
		},
		{
			name:   "a seed",
			node:   node,
			ledger: []step{quickPause(a)},
			ts:     []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added, Private: true, Complete: true}},
			now:    2 * time.Minute,
			want:   plan.Plan{States: map[bencode.InfoHash]plan.State{a: window}, Wake: at(30 * time.Minute)},
		},
		{
			name:   "a public torrent does not wait",
			node:   node,
			ledger: []step{quickPause(a)},
			ts:     []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added}},
			now:    2 * time.Minute,
			want:   plan.Plan{Commands: []plan.Command{plan.Resume{Hash: a}}},
		},
		{
			name: "the start spread",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added, Complete: true, NotBefore: at(90 * time.Second)}},
			want: plan.Plan{
				States: map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldWindow, Until: at(90 * time.Second)}},
				Wake:   at(90 * time.Second),
			},
		},
		{
			name:   "the later of the spread and the window",
			node:   node,
			ledger: []step{quickPause(a)},
			ts:     []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added, Private: true, NotBefore: at(3 * time.Minute)}},
			now:    2 * time.Minute,
			want:   plan.Plan{States: map[bencode.InfoHash]plan.State{a: window}, Wake: at(30 * time.Minute)},
		},
		{
			name:   "the window has passed",
			node:   node,
			ledger: []step{quickPause(a)},
			ts:     []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added, Private: true}},
			now:    30 * time.Minute,
			want:   plan.Plan{Commands: []plan.Command{plan.Resume{Hash: a}}},
		},
	})
}

func TestARecheckPausesARunningTorrentFirst(t *testing.T) {
	running := []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Running, Complete: true, NeedsCheck: true}}
	check(t, []decision{
		{
			name: "pause",
			node: node, ts: running, now: time.Hour,
			want: plan.Plan{Commands: []plan.Command{plan.Pause{Hash: a}}},
		},
		{
			name:   "the pause waits out the duplicate rule",
			node:   node,
			ledger: []step{stoppedAt(a, time.Hour-10*time.Second)},
			ts:     running, now: time.Hour,
			want: plan.Plan{Wake: at(time.Hour + 20*time.Second)},
		},
		{
			name: "pausing",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Pausing, Complete: true, NeedsCheck: true}},
			now:  time.Hour,
		},
		{
			name: "paused, then the recheck",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added, Complete: true, NeedsCheck: true}},
			now:  time.Hour,
			want: plan.Plan{Commands: []plan.Command{plan.Recheck{Hash: a}}},
		},
		{
			name: "checking",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Checking, Complete: true}},
			now:  time.Hour,
		},
	})
}

func TestARecheckResumesThroughThePolicy(t *testing.T) {
	checked := []plan.Torrent{{Hash: a, Order: 1, Phase: plan.Added, Private: true, Complete: true}}
	check(t, []decision{
		{
			name:   "inside the window",
			node:   node,
			ledger: []step{quickPause(a)},
			ts:     checked, now: 5 * time.Minute,
			want: plan.Plan{
				States: map[bencode.InfoHash]plan.State{a: {Hold: plan.HoldWindow, Until: at(30 * time.Minute)}},
				Wake:   at(30 * time.Minute),
			},
		},
		{
			name: "outside it",
			node: node, ts: checked, now: 5 * time.Minute,
			want: plan.Plan{Commands: []plan.Command{plan.Resume{Hash: a}}},
		},
	})
}

func TestADiscardTheNodeDoesNotHoldPartsAtOnce(t *testing.T) {
	check(t, []decision{
		{
			name: "not held",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Phase: plan.Fetching, Discard: plan.DeleteFiles}},
			want: plan.Plan{Commands: []plan.Command{plan.Parted{Hash: a}}},
		},
		{
			name: "held and paused",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Phase: plan.Added, Discard: plan.DeleteFiles}},
			want: plan.Plan{Commands: []plan.Command{plan.Remove{Hash: a, DeleteFiles: true}}},
		},
		{
			name: "held and running",
			node: node,
			ts: []plan.Torrent{
				{Hash: a, Phase: plan.Running, Discard: plan.KeepFiles, Resumed: t0},
				{Hash: b, Order: 1, Phase: plan.Added},
			},
			now:  time.Minute,
			want: plan.Plan{Commands: []plan.Command{plan.Remove{Hash: a}, plan.Resume{Hash: b}}},
		},
		{
			name:   "running, a stopped went out 10 s ago",
			node:   node,
			ledger: []step{stoppedAt(a, 50*time.Second)},
			ts:     []plan.Torrent{{Hash: a, Phase: plan.Running, Discard: plan.KeepFiles, Resumed: t0}},
			now:    time.Minute,
			want:   plan.Plan{Wake: at(80 * time.Second)},
		},
		{
			name: "parting",
			node: node,
			ts:   []plan.Torrent{{Hash: a, Phase: plan.Parting, Discard: plan.KeepFiles}},
		},
	})
}

func TestWakeIsTheEarliestDeadline(t *testing.T) {
	unlimited := node
	unlimited.ActiveDownloads = 0
	check(t, []decision{
		{
			name:   "the start spread before a stall and a window",
			node:   unlimited,
			ledger: []step{quickPause(b)},
			ts: []plan.Torrent{
				{Hash: a, Order: 1, Phase: plan.Running, Resumed: t0, LastPayload: at(5 * time.Minute)},
				{Hash: b, Order: 2, Phase: plan.Added, Private: true},
				{Hash: c, Order: 3, Phase: plan.Added, NotBefore: at(9 * time.Minute)},
			},
			now: 2 * time.Minute,
			want: plan.Plan{
				States: map[bencode.InfoHash]plan.State{
					b: {Hold: plan.HoldWindow, Until: at(30 * time.Minute)},
					c: {Hold: plan.HoldWindow, Until: at(9 * time.Minute)},
				},
				Wake: at(9 * time.Minute),
			},
		},
		{
			name: "nothing due",
			node: node,
			ts: []plan.Torrent{
				{Hash: a, Order: 1, Phase: plan.Running, Complete: true, Resumed: t0},
				{Hash: b, Order: 2, Phase: plan.Added, Paused: true},
			},
			want: plan.Plan{States: map[bencode.InfoHash]plan.State{b: {Hold: plan.HoldWanted}}},
		},
	})
}
