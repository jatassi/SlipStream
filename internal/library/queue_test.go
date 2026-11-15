package library_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// asked is a Queue that writes each call it takes into the caller's
// Write, so a Write that rolls back takes its asks with it, and answers
// refuse.
type asked struct {
	lib    *storage.DB[storage.Library]
	calls  int
	refuse error
}

func newAsked(t *testing.T, lib *storage.DB[storage.Library]) *asked {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "CREATE TABLE asked (call INTEGER NOT NULL, trigger TEXT NOT NULL, work TEXT NOT NULL, items TEXT NOT NULL)")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return &asked{lib: lib}
}

func (q *asked) Queue(ctx context.Context, tx *storage.Tx[storage.Library], tr searches.Trigger, asks ...searches.Ask) error {
	q.calls++
	for _, a := range asks {
		var items []string
		for _, it := range a.Items {
			items = append(items, it.String())
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO asked (call, trigger, work, items) VALUES (?, ?, ?, ?)", q.calls, string(tr), a.Work.String(), strings.Join(items, " ")); err != nil {
			return err
		}
	}
	return q.refuse
}

// call is one committed Queue call.
type call struct {
	trigger searches.Trigger
	asks    []searches.Ask
}

// took is every call a committed Write queued, in order.
func (q *asked) took(t *testing.T) []call {
	t.Helper()
	var out []call
	err := q.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, "SELECT call, trigger, work, items FROM asked ORDER BY rowid")
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		last := 0
		for rows.Next() {
			var (
				n                    int
				trigger, work, items string
			)
			if err := rows.Scan(&n, &trigger, &work, &items); err != nil {
				return fmt.Errorf("read an ask: %w", err)
			}
			if n != last {
				out = append(out, call{trigger: searches.Trigger(trigger)})
				last = n
			}
			a := searches.Ask{Work: works.WorkID{UUID: mustUUID(t, work)}}
			for _, s := range strings.Fields(items) {
				a.Items = append(a.Items, works.ItemID{UUID: mustUUID(t, s)})
			}
			slices.SortFunc(a.Items, func(x, y works.ItemID) int { return strings.Compare(x.String(), y.String()) })
			out[len(out)-1].asks = append(out[len(out)-1].asks, a)
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func mustUUID(t *testing.T, s string) storage.UUID {
	t.Helper()
	u, err := storage.ParseUUID(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// idsOf is the ids of items, in the order took sorts them.
func idsOf(items ...works.Item) []works.ItemID {
	out := make([]works.ItemID, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	slices.SortFunc(out, func(x, y works.ItemID) int { return strings.Compare(x.String(), y.String()) })
	return out
}
