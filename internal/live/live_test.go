package live_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// start is synctest's clock at the bubble's start, 2000-01-01 UTC, in Unix
// microseconds: the id of the head before any publish.
const start = 946684800000000

type nodesChanged struct{ Node string }

type sessionsChanged struct{ Account storage.UUID }

type notice struct{ Text string }

var (
	nodesChangedEvent    = live.AdminEvent[nodesChanged]("nodes.changed")
	sessionsChangedEvent = live.AccountEvent[sessionsChanged]("identity.sessions-changed")
	noticeEvent          = live.EveryoneEvent[notice]("notices.posted")
	events               = []live.Event{nodesChangedEvent, sessionsChangedEvent, noticeEvent}
)

var (
	alice        = mustUUID("01920000-0000-7000-8000-00000000000a")
	bob          = mustUUID("01920000-0000-7000-8000-00000000000b")
	aliceSession = mustUUID("01920000-0000-7000-8000-0000000000a1")
	aliceTab2    = mustUUID("01920000-0000-7000-8000-0000000000a2")
	bobSession   = mustUUID("01920000-0000-7000-8000-0000000000b1")
	member       = live.Viewer{Account: alice, Session: aliceSession}
	admin        = live.Viewer{Account: bob, Session: bobSession, Admin: true}
)

func mustUUID(s string) storage.UUID {
	u, err := storage.ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return u
}

type publishers struct {
	nodes    live.AdminPublisher[nodesChanged]
	sessions live.AccountPublisher[sessionsChanged]
	notices  live.EveryonePublisher[notice]
}

func newHub(t *testing.T) (*live.Hub, publishers) {
	t.Helper()
	h := live.NewHub(events)
	t.Cleanup(h.Close)
	return h, publishers{
		nodes:    nodesChangedEvent.Bind(h),
		sessions: sessionsChangedEvent.Bind(h),
		notices:  noticeEvent.Bind(h),
	}
}

func next(t *testing.T, s *live.Subscription) live.Message {
	t.Helper()
	msg, err := s.Next(t.Context())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	return msg
}

func expect(t *testing.T, s *live.Subscription, want ...live.Message) {
	t.Helper()
	for i, w := range want {
		if got := next(t, s); got != w {
			t.Fatalf("message %d = %+v, want %+v", i, got, w)
		}
	}
}

func resyncAt(id int64) live.Message {
	return live.Message{ID: id, Name: "live.resync", Data: live.Resync{}}
}

func TestMemberSkipsAdminEntriesAndReceivesItsAccounts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		s := h.Subscribe(member, "")
		pub.nodes.Publish(t.Context(), nodesChanged{Node: "seedbox"})
		pub.sessions.Publish(t.Context(), bob, sessionsChanged{Account: bob})
		pub.sessions.Publish(t.Context(), alice, sessionsChanged{Account: alice})
		pub.notices.Publish(t.Context(), notice{Text: "hello"})
		expect(t, s,
			resyncAt(start),
			live.Message{ID: start + 3, Name: "identity.sessions-changed", Data: sessionsChanged{Account: alice}},
			live.Message{ID: start + 4, Name: "notices.posted", Data: notice{Text: "hello"}},
		)
	})
}

func TestAdminReceivesAdminEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		s := h.Subscribe(admin, "")
		pub.sessions.Publish(t.Context(), alice, sessionsChanged{Account: alice})
		pub.nodes.Publish(t.Context(), nodesChanged{Node: "seedbox"})
		expect(t, s,
			resyncAt(start),
			live.Message{ID: start + 2, Name: "nodes.changed", Data: nodesChanged{Node: "seedbox"}},
		)
	})
}

func TestNextWaitsForTheNextPublish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		s := h.Subscribe(member, "")
		expect(t, s, resyncAt(start))
		got := make(chan live.Message)
		go func() { got <- next(t, s) }()
		synctest.Wait()
		pub.notices.Publish(t.Context(), notice{Text: "late"})
		if msg, want := <-got, (live.Message{ID: start + 1, Name: "notices.posted", Data: notice{Text: "late"}}); msg != want {
			t.Errorf("Next after a wait = %+v, want %+v", msg, want)
		}
	})
}

func TestNextReturnsWhenItsContextEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := newHub(t)
		s := h.Subscribe(member, "")
		expect(t, s, resyncAt(start))
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if _, err := s.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Next past its deadline = %v, want %v", err, context.DeadlineExceeded)
		}
	})
}

func TestReplayAfterLastEventID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		for _, text := range []string{"one", "two", "three"} {
			pub.notices.Publish(t.Context(), notice{Text: text})
		}
		s := h.Subscribe(member, "946684800000001")
		pub.notices.Publish(t.Context(), notice{Text: "four"})
		expect(t, s,
			live.Message{ID: start + 2, Name: "notices.posted", Data: notice{Text: "two"}},
			live.Message{ID: start + 3, Name: "notices.posted", Data: notice{Text: "three"}},
			live.Message{ID: start + 4, Name: "notices.posted", Data: notice{Text: "four"}},
		)
	})
}

