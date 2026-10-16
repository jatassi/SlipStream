package storage_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/example/clip"
	"github.com/jatassi/slipstream/internal/storage/example/shelf"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

var timings = storage.Timings{BusyTimeout: 5 * time.Second}

type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, _ := b.buf.Write(p)
	return n, nil
}

func (b *logBuffer) lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Split(strings.TrimSpace(b.buf.String()), "\n")
}

func newLog() (*slog.Logger, *logBuffer) {
	b := &logBuffer{}
	return slog.New(slog.NewJSONHandler(b, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}})), b
}

type hookCalls struct {
	plans []storage.Plan
	err   error
	check func(context.Context, *storage.Unmigrated)
}

func (h *hookCalls) hook(ctx context.Context, u *storage.Unmigrated, plan storage.Plan) error {
	h.plans = append(h.plans, plan)
	if h.check != nil {
		h.check(ctx, u)
	}
	return h.err
}

func openLibrary(t *testing.T, path string, schemas []storage.Schema[storage.Library], hook storage.PreMigrationHook, log *slog.Logger) *storage.DB[storage.Library] {
	t.Helper()
	db, err := storage.OpenLibrary(t.Context(), path, schemas, hook, timings, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func migrated(t *testing.T, path string, schemas ...storage.Schema[storage.Library]) {
	t.Helper()
	db, err := storage.OpenLibrary(t.Context(), path, schemas, nil, timings, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func libraryPath(t *testing.T) string { return filepath.Join(t.TempDir(), storage.LibraryFile) }

func both() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{shelf.Schema(), clip.Schema()}
}

func addShelf(t *testing.T, db *storage.DB[storage.Library], name string) shelf.ID {
	t.Helper()
	var id shelf.ID
	if err := db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		id, err = shelf.Add(ctx, tx, name)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestFreshFileMigratesEverySetInOrder(t *testing.T) {
	log, records := newLog()
	var h hookCalls
	db := openLibrary(t, libraryPath(t), both(), h.hook, log)

	want := []string{`{"level":"INFO","msg":"migrated","file":"library.db","sets":[{"set":"shelf","from":0,"to":1},{"set":"clip","from":0,"to":2}]}`}
	if got := records.lines(); !slices.Equal(got, want) {
		t.Errorf("log = %q, want %q", got, want)
	}
	if len(h.plans) != 0 {
		t.Errorf("hook ran on a fresh file with %v", h.plans)
	}
	c, err := db.Catalog(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for _, o := range c.Objects {
		if o.Type == "table" {
			tables = append(tables, o.Name)
		}
	}
	if want := []string{"clip", "goose_clip", "goose_shelf", "shelf"}; !slices.Equal(tables, want) {
		t.Errorf("tables = %v, want %v", tables, want)
	}
}

func TestRestartAppliesNothing(t *testing.T) {
	path := libraryPath(t)
	db := openLibrary(t, path, both(), nil, slog.New(slog.DiscardHandler))
	addShelf(t, db, "Films")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	log, records := newLog()
	var h hookCalls
	db = openLibrary(t, path, both(), h.hook, log)
	if got := records.lines(); !slices.Equal(got, []string{""}) {
		t.Errorf("restart logged %q", got)
	}
	if len(h.plans) != 0 {
		t.Errorf("hook ran on a restart with %v", h.plans)
	}
	names, err := shelf.Names(t.Context(), db)
	if err != nil || !slices.Equal(names, []string{"Films"}) {
		t.Errorf("names = %v, %v; want [Films]", names, err)
	}
}

func TestUpgradeCallsTheHookBeforeMigrating(t *testing.T) {
	path := libraryPath(t)
	old := []storage.Schema[storage.Library]{shelf.Schema(), clip.Schema().Through(1)}
	db := openLibrary(t, path, old, nil, slog.New(slog.DiscardHandler))
	addShelf(t, db, "Films")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	refuse := hookCalls{err: errors.New("backup failed")}
	_, err := storage.OpenLibrary(t.Context(), path, both(), refuse.hook, timings, slog.New(slog.DiscardHandler))
	if want := "before migrating library.db: backup failed"; err == nil || err.Error() != want {
		t.Fatalf("open with a failing hook = %v, want %q", err, want)
	}
	migrated(t, path, old...)

	var held storage.Versions
	h := hookCalls{check: func(ctx context.Context, u *storage.Unmigrated) {
		var err error
		if held, err = u.Versions(ctx); err != nil {
			t.Error(err)
		}
	}}
	db = openLibrary(t, path, both(), h.hook, slog.New(slog.DiscardHandler))
	if want := []storage.Plan{{{Set: "clip", From: 1, To: 2}}}; !slices.EqualFunc(h.plans, want, slices.Equal) {
		t.Errorf("hook plans = %v, want %v", h.plans, want)
	}
	if want := (storage.Versions{"shelf": 1, "clip": 1}); len(held) != 2 || held["shelf"] != 1 || held["clip"] != 1 {
		t.Errorf("hook saw versions %v, want %v", held, want)
	}
	names, err := shelf.Names(t.Context(), db)
	if err != nil || !slices.Equal(names, []string{"Films"}) {
		t.Errorf("names = %v, %v; want [Films]", names, err)
	}
}

func TestNewerLibraryIsRefused(t *testing.T) {
	path := libraryPath(t)
	migrated(t, path, both()...)

	_, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{shelf.Schema(), clip.Schema().Through(1)}, nil, timings, slog.New(slog.DiscardHandler))
	assertRefused(t, err, "library.db holds schema clip at version 2, but this binary knows only up to 1; restore a Backup this version took, or update SlipStream")

	_, err = storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{shelf.Schema()}, nil, timings, slog.New(slog.DiscardHandler))
	assertRefused(t, err, "library.db holds schema clip, which this binary does not have")
}

func TestUnreadableLibraryIsRefused(t *testing.T) {
	path := libraryPath(t)
	if err := os.WriteFile(path, bytes.Repeat([]byte("not a database "), 512), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := storage.OpenLibrary(t.Context(), path, both(), nil, timings, slog.New(slog.DiscardHandler))
	assertRefused(t, err, "library.db is not a readable SQLite database; restore a Backup")
}

func assertRefused(t *testing.T, err error, want string) {
	t.Helper()
	refused, ok := errors.AsType[*storage.RefusedError](err)
	if !ok || refused.Error() != want {
		t.Errorf("err = %v, want *storage.RefusedError %q", err, want)
	}
}

func outbox() storage.Schema[storage.Local] {
	return storage.NewSchema[storage.Local]("outbox", fstest.MapFS{
		"schema/00001_outbox.sql":      {Data: []byte("-- +goose Up\nCREATE TABLE outbox (id INTEGER PRIMARY KEY, body TEXT NOT NULL) STRICT;\n")},
		"schema/00002_outbox_sent.sql": {Data: []byte("-- +goose Up\nALTER TABLE outbox ADD COLUMN sent_at INTEGER;\n")},
	})
}

func openLocal(t *testing.T, path string, schemas []storage.Schema[storage.Local], log *slog.Logger) *storage.DB[storage.Local] {
	t.Helper()
	db, err := storage.OpenLocal(t.Context(), path, schemas, timings, log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countOutbox(t *testing.T, db *storage.DB[storage.Local]) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM outbox").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestNewerLocalIsDiscarded(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LocalFile)
	db := openLocal(t, path, []storage.Schema[storage.Local]{outbox()}, slog.New(slog.DiscardHandler))
	if err := db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO outbox (body) VALUES ('hello')")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	log, records := newLog()
	db = openLocal(t, path, []storage.Schema[storage.Local]{outbox().Through(1)}, log)
	want := []string{`{"level":"WARN","msg":"local state discarded","file":"local.db","reason":"local.db holds schema outbox at version 2, but this binary knows only up to 1"}`,
		`{"level":"INFO","msg":"migrated","file":"local.db","sets":[{"set":"outbox","from":0,"to":1}]}`}
	if got := records.lines(); !slices.Equal(got, want) {
		t.Errorf("log = %q, want %q", got, want)
	}
	if n := countOutbox(t, db); n != 0 {
		t.Errorf("outbox holds %d rows after the discard, want 0", n)
	}
	kept := openLocal(t, path+".discarded", []storage.Schema[storage.Local]{outbox()}, slog.New(slog.DiscardHandler))
	if n := countOutbox(t, kept); n != 1 {
		t.Errorf("local.db.discarded holds %d rows, want 1", n)
	}
}

func TestUnreadableLocalOpensEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, storage.LocalFile)
	if err := os.WriteFile(path+".discarded", []byte("an older discard"), 0o600); err != nil {
		t.Fatal(err)
	}
	garbage := bytes.Repeat([]byte("not a database "), 512)
	if err := os.WriteFile(path, garbage, 0o600); err != nil {
		t.Fatal(err)
	}
	log, records := newLog()
	db := openLocal(t, path, []storage.Schema[storage.Local]{outbox()}, log)
	want := []string{`{"level":"WARN","msg":"local state discarded","file":"local.db","reason":"local.db is not a readable SQLite database"}`,
		`{"level":"INFO","msg":"migrated","file":"local.db","sets":[{"set":"outbox","from":0,"to":2}]}`}
	if got := records.lines(); !slices.Equal(got, want) {
		t.Errorf("log = %q, want %q", got, want)
	}
	if n := countOutbox(t, db); n != 0 {
		t.Errorf("outbox holds %d rows, want 0", n)
	}
	if kept, err := os.ReadFile(path + ".discarded"); err != nil || !bytes.Equal(kept, garbage) {
		t.Errorf("local.db.discarded is not the garbage file (%v)", err)
	}
}

func TestFailedLocalMigrationStopsTheOpen(t *testing.T) {
	broken := storage.NewSchema[storage.Local]("outbox", fstest.MapFS{
		"schema/00001_outbox.sql": {Data: []byte("-- +goose Up\nCREATE TABLE outbox (id INTEGER PRIMARY KEY) STRICT;\n")},
		"schema/00002_broken.sql": {Data: []byte("-- +goose Up\nALTER TABLE nowhere ADD COLUMN x INTEGER;\n")},
	})
	_, err := storage.OpenLocal(t.Context(), filepath.Join(t.TempDir(), storage.LocalFile), []storage.Schema[storage.Local]{broken}, timings, slog.New(slog.DiscardHandler))
	if err == nil {
		t.Fatal("open succeeded")
	}
	const prefix, suffix = "local.db migration outbox 2 failed: ", "; local.db is safe to delete"
	if msg := err.Error(); !strings.HasPrefix(msg, prefix) || !strings.HasSuffix(msg, suffix) || !strings.Contains(msg, "no such table: nowhere") {
		t.Errorf("err = %q, want %q + the SQLite error + %q", msg, prefix, suffix)
	}
	if _, ok := errors.AsType[*storage.RefusedError](err); ok {
		t.Error("a failed migration is a *storage.RefusedError")
	}
}

func TestRowsLeftWithoutTheirParentFailTheOpen(t *testing.T) {
	orphan := storage.NewSchema[storage.Library]("pin", fstest.MapFS{
		"schema/00001_pin.sql": {Data: []byte("-- +goose Up\nCREATE TABLE pin (id INTEGER PRIMARY KEY, shelf_id INTEGER NOT NULL REFERENCES shelf (id)) STRICT;\nINSERT INTO pin (shelf_id) VALUES (42);\n")},
	})
	_, err := storage.OpenLibrary(t.Context(), libraryPath(t), []storage.Schema[storage.Library]{shelf.Schema(), orphan}, nil, timings, slog.New(slog.DiscardHandler))
	if want := "library.db: set pin left rows in pin whose foreign key into shelf matches nothing"; err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestTableRebuildKeepsChildRowsOfLaterSets(t *testing.T) {
	parent := storage.NewSchema[storage.Library]("box", fstest.MapFS{
		"schema/00001_box.sql":     {Data: []byte("-- +goose Up\nCREATE TABLE box (id INTEGER PRIMARY KEY, label TEXT NOT NULL) STRICT;\nINSERT INTO box (id, label) VALUES (1, 'a');\n")},
		"schema/00002_rebuild.sql": {Data: []byte("-- +goose Up\nCREATE TABLE box_new (id INTEGER PRIMARY KEY, label TEXT NOT NULL, size INTEGER NOT NULL DEFAULT 0) STRICT;\nINSERT INTO box_new (id, label) SELECT id, label FROM box;\nDROP TABLE box;\nALTER TABLE box_new RENAME TO box;\n")},
	})
	child := storage.NewSchema[storage.Library]("item", fstest.MapFS{
		"schema/00001_item.sql": {Data: []byte("-- +goose Up\nCREATE TABLE item (id INTEGER PRIMARY KEY, box_id INTEGER NOT NULL REFERENCES box (id) ON DELETE CASCADE) STRICT;\nINSERT INTO item (box_id) VALUES (1);\n")},
	})
	path := libraryPath(t)
	migrated(t, path, parent.Through(1), child)
	db := openLibrary(t, path, []storage.Schema[storage.Library]{parent, child}, nil, slog.New(slog.DiscardHandler))
	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM item").Scan(&n); err != nil || n != 1 {
		t.Errorf("item holds %d rows after box's rebuild (%v), want 1", n, err)
	}
}

func TestConcurrentWritesAllCommit(t *testing.T) {
	db := storagetest.Library(t, shelf.Schema())
	const writers, writes, readers = 16, 50, 4
	ctx := t.Context()
	errs := make(chan error, writers*writes+readers)
	var wg sync.WaitGroup
	for w := range writers {
		wg.Go(func() {
			for i := range writes {
				errs <- db.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
					_, err := shelf.Add(ctx, tx, fmt.Sprintf("shelf %d-%d", w, i))
					return err
				})
			}
		})
	}
	done := make(chan struct{})
	var rg sync.WaitGroup
	for range readers {
		rg.Go(func() { errs <- readUntil(ctx, db, done) })
	}
	wg.Wait()
	close(done)
	rg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	if n, err := shelf.Count(ctx, db); err != nil || n != writers*writes {
		t.Errorf("count = %d, %v; want %d", n, err, writers*writes)
	}
}

func readUntil(ctx context.Context, db *storage.DB[storage.Library], done <-chan struct{}) error {
	for {
		select {
		case <-done:
			return nil
		default:
		}
		if _, err := shelf.Count(ctx, db); err != nil {
			return err
		}
	}
}

func assertPanics(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if got := recover(); got != want {
			t.Errorf("panic = %v, want %q", got, want)
		}
	}()
	fn()
}

