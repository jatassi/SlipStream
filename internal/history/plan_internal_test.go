package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

const (
	planEntries = 20000
	rareKind    = entry.Kind("rare")
	rareType    = works.TypeID("rarer")
)

// skewed is a History of 20,000 entries after ANALYZE: almost all one kind
// under one type, a rare kind under a second type every 1,000, a Health
// entry every 400, and the Work answered on about 300.
func skewed(t *testing.T) (lib *storage.DB[storage.Library], busy works.WorkID) {
	t.Helper()
	lib = storagetest.Library(t, Schema())
	busy = works.WorkID{UUID: storage.NewUUID()}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
				INSERT INTO history_event (id, kind, at, content_type, subject)
				SELECT i,
					CASE WHEN i % 1000 = 0 THEN ? WHEN i % 400 = 7 THEN ? ELSE 'common' END,
					1700000000000000 + i * 60000000,
					CASE WHEN i % 1000 = 0 THEN ? WHEN i % 400 = 7 THEN NULL ELSE 'films' END,
					'{}'
				FROM n`, []any{planEntries, rareKind, NewItemsRow.Kind, rareType}},
			{`INSERT INTO history_work (event_id, work_id)
				SELECT id, CASE WHEN id % 66 = 1 THEN ? ELSE CAST(printf('%016d', id % 5000) AS BLOB) END
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
	return lib, busy
}

func TestEveryHistoryReadUsesItsIndexOnSkewedData(t *testing.T) {
	lib, busy := skewed(t)
	const floor, before, limit = EventID(planEntries / 2), EventID(planEntries - 50), 51
	at := storage.FromTime(time.UnixMicro(1700000000000000 + 15000*60000000))
	kindsOf := func(n int) string {
		ks := []entry.Kind{rareKind, "common", NewItemsRow.Kind, ClearedRow.Kind, "a", "b", "c", "d", "e"}[:n]
		b, err := json.Marshal(ks)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	shapes := []shape{
		{"ListIDs", []string{"INTEGER PRIMARY KEY"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListIDs(ctx, db.ListIDsParams{Floor: floor, Before: before, Limit: limit})
			return err
		}},
		{"ListKindIDs", []string{"history_event_kind"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListKindIDs(ctx, db.ListKindIDsParams{Kind: rareKind, Floor: floor, Before: before, Limit: limit})
			return err
		}},
		{"ListTypeKindIDs", []string{"history_event_type"}, func(ctx context.Context, q *db.Queries) error {
			typ := rareType
			_, err := q.ListTypeKindIDs(ctx, db.ListTypeKindIDsParams{ContentType: &typ, Kind: rareKind, Floor: floor, Before: before, Limit: limit})
			return err
		}},
		{"FirstIDFrom", []string{"history_event_at"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.FirstIDFrom(ctx, at)
			return ignoreNoRows(err)
		}},
		{"LastIDBefore", []string{"history_event_at"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.LastIDBefore(ctx, at)
			return ignoreNoRows(err)
		}},
		{"CountKindFrom", []string{"history_event_kind"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.CountKindFrom(ctx, db.CountKindFromParams{Kind: rareKind, Floor: floor})
			return err
		}},
		{"ListSweep", []string{"INTEGER PRIMARY KEY"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListSweep(ctx, db.ListSweepParams{After: floor, Ceiling: before, Limit: 500})
			return err
		}},
		{"NewestItem", []string{"history_item_key"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.NewestItem(ctx, "library.disk3")
			return err
		}},
		{"ListByIDs", []string{"INTEGER PRIMARY KEY"}, func(ctx context.Context, q *db.Queries) error {
			_, err := q.ListByIDs(ctx, []EventID{floor, floor + 1, before - 1})
			return err
		}},
	}
	for _, n := range []int{1, 3, 9} {
		for _, typed := range []bool{false, true} {
			var typ any
			if typed {
				typ = string(rareType)
			}
			shapes = append(shapes, shape{fmt.Sprintf("ListWorkIDs %d kinds typed %t", n, typed), []string{"history_work_work", "INTEGER PRIMARY KEY"},
				func(ctx context.Context, q *db.Queries) error {
					_, err := q.ListWorkIDs(ctx, db.ListWorkIDsParams{WorkID: busy, Floor: floor, Before: before, Kinds: kindsOf(n), ContentType: typ, Limit: limit})
					return err
				}})
		}
	}
	for _, s := range shapes {
		wantPlan(t, lib, s)
	}
}

// shape is one History read and the indexes it must read through.
type shape struct {
	name    string
	indexes []string
	read    func(ctx context.Context, q *db.Queries) error
}

func wantPlan(t *testing.T, lib *storage.DB[storage.Library], s shape) {
	t.Helper()
	var p storagetest.QueryPlan
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		plans := storagetest.Plan(t, r, func(r storage.Reader[storage.Library]) error { return s.read(ctx, db.New(r)) })
		if len(plans) != 1 {
			t.Fatalf("%s ran %d queries, want 1", s.name, len(plans))
		}
		p = plans[0]
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range s.indexes {
		if !uses(p, index) {
			t.Errorf("%s doesn't read through %s: %q", s.name, index, p.Lines)
		}
	}
	for _, l := range p.Lines {
		// The kinds a Work's read names are a JSON list of at most nine,
		// which json_each walks once.
		if (strings.Contains(l, "SCAN") && !strings.Contains(l, "SCAN json_each")) || strings.Contains(l, "USE TEMP B-TREE") {
			t.Errorf("%s plans %q", s.name, p.Lines)
		}
	}
}

func uses(p storagetest.QueryPlan, index string) bool {
	if index == "INTEGER PRIMARY KEY" {
		for _, l := range p.Lines {
			if strings.Contains(l, "USING INTEGER PRIMARY KEY") {
				return true
			}
		}
		return false
	}
	return p.Uses(index)
}

func ignoreNoRows(err error) error {
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	return err
}
