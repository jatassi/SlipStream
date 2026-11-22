package storage_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/clip"
	"github.com/jatassi/slipstream/internal/storage/example/shelf"
)

// keyed is the secret set before both example sets, as every real library
// lists it first.
func keyed(rest ...storage.Schema[storage.Library]) []storage.Schema[storage.Library] {
	return append([]storage.Schema[storage.Library]{secret.Schema()}, rest...)
}

// fence stands in for the nodes set, whose nodes_fence table OpenCandidate
// reads.
func fence() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("nodes", fstest.MapFS{
		"schema/00001_fence.sql": {Data: []byte("-- +goose Up\nCREATE TABLE nodes_fence (id INTEGER PRIMARY KEY CHECK (id = 1), state TEXT NOT NULL) STRICT;\n")},
	})
}

func bulk() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("bulk", fstest.MapFS{
		"schema/00001_bulk.sql": {Data: []byte("-- +goose Up\nCREATE TABLE bulk (id INTEGER PRIMARY KEY, body BLOB NOT NULL) STRICT;\n")},
	})
}

func run(t *testing.T, db *storage.DB[storage.Library], query string, args ...any) {
	t.Helper()
	if err := db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, r storage.Reader[storage.Library], table string) int64 {
	t.Helper()
	var n int64
	if err := r.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func reopen(t *testing.T, path string, schemas []storage.Schema[storage.Library]) *storage.DB[storage.Library] {
	t.Helper()
	return openLibrary(t, path, schemas, nil, slog.New(slog.DiscardHandler))
}

func TestASnapshotWhileTheWriterCommits(t *testing.T) {
	path := libraryPath(t)
	schemas := keyed(bulk())
	db := reopen(t, path, schemas)
	const rows, size = 50, 1 << 20
	for range rows {
		run(t, db, "INSERT INTO bulk (body) VALUES (randomblob(?))", size)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = reopen(t, path, schemas)
	if est, err := db.Estimate(t.Context()); err != nil || est < rows*size || est > rows*size*11/10 {
		t.Fatalf("estimate = %d, %v; want about %d", est, err, rows*size)
	}

	stop := make(chan struct{})
	var worst time.Duration
	var writes int
	var wg sync.WaitGroup
	wg.Go(func() {
		tick := time.NewTicker(10 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
			}
			began := time.Now()
			if err := db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := tx.ExecContext(ctx, "INSERT INTO bulk (body) VALUES (x'00')")
				return err
			}); err != nil {
				t.Error(err)
				return
			}
			worst = max(worst, time.Since(began))
			writes++
		}
	})
	dst := filepath.Join(t.TempDir(), "snapshot.db")
	began := time.Now()
	err := db.Snapshot(t.Context(), dst)
	took := time.Since(began)
	close(stop)
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if worst > took/2 || writes < 2 {
		t.Fatalf("%d Writes committed during the %s snapshot, the slowest in %s; want some, none past half the snapshot", writes, took, worst)
	}
	inspected, err := storage.Inspect(t.Context(), dst, schemas)
	if err != nil || len(inspected.Plan) != 0 {
		t.Fatalf("inspect the snapshot = %+v, %v; want a whole file at today's schemas", inspected, err)
	}
	snap := reopen(t, dst, schemas)
	if n := count(t, snap, "bulk"); n < rows {
		t.Fatalf("snapshot holds %d rows, want at least %d", n, rows)
	}
}

func TestASnapshotInsideAWritePanics(t *testing.T) {
	db := reopen(t, libraryPath(t), keyed())
	assertPanics(t, "storage: Write inside Write on library.db deadlocks; pass the Tx", func() {
		_ = db.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
			return db.Snapshot(ctx, filepath.Join(t.TempDir(), "snapshot.db"))
		})
	})
}

func TestSnapshotFileCopiesAClosedLibrary(t *testing.T) {
	path := libraryPath(t)
	db := reopen(t, path, keyed(shelf.Schema()))
	addShelf(t, db, "Films")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "snapshot.db")
	if err := storage.SnapshotFile(t.Context(), path, dst); err != nil {
		t.Fatal(err)
	}
	names, err := shelf.Names(t.Context(), reopen(t, dst, keyed(shelf.Schema())))
	if err != nil || !slices.Equal(names, []string{"Films"}) {
		t.Fatalf("snapshot names = %v, %v; want [Films]", names, err)
	}
}