func TestNestedWriteOnTheSameFilePanics(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema())
	assertPanics(t, "storage: Write inside Write on library.db deadlocks; pass the Tx", func() {
		_ = lib.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
			return lib.Write(ctx, func(context.Context, *storage.Tx[storage.Library]) error { return nil })
		})
	})
	if n, err := shelf.Count(t.Context(), lib); err != nil || n != 0 {
		t.Errorf("count after the panic = %d, %v; want 0", n, err)
	}
	addShelf(t, lib, "after")
}

func TestReadThroughTheDBInsideWritePanics(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema())
	assertPanics(t, "storage: read through the Tx inside Write", func() {
		_ = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			if _, err := shelf.Add(ctx, tx, "Films"); err != nil {
				return err
			}
			_, err := shelf.Count(ctx, lib)
			return err
		})
	})
	if n, err := shelf.Count(t.Context(), lib); err != nil || n != 0 {
		t.Errorf("the panicked Write committed: count = %d, %v", n, err)
	}
}

func TestReadSeesOneSnapshot(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema())
	addShelf(t, lib, "Films")
	var before, after int64
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if before, err = shelf.Count(ctx, r); err != nil {
			return err
		}
		addShelf(t, lib, "Series")
		after, err = shelf.Count(ctx, r)
		return err
	})
	if err != nil || before != 1 || after != 1 {
		t.Errorf("counts inside one Read = %d then %d (%v), want 1 then 1", before, after, err)
	}
	if n, err := shelf.Count(t.Context(), lib); err != nil || n != 2 {
		t.Errorf("count after the Read = %d, %v; want 2", n, err)
	}
	assertPanics(t, "storage: read through the Tx inside Write", func() {
		_ = lib.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
			return lib.Read(ctx, func(context.Context, storage.Reader[storage.Library]) error { return nil })
		})
	})
	assertPanics(t, "storage: write inside Read; take a *storage.Tx", func() {
		_ = lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			_, err := r.ExecContext(ctx, "DELETE FROM shelf")
			return err
		})
	})
}

