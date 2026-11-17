package notifications_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

type world = notificationstest.World

var family = notificationstest.URL("family")

func ping(name string) notificationstest.Ping { return notificationstest.Ping{Name: name} }

func pings(names ...string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "Ping " + n
	}
	return out
}

func titles(rs []notificationstest.Received) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Title
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func wantTitles(t *testing.T, w *world, target string, want ...string) {
	t.Helper()
	if got := titles(w.Courier.Received(target)); !slices.Equal(got, want) {
		t.Fatalf("%s received %q, want %q", target, got, want)
	}
}

func run(t *testing.T, f func(t *testing.T, w *world)) {
	synctest.Test(t, func(t *testing.T) { f(t, notificationstest.New(t, notifications.DefaultTimings())) })
}

func TestARestartBetweenTheRecordAndTheSendDeliversOnce(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		w.Stop()
		w.Record(notificationstest.PingRow, ping("one"))
		wantTitles(t, w, family)
		w.Restart()
		w.Advance(time.Hour)
		wantTitles(t, w, family, pings("one")...)
	})
}

func TestARestartBeforeTheSettleSendsThatMessageAtMostTwice(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Hold: make(chan struct{})})
		w.Record(notificationstest.PingRow, ping("one"), ping("two"))
		wantTitles(t, w, family, pings("one")...)
		w.Restart()
		w.Advance(time.Hour)
		wantTitles(t, w, family, pings("one", "one", "two")...)
	})
}

func TestAFailedSettleSendsNothingAgainAndTheNextPassSettles(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		hold := make(chan struct{})
		w.Courier.Queue(family, notificationstest.Reply{Hold: hold})
		w.Record(notificationstest.PingRow, ping("one"), ping("two"))
		w.FailNextWrite()
		close(hold)
		synctest.Wait()
		wantTitles(t, w, family, pings("one")...)
		w.Advance(notifications.DefaultTimings().Ladder[0])
		wantTitles(t, w, family, pings("one", "two")...)
		w.Advance(time.Hour)
		wantTitles(t, w, family, pings("one", "two")...)
	})
}

func TestASlowDownWaitsAndWarnsOnlyOnceItPassesTheWarnTime(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"2"}}})
		w.Record(notificationstest.PingRow, ping("one"))
		if st := w.Status(id); st.Shown != notifications.ShownWaiting || st.Level != health.None {
			t.Fatalf("after a 429 the connection reads %+v, want waiting with no level", st)
		}
		if a := w.Health(); len(a.Items) != 0 {
			t.Fatalf("a 2 s wait raised %+v", a.Items)
		}
		w.Advance(2*time.Second - time.Millisecond)
		wantTitles(t, w, family, pings("one")...)
		w.Advance(time.Millisecond)
		wantTitles(t, w, family, pings("one", "one")...)

		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"3600"}}})
		w.Record(notificationstest.PingRow, ping("two"))
		start := time.Now()
		a := w.Health()
		if len(a.Items) != 0 || !a.Next.Equal(start.Add(notifications.WarnAfter.Default)) {
			t.Fatalf("an hour's wait answers %+v, want no item until the warn time", a)
		}
		w.AdvanceTo(a.Next)
		a = w.Health()
		if len(a.Items) != 1 || a.Items[0].Level != health.Warn || a.Items[0].Subject != "Messages to Hook 'family' are waiting" {
			t.Fatalf("past the warn time Health reads %+v", a.Items)
		}
	})
}

