// Package storagetest opens files for owners' tests and audits schema
// lists. internal/node's test runs the audit over the composition root's
// real lists, so a new owner's set is audited by adding its line there.
package storagetest

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

// timings is a literal: config-sections lets only internal/node/config.go
// call storage.DefaultTimings.
var timings = storage.Timings{BusyTimeout: 5 * time.Second}

// Library opens a fresh library.db in t.TempDir() with schemas migrated and
// closes it when t ends. An owner's test writes and reads through it with
// the owner's own exported functions.
func Library(t testing.TB, schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	t.Helper()
	return opened(t, func(ctx context.Context, dir string) (*storage.DB[storage.Library], error) {
		return openLibrary(ctx, filepath.Join(dir, storage.LibraryFile), schemas, logger(t))
	})
}

// Local is Library for local.db.
func Local(t testing.TB, schemas ...storage.Schema[storage.Local]) *storage.DB[storage.Local] {
	t.Helper()
	return opened(t, func(ctx context.Context, dir string) (*storage.DB[storage.Local], error) {
		return openLocal(ctx, filepath.Join(dir, storage.LocalFile), schemas, logger(t))
	})
}

func opened[F storage.File](t testing.TB, open func(ctx context.Context, dir string) (*storage.DB[F], error)) *storage.DB[F] {
	t.Helper()
	db, err := open(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	return db
}

func openLibrary(ctx context.Context, path string, schemas []storage.Schema[storage.Library], log *slog.Logger) (*storage.DB[storage.Library], error) {
	return storage.OpenLibrary(ctx, path, schemas, noBackup, timings, log)
}

func openLocal(ctx context.Context, path string, schemas []storage.Schema[storage.Local], log *slog.Logger) (*storage.DB[storage.Local], error) {
	return storage.OpenLocal(ctx, path, schemas, timings, log)
}

func noBackup(context.Context, *storage.Unmigrated, storage.Plan) error { return nil }

func logger(t testing.TB) *slog.Logger {
	return slog.New(slog.NewTextHandler(t.Output(), nil))
}
