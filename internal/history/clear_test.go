package history_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
)

// sweeping runs w's History, which sweeps what a Clear left, until the
// test ends.
func sweeping(t *testing.T, w *world) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = w.History.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	synctest.Wait()
}

func clearAll(t *testing.T, w *world) {
	t.Helper()
	if err := w.History.Clear(t.Context(), identity.Admin{}); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

func listed(t *testing.T, w *world) []string {
	t.Helper()
	p, err := w.History.Page(t.Context(), identity.Admin{}, history.Filter{}, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range p.Entries {
		out = append(out, e.Message.Title)
	}
	return out
}

func TestAClearEmptiesHistoryAndTheBell(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		sweeping(t, w)
		admin := w.Admin("sam@example.test")
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "one"}, notificationstest.Ping{Name: "two"})
		w.Record(notificationstest.PongRow, notificationstest.Ping{Name: "three"})
		if f := w.Feed(admin); len(f.Rows) != 3 {
			t.Fatalf("the bell shows %d rows before the Clear, want 3", len(f.Rows))
		}
		clearAll(t, w)
		if got := listed(t, w); len(got) != 0 {
			t.Fatalf("after a Clear History lists %q", got)
		}
		if f := w.Feed(admin); len(f.Rows) != 0 {
			t.Fatalf("after a Clear the bell shows %+v", f.Rows)
		}
		if got := names(t, w); len(got) != 0 {
			t.Fatalf("after a Clear History holds %q", got)
		}
	})
}

func TestAnEntryADeliveryOwesLeavesHistoryAtOnceAndTheBellOnceSent(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		sweeping(t, w)
		admin := w.Admin("sam@example.test")
		stuck(w, "ping")
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "owed"})
		w.Record(notificationstest.PongRow, notificationstest.Ping{Name: "sent"})
		clearAll(t, w)
		if got := listed(t, w); len(got) != 0 {
			t.Fatalf("after a Clear History lists %q, want nothing, the owed entry included", got)
		}
		if f := w.Feed(admin); len(f.Rows) != 1 || f.Rows[0].Title != "Ping owed" {
			t.Fatalf("after a Clear the bell shows %+v, want the owed Ping alone", f.Rows)
		}
		w.Advance(2 * time.Hour)
		w.Housekeeping()
		if f := w.Feed(admin); len(f.Rows) != 0 {
			t.Fatalf("once its delivery ended the bell shows %+v", f.Rows)
		}
	})
}

func TestAClearThatRunStoppedBeforeSweepingFinishesWhenRunStarts(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		admin := w.Admin("sam@example.test")
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "one"}, notificationstest.Ping{Name: "two"})
		clearAll(t, w)
		if f := w.Feed(admin); len(f.Rows) != 2 {
			t.Fatalf("with no sweeper running the bell shows %d rows, want both still", len(f.Rows))
		}
		sweeping(t, w)
		if f := w.Feed(admin); len(f.Rows) != 0 {
			t.Fatalf("after History started the bell shows %+v", f.Rows)
		}
	})
}

func TestASecondClearIsTheSameAsOneAndALaterClearRaisesTheFloor(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		sweeping(t, w)
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "old"})
		clearAll(t, w)
		clearAll(t, w)
		if got := listed(t, w); len(got) != 0 {
			t.Fatalf("after two Clears History lists %q", got)
		}
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "new"})
		if got := listed(t, w); !slices.Equal(got, []string{"Ping new"}) {
			t.Fatalf("after the Clears History lists %q, want the new Ping", got)
		}
		clearAll(t, w)
		if got := listed(t, w); len(got) != 0 {
			t.Fatalf("after a later Clear History lists %q", got)
		}
	})
}

func TestAClearIsAnEntryThatIsGone(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Record(notificationstest.PingRow, notificationstest.Ping{Name: "one"})
		p, err := w.History.Page(t.Context(), identity.Admin{}, history.Filter{}, 0, 1)
		if err != nil || len(p.Entries) != 1 {
			t.Fatalf("History lists %+v (%v)", p, err)
		}
		clearAll(t, w)
		if _, err := w.History.One(t.Context(), identity.Admin{}, p.Entries[0].ID, time.UTC); !errors.Is(err, history.ErrGone) {
			t.Fatalf("a cleared entry read %v, want it gone", err)
		}
	})
}
