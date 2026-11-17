// Package historytest reads the history set as stored, for tests that
// assert what was recorded before an API lists it.
package historytest

import (
	"context"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/internal/db"
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
		q := db.New(r)
		rows, err := q.ListAll(ctx)
		if err != nil {
			return err
		}
		at := make(map[history.EventID]int, len(rows))
		ids := make([]history.EventID, len(rows))
		for i, row := range rows {
			e := Entry{ID: row.ID, Kind: row.Kind, At: row.At.Time(), Work: row.WorkID, Grab: row.GrabID, Import: row.ImportID, Subject: row.Subject}
			if row.ContentType != nil {
				e.Type = *row.ContentType
			}
			out = append(out, e)
			at[row.ID], ids[i] = i, row.ID
		}
		ws, err := q.ListWorksOf(ctx, ids)
		if err != nil {
			return err
		}
		for _, w := range ws {
			out[at[w.EventID]].Works = append(out[at[w.EventID]].Works, w.WorkID)
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
