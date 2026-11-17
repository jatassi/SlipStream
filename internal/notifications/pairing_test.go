package notifications_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/historytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
)

const wait = 5 * time.Minute

func disk(key string, level health.Level) health.Transition {
	now := time.Now()
	return health.Transition{
		Category: "storage", Key: health.Key(key), Kind: health.Raised, From: health.None, To: level,
		Subject: "Disk " + key + " is almost full", Text: "It has 1 GB left.", At: now, Since: now,
	}
}

func moved(t health.Transition, kind health.Kind, to health.Level) health.Transition {
	t.Kind, t.From, t.To, t.At = kind, t.To, to, time.Now()
	return t
}

func cleared(t health.Transition) health.Transition {
	t.Kind, t.From, t.To, t.At = health.Cleared, t.To, health.None, time.Now()
	return t
}

func paused(t *testing.T, w *world, name string) notifications.ConnectionID {
	t.Helper()
	id := w.Connect(name)
	w.Courier.Queue(notificationstest.URL(name), notificationstest.Reply{Status: http.StatusNotFound})
	w.Record(notificationstest.PingRow, ping("one"))
	return id
}

func unpause(t *testing.T, w *world, id notifications.ConnectionID) {
	t.Helper()
	if res, err := w.Service.TestSaved(t.Context(), id, notifications.Edit{}); err != nil || !res.Passed {
		t.Fatalf("TestSaved = %+v, %v", res, err)
	}
	w.Advance(time.Hour)
}

func TestAnItemThatClearsInsideTheWaitSendsNothing(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		admin := w.Admin("sam@example.test")
		d := disk("d", health.Fail)
		w.Transitions(d)
		w.Advance(wait / 2)
		w.Transitions(cleared(d))
		w.Advance(time.Hour)
		wantTitles(t, w, family)
		if f := w.Feed(admin); len(f.Rows) != 0 {
			t.Fatalf("the bell shows %+v", f.Rows)
		}
	})
}

func TestAWorseInsideTheWaitSendsOneMessageAtTheFirstDueTimeAtError(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		d := disk("d", health.Warn)
		due := time.Now().Add(wait)
		w.Transitions(d)
		w.Advance(wait / 2)
		w.Transitions(moved(d, health.Worse, health.Fail))
		w.Advance(time.Hour)
		got := w.Courier.Received(family)
		if len(got) != 1 || got[0].Title != "New Health item for Disk d is almost full" || got[0].Tone != entry.Fail || !got[0].At.Equal(due) {
			t.Fatalf("received %+v, want one message at Error at %v", got, due)
		}
		if raises := kindsOf(historytest.Entries(t, w.Lib)); !slices.Equal(raises, []entry.Kind{"new_health_item", "new_health_item"}) {
			t.Fatalf("History holds %v, want both raises", raises)
		}
	})
}

func TestAWorseBesideAFreshRaiseSendsEachAtItsOwnDueTime(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		d := disk("d", health.Warn)
		first := time.Now().Add(wait)
		w.Transitions(d)
		w.Advance(wait / 2)
		second := time.Now().Add(wait)
		w.Transitions(moved(d, health.Worse, health.Fail), disk("e", health.Fail))
		w.Advance(time.Hour)
		got := w.Courier.Received(family)
		if len(got) != 2 || got[0].Title != "New Health item for Disk d is almost full" || got[0].Tone != entry.Fail || !got[0].At.Equal(first) ||
			got[1].Title != "New Health item for Disk e is almost full" || !got[1].At.Equal(second) {
			t.Fatalf("received %+v, want d at Error at %v, then e at %v", got, first, second)
		}
		if strings.Contains(w.Logs(), "level=ERROR") {
			t.Fatalf("a dispatcher pass failed:\n%s", w.Logs())
		}
	})
}

