package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Explaining is r, which runs EXPLAIN QUERY PLAN on each query before it
// runs it and hands explained the query and its plan's lines. It exists
// for the plan tests that pin each read to an index.
func Explaining[F File](r Reader[F], explained func(query string, lines []string)) Reader[F] {
	return &explaining[F]{Reader: r, explained: explained}
}

type explaining[F File] struct {
	Reader[F]
	explained func(query string, lines []string)
}

func (e *explaining[F]) explain(ctx context.Context, query string, args []any) error {
	rows, err := e.Reader.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		return fmt.Errorf("explain %s: %w", query, err)
	}
	var lines []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			return errors.Join(fmt.Errorf("explain %s: %w", query, err), rows.Close())
		}
		lines = append(lines, detail)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return fmt.Errorf("explain %s: %w", query, err)
	}
	e.explained(query, lines)
	return nil
}

func (e *explaining[F]) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if err := e.explain(ctx, query, args); err != nil {
		return nil, err
	}
	return e.Reader.QueryContext(ctx, query, args...)
}

// QueryRowContext can't answer an explain's error, so it panics with it.
func (e *explaining[F]) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if err := e.explain(ctx, query, args); err != nil {
		panic(err)
	}
	return e.Reader.QueryRowContext(ctx, query, args...)
}