func TestA503RunClimbsTheLadderAndResetsAfterASuccess(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		down := notificationstest.Reply{Status: http.StatusServiceUnavailable}
		w.Courier.Queue(family, down, down, down)
		w.Record(notificationstest.PingRow, ping("one"))
		ladder := notifications.DefaultTimings().Ladder
		for i, rung := range ladder[:3] {
			// A retry comes at least half its rung later.
			w.Advance(rung / 2)
			if got := len(w.Courier.Received(family)); got != i+1 {
				t.Fatalf("within half of rung %d the Courier had %d sends, want %d", i, got, i+1)
			}
			w.Advance(rung / 2)
			if got := len(w.Courier.Received(family)); got != i+2 {
				t.Fatalf("after rung %d the Courier had %d sends, want %d", i, got, i+2)
			}
		}
		w.Courier.Queue(family, down)
		w.Record(notificationstest.PingRow, ping("two"))
		w.Advance(ladder[0])
		wantTitles(t, w, family, pings("one", "one", "one", "one", "two", "two")...)
	})
}

func TestA404PausesAndAPassingTestSendsTheWaitingMessagesInOrder(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusNotFound})
		w.Record(notificationstest.PingRow, ping("one"))
		w.Record(notificationstest.PingRow, ping("two"), ping("three"))
		w.Advance(time.Hour)
		wantTitles(t, w, family, pings("one")...)
		a := w.Health()
		want := health.Item{Key: health.KeyOf(id.String()), Level: health.Fail}
		if len(a.Items) != 1 || a.Items[0].Key != want.Key || a.Items[0].Level != want.Level ||
			a.Items[0].Subject != "Hook 'family' is paused" ||
			a.Items[0].Text != "Hook responded that this address no longer exists. Edit the connection with a new one. Edit the connection to fix it, then test it." {
			t.Fatalf("a paused connection reads %+v", a.Items)
		}

		res, err := w.Service.TestSaved(t.Context(), id, notifications.Edit{})
		if err != nil || !res.Passed || res.Node != "Den" {
			t.Fatalf("TestSaved = %+v, %v", res, err)
		}
		synctest.Wait()
		wantTitles(t, w, family, "Ping one", "Test from SlipStream", "Ping one", "Ping two", "Ping three")
		if a := w.Health(); len(a.Items) != 0 {
			t.Fatalf("after the Test Health reads %+v", a.Items)
		}
	})
}

func TestAPassingTestWithEditsDoesntUnpauseAndTheSaveDoes(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusNotFound})
		w.Record(notificationstest.PingRow, ping("one"))
		moved := notificationstest.URL("family-moved")
		edit := notifications.Edit{Settings: providers.Values{"url": notificationstest.Text(moved)}}
		res, err := w.Service.TestSaved(t.Context(), id, edit)
		if err != nil || !res.Passed {
			t.Fatalf("TestSaved = %+v, %v", res, err)
		}
		w.Advance(time.Hour)
		wantTitles(t, w, moved, "Test from SlipStream")
		if st := w.Status(id); st.Shown != notifications.ShownPaused {
			t.Fatalf("after a Test with edits the connection reads %v, want paused", st.Shown)
		}
		w.Edit(id, edit)
		wantTitles(t, w, moved, "Test from SlipStream", "Ping one")
		wantTitles(t, w, family, pings("one")...)
	})
}

func TestAFailingTestNamesTheReasonAndTheField(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		draft := notifications.Draft{
			Name: "family", Provider: notificationstest.HookID, On: true,
			Settings: providers.Values{"url": notificationstest.Text(family)}, Secrets: map[string]string{"token": "wrong"},
		}
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusUnauthorized})
		res, err := w.Service.Test(t.Context(), draft)
		want := notifications.TestResult{Node: "Den", Reason: "Hook responded that it doesn't accept these credentials. Edit the connection with new ones.", Field: "token"}
		if err != nil || res != want {
			t.Fatalf("Test = %+v, %v; want %+v", res, err, want)
		}
		if views, err := w.Service.Connections(t.Context()); err != nil || len(views) != 0 {
			t.Fatalf("a Test left connections %+v, %v", views, err)
		}
	})
}

func TestA400GivesUpOneMessageAndTheNextGoes(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusBadRequest})
		w.Record(notificationstest.PingRow, ping("one"), ping("two"))
		w.Advance(time.Hour)
		wantTitles(t, w, family, pings("one", "two")...)
		if st := w.Status(id); st.Shown != notifications.ShownOK || st.GivenUp != 1 || st.Waiting != 0 {
			t.Fatalf("after a 400 the connection reads %+v, want ok with one given up", st)
		}
	})
}