func TestAWorseAfterTheRaiseWasSentSendsAgainAfterANewWait(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family", func(d *notifications.Draft) { d.Warnings = true })
		d := disk("d", health.Warn)
		w.Transitions(d)
		w.Advance(wait)
		w.Transitions(moved(d, health.Worse, health.Fail))
		w.Advance(wait - time.Second)
		wantTitles(t, w, family, "New Health item for Disk d is almost full")
		w.Advance(time.Second)
		got := w.Courier.Received(family)
		if len(got) != 2 || got[1].Tone != entry.Fail {
			t.Fatalf("received %+v, want the Worse at Error after its own wait", got)
		}
	})
}

func TestAClearAfterTheRaiseWasSentReachesOnlyTheTargetsThatWereTold(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		d := disk("d", health.Fail)
		w.Transitions(d)
		w.Advance(wait)
		w.Connect("late")
		w.Transitions(cleared(d))
		w.Advance(time.Hour)
		wantTitles(t, w, family, "New Health item for Disk d is almost full", "Health item for Disk d is almost full cleared")
		wantTitles(t, w, notificationstest.URL("late"))
	})
}

func kindsOf(entries []historytest.Entry) []entry.Kind {
	out := make([]entry.Kind, len(entries))
	for i, e := range entries {
		out[i] = e.Kind
	}
	return out
}

func TestThreeItemsInOneEvaluationAreOneMessagePerConnection(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		w.Connect("other")
		w.Transitions(disk("c", health.Fail), disk("d", health.Fail), disk("e", health.Fail))
		w.Advance(wait)
		wantTitles(t, w, family, "3 new Health items")
		wantTitles(t, w, notificationstest.URL("other"), "3 new Health items")
	})
}

func TestAWarningWithWarningsOffSendsNeitherTheItemNorItsClear(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		d := disk("d", health.Warn)
		w.Transitions(d)
		w.Advance(wait)
		w.Transitions(cleared(d))
		w.Advance(time.Hour)
		wantTitles(t, w, family)
	})
}

func TestAnErrorThatGotBetterStillSendsItsClearWithWarningsOff(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		d := disk("d", health.Fail)
		w.Transitions(d)
		w.Advance(wait)
		better := moved(d, health.Better, health.Warn)
		w.Transitions(better)
		w.Advance(time.Minute)
		w.Transitions(cleared(better))
		w.Advance(time.Hour)
		wantTitles(t, w, family, "New Health item for Disk d is almost full", "Health item for Disk d is almost full cleared")
	})
}

func TestARaiseGivenUpByARefusalIsFollowedByNoClear(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := paused(t, w, "family")
		d := disk("d", health.Fail)
		w.Transitions(d)
		w.Advance(wait)
		w.Transitions(cleared(d))
		w.Courier.Queue(family, notificationstest.Reply{}, notificationstest.Reply{}, notificationstest.Reply{Status: http.StatusBadRequest})
		unpause(t, w, id)
		wantTitles(t, w, family, "Ping one", "Test from SlipStream", "Ping one", "New Health item for Disk d is almost full")
		if st := w.Status(id); st.Waiting != 0 || st.GivenUp != 1 {
			t.Fatalf("the connection reads %+v, want the raise given up and the clear dropped", st)
		}
	})
}

func TestARaiseGivenUpByAgeIsFollowedByNoClear(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := paused(t, w, "family")
		d := disk("d", health.Fail)
		w.Transitions(d)
		w.Advance(time.Hour)
		w.Transitions(cleared(d))
		w.Advance(notifications.GiveUp.Default - 30*time.Minute)
		unpause(t, w, id)
		wantTitles(t, w, family, "Ping one", "Test from SlipStream")
		if st := w.Status(id); st.Waiting != 0 {
			t.Fatalf("the connection reads %+v, want the clear dropped", st)
		}
	})
}

func TestARaiseQueuedBehindAPauseArrivesThenItsClear(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := paused(t, w, "family")
		d := disk("d", health.Fail)
		w.Transitions(d)
		w.Advance(wait)
		w.Transitions(cleared(d))
		unpause(t, w, id)
		wantTitles(t, w, family, "Ping one", "Test from SlipStream", "Ping one", "New Health item for Disk d is almost full", "Health item for Disk d is almost full cleared")
	})
}

