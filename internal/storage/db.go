package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type DB[F File] struct {
	write *sql.DB
	read  *sql.DB
}

type Tx[F File] struct {
	tx *sql.Tx
}

// The four exported methods exist for sqlc's generated DBTX and nothing
// else.
type Reader[F File] interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	file() F
}

var (
	_ Reader[Library] = (*DB[Library])(nil)
	_ Reader[Library] = (*Tx[Library])(nil)
)

// ErrNoRows is what a generated :one query returns when no row matches.
var ErrNoRows = sql.ErrNoRows

// Write runs fn in one transaction on the file's writer connection. It
// commits when fn returns nil and rolls back when fn returns an error or
// panics; fn's error comes back unchanged. Writers wait for the connection
// under ctx, in Go, never inside SQLite.
//
// fn gets a ctx marking that this call chain holds F's writer. With it, a
// nested Write on F panics with
//
//	storage: Write inside Write on library.db deadlocks; pass the Tx
//
// a read through any DB[F] panics with
//
//	storage: read through the Tx inside Write
//
// and a Write on a file of lower rank (library.db inside local.db) panics
// with
//
//	storage: Write on library.db inside Write on local.db deadlocks; write library.db first
//
// A Write on local.db inside a library.db Write is allowed.
func (d *DB[F]) Write(ctx context.Context, fn func(ctx context.Context, tx *Tx[F]) error) error {
	ctx = enter[F](ctx)
	name := policyOf[F]().name
	sqlTx, err := d.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin: %w", name, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = sqlTx.Rollback()
		}
	}()
	if err := fn(ctx, &Tx[F]{tx: sqlTx}); err != nil {
		return err
	}
	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", name, err)
	}
	committed = true
	return nil
}

func (d *DB[F]) Close() error {
	name := policyOf[F]().name
	var errs []error
	if err := d.read.Close(); err != nil {
		errs = append(errs, fmt.Errorf("%s: close readers: %w", name, err))
	}
	if err := d.write.Close(); err != nil {
		errs = append(errs, fmt.Errorf("%s: close writer: %w", name, err))
	}
	return errors.Join(errs...)
}

// QueryContext reads through the reader pool. Inside a Write on F it
// panics: the reader's snapshot would miss the Write's own rows.
func (d *DB[F]) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	mustNotHold[F](ctx)
	rows, err := d.read.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", policyOf[F]().name, err)
	}
	return rows, nil
}

func (d *DB[F]) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	mustNotHold[F](ctx)
	return d.read.QueryRowContext(ctx, query, args...)
}

func (d *DB[F]) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	mustNotHold[F](ctx)
	stmt, err := d.read.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", policyOf[F]().name, err)
	}
	return stmt, nil
}

// ExecContext panics: the DB reads, and writes go through a Tx. It exists
// only so *DB satisfies sqlc's DBTX; reaching it means an owner's read
// function ran a write query.
func (d *DB[F]) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	panic("storage: write through the DB outside Write; take a *storage.Tx")
}

func (d *DB[F]) file() F {
	var f F
	return f
}

func (t *Tx[F]) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", policyOf[F]().name, err)
	}
	return res, nil
}

func (t *Tx[F]) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	stmt, err := t.tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", policyOf[F]().name, err)
	}
	return stmt, nil
}

func (t *Tx[F]) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", policyOf[F]().name, err)
	}
	return rows, nil
}

func (t *Tx[F]) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *Tx[F]) file() F {
	var f F
	return f
}

type holdingKey struct{}

type holding [files]string

func enter[F File](ctx context.Context) context.Context {
	p := policyOf[F]()
	h, _ := ctx.Value(holdingKey{}).(holding)
	if h[p.rank] != "" {
		panic("storage: Write inside Write on " + p.name + " deadlocks; pass the Tx")
	}
	for _, held := range h[p.rank+1:] {
		if held != "" {
			panic("storage: Write on " + p.name + " inside Write on " + held + " deadlocks; write " + p.name + " first")
		}
	}
	h[p.rank] = p.name
	return context.WithValue(ctx, holdingKey{}, h)
}

func mustNotHold[F File](ctx context.Context) {
	if h, _ := ctx.Value(holdingKey{}).(holding); h[policyOf[F]().rank] != "" {
		panic("storage: read through the Tx inside Write")
	}
}