func TestA413WithAPosterIsSentAgainWithoutIt(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		tooLarge := notificationstest.Reply{Status: http.StatusRequestEntityTooLarge, Body: "too large for " + notificationstest.Token("family")}
		withPoster := notificationstest.Ping{Name: "one", Poster: notificationstest.APoster}
		w.Courier.Queue(family, tooLarge)
		w.Record(notificationstest.PingRow, withPoster)
		got := w.Courier.Received(family)
		if len(got) != 2 || got[0].Poster != "poster.jpg" || got[1].Poster != "" {
			t.Fatalf("received %+v, want the poster then the same message without it", got)
		}
		if st := w.Status(id); st.GivenUp != 0 || st.Shown != notifications.ShownOK {
			t.Fatalf("after the retry passed the connection reads %+v", st)
		}

		w.Courier.Queue(family, tooLarge, tooLarge)
		withPoster.Name = "two"
		w.Record(notificationstest.PingRow, withPoster, ping("three"))
		wantTitles(t, w, family, "Ping one", "Ping one", "Ping two", "Ping two", "Ping three")
		if st := w.Status(id); st.GivenUp != 1 || st.Shown != notifications.ShownOK {
			t.Fatalf("after a second 413 the connection reads %+v, want one given up", st)
		}
	})
}

func TestAMessageOlderThanGiveUpIsGivenUpAndCountedUntilASave(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		for range 100 {
			w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusServiceUnavailable})
		}
		w.Record(notificationstest.PingRow, ping("one"))
		w.Advance(notifications.GiveUp.Default + time.Second)
		if st := w.Status(id); st.GivenUp != 1 || st.Waiting != 0 {
			t.Fatalf("past Give up the connection reads %+v, want one given up and none waiting", st)
		}
		w.Edit(id, notifications.Edit{Warnings: ptr(true)})
		if st := w.Status(id); st.GivenUp != 0 {
			t.Fatalf("after a Save the connection counts %d given up, want 0", st.GivenUp)
		}
	})
}

func TestASentResponseIsLoggedWithSecretsMasked(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusFailedDependency, Body: "one target refused " + notificationstest.Token("family")})
		w.Record(notificationstest.PingRow, ping("one"))
		logs := w.Logs()
		if !strings.Contains(logs, "one target refused [secret]") || strings.Contains(logs, notificationstest.Token("family")) {
			t.Fatalf("the log reads %q, want the response with the token masked", logs)
		}
		if st := w.Status(id); st.Shown != notifications.ShownOK || st.Waiting != 0 || st.GivenUp != 0 {
			t.Fatalf("after a 424 the connection reads %+v, want ok", st)
		}
		if n := deliveries(t, w, "%"); n != 0 {
			t.Fatalf("%d delivery rows remain", n)
		}
	})
}

func TestOnlyDueAndGivenUpDeliveriesRemainAfterARunOfSends(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		other := w.Connect("other")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusBadRequest})
		w.Record(notificationstest.PingRow, ping("one"), ping("two"), ping("three"))
		w.Edit(other, notifications.Edit{On: ptr(false)})
		w.Advance(time.Hour)
		if due, given, all := deliveries(t, w, "due"), deliveries(t, w, "given_up"), deliveries(t, w, "%"); due != 0 || given != 1 || all != 1 {
			t.Fatalf("delivery rows: %d due, %d given up, %d in all; want only the one given up", due, given, all)
		}
	})
}

