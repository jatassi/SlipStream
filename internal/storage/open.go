package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

type Timings struct {
	BusyTimeout time.Duration `yaml:"busy_timeout"`
}

func DefaultTimings() Timings { return Timings{BusyTimeout: 5 * time.Second} }

// RefusedError is a library.db the Node must not start on: a *SchemaError
// (newer schema or unknown set) or a file SQLite cannot read. internal/node
// maps it to ExitLibraryRefused. Its text is the wrapped error's, for an
// unreadable file:
//
//	library.db isn't a readable SQLite database. Restore a Backup
type RefusedError struct{ Err error }

func (e *RefusedError) Error() string { return e.Err.Error() }

func (e *RefusedError) Unwrap() error { return e.Err }

// OpenLibrary opens library.db at path and brings it to the latest version
// of every schema, in list order, before returning. A file holding a newer
// schema, an unknown set, or bytes SQLite cannot read is a *RefusedError.
// ctx cancels a migration in flight; a canceled migration's file stays at
// the last whole migration.
func OpenLibrary(ctx context.Context, path string, schemas []Schema[Library], beforeMigrate PreMigrationHook, t Timings, log *slog.Logger) (*DB[Library], error) {
	return open(ctx, path, schemas, beforeMigrate, t, log)
}

// OpenLocal opens local.db at path like OpenLibrary, with no hook. A file
// that is corrupt, fails quick_check or holds a schema Compare refuses is
// renamed to local.db.discarded (with its -wal and -shm), replacing any
// earlier one, and recreated empty, with one "local state discarded"
// warning. A migration that fails is still an error:
//
//	local.db migration <set> <version> failed: <error>. local.db is safe to delete
func OpenLocal(ctx context.Context, path string, schemas []Schema[Local], t Timings, log *slog.Logger) (*DB[Local], error) {
	return open(ctx, path, schemas, nil, t, log)
}

func open[F File](ctx context.Context, path string, schemas []Schema[F], hook PreMigrationHook, t Timings, log *slog.Logger) (*DB[F], error) {
	p := policyOf[F]()
	err := migrate(ctx, path, schemas, hook, t, log)
	if refused(err) && p.discardRefused {
		if derr := discard(path); derr != nil {
			return nil, derr
		}
		log.Warn("local state discarded", "file", p.name, "reason", err.Error())
		err = migrate(ctx, path, schemas, hook, t, log)
	}
	if refused(err) {
		return nil, &RefusedError{Err: err}
	}
	if err != nil {
		return nil, err
	}
	return connect[F](ctx, path, t)
}

func refused(err error) bool {
	var schema *SchemaError
	var corrupt *corruptError
	return errors.As(err, &schema) || errors.As(err, &corrupt)
}

// discard renames path and its -wal and -shm to path.discarded and its
// -wal and -shm, first removing any earlier discarded copy, so SQLite can
// still open the discarded file beside the new one.
func discard(path string) error {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		to := path + ".discarded" + suffix
		if err := os.Remove(to); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("discard %s: %w", filepath.Base(path), err)
		}
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		err := os.Rename(path+suffix, path+".discarded"+suffix)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("discard %s: %w", filepath.Base(path), err)
		}
	}
	return nil
}

func connect[F File](ctx context.Context, path string, t Timings) (*DB[F], error) {
	p := policyOf[F]()
	write, err := sql.Open("sqlite", dsn(path, p, t, roleWrite))
	if err != nil {
		return nil, fmt.Errorf("%s: open writer: %w", p.name, err)
	}
	write.SetMaxOpenConns(1)
	write.SetMaxIdleConns(1)
	if err := write.PingContext(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("%s: open writer: %w", p.name, err), write.Close())
	}
	read, err := sql.Open("sqlite", dsn(path, p, t, roleRead))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%s: open readers: %w", p.name, err), write.Close())
	}
	readers := runtime.GOMAXPROCS(0)
	read.SetMaxOpenConns(readers)
	read.SetMaxIdleConns(readers)
	return &DB[F]{write: write, read: read}, nil
}

type role int

const (
	roleMigrate role = iota
	roleWrite
	roleRead
)

// foreign_keys is off on the migration connection, because SQLite's table-rebuild
// procedure drops a table, and with foreign keys on that drop would
// cascade-delete child rows in other sets.
//
// synchronous is NORMAL on the migration connection, so a migration commits
// without a sync. Closing that connection, the file's only one, checkpoints
// and syncs once, so opening costs a few syncs however many migrations run.
func dsn(path string, p policy, t Timings, r role) string {
	busy := (t.BusyTimeout + time.Millisecond - 1) / time.Millisecond
	q := url.Values{}
	q.Add("_pragma", "busy_timeout("+strconv.FormatInt(int64(busy), 10)+")")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "reverse_unordered_selects(1)")
	switch r {
	case roleMigrate:
		q.Add("_pragma", "synchronous(NORMAL)")
		q.Add("_pragma", "foreign_keys(0)")
	case roleWrite:
		q.Add("_pragma", "synchronous("+p.synchronous+")")
		q.Add("_pragma", "foreign_keys(1)")
		q.Set("_txlock", "immediate")
	case roleRead:
		q.Add("_pragma", "synchronous("+p.synchronous+")")
		q.Add("_pragma", "foreign_keys(1)")
		q.Add("_pragma", "query_only(1)")
	}
	slashed := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" {
		slashed = "/" + slashed
	}
	return "file:" + (&url.URL{Path: slashed}).EscapedPath() + "?" + q.Encode()
}
