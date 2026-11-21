package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Snapshot writes a copy of the file to dst, which must not exist, with
// VACUUM INTO on a short-lived connection of its own, so Writes keep
// committing while it runs. Then the writer truncates the WAL that grew
// meanwhile, as Optimize claims it. Inside a Write on F it panics as a
// nested Write does.
func (d *DB[F]) Snapshot(ctx context.Context, dst string) error {
	ctx = claimWriter[F](ctx)
	p := policyOf[F]()
	if err := snapshotPath(ctx, d.path, p, d.timings, dst); err != nil {
		return err
	}
	if _, err := d.write.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return fmt.Errorf("%s: checkpoint after the snapshot: %w", p.name, err)
	}
	return nil
}

// unshared is the Timings of a connection to a file no other connection
// opens, which never waits on a lock.
var unshared = Timings{}

// SnapshotFile is Snapshot of a library.db at path that no Node has open.
func SnapshotFile(ctx context.Context, path, dst string) error {
	return snapshotPath(ctx, path, policyOf[Library](), unshared, dst)
}

func snapshotPath(ctx context.Context, path string, p policy, t Timings, dst string) (err error) {
	conn, err := sql.Open("sqlite", dsn(path, p, t, roleSnapshot))
	if err != nil {
		return fmt.Errorf("%s: open for a snapshot: %w", p.name, err)
	}
	defer func() { err = errors.Join(err, closeConn(p, conn)) }()
	conn.SetMaxOpenConns(1)
	return vacuumInto(ctx, conn, p, dst)
}

func vacuumInto(ctx context.Context, conn *sql.DB, p policy, dst string) error {
	if _, err := conn.ExecContext(ctx, "VACUUM INTO ?", dst); err != nil {
		return fmt.Errorf("%s: snapshot to %s: %w", p.name, dst, err)
	}
	return nil
}

func closeConn(p policy, conn *sql.DB) error {
	if err := conn.Close(); err != nil {
		return fmt.Errorf("%s: close: %w", p.name, err)
	}
	return nil
}

// Estimate is how many bytes a snapshot of the file takes: its pages less
// the free ones, which VACUUM INTO leaves out.
func (d *DB[F]) Estimate(ctx context.Context) (int64, error) {
	mustNotHold[F](ctx)
	return estimate(ctx, d.read, policyOf[F]())
}

// Snapshot is DB.Snapshot on the migration connection, before any
// migration runs.
func (u *Unmigrated) Snapshot(ctx context.Context, dst string) error {
	return vacuumInto(ctx, u.conn, policyOf[Library](), dst)
}

// Estimate is DB.Estimate before any migration runs.
func (u *Unmigrated) Estimate(ctx context.Context) (int64, error) {
	return estimate(ctx, u.conn, policyOf[Library]())
}

// SecretCheck is the secret set's check row, id 1 of secret_check, which
// library.key must open. The statement is fixed: schema 1 of the secret
// set made the table, and no later schema may change it, since the hook
// reads it before any migration. ok is false when the row is absent.
func (u *Unmigrated) SecretCheck(ctx context.Context) (value []byte, ok bool, err error) {
	return secretCheck(ctx, u.conn)
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func secretCheck(ctx context.Context, q queryRower) (value []byte, ok bool, err error) {
	err = q.QueryRowContext(ctx, "SELECT value FROM secret_check WHERE id = 1").Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read the secret check row: %w", err)
	}
	return value, true, nil
}

func estimate(ctx context.Context, q queryRower, p policy) (int64, error) {
	var n int64
	err := q.QueryRowContext(ctx, "SELECT (c.page_count - f.freelist_count) * s.page_size FROM pragma_page_count() AS c, pragma_freelist_count() AS f, pragma_page_size() AS s").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("%s: estimate a snapshot: %w", p.name, err)
	}
	return n, nil
}