func TestLastEventIDAtTheHeadContinuesLive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		pub.notices.Publish(t.Context(), notice{Text: "seen"})
		s := h.Subscribe(member, "946684800000001")
		pub.notices.Publish(t.Context(), notice{Text: "new"})
		expect(t, s, live.Message{ID: start + 2, Name: "notices.posted", Data: notice{Text: "new"}})
	})
}

func TestLastEventIDTheRingCannotServeResyncs(t *testing.T) {
	for _, tc := range []struct {
		name, lastEventID string
	}{
		{"absent", ""},
		{"not an integer", "abc"},
		{"from another process", "5"},
		{"above the head", "946684800004099"},
		{"older than the ring", "946684800000001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h, pub := newHub(t)
				for range 4098 {
					pub.notices.Publish(t.Context(), notice{Text: "filler"})
				}
				s := h.Subscribe(member, tc.lastEventID)
				pub.notices.Publish(t.Context(), notice{Text: "live"})
				expect(t, s,
					resyncAt(start+4098),
					live.Message{ID: start + 4099, Name: "notices.posted", Data: notice{Text: "live"}},
				)
			})
		})
	}
}

func TestOldestEntryInTheRingReplays(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		for range 4098 {
			pub.notices.Publish(t.Context(), notice{Text: "filler"})
		}
		s := h.Subscribe(member, "946684800000002")
		expect(t, s, live.Message{ID: start + 3, Name: "notices.posted", Data: notice{Text: "filler"}})
	})
}

func TestLappedReaderGetsOneResyncAtTheHead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		s := h.Subscribe(member, "946684800000000")
		for range 4097 {
			pub.notices.Publish(t.Context(), notice{Text: "filler"})
		}
		pub.sessions.Publish(t.Context(), alice, sessionsChanged{Account: alice})
		expect(t, s,
			resyncAt(start+4098),
		)
		pub.notices.Publish(t.Context(), notice{Text: "after"})
		expect(t, s, live.Message{ID: start + 4099, Name: "notices.posted", Data: notice{Text: "after"}})
	})
}

func TestReaderAFullRingBehindIsNotLapped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		s := h.Subscribe(member, "946684800000000")
		for range 4096 {
			pub.notices.Publish(t.Context(), notice{Text: "filler"})
		}
		expect(t, s, live.Message{ID: start + 1, Name: "notices.posted", Data: notice{Text: "filler"}})
	})
}

func TestEndSessionEndsOnlyThatSessionsSubscriptions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		tab1 := h.Subscribe(member, "946684800000000")
		tab2 := h.Subscribe(member, "946684800000000")
		other := h.Subscribe(live.Viewer{Account: alice, Session: aliceTab2}, "946684800000000")
		pub.notices.Publish(t.Context(), notice{Text: "before"})
		h.EndSession(aliceSession)
		pub.notices.Publish(t.Context(), notice{Text: "after"})
		for _, s := range []*live.Subscription{tab1, tab2} {
			expect(t, s, live.Message{Name: "live.session-ended", Data: live.SessionEnded{Reason: "signed-out"}})
			if _, err := s.Next(t.Context()); !errors.Is(err, live.ErrEnded) {
				t.Errorf("Next after session-ended = %v, want %v", err, live.ErrEnded)
			}
		}
		expect(t, other,
			live.Message{ID: start + 1, Name: "notices.posted", Data: notice{Text: "before"}},
			live.Message{ID: start + 2, Name: "notices.posted", Data: notice{Text: "after"}},
		)
	})
}

func TestEndSessionWakesAWaitingReader(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := newHub(t)
		s := h.Subscribe(member, "946684800000000")
		got := make(chan live.Message)
		go func() { got <- next(t, s) }()
		synctest.Wait()
		h.EndSession(aliceSession)
		if msg, want := <-got, (live.Message{Name: "live.session-ended", Data: live.SessionEnded{Reason: "signed-out"}}); msg != want {
			t.Errorf("Next = %+v, want %+v", msg, want)
		}
	})
}

func TestCloseEndsEverySubscription(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		waiting := h.Subscribe(member, "946684800000000")
		behind := h.Subscribe(admin, "946684800000000")
		pub.notices.Publish(t.Context(), notice{Text: "unread"})
		expect(t, waiting, live.Message{ID: start + 1, Name: "notices.posted", Data: notice{Text: "unread"}})
		errs := make(chan error)
		go func() {
			_, err := waiting.Next(t.Context())
			errs <- err
		}()
		synctest.Wait()
		h.Close()
		if err := <-errs; !errors.Is(err, live.ErrEnded) {
			t.Errorf("waiting Next after Close = %v, want %v", err, live.ErrEnded)
		}
		if _, err := behind.Next(t.Context()); !errors.Is(err, live.ErrEnded) {
			t.Errorf("unread Next after Close = %v, want %v", err, live.ErrEnded)
		}
		if _, err := h.Subscribe(member, "").Next(t.Context()); !errors.Is(err, live.ErrEnded) {
			t.Errorf("Next on a subscription made after Close = %v, want %v", err, live.ErrEnded)
		}
	})
}

func TestSubscriptionCloseEndsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := newHub(t)
		s := h.Subscribe(member, "")
		s.Close()
		if _, err := s.Next(t.Context()); !errors.Is(err, live.ErrEnded) {
			t.Errorf("Next after Close = %v, want %v", err, live.ErrEnded)
		}
	})
}

func TestEmptyHubResyncs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := live.NewHub(nil)
		defer h.Close()
		expect(t, h.Subscribe(member, ""), resyncAt(start))
	})
}

func assertPanics(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if got := recover(); got != want {
			t.Errorf("panic = %v, want %q", got, want)
		}
	}()
	fn()
}

func TestBindOfAnUnlistedEventPanics(t *testing.T) {
	h := live.NewHub([]live.Event{sessionsChangedEvent})
	assertPanics(t, "live: event nodes.changed is not in surface.Events", func() {
		nodesChangedEvent.Bind(h)
	})
}

func TestBindOfAnotherDeclarationWithAListedNamePanics(t *testing.T) {
	h := live.NewHub(events)
	assertPanics(t, "live: event nodes.changed is not in surface.Events", func() {
		live.AdminEvent[nodesChanged]("nodes.changed").Bind(h)
	})
}

func TestTwoEventsWithOnePayloadTypePanic(t *testing.T) {
	assertPanics(t, "live: events nodes.changed and nodes.removed share payload type live_test.nodesChanged; give each event its own type", func() {
		live.NewHub([]live.Event{nodesChangedEvent, live.AdminEvent[nodesChanged]("nodes.removed")})
	})
}

func TestADuplicateNamePanics(t *testing.T) {
	assertPanics(t, "live: event nodes.changed is declared twice", func() {
		live.NewHub([]live.Event{nodesChangedEvent, live.AdminEvent[notice]("nodes.changed")})
	})
}

func TestAControlEventInTheListPanics(t *testing.T) {
	assertPanics(t, "live: event live.resync is declared twice", func() {
		live.NewHub([]live.Event{live.ResyncEvent})
	})
}

func TestPublishInsideWritePanics(t *testing.T) {
	_, pub := newHub(t)
	lib := storagetest.Library(t)
	assertPanics(t, "live: Publish inside Write; publish after the commit", func() {
		_ = lib.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
			pub.notices.Publish(ctx, notice{Text: "too soon"})
			return nil
		})
	})
}

func TestEventsListsEveryEventWithTheControlEvents(t *testing.T) {
	h, _ := newHub(t)
	var names []string
	for _, e := range h.Events() {
		names = append(names, e.Name())
	}
	want := "identity.sessions-changed live.resync live.session-ended nodes.changed notices.posted"
	if got := strings.Join(names, " "); got != want {
		t.Fatalf("Events = %s, want %s", got, want)
	}
}

func TestWatchWakesOnceForEveryPublishSinceItLastReceived(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, p := newHub(t)
		ctx, stop := context.WithCancel(t.Context())
		woken := h.Watch(ctx, nodesChangedEvent, noticeEvent)
		p.sessions.Publish(t.Context(), alice, sessionsChanged{Account: alice})
		if pending(woken) != 0 {
			t.Fatal("an event it doesn't watch woke the watch")
		}
		p.nodes.Publish(t.Context(), nodesChanged{Node: "seedbox"})
		p.notices.Publish(t.Context(), notice{Text: "hi"})
		p.nodes.Publish(t.Context(), nodesChanged{Node: "home"})
		if got := pending(woken); got != 1 {
			t.Fatalf("three publishes woke the watch %d times, want 1", got)
		}
		stop()
		synctest.Wait()
		p.nodes.Publish(t.Context(), nodesChanged{Node: "seedbox"})
		if got := pending(woken); got != 0 {
			t.Fatalf("a publish after the watch's context ended woke it %d times", got)
		}
	})
}

func pending(woken <-chan struct{}) int {
	for n := 0; ; n++ {
		select {
		case <-woken:
		default:
			return n
		}
	}
}

func TestWatchOfAnUnlistedEventPanics(t *testing.T) {
	h := live.NewHub([]live.Event{sessionsChangedEvent})
	assertPanics(t, "live: event nodes.changed is not in surface.Events", func() {
		h.Watch(t.Context(), nodesChangedEvent)
	})
}
