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

func TestAnAdminEventListensWhileAnAdminSubscriptionIsOpen(t *testing.T) {
	h, pub := newHub(t)
	if pub.nodes.Listening() {
		t.Fatal("an empty hub listens for Admin events")
	}
	h.Subscribe(member, "")
	if pub.nodes.Listening() {
		t.Fatal("a member's subscription listens for Admin events")
	}
	s := h.Subscribe(admin, "")
	if !pub.nodes.Listening() {
		t.Fatal("an Admin's open subscription doesn't listen")
	}
	s.Close()
	if pub.nodes.Listening() {
		t.Fatal("a closed Admin subscription still listens")
	}
	if _, err := s.Next(t.Context()); !errors.Is(err, live.ErrEnded) {
		t.Errorf("Next after Close = %v, want %v", err, live.ErrEnded)
	}
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
		atHead := h.Subscribe(member, "946684800000003")
		pub.notices.Publish(t.Context(), notice{Text: "four"})
		four := live.Message{ID: start + 4, Name: "notices.posted", Data: notice{Text: "four"}}
		expect(t, s,
			live.Message{ID: start + 2, Name: "notices.posted", Data: notice{Text: "two"}},
			live.Message{ID: start + 3, Name: "notices.posted", Data: notice{Text: "three"}},
			four,
		)
		expect(t, atHead, four)
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
		h.EndSession(t.Context(), aliceSession, live.Revoked)
		pub.notices.Publish(t.Context(), notice{Text: "after"})
		for _, s := range []*live.Subscription{tab1, tab2} {
			expect(t, s, live.Message{Name: "live.session-ended", Data: live.SessionEnded{Reason: live.Revoked}})
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

func TestEndAccountStreamsReGatesOnlyThatAccountsTabs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, pub := newHub(t)
		tab1 := h.Subscribe(member, "946684800000000")
		tab2 := h.Subscribe(live.Viewer{Account: alice, Session: aliceTab2}, "946684800000000")
		signedOut := h.Subscribe(live.Viewer{Account: alice, Session: aliceTab2}, "946684800000000")
		other := h.Subscribe(admin, "946684800000000")
		h.EndSession(t.Context(), aliceTab2, live.Revoked)
		h.EndAccountStreams(t.Context(), alice)
		pub.notices.Publish(t.Context(), notice{Text: "after"})
		expect(t, tab1, live.Message{Name: "live.access-changed", Data: live.AccessChanged{}})
		expect(t, tab2, live.Message{Name: "live.session-ended", Data: live.SessionEnded{Reason: live.Revoked}})
		expect(t, signedOut, live.Message{Name: "live.session-ended", Data: live.SessionEnded{Reason: live.Revoked}})
		if _, err := tab1.Next(t.Context()); !errors.Is(err, live.ErrEnded) {
			t.Errorf("Next after access-changed = %v, want %v", err, live.ErrEnded)
		}
		expect(t, other, live.Message{ID: start + 1, Name: "notices.posted", Data: notice{Text: "after"}})
	})
}

func TestEndSessionWakesAWaitingReader(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, _ := newHub(t)
		s := h.Subscribe(member, "946684800000000")
		got := make(chan live.Message)
		go func() { got <- next(t, s) }()
		synctest.Wait()
		h.EndSession(t.Context(), aliceSession, live.SignedOut)
		if msg, want := <-got, (live.Message{Name: "live.session-ended", Data: live.SessionEnded{Reason: live.SignedOut}}); msg != want {
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

func TestMisuseOfTheHubPanics(t *testing.T) {
	inWrite := func(t *testing.T, fn func(ctx context.Context)) {
		_ = storagetest.Library(t).Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
			fn(ctx)
			return nil
		})
	}
	for _, tc := range []struct {
		name, want string
		misuse     func(t *testing.T)
	}{
		{"bind of an unlisted event", "live: event nodes.changed is not in surface.Events", func(*testing.T) {
			nodesChangedEvent.Bind(live.NewHub([]live.Event{sessionsChangedEvent}))
		}},
		{"bind of another declaration with a listed name", "live: event nodes.changed is not in surface.Events", func(*testing.T) {
			live.AdminEvent[nodesChanged]("nodes.changed").Bind(live.NewHub(events))
		}},
		{"watch of an unlisted event", "live: event nodes.changed is not in surface.Events", func(t *testing.T) {
			live.NewHub([]live.Event{sessionsChangedEvent}).Watch(t.Context(), nodesChangedEvent)
		}},
		{"two events with one payload type", "live: events nodes.changed and nodes.removed share payload type live_test.nodesChanged; give each event its own type", func(*testing.T) {
			live.NewHub([]live.Event{nodesChangedEvent, live.AdminEvent[nodesChanged]("nodes.removed")})
		}},
		{"a duplicate name", "live: event nodes.changed is declared twice", func(*testing.T) {
			live.NewHub([]live.Event{nodesChangedEvent, live.AdminEvent[notice]("nodes.changed")})
		}},
		{"a control event in the list", "live: event live.resync is declared twice", func(*testing.T) {
			live.NewHub([]live.Event{live.ResyncEvent})
		}},
		{"publish inside write", "live: Publish inside Write; publish after the commit", func(t *testing.T) {
			_, pub := newHub(t)
			inWrite(t, func(ctx context.Context) { pub.notices.Publish(ctx, notice{Text: "too soon"}) })
		}},
		{"end session inside write", "live: EndSession inside Write; end sessions after the commit", func(t *testing.T) {
			h, _ := newHub(t)
			inWrite(t, func(ctx context.Context) { h.EndSession(ctx, aliceSession, live.Revoked) })
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if got := recover(); got != tc.want {
					t.Errorf("panic = %v, want %q", got, tc.want)
				}
			}()
			tc.misuse(t)
		})
	}
}

func TestEventsListsEveryEventWithTheControlEvents(t *testing.T) {
	h, _ := newHub(t)
	var names []string
	for _, e := range h.Events() {
		names = append(names, e.Name())
	}
	want := "identity.sessions-changed live.access-changed live.resync live.session-ended nodes.changed notices.posted"
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

func TestTapKeepsWhatEachPublishNamed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h, p := newHub(t)
		tap := h.Tap(t.Context(), nodesChangedEvent)
		p.notices.Publish(t.Context(), notice{Text: "hi"})
		p.nodes.Publish(t.Context(), nodesChanged{Node: "seedbox"})
		p.nodes.Publish(t.Context(), nodesChanged{Node: "home"})
		if got := pending(tap.Woken()); got != 1 {
			t.Fatalf("two publishes woke the tap %d times, want 1", got)
		}
		msgs, lost := tap.Take()
		if lost || len(msgs) != 2 || msgs[0].Data != (nodesChanged{Node: "seedbox"}) || msgs[1].Data != (nodesChanged{Node: "home"}) {
			t.Fatalf("Take = %+v, %v; want seedbox then home", msgs, lost)
		}
		for range 1025 {
			p.nodes.Publish(t.Context(), nodesChanged{Node: "den"})
		}
		if msgs, lost := tap.Take(); !lost || msgs != nil {
			t.Fatalf("after 1,025 publishes Take = %d messages, lost %v; want none, lost", len(msgs), lost)
		}
	})
}
