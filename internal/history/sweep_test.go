package history_test

import (
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/historytest"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
)

type world = notificationstest.World

func run(t *testing.T, f func(t *testing.T, w *world)) {
	synctest.Test(t, func(t *testing.T) { f(t, notificationstest.New(t, notifications.DefaultTimings())) })
}

// old is past Keep history for's default.
var old = history.KeepFor.Default + 24*time.Hour

func names(t *testing.T, w *world) []string {
	t.Helper()
	var out []string
	for _, e := range historytest.Entries(t, w.Lib) {
		out = append(out, string(e.Kind)+" "+e.Subject)
	}
	return out
}

// stuck connects a Hook for events whose first send fails, so what it
// is sent stays due until the dispatcher tries again.
func stuck(w *world, events ...entry.Kind) {
	w.Connect("stuck", func(d *notifications.Draft) { d.Events = events })
	w.Courier.Queue(notificationstest.URL("stuck"), notificationstest.Reply{Status: http.StatusServiceUnavailable})
}

func TestAnEntryOlderThanKeepHistoryForGoesWithItsBellRow(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		admin := w.Admin("sam@example.test")
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "old"})
		w.Advance(old)
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "new"})
		w.Housekeeping()
		if got := names(t, w); len(got) != 1 || got[0] != `ping {"name":"new"}` {
			t.Fatalf("History holds %q, want the new Ping alone", got)
		}
		if f := w.Feed(admin); len(f.Rows) != 1 || f.Rows[0].Title != "Ping new" {
			t.Fatalf("the bell shows %+v, want the new Ping alone", f.Rows)
		}
	})
}

func TestAnEntryADeliveryStillOwesStaysUntilTheDeliveryEnds(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		stuck(w, "ping")
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "owed"}, notificationstest.Ping{Name: "owed too"})
		for range 3 {
			w.Record(notificationstest.PongRow, notificationstest.Ping{Name: "sent"})
		}
		w.Stop()
		w.Advance(old)
		w.Housekeeping()
		if got := names(t, w); len(got) != 2 {
			t.Fatalf("History holds %q, want the two Pings the stuck connection still owes, past a batch of them", got)
		}
		w.Restart()
		w.Advance(time.Hour)
		w.Housekeeping()
		if got := names(t, w); len(got) != 0 {
			t.Fatalf("after the deliveries ended History holds %q", got)
		}
	})
}

func TestTheEntryAWaitingRaiseRendersFromStaysWhileTheRaiseIsOwed(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		now := time.Now()
		d := health.Transition{
			Category: "storage", Key: "d", Kind: health.Raised, To: health.Warn,
			Subject: "Disk d is almost full", Text: "It has 1 GB left.", At: now, Since: now,
		}
		w.Transitions(d)
		w.Stop()
		w.Advance(time.Minute)
		worse := d
		worse.Kind, worse.From, worse.To, worse.At = health.Worse, health.Warn, health.Fail, time.Now()
		w.Transitions(worse)
		w.Advance(old)
		w.Housekeeping()
		if got := names(t, w); len(got) != 2 {
			t.Fatalf("History holds %q, want the raise and the Worse it renders from", got)
		}
	})
}