func TestWriteThroughTheDBPanics(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema())
	assertPanics(t, "storage: write through the DB outside Write; take a *storage.Tx", func() {
		_, _ = lib.ExecContext(t.Context(), "DELETE FROM shelf")
	})
	rows, err := lib.QueryContext(t.Context(), "INSERT INTO shelf (name) VALUES ('x') RETURNING id")
	if err == nil {
		_ = rows.Close()
	}
	if want := "library.db: attempt to write a readonly database (8)"; err == nil || err.Error() != want {
		t.Errorf("a write through the readers = %v, want %q", err, want)
	}
}

func TestLocalWriteInsideLibraryWrite(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema())
	local := storagetest.Local(t, outbox())
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := shelf.Add(ctx, tx, "Films"); err != nil {
			return err
		}
		return local.Write(ctx, func(ctx context.Context, ltx *storage.Tx[storage.Local]) error {
			_, err := ltx.ExecContext(ctx, "INSERT INTO outbox (body) VALUES ('shelf added')")
			return err
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := shelf.Count(t.Context(), lib); err != nil || n != 1 {
		t.Errorf("shelf count = %d, %v; want 1", n, err)
	}
	if n := countOutbox(t, local); n != 1 {
		t.Errorf("outbox count = %d, want 1", n)
	}
	assertPanics(t, "storage: Write on library.db inside Write on local.db deadlocks; write library.db first", func() {
		_ = local.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Local]) error {
			return lib.Write(ctx, func(context.Context, *storage.Tx[storage.Library]) error { return nil })
		})
	})
}

