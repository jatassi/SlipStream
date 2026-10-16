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

// Reader is F read through the *DB outside a Write, the *Tx inside one, or
// the snapshot inside a Read.
// Its four exported methods exist for sqlc's generated DBTX and nothing else.
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
	_ Reader[Library] = (*snapshot[Library])(nil)
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
// and a Write on a file earlier in write order (library.db inside local.db) panics
// with
//
//	storage: Write on library.db inside Write on local.db deadlocks; write library.db first
//
// A Write on local.db inside a library.db Write is allowed.
func (d *DB[F]) Write(ctx context.Context, fn func(ctx context.Context, tx *Tx[F]) error) error {
	ctx = claimWriter[F](ctx)
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

// Read runs fn on one reader connection inside one transaction, so every
// query fn makes reads the same snapshot. Inside a Write on F it panics, as
// any read through the DB does.
func (d *DB[F]) Read(ctx context.Context, fn func(ctx context.Context, r Reader[F]) error) error {
	mustNotHold[F](ctx)
	sqlTx, err := d.read.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin read: %w", policyOf[F]().name, err)
	}
	defer func() { _ = sqlTx.Rollback() }()
	return fn(ctx, &snapshot[F]{Tx[F]{tx: sqlTx}})
}

// Optimize runs PRAGMA optimize on the file's writer connection, which
// refreshes the statistics SQLite's query planner reads where they have
// gone stale. Inside a Write on F it panics as a nested Write does.
func (d *DB[F]) Optimize(ctx context.Context) error {
	ctx = claimWriter[F](ctx)
	if _, err := d.write.ExecContext(ctx, "PRAGMA optimize"); err != nil {
		return fmt.Errorf("%s: optimize: %w", policyOf[F]().name, err)
	}
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

// snapshot is the Reader inside a Read: a Tx on a reader connection that
// refuses writes.
type snapshot[F File] struct{ Tx[F] }

func (*snapshot[F]) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	panic("storage: write inside Read; take a *storage.Tx")
}

type holdingKey struct{}

type holding [fileCount]string

func heldIn(ctx context.Context) holding {
	h, _ := ctx.Value(holdingKey{}).(holding)
	return h
}

func (h holding) holds(writeOrder int) bool { return h[writeOrder] != "" }

func (h holding) heldAfter(writeOrder int) (string, bool) {
	for _, held := range h[writeOrder+1:] {
		if held != "" {
			return held, true
		}
	}
	return "", false
}

func claimWriter[F File](ctx context.Context) context.Context {
	p := policyOf[F]()
	h := heldIn(ctx)
	if h.holds(p.writeOrder) {
		panic("storage: Write inside Write on " + p.name + " deadlocks; pass the Tx")
	}
	if held, ok := h.heldAfter(p.writeOrder); ok {
		panic("storage: Write on " + p.name + " inside Write on " + held + " deadlocks; write " + p.name + " first")
	}
	h[p.writeOrder] = p.name
	return context.WithValue(ctx, holdingKey{}, h)
}

func mustNotHold[F File](ctx context.Context) {
	if heldIn(ctx).holds(policyOf[F]().writeOrder) {
		panic("storage: read through the Tx inside Write")
	}
}

// InWrite reports whether ctx is inside a Write on either file.
func InWrite(ctx context.Context) bool { return heldIn(ctx) != holding{} }
