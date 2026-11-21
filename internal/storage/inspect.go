package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ErrDamaged is a library.db that fails integrity_check or isn't a SQLite
// database at all.
var ErrDamaged = errors.New("the library database is damaged")

// Inspected is what a library.db that isn't open says about itself.
type Inspected struct {
	Versions Versions
	// Plan is what opening it with today's schemas would apply.
	Plan Plan
	// Check is the secret set's check row, nil when the row is absent.
	Check []byte
}

// Inspect opens the library.db at path read-only, runs integrity_check and
// reads its versions and check row. It never writes the file. A file
// holding a newer schema or a set schemas lacks is a *RefusedError naming
// the set and its version.
func Inspect(ctx context.Context, path string, schemas []Schema[Library]) (_ Inspected, err error) {
	p := policyOf[Library]()
	conn, err := sql.Open("sqlite", dsn(path, p, unshared, roleInspect))
	if err != nil {
		return Inspected{}, fmt.Errorf("%s: open to inspect: %w", p.name, err)
	}
	defer func() { err = errors.Join(err, closeConn(p, conn)) }()
	conn.SetMaxOpenConns(1)
	if err := integrityCheck(ctx, conn); err != nil {
		return Inspected{}, err
	}
	held, err := readVersions(ctx, conn)
	if err != nil {
		return Inspected{}, damagedOr(err)
	}
	plan, err := Compare(held, schemas)
	if err != nil {
		return Inspected{}, &RefusedError{Err: err}
	}
	check, _, err := secretCheck(ctx, conn)
	if err != nil {
		return Inspected{}, damagedOr(err)
	}
	return Inspected{Versions: held, Plan: plan, Check: check}, nil
}

func integrityCheck(ctx context.Context, conn *sql.DB) error {
	rows, err := conn.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return damagedOr(err)
	}
	var found []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			_ = rows.Close()
			return damagedOr(err)
		}
		found = append(found, line)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return damagedOr(err)
	}
	if len(found) != 1 || found[0] != "ok" {
		return fmt.Errorf("%w: %s", ErrDamaged, strings.Join(found, "; "))
	}
	return nil
}

// damagedOr is err as ErrDamaged when SQLite found the file corrupt or not
// a database.
func damagedOr(err error) error {
	var serr *sqlite.Error
	if errors.As(err, &serr) {
		switch serr.Code() & 0xff {
		case sqlite3.SQLITE_NOTADB, sqlite3.SQLITE_CORRUPT:
			return fmt.Errorf("%w: %w", ErrDamaged, err)
		}
	}
	return fmt.Errorf("%s: inspect: %w", LibraryFile, err)
}

// Candidate is a staged library.db, migrated to today's schemas and open
// for the reads a restore's confirm step makes with today's queries.
type Candidate struct {
	DB *DB[Library]
	// Fenced is whether the staged library holds a nodes_fence row: a
	// Backup of a Controller that was fenced, whose fence secret the
	// restore must replace.
	Fenced bool
}

// OpenCandidate migrates the library.db at path with no pre-migration
// hook, since the file isn't live, and opens it. Its schemas hold the
// nodes set. Closing the DB removes nothing.
func OpenCandidate(ctx context.Context, path string, schemas []Schema[Library], t Timings, log *slog.Logger) (*Candidate, error) {
	db, err := open(ctx, path, schemas, nil, t, log)
	if err != nil {
		return nil, err
	}
	var fenced bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM nodes_fence)").Scan(&fenced); err != nil {
		return nil, errors.Join(fmt.Errorf("%s: read the fence: %w", LibraryFile, err), db.Close())
	}
	return &Candidate{DB: db, Fenced: fenced}, nil
}
