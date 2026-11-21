package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Carry names one table CarryFrom copies, and the column it matches rows
// by, which is the table's primary key or unique.
type Carry struct{ Table, Key string }

// CarryFrom makes each table in tx hold exactly the rows the library.db at
// from holds: it deletes the rows whose key from lacks, then upserts every
// row of from by key, so a row kept on both sides keeps every reference to
// it. Columns are read from both files, and a table whose columns differ
// is refused with nothing written.
//
// from is read on its own read-only connection first, and its rows sit in
// memory, since ATTACH can't run inside the Write. It suits small tables.
func CarryFrom(ctx context.Context, tx *Tx[Library], from string, tables []Carry) (err error) {
	p := policyOf[Library]()
	conn, err := sql.Open("sqlite", dsn(from, p, unshared, roleInspect))
	if err != nil {
		return fmt.Errorf("carry from %s: %w", from, err)
	}
	defer func() { err = errors.Join(err, closeConn(p, conn)) }()
	conn.SetMaxOpenConns(1)
	for _, c := range tables {
		if err := carry(ctx, tx, conn, c); err != nil {
			return fmt.Errorf("carry %s from %s: %w", c.Table, from, err)
		}
	}
	return nil
}

func carry(ctx context.Context, tx *Tx[Library], from *sql.DB, c Carry) error {
	cols, err := columns(ctx, from, c.Table)
	if err != nil {
		return err
	}
	here, err := columns(ctx, tx, c.Table)
	if err != nil {
		return err
	}
	if !slices.Equal(cols, here) {
		return fmt.Errorf("the columns differ: %s there, %s here", strings.Join(cols, ", "), strings.Join(here, ", "))
	}
	names := make([]string, len(cols))
	for i, col := range cols {
		names[i], _, _ = strings.Cut(col, " ")
	}
	key := slices.Index(names, c.Key)
	if key < 0 {
		return fmt.Errorf("no column %s", c.Key)
	}
	rows, err := readRows(ctx, from, "SELECT "+quoted(names)+" FROM "+quote(c.Table), len(names))
	if err != nil {
		return err
	}
	kept := map[string]bool{}
	for _, row := range rows {
		kept[keyOf(row[key])] = true
	}
	present, err := readRows(ctx, tx, "SELECT "+quote(c.Key)+" FROM "+quote(c.Table), 1)
	if err != nil {
		return err
	}
	for _, row := range present {
		if kept[keyOf(row[0])] {
			continue
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+quote(c.Table)+" WHERE "+quote(c.Key)+" = ?", row[0]); err != nil {
			return err
		}
	}
	upsert := upsertStatement(c, names)
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, upsert, row...); err != nil {
			return err
		}
	}
	return nil
}

func upsertStatement(c Carry, names []string) string {
	var set []string
	for _, n := range names {
		if n != c.Key {
			set = append(set, quote(n)+" = excluded."+quote(n))
		}
	}
	conflict := "DO NOTHING"
	if len(set) > 0 {
		conflict = "DO UPDATE SET " + strings.Join(set, ", ")
	}
	params := strings.TrimSuffix(strings.Repeat("?, ", len(names)), ", ")
	return "INSERT INTO " + quote(c.Table) + " (" + quoted(names) + ") VALUES (" + params + ") ON CONFLICT (" + quote(c.Key) + ") " + conflict
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// columns is each column of table as "name TYPE", in table order.
func columns(ctx context.Context, q queryer, table string) ([]string, error) {
	rows, err := readRows(ctx, q, "SELECT name, type FROM pragma_table_info(?) ORDER BY cid", 2, table)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no table %s", table)
	}
	cols := make([]string, len(rows))
	for i, r := range rows {
		cols[i] = fmt.Sprintf("%s %s", r[0], r[1])
	}
	return cols, nil
}

func readRows(ctx context.Context, q queryer, query string, width int, args ...any) ([][]any, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var out [][]any
	for rows.Next() {
		row := make([]any, width)
		ptrs := make([]any, width)
		for i := range row {
			ptrs[i] = &row[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("read: %w", err)
		}
		out = append(out, row)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return out, nil
}

func keyOf(v any) string {
	switch v := v.(type) {
	case int64:
		return "i" + strconv.FormatInt(v, 10)
	case []byte:
		return "b" + string(v)
	case string:
		return "s" + v
	default:
		return fmt.Sprintf("%T%v", v, v)
	}
}

func quote(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func quoted(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = quote(n)
	}
	return strings.Join(q, ", ")
}
