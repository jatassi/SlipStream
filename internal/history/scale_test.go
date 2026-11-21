//go:build scale

package history_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

const scaleEntries = 200000

const budget = 100 * time.Millisecond

// TestScaleHistory seeds 200,000 entries five minutes apart, ending now: a
// rare kind under a second type every 1,000, a Health entry every 1,000, and
// a Work on 300. It times each History page, the Summary, a prune and a Clear.
func TestScaleHistory(t *testing.T) {
	lib := storagetest.Library(t, history.Schema())
	h := history.New(history.Config{
		Library: lib, Catalog: entry.NewCatalog(), Outbox: quietOutbox{}, Changed: &published{}, Batch: 1000, Log: slog.New(slog.DiscardHandler),
	})
	h.Recorder("noter", []entry.Entry{fetchedRow, refusedRow, landedRow, leftRow})
	busy := works.WorkID{UUID: storage.NewUUID()}
	now := time.Now()
	seed(t, lib, now, busy)

	for _, f := range []history.Filter{
		{},
		{Type: "films"},
		{Groups: []entry.Group{entry.Grabs}},
		{Groups: []entry.Group{entry.Health}},
		{Kinds: []entry.Kind{refusedRow.Kind}},
		{Type: "shows", Kinds: []entry.Kind{refusedRow.Kind}},
		{Since: now.Add(-12 * time.Minute)},
		{Work: busy},
		{Work: busy, Type: "shows", Kinds: []entry.Kind{refusedRow.Kind}},
	} {
		start := time.Now()
		p, err := h.Page(t.Context(), identity.Admin{}, f, 0, 50)
		took := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("a page of %+v: %d entries in %s", f, len(p.Entries), took)
		if took > budget {
			t.Errorf("a page of %+v took %s, over %s", f, took, budget)
		}
	}

	start := time.Now()
	if _, err := h.Summary(t.Context(), identity.Admin{}, now.Add(-7*24*time.Hour), 0); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > budget {
		t.Errorf("the Summary took %s, over %s", took, budget)
	} else {
		t.Logf("the Summary: %s", took)
	}

	start = time.Now()
	task := housekeeping.Task(housekeeping.Config{Reconciling: func(context.Context) (bool, error) { return false, nil }, Keep: h, Batch: 1000, Steps: []housekeeping.Step{h.Retention()}})
	if err := task.Run(t.Context(), tasks.Unit{Key: "history"}, &tasks.Progress{}); err != nil {
		t.Fatal(err)
	}
	t.Logf("a full prune to %d entries: %s", left(t, lib), time.Since(start))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = h.Run(ctx) }()
	start = time.Now()
	if err := h.Clear(t.Context(), identity.Admin{}); err != nil {
		t.Fatal(err)
	}
	for left(t, lib) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("a full Clear: %s", time.Since(start))
}

func seed(t *testing.T, lib *storage.DB[storage.Library], now time.Time, busy works.WorkID) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?1)
				INSERT INTO history_event (id, kind, at, content_type, subject)
				SELECT i,
					CASE WHEN i % 1000 = 0 THEN ?2 WHEN i % 1000 = 500 THEN ?3 WHEN i % 7 = 0 THEN ?4 ELSE ?5 END,
					?6 - (?1 - i) * 300000000,
					CASE WHEN i % 1000 = 0 THEN 'shows' WHEN i % 1000 = 500 THEN NULL ELSE 'films' END,
					'{"name":"entry"}'
				FROM n`, []any{scaleEntries, refusedRow.Kind, history.NewItemsRow.Kind, landedRow.Kind, fetchedRow.Kind, now.UnixMicro()}},
			{`INSERT INTO history_work (event_id, work_id)
				SELECT id, CASE WHEN id % 666 = 0 THEN ? ELSE CAST(printf('%016d', id % 20000) AS BLOB) END
				FROM history_event WHERE content_type IS NOT NULL`, []any{busy}},
			{`INSERT INTO history_item (event_id, key, level, subject, text, since_at)
				SELECT id, 'library.disk' || (id % 7), 'warn', 'Disk', 'It is low.', at
				FROM history_event WHERE content_type IS NULL`, nil},
			{`ANALYZE`, nil},
		} {
			if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func left(t *testing.T, lib *storage.DB[storage.Library]) int {
	t.Helper()
	var n int
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT COUNT(*) FROM history_event`).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