func TestTimesReadBackAsUTCInstants(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema(), clip.Schema())
	on := addShelf(t, lib, "Films")
	zone := time.FixedZone("UTC+5", 5*60*60)
	watched := time.Date(2026, 10, 12, 18, 30, 15, 123456789, zone)
	var heat, ronin clip.Clip
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if heat, err = clip.Add(ctx, tx, on, "Heat"); err != nil {
			return err
		}
		if ronin, err = clip.Add(ctx, tx, on, "Ronin"); err != nil {
			return err
		}
		return clip.MarkWatched(ctx, tx, heat.ID, storage.FromTime(watched))
	}); err != nil {
		t.Fatal(err)
	}
	got, err := clip.Get(t.Context(), lib, heat.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 12, 13, 30, 15, 123456000, time.UTC)
	if at := got.WatchedAt.Time(); !at.Equal(want) || at.Location() != time.UTC || got.WatchedAt != storage.FromTime(watched) {
		t.Errorf("watched_at = %v, want %v", got.WatchedAt, want)
	}
	unwatched, err := clip.Unwatched(t.Context(), lib, on)
	if err != nil || len(unwatched) != 1 || unwatched[0] != ronin || !ronin.WatchedAt.IsZero() {
		t.Errorf("unwatched = %v, %v; want only Ronin with a zero watched_at", unwatched, err)
	}
}

