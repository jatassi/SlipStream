package notifications_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestDeliveriesBeforeTheBackupAreGivenUp restores a library holding a
// message due after a failed send and one not yet released, both from
// before the Backup was taken, and one from after it: only the last is
// sent, since the old Controller most likely sent the others.
func TestDeliveriesBeforeTheBackupAreGivenUp(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		w.Connect("family")
		w.Courier.Queue(family, notificationstest.Reply{Status: http.StatusServiceUnavailable})
		w.Record(notificationstest.PingRow, ping("due"))
		w.Stop()
		w.Record(notificationstest.PingRow, ping("unreleased"))
		time.Sleep(time.Minute)
		takenAt := time.Now()
		time.Sleep(time.Minute)
		w.Record(notificationstest.PingRow, ping("after"))
		err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return notifications.GiveUpBefore(ctx, tx, takenAt)
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Restart()
		w.Advance(time.Hour)
		wantTitles(t, w, family, pings("due", "after")...)
		if given := deliveries(t, w, "given_up"); given != 1 {
			t.Fatalf("%d deliveries are given up, want the failed one", given)
		}
	})
}
