// Package storage is the mechanism behind every SlipStream table. It owns
// no tables.
//
// A Node keeps two SQLite files. library.db (Library) holds everything the
// Controller owns and a Backup copies. local.db (Local) holds Role-local
// runtime state any Node may lose. Every schema, database, transaction and
// reader is generic over one of the two, so code aimed at the wrong file
// does not compile.
//
// # Owning tables
//
// Copy internal/storage/example/clip. An owner is a package with:
//
//	schema/00001_<slug>.sql   forward-only goose migrations, no Down
//	queries/<name>.sql        sqlc queries; every :many and LIMIT has a top-level ORDER BY
//	sqlc.yaml                 output in internal/db, *.*_at -> storage.Time first
//	internal/db/ids.go        hand-written typed ids (type ClipID int64)
//	internal/db/*.go          sqlc output, importable only by the owner
//	<owner>.go                type file = storage.Library (the one line naming the file),
//	                          Schema(), and the typed functions callers use
//
// Run `go run ./cmd/release sqlc` after editing schema/ or queries/.
//
// Write functions take *storage.Tx[file]. Read functions take
// storage.Reader[file], which both the *DB and a *Tx satisfy, so one read
// function serves callers outside a Write and inside one. Both build the
// generated Queries with db.New(tx) or db.New(r) and never export it.
//
// The set name an owner passes to NewSchema names its goose_<set> version
// table and prefixes every table it creates. A set references only its own
// tables or tables of sets listed before it, and a foreign key into another
// set targets that table's id. storagetest's audit checks these rules.
//
// # Writing
//
// DB.Write runs fn in one BEGIN IMMEDIATE transaction on the file's single
// writer connection and commits when fn returns nil. Writers queue in Go,
// so no write sees SQLITE_BUSY and nobody writes a retry loop.
//
// Inside fn, use fn's ctx and tx:
//   - a Write on the same file panics (it would wait on itself forever);
//   - a read through the DB panics (its snapshot misses fn's own rows);
//     read through the tx instead;
//   - a Write on local.db is allowed, and commits before the library.db
//     Write that encloses it. The reverse order panics, because two
//     goroutines nesting in opposite orders deadlock.
//
// fn does no network or file IO: it holds the file's only writer for its
// whole run. Nothing checks this; it is the one rule written only here.
package storage