func TestSecretCheckReadsSchemaOnesRow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, storage.LibraryFile)
	old := keyed(shelf.Schema(), clip.Schema().Through(1))
	db := reopen(t, path, old)
	if _, err := secret.Open(t.Context(), db, filepath.Join(dir, secret.KeyFile)); err != nil {
		t.Fatal(err)
	}
	var stored []byte
	if err := db.QueryRowContext(t.Context(), "SELECT value FROM secret_check").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	var got []byte
	var ok bool
	h := hookCalls{check: func(ctx context.Context, u *storage.Unmigrated) {
		var err error
		if got, ok, err = u.SecretCheck(ctx); err != nil {
			t.Error(err)
		}
	}}
	openLibrary(t, path, keyed(both()...), h.hook, slog.New(slog.DiscardHandler))
	if !ok || len(stored) == 0 || !slices.Equal(got, stored) {
		t.Fatalf("SecretCheck = %x, %v; want the stored row %x", got, ok, stored)
	}

	bare := libraryPath(t)
	migrated(t, bare, old...)
	h = hookCalls{check: func(ctx context.Context, u *storage.Unmigrated) {
		var err error
		if got, ok, err = u.SecretCheck(ctx); err != nil {
			t.Error(err)
		}
	}}
	openLibrary(t, bare, keyed(both()...), h.hook, slog.New(slog.DiscardHandler))
	if ok || got != nil {
		t.Fatalf("SecretCheck of a library with no row = %x, %v; want none", got, ok)
	}
}

func TestTheHookSnapshotsTheUnmigratedLibrary(t *testing.T) {
	path := libraryPath(t)
	old := keyed(shelf.Schema(), clip.Schema().Through(1))
	db := reopen(t, path, old)
	addShelf(t, db, "Films")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "snapshot.db")
	var est int64
	h := hookCalls{check: func(ctx context.Context, u *storage.Unmigrated) {
		var err error
		if est, err = u.Estimate(ctx); err != nil {
			t.Error(err)
		}
		if err := u.Snapshot(ctx, dst); err != nil {
			t.Error(err)
		}
	}}
	openLibrary(t, path, keyed(both()...), h.hook, slog.New(slog.DiscardHandler))
	inspected, err := storage.Inspect(t.Context(), dst, keyed(both()...))
	if err != nil {
		t.Fatal(err)
	}
	if inspected.Versions["clip"] != 1 || !slices.Equal(inspected.Plan, storage.Plan{{Set: "clip", From: 1, To: 2}}) {
		t.Fatalf("snapshot is at %v with plan %v; want clip 1, before the migration", inspected.Versions, inspected.Plan)
	}
	if info, err := os.Stat(dst); err != nil || est <= 0 || est > info.Size() {
		t.Fatalf("estimate %d for a %v-byte snapshot (%v); want no more than its size", est, info.Size(), err)
	}
}