func deliveries(t *testing.T, w *world, state string) int {
	t.Helper()
	var n int
	if err := w.Lib.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM notifications_delivery WHERE state LIKE ?", state).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTurningOffDeletingAndTurningAKindOffDropMessages(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		off, gone, kind := w.Connect("off"), w.Connect("gone"), w.Connect("kind")
		for _, name := range []string{"off", "gone", "kind"} {
			w.Courier.Queue(notificationstest.URL(name), notificationstest.Reply{Status: http.StatusNotFound})
		}
		w.Record(notificationstest.PingRow, ping("one"))
		w.Record(notificationstest.PongRow, ping("two"))
		w.Edit(off, notifications.Edit{On: ptr(false)})
		if err := w.Service.Delete(t.Context(), gone); err != nil {
			t.Fatal(err)
		}
		w.Edit(kind, notifications.Edit{Events: ptr([]notice.Kind{notificationstest.PingRow.Kind})})
		w.Edit(off, notifications.Edit{On: ptr(true)})
		w.Advance(time.Hour)
		wantTitles(t, w, notificationstest.URL("off"), pings("one")...)
		wantTitles(t, w, notificationstest.URL("gone"), pings("one")...)
		wantTitles(t, w, notificationstest.URL("kind"), pings("one", "one")...)

		w.Record(notificationstest.PingRow, ping("three"))
		wantTitles(t, w, notificationstest.URL("off"), pings("one", "three")...)
	})
}

func TestAConnectionTurnedOnLaterGetsNoOldEvents(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		late := w.Connect("late", func(d *notifications.Draft) { d.On = false })
		w.Record(notificationstest.PingRow, ping("before"))
		w.Advance(time.Minute)
		w.Edit(late, notifications.Edit{On: ptr(true)})
		w.Record(notificationstest.PingRow, ping("after"))
		w.Advance(time.Hour)
		wantTitles(t, w, notificationstest.URL("late"), pings("after")...)
	})
}

func TestNoPassGivesUpMoreThanItsBound(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		id := w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusNotFound})
		w.Record(notificationstest.PingRow, ping("first"))
		backlog := make([]notificationstest.Ping, 4999)
		for i := range backlog {
			backlog[i] = ping(strconv.Itoa(i))
		}
		w.Record(notificationstest.PingRow, backlog...)
		if st := w.Status(id); st.Waiting != 5000 {
			t.Fatalf("%d messages wait, want 5000", st.Waiting)
		}
		var (
			mu     sync.Mutex
			counts []int
		)
		w.EachPass(func() {
			v, err := w.Service.Connection(context.Background(), id)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			counts = append(counts, v.Status.GivenUp)
			mu.Unlock()
		})
		w.Advance(notifications.GiveUp.Default + time.Second)
		w.EachPass(nil)
		mu.Lock()
		defer mu.Unlock()
		if st := w.Status(id); st.GivenUp != 5000 || st.Waiting != 0 {
			t.Fatalf("past Give up the connection reads %+v, want 5000 given up", st)
		}
		passes := 0
		for i := 1; i < len(counts); i++ {
			if step := counts[i] - counts[i-1]; step > 1000 {
				t.Fatalf("one pass gave up %d messages", step)
			} else if step > 0 {
				passes++
			}
		}
		if passes < 5 {
			t.Fatalf("the give-ups took %d passes, want at least 5", passes)
		}
	})
}

func TestAWeekOfBacklogDrainsInOrderAndGivesUpWhatGrowsTooOld(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		week := 7 * 24 * time.Hour
		w.Set(notifications.GiveUp, week)
		id := w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusNotFound})
		start := time.Now()
		w.Record(notificationstest.PingRow, ping("0"))
		at := map[string]time.Time{"Ping 0": start}
		for i := 1; i <= 100; i++ {
			w.Advance(20 * time.Second)
			w.Record(notificationstest.PingRow, ping(strconv.Itoa(i)))
			at[fmt.Sprintf("Ping %d", i)] = time.Now()
		}
		for range 200 {
			w.Courier.Queue(family, notificationstest.Reply{Delay: 50 * time.Second})
		}
		w.AdvanceTo(start.Add(week - 5*time.Second))
		w.Edit(id, notifications.Edit{Warnings: ptr(true)})
		w.Advance(2 * time.Hour)

		got := w.Courier.Received(family)[1:]
		last := -1
		for _, r := range got {
			n, err := strconv.Atoi(strings.TrimPrefix(r.Title, "Ping "))
			if err != nil || n <= last {
				t.Fatalf("the drain sent %q after Ping %d", r.Title, last)
			}
			if age := r.At.Sub(at[r.Title]); age > week {
				t.Fatalf("%s went out %v after it happened, past Give up", r.Title, age)
			}
			last = n
		}
		st := w.Status(id)
		if st.Waiting != 0 || st.GivenUp == 0 || len(got) < 2 || st.GivenUp+len(got) != 101 {
			t.Fatalf("the drain sent %d and gave up %d with %d waiting, want all 101 accounted for", len(got), st.GivenUp, st.Waiting)
		}
	})
}

