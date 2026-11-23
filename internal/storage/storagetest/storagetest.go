package storagetest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

var timings = storage.Timings{BusyTimeout: 5 * time.Second}

// Library opens a fresh library.db in t.TempDir() with schemas migrated and
// closes it when t ends.
func Library(t testing.TB, schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	t.Helper()
	return opened(t, storage.LibraryFile, schemas, openLibrary)
}

func Local(t testing.TB, schemas ...storage.Schema[storage.Local]) *storage.DB[storage.Local] {
	t.Helper()
	return opened(t, storage.LocalFile, schemas, openLocal)
}

func opened[F storage.File](t testing.TB, name string, schemas []storage.Schema[F], open opener[F]) *storage.DB[F] {
	t.Helper()
	file, err := imageOf(name, schemas, open)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, file, 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := open(t.Context(), path, schemas, logger(t))
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

// SeedLibrary writes a library.db at path that schemas have already
// migrated, unless path holds a file, for node.Test's SeedLibrary. It may
// run off the test goroutine, so it returns its error.
func SeedLibrary(path string, schemas []storage.Schema[storage.Library]) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("seed library.db: %w", err)
	}
	file, err := imageOf(storage.LibraryFile, schemas, openLibrary)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, file, 0o600); err != nil {
		return fmt.Errorf("seed library.db: %w", err)
	}
	return nil
}

// image is a file freshly migrated with schemas. Migrating costs seconds
// under -race, so each schema list migrates once per test binary and every
// later open starts from a copy of the file already at its target. A clean
// close checkpoints the WAL, so the main file holds everything.
type image struct {
	schemas any
	file    []byte
}

var images struct {
	mu   sync.Mutex
	list []image
}

func imageOf[F storage.File](name string, schemas []storage.Schema[F], open opener[F]) (file []byte, err error) {
	images.mu.Lock()
	defer images.mu.Unlock()
	for _, img := range images.list {
		if reflect.DeepEqual(img.schemas, schemas) {
			return img.file, nil
		}
	}
	dir, err := os.MkdirTemp("", "image")
	if err != nil {
		return nil, fmt.Errorf("migrate %s image: %w", name, err)
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	path := filepath.Join(dir, name)
	db, err := open(context.Background(), path, schemas, slog.New(slog.DiscardHandler))
	if err != nil {
		return nil, err
	}
	if err := db.Close(); err != nil {
		return nil, err
	}
	file, err = os.ReadFile(path) //nolint:gosec // G304: a file in our own temporary directory
	if err != nil {
		return nil, fmt.Errorf("migrate %s image: %w", name, err)
	}
	images.list = append(images.list, image{schemas: slices.Clone(schemas), file: file})
	return file, nil
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