func TestInspectChangesNothingOnDisk(t *testing.T) {
	path := libraryPath(t)
	migrated(t, path, keyed(both()...)...)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := storage.Inspect(t.Context(), path, keyed(both()...))
	if err != nil || len(inspected.Plan) != 0 || inspected.Check != nil || inspected.Versions["clip"] != 2 {
		t.Fatalf("inspect = %+v, %v; want clip 2, nothing to apply and no check row", inspected, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !slices.Equal(before, after) {
		t.Fatalf("library.db changed under Inspect (%v)", err)
	}
}

func TestInspectRefusesADamagedFile(t *testing.T) {
	path := libraryPath(t)
	db := reopen(t, path, keyed(shelf.Schema()))
	for i := range 200 {
		addShelf(t, db, "Shelf "+strconv.Itoa(i))
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	page := int(b[16])<<8 | int(b[17])
	for i := len(b) - page; i < len(b); i++ {
		b[i] = 0xa5
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Inspect(t.Context(), path, keyed(shelf.Schema())); !errors.Is(err, storage.ErrDamaged) {
		t.Fatalf("inspect a file with a damaged page = %v, want ErrDamaged", err)
	}

	junk := libraryPath(t)
	if err := os.WriteFile(junk, slices.Repeat([]byte("not a database "), 512), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Inspect(t.Context(), junk, keyed()); !errors.Is(err, storage.ErrDamaged) {
		t.Fatalf("inspect a file that isn't a database = %v, want ErrDamaged", err)
	}
}

func TestInspectRefusesANewerSchemaNamingTheVersion(t *testing.T) {
	path := libraryPath(t)
	migrated(t, path, keyed(both()...)...)
	_, err := storage.Inspect(t.Context(), path, keyed(shelf.Schema(), clip.Schema().Through(1)))
	refused, ok := errors.AsType[*storage.RefusedError](err)
	schema, isSchema := errors.AsType[*storage.SchemaError](err)
	if !ok || !isSchema || schema.Set != "clip" || schema.Held != 2 || schema.Known != 1 {
		t.Fatalf("inspect = %v, want a *storage.RefusedError naming clip at version 2", err)
	}
	if want := "library.db holds schema clip at version 2, but this binary supports only up to 1. Restore a Backup this version took, or update SlipStream"; refused.Error() != want {
		t.Fatalf("refusal = %q, want %q", refused.Error(), want)
	}
}

func TestOpenCandidateMigratesWithoutTheHook(t *testing.T) {
	path := libraryPath(t)
	db := reopen(t, path, keyed(fence(), shelf.Schema(), clip.Schema().Through(1)))
	addShelf(t, db, "Films")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	today := keyed(fence(), shelf.Schema(), clip.Schema())
	c, err := storage.OpenCandidate(t.Context(), path, today, timings, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	names, err := shelf.Names(t.Context(), c.DB)
	if err != nil || !slices.Equal(names, []string{"Films"}) || c.Fenced {
		t.Fatalf("candidate names = %v, %v, fenced %v; want [Films], not fenced", names, err, c.Fenced)
	}
	if err := c.DB.Close(); err != nil {
		t.Fatal(err)
	}
	inspected, err := storage.Inspect(t.Context(), path, today)
	if err != nil || len(inspected.Plan) != 0 {
		t.Fatalf("after OpenCandidate, inspect = %+v, %v; want today's schemas", inspected, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("closing the candidate removed it: %v", err)
	}
}

func TestOpenCandidateReportsAFenceRow(t *testing.T) {
	path := libraryPath(t)
	schemas := keyed(fence(), shelf.Schema())
	db := reopen(t, path, schemas)
	run(t, db, "INSERT INTO nodes_fence (id, state) VALUES (1, 'proven')")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	c, err := storage.OpenCandidate(t.Context(), path, schemas, timings, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.DB.Close() })
	if !c.Fenced {
		t.Fatal("a library with a nodes_fence row opened as a candidate that isn't fenced")
	}
}

func TestCarryFromUpsertsAndDeletesAbsentRows(t *testing.T) {
	schemas := keyed(both()...)
	from := libraryPath(t)
	src := reopen(t, from, schemas)
	run(t, src, "INSERT INTO shelf (id, name) VALUES (1, 'Films'), (2, 'Series, renamed')")
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}

	db := reopen(t, libraryPath(t), schemas)
	run(t, db, "INSERT INTO shelf (id, name) VALUES (2, 'Series'), (3, 'Revoked')")
	run(t, db, "INSERT INTO clip (shelf_id, public_id, title, created_at) VALUES (2, randomblob(16), 'Kept', 0)")
	if err := db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return storage.CarryFrom(ctx, tx, from, []storage.Carry{{Table: "shelf", Key: "id"}})
	}); err != nil {
		t.Fatal(err)
	}
	names, err := shelf.Names(t.Context(), db)
	if err != nil || !slices.Equal(names, []string{"Films", "Series, renamed"}) {
		t.Fatalf("names = %v, %v; want [Films, Series, renamed]", names, err)
	}
	if clips, err := clip.OnShelf(t.Context(), db, 2); err != nil || len(clips) != 1 {
		t.Fatalf("clips on the carried shelf = %v, %v; want the one kept", clips, err)
	}
}

func TestCarryFromRefusesDifferentColumns(t *testing.T) {
	from := libraryPath(t)
	src := reopen(t, from, keyed(shelf.Schema(), clip.Schema().Through(1)))
	run(t, src, "INSERT INTO shelf (id, name) VALUES (1, 'Films')")
	run(t, src, "INSERT INTO clip (shelf_id, public_id, title, created_at) VALUES (1, randomblob(16), 'From', 0)")
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}

	db := reopen(t, libraryPath(t), keyed(both()...))
	run(t, db, "INSERT INTO shelf (id, name) VALUES (1, 'Films')")
	run(t, db, "INSERT INTO clip (shelf_id, public_id, title, created_at) VALUES (1, randomblob(16), 'Here', 0)")
	err := db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return storage.CarryFrom(ctx, tx, from, []storage.Carry{{Table: "clip", Key: "id"}})
	})
	if err == nil {
		t.Fatal("carry across different columns succeeded")
	}
	clips, err := clip.OnShelf(t.Context(), db, 1)
	if err != nil || len(clips) != 1 || clips[0].Title != "Here" {
		t.Fatalf("clips after the refusal = %v, %v; want only Here", clips, err)
	}
}
