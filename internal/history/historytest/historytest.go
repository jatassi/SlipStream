// Package historytest reads the history set as stored, for tests that
// assert what was recorded before an API lists it.
package historytest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Entry is one history_event row with its Works and Health items.
type Entry struct {
	ID      history.EventID
	Kind    entry.Kind
	At      time.Time
	Type    works.TypeID
	Work    works.WorkID
	Grab    storage.UUID
	Import  storage.UUID
	Subject string
	Works   []works.WorkID
	Items   []history.Item
}

// Entries is every entry in id order.
func Entries(t testing.TB, lib *storage.DB[storage.Library]) []Entry {
	t.Helper()
	var out []Entry
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if out, err = events(ctx, r); err != nil {
			return err
		}
		at := make(map[history.EventID]int, len(out))
		ids := make([]history.EventID, len(out))
		for i, e := range out {
			at[e.ID], ids[i] = i, e.ID
		}
		if err := eachRow(ctx, r, "SELECT event_id, work_id FROM history_work ORDER BY event_id, work_id", func(rows *sql.Rows) error {
			var id history.EventID
			var work works.WorkID
			if err := rows.Scan(&id, &work); err != nil {
				return fmt.Errorf("read a history_work row: %w", err)
			}
			out[at[id]].Works = append(out[at[id]].Works, work)
			return nil
		}); err != nil {
			return err
		}
		items, err := history.Items(ctx, r, ids)
		for id, its := range items {
			out[at[id]].Items = its
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func events(ctx context.Context, r storage.Reader[storage.Library]) ([]Entry, error) {
	var out []Entry
	err := eachRow(ctx, r, `SELECT id, kind, at, COALESCE(content_type, ''), work_id, grab_id, import_id, subject
		FROM history_event ORDER BY id`, func(rows *sql.Rows) error {
		var e Entry
		var at storage.Time
		if err := rows.Scan(&e.ID, &e.Kind, &at, &e.Type, &e.Work, &e.Grab, &e.Import, &e.Subject); err != nil {
			return fmt.Errorf("read a history_event row: %w", err)
		}
		e.At = at.Time()
		out = append(out, e)
		return nil
	})
	return out, err
}

func eachRow(ctx context.Context, r storage.Reader[storage.Library], query string, scan func(*sql.Rows) error) error {
	rows, err := r.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read %q: %w", query, err)
	}
	return nil
}