func TestUUIDsRoundTripAndSortByCreation(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema(), clip.Schema())
	on := addShelf(t, lib, "Films")
	var added []clip.Clip
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, title := range []string{"Heat", "Alien", "Ronin", "Brazil"} {
			c, err := clip.Add(ctx, tx, on, title)
			if err != nil {
				return err
			}
			added = append(added, c)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	listed, err := clip.OnShelf(t.Context(), lib, on)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(listed, added) {
		t.Errorf("clips by public_id = %v, want creation order %v", listed, added)
	}
	parsed, err := storage.ParseUUID(added[0].PublicID.String())
	if err != nil || parsed != added[0].PublicID.UUID {
		t.Errorf("ParseUUID(%s) = %v, %v", added[0].PublicID, parsed, err)
	}
	fromBytes, err := storage.UUIDFromBytes(added[0].PublicID.Bytes())
	if err != nil || fromBytes != added[0].PublicID.UUID {
		t.Errorf("UUIDFromBytes(%x) = %v, %v", added[0].PublicID.Bytes(), fromBytes, err)
	}
	if _, err := storage.UUIDFromBytes([]byte{1, 2, 3}); err == nil || err.Error() != "storage.UUID: 3 bytes, want 16" {
		t.Errorf("UUIDFromBytes of 3 bytes = %v", err)
	}
	if _, err := clip.Get(t.Context(), lib, added[3].ID+1); !errors.Is(err, storage.ErrNoRows) {
		t.Errorf("Get of a missing clip = %v, want storage.ErrNoRows", err)
	}
}

