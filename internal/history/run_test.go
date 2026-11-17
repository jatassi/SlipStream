package history_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

// quietOutbox owes nothing and was told nothing.
type quietOutbox struct{}

func (quietOutbox) Queue(context.Context, *storage.Tx[storage.Library], history.Captured) error {
	return nil
}

func (quietOutbox) Owed(context.Context, storage.Reader[storage.Library], []history.EventID) (map[history.EventID]bool, error) {
	return nil, nil
}

func (quietOutbox) Told(context.Context, storage.Reader[storage.Library], string) (history.Told, bool, error) {
	return history.Told{}, false, nil
}

// published counts history.changed.
type published struct {
	n atomic.Int64
}

func (p *published) Publish(context.Context, history.Changed) { p.n.Add(1) }

type running struct {
	lib  *storage.DB[storage.Library]
	h    *history.Service
	rec  entry.Recorder
	sent *published
}

func start(t *testing.T) running {
	t.Helper()
	lib := storagetest.Library(t, history.Schema())
	sent := &published{}
	h := history.New(history.Config{
		Library: lib, Catalog: entry.NewCatalog(), Outbox: quietOutbox{}, Changed: sent, Batch: 10, Log: slog.New(slog.DiscardHandler),
	})
	rec := h.Recorder("pings", []entry.Entry{notificationstest.PingRow})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	synctest.Wait()
	return running{lib: lib, h: h, rec: rec, sent: sent}
}

var errRolledBack = errors.New("rolled back")

// record records a Ping in a Write that commits unless rollBack.
func (r running) record(t *testing.T, rollBack bool) {
	t.Helper()
	err := r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := notificationstest.PingRow.Record(ctx, tx, r.rec, notificationstest.Ping{Name: "one"}); err != nil {
			return err
		}
		if rollBack {
			return errRolledBack
		}
		return nil
	})
	if !errors.Is(err, errRolledBack) && err != nil {
		t.Fatal(err)
	}
}

func (r running) wantPublished(t *testing.T, want int64, after string) {
	t.Helper()
	time.Sleep(time.Hour)
	synctest.Wait()
	if got := r.sent.n.Load(); got != want {
		t.Fatalf("after %s history.changed was published %d times, want %d", after, got, want)
	}
}

func TestHistoryChangedIsPublishedOncePerBurstAfterItsCommits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := start(t)
		for range 5 {
			r.record(t, false)
			time.Sleep(100 * time.Millisecond)
		}
		r.wantPublished(t, 1, "a burst of five captures")
		r.record(t, true)
		r.wantPublished(t, 1, "a rolled-back capture")
	})
}

func TestHistoryChangedIsPublishedAfterASweepThatDeletedEntries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := start(t)
		r.record(t, false)
		r.wantPublished(t, 1, "a capture")
		sweep := housekeeping.Task(housekeeping.Config{Keep: r.h, Batch: 10, Steps: []housekeeping.Step{r.h.Retention()}})
		if err := sweep.Run(t.Context(), tasks.Unit{Key: "history"}, &tasks.Progress{}); err != nil {
			t.Fatal(err)
		}
		r.wantPublished(t, 1, "a sweep that deleted nothing")
		time.Sleep(history.KeepFor.Default)
		if err := sweep.Run(t.Context(), tasks.Unit{Key: "history"}, &tasks.Progress{}); err != nil {
			t.Fatal(err)
		}
		r.wantPublished(t, 2, "a sweep that deleted the entry")
	})
}

func TestHistoryChangedIsPublishedAfterAClear(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := start(t)
		r.record(t, false)
		r.wantPublished(t, 1, "a capture")
		if err := r.h.Clear(t.Context(), identity.Admin{}); err != nil {
			t.Fatal(err)
		}
		r.wantPublished(t, 2, "a Clear")
	})
}