func TestACustomScriptHoldsWhileItsNodeIsOfflineAndPausesWhenItIsRemoved(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		seedbox := w.AddNode("seedbox")
		id := w.ConnectScript("den", seedbox)
		path := notificationstest.ScriptPath("den")
		w.Roster.SetOnline(seedbox, false)
		synctest.Wait()
		w.Record(notificationstest.PingRow, ping("one"))
		w.Advance(time.Hour)
		wantTitles(t, w, path)
		if st := w.Status(id); st.Shown != notifications.ShownHolding || st.Level != health.None {
			t.Fatalf("on an offline Node the script reads %+v, want holding", st)
		}
		if a := w.Health(); len(a.Items) != 0 {
			t.Fatalf("a script holding raised %+v", a.Items)
		}
		w.Roster.SetOnline(seedbox, true)
		synctest.Wait()
		got := w.Courier.Received(path)
		if len(got) != 1 || got[0].Title != "Ping one" || got[0].Node != seedbox {
			t.Fatalf("back online the script ran %+v", got)
		}

		w.Roster.Remove(seedbox)
		synctest.Wait()
		a := w.Health()
		if len(a.Items) != 1 || a.Items[0].Level != health.Fail || a.Items[0].Subject != "Custom Script 'den' is paused" ||
			!strings.HasPrefix(a.Items[0].Text, "seedbox was removed. Pick another Node for this script.") {
			t.Fatalf("with its Node removed the script reads %+v", a.Items)
		}
	})
}

func TestAPausedLinesWordsFollowTheProviderWithoutAWrite(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusNotFound})
		w.Record(notificationstest.PingRow, ping("one"))
		passes := w.Passes()
		w.Words[providers.Gone] = "Hook has no hook at this URL. Paste a new one."
		a := w.Health()
		if len(a.Items) != 1 || !strings.HasPrefix(a.Items[0].Text, "Hook has no hook at this URL. Paste a new one.") {
			t.Fatalf("after the words changed Health reads %+v", a.Items)
		}
		if w.Passes() != passes {
			t.Fatal("rewording a reason ran a pass")
		}
	})
}

// TestAScriptLeftAtItsDefaultStopRunsForItAndAFailureNamesItsNode adds a
// script with no stop, which runs for the field's 30 s, and a Test whose
// script exits 3 says which Node it ran on.
func TestAScriptLeftAtItsDefaultStopRunsForItAndAFailureNamesItsNode(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		seedbox := w.AddNode("seedbox")
		id := w.ConnectScript("den", seedbox)
		path := notificationstest.ScriptPath("den")
		w.Courier.Queue(path, notificationstest.Reply{Exit: 3, Body: "it broke"})
		res, err := w.Service.TestSaved(t.Context(), id, notifications.Edit{})
		if err != nil || res.Passed || res.Reason != "The script on seedbox exited with code 3: it broke" {
			t.Fatalf("a Test of a failing script = %+v, %v, want it to name seedbox", res, err)
		}
		if got := w.Courier.Received(path); len(got) != 1 || got[0].Stop != 30*time.Second {
			t.Fatalf("the script ran %+v, want once with 30 s to run", got)
		}
	})
}