func TestCompareRefusesWhatOpenRefuses(t *testing.T) {
	schemas := both()
	plan, err := storage.Compare(storage.Versions{"shelf": 1}, schemas)
	if err != nil || !slices.Equal(plan, storage.Plan{{Set: "clip", From: 0, To: 2}}) {
		t.Errorf("Compare = %v, %v", plan, err)
	}
	_, err = storage.Compare(storage.Versions{"shelf": 1, "clip": 3, "zebra": 1, "aardvark": 1}, schemas)
	if want := "library.db holds schema clip at version 3, but this binary knows only up to 2; restore a Backup this version took, or update SlipStream"; err == nil || err.Error() != want {
		t.Errorf("Compare = %v, want %q", err, want)
	}
	_, err = storage.Compare(storage.Versions{"zebra": 1, "aardvark": 1}, schemas)
	if want := "library.db holds schema aardvark, which this binary does not have"; err == nil || err.Error() != want {
		t.Errorf("Compare = %v, want %q", err, want)
	}
}

func TestWrongFileDoesNotCompile(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "build", "-o", os.DevNull, "./testdata/wrongfile").CombinedOutput()
	if err == nil {
		t.Fatal("testdata/wrongfile compiled")
	}
	for _, want := range []string{
		"cannot use tx (variable of type *storage.Tx[storage.Library]) as *storage.Tx[storage.Local] value in argument to putOutbox",
		"cannot use tx (variable of type *storage.Tx[storage.Library]) as storage.Reader[storage.Local] value in argument to readOutbox: *storage.Tx[storage.Library] does not implement storage.Reader[storage.Local] (wrong type for method file)",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("build output lacks %q:\n%s", want, out)
		}
	}
}

func TestInWriteReportsEitherFilesWriter(t *testing.T) {
	lib := storagetest.Library(t)
	local := storagetest.Local(t)
	if storage.InWrite(t.Context()) {
		t.Error("InWrite outside any Write = true")
	}
	var inLibrary, inLocal, inBoth bool
	if err := lib.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
		inLibrary = storage.InWrite(ctx)
		return local.Write(ctx, func(ctx context.Context, _ *storage.Tx[storage.Local]) error {
			inBoth = storage.InWrite(ctx)
			return nil
		})
	}); err != nil {
		t.Fatal(err)
	}
	if err := local.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Local]) error {
		inLocal = storage.InWrite(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got := [3]bool{inLibrary, inLocal, inBoth}; got != [3]bool{true, true, true} {
		t.Errorf("InWrite inside library, local, both = %v; want all true", got)
	}
}

func TestOptimizeRunsOnBothFiles(t *testing.T) {
	lib := storagetest.Library(t, shelf.Schema())
	local := storagetest.Local(t, outbox())
	if err := lib.Optimize(t.Context()); err != nil {
		t.Errorf("library.db: %v", err)
	}
	if err := local.Optimize(t.Context()); err != nil {
		t.Errorf("local.db: %v", err)
	}
	assertPanics(t, "storage: Write inside Write on library.db deadlocks; pass the Tx", func() {
		_ = lib.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
			return lib.Optimize(ctx)
		})
	})
}