func TestAConnectionsOwnItemIsNeverSentToIt(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		w.Connect("other")
		own := health.Transition{
			Category: notifications.HealthAbout.ID, Key: health.KeyOf(id.String()), Kind: health.Raised, To: health.Fail,
			Subject: "Hook 'family' is paused", At: time.Now(), Since: time.Now(),
		}
		w.Transitions(own)
		w.Advance(wait)
		wantTitles(t, w, family)
		wantTitles(t, w, notificationstest.URL("other"), "New Health item for Hook 'family' is paused")
	})
}

func TestARaiseAndAClearInOneEvaluationAreTwoMessages(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		c := disk("c", health.Fail)
		w.Transitions(c)
		w.Advance(wait)
		w.Transitions(disk("d", health.Fail), cleared(c))
		w.Advance(wait)
		wantTitles(t, w, family, "New Health item for Disk c is almost full", "Health item for Disk c is almost full cleared", "New Health item for Disk d is almost full")
	})
}

func TestAClearAfterItsRaiseWasPrunedTakesTheWordsATargetWasTold(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		admin := w.Admin("sam@example.test")
		w.Set(notifications.GiveUp, 7*24*time.Hour)
		w.Connect("family")
		stuck := paused(t, w, "stuck")
		d := disk("d", health.Fail)
		w.Transitions(d)
		w.Advance(wait)
		wantTitles(t, w, family, "Ping one", "New Health item for Disk d is almost full")

		w.Advance(history.KeepFor.Default + 24*time.Hour)
		if st := w.Status(stuck); st.GivenUp != 2 || st.Waiting != 0 {
			t.Fatalf("the paused connection reads %+v, want its two messages given up", st)
		}
		w.Housekeeping()
		if f := w.Feed(admin); len(f.Rows) != 0 {
			t.Fatalf("after the prune the bell shows %+v", f.Rows)
		}
		restart := cleared(d)
		restart.Subject = "Storage: a problem that cleared while SlipStream was stopped"
		w.Transitions(restart)
		got := w.Courier.Received(family)
		if last := got[len(got)-1]; last.Title != "Health item for Disk d is almost full cleared" || !slices.Equal(last.Lines, []string{"It was on Health from 00:00 to 00:05."}) {
			t.Fatalf("the clear after the prune reads %+v", last)
		}
		if f := w.Feed(admin); len(f.Rows) != 1 || f.Rows[0].Title != "Health item for Disk d is almost full cleared" {
			t.Fatalf("the bell shows %+v, want the clear", f.Rows)
		}
	})
}

func TestAClearDuringTheWaitKeepsTheRaiseAndItsWorseUntilTheyAreSent(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		d := disk("d", health.Warn)
		w.Transitions(d)
		w.Advance(wait / 2)
		w.Transitions(moved(d, health.Worse, health.Fail))
		if err := w.History.Clear(t.Context(), identity.Admin{}); err != nil {
			t.Fatal(err)
		}
		w.Housekeeping()
		if p, err := w.History.Page(t.Context(), identity.Admin{}, history.Filter{}, 0, 10); err != nil || len(p.Entries) != 0 {
			t.Fatalf("after a Clear History lists %+v (%v), want nothing", p.Entries, err)
		}
		if held := kindsOf(historytest.Entries(t, w.Lib)); len(held) != 2 {
			t.Fatalf("after a Clear's sweep History holds %v, want the owed raise and the Worse it renders from", held)
		}
		w.Advance(time.Hour)
		if got := w.Courier.Received(family); len(got) != 1 || got[0].Tone != entry.Fail {
			t.Fatalf("received %+v, want one message at Error", got)
		}
		w.Housekeeping()
		if held := kindsOf(historytest.Entries(t, w.Lib)); len(held) != 0 {
			t.Fatalf("once sent History holds %v, want the cleared entries gone", held)
		}
	})
}
