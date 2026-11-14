package storage

import (
	"context"
	"database/sql"
	"errors"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Failed is whether err is the database or the call failing rather than
// what a Write wrote: the call ended, its transaction or connection is
// gone, or SQLite couldn't lock, read, write or trust the file. The same
// Write may succeed when run again. A constraint, a type mismatch or a
// statement SQLite rejects fails the same way every time, so it isn't.
func Failed(err error) bool {
	for _, e := range []error{context.Canceled, context.DeadlineExceeded, sql.ErrTxDone, sql.ErrConnDone} {
		if errors.Is(err, e) {
			return true
		}
	}
	var serr *sqlite.Error
	if !errors.As(err, &serr) {
		return false
	}
	switch serr.Code() & 0xff {
	case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_LOCKED, sqlite3.SQLITE_NOMEM, sqlite3.SQLITE_READONLY,
		sqlite3.SQLITE_INTERRUPT, sqlite3.SQLITE_IOERR, sqlite3.SQLITE_CORRUPT, sqlite3.SQLITE_FULL,
		sqlite3.SQLITE_CANTOPEN, sqlite3.SQLITE_PROTOCOL, sqlite3.SQLITE_NOTADB:
		return true
	}
	return false
}
