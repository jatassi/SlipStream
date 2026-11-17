package notifications_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
)

func unread(t *testing.T, f notifications.Feed) int {
	t.Helper()
	if f.Unread == nil {
		t.Fatal("an Admin's feed has no unread count")
	}
	return *f.Unread
}

func seen(t *testing.T, w *world, by identity.Admin, through notifications.Seq) {
	t.Helper()
	if err := w.Service.Seen(t.Context(), by, through); err != nil {
		t.Fatal(err)
	}
}

func TestAnItemShownAfterItsWaitIsUnreadPastAMovedCursor(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		address, err := identity.ParseAddress("https://media.example.com")
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Identity.SetAddress(t.Context(), identity.Admin{}, address); err != nil {
			t.Fatal(err)
		}
		sam := w.Admin("sam@example.test")
		w.Advance(time.Minute)
		w.Record(notificationstest.PingRow, ping("a"))
		w.Transitions(disk("d", health.Fail))
		w.Record(notificationstest.PingRow, ping("b"))
		f := w.Feed(sam)
		seen(t, w, sam, f.Rows[0].Seq)
		if f := w.Feed(sam); unread(t, f) != 0 {
			t.Fatalf("after Seen %d rows are unread", unread(t, f))
		}
		w.Advance(wait)
		f = w.Feed(sam)
		if unread(t, f) != 1 || len(f.Rows) != 3 || f.Rows[0].Title != "New Health item for Disk d is almost full" || f.Rows[0].Read || !f.Rows[1].Read {
			t.Fatalf("after the wait the feed reads %+v with %d unread, want the item unread on top", f.Rows, unread(t, f))
		}
		if f.Rows[1].Link != "https://media.example.com/pings/b" {
			t.Fatalf("a row links to %q, want the path on the Address", f.Rows[1].Link)
		}
	})
}

func TestEachAdminHasTheirOwnCursor(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		sam, lee := w.Admin("sam@example.test"), w.Admin("lee@example.test")
		w.Advance(time.Minute)
		w.Record(notificationstest.PingRow, ping("a"), ping("b"))
		w.Advance(time.Minute)
		kim := w.Admin("kim@example.test")
		if f := w.Feed(kim); unread(t, f) != 0 || !f.Rows[0].Read {
			t.Fatalf("a new Admin's feed reads %+v with %d unread, want nothing unread", f.Rows, unread(t, f))
		}
		seen(t, w, sam, 1<<62)
		if n, m := unread(t, w.Feed(sam)), unread(t, w.Feed(lee)); n != 0 || m != 2 {
			t.Fatalf("after Sam read, Sam has %d unread and Lee %d, want 0 and 2", n, m)
		}
		key := identity.Admin{}
		if f := w.Feed(key); f.Unread != nil || len(f.Rows) != 2 {
			t.Fatalf("the API key's feed reads %+v with unread %v, want rows and no count", f.Rows, f.Unread)
		}
		if err := w.Service.Seen(t.Context(), key, 1); !errors.Is(err, notifications.ErrNeedsAccount) {
			t.Fatalf("Seen with the API key = %v, want ErrNeedsAccount", err)
		}
	})
}

func TestAFeedRowKeepsTheItemsItShowedAfterTheWarningsSwitchChanges(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		sam := w.Admin("sam@example.test")
		if _, err := w.Service.SetBell(t.Context(), nil, ptr(true)); err != nil {
			t.Fatal(err)
		}
		w.Transitions(disk("c", health.Warn), disk("d", health.Fail))
		w.Advance(wait)
		before := w.Feed(sam).Rows
		if len(before) != 1 || before[0].Title != "2 new Health items" {
			t.Fatalf("with warnings on the bell shows %+v", before)
		}
		if _, err := w.Service.SetBell(t.Context(), nil, ptr(false)); err != nil {
			t.Fatal(err)
		}
		w.Transitions(disk("e", health.Warn), disk("f", health.Fail))
		w.Advance(wait)
		after := w.Feed(sam).Rows
		if len(after) != 2 || after[1] != before[0] || after[0].Title != "New Health item for Disk f is almost full" {
			t.Fatalf("after warnings went off the bell shows %+v, want the old row as it was and the new one without the Warning", after)
		}
	})
}
