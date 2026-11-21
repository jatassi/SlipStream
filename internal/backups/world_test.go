package backups_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// world is a Controller's data dir with a library.db, its library.key and
// a Backups folder, and a Scheduler that runs each Start's unit in a
// goroutine of its own, one at a time unless parallel is set.
type world struct {
	t        *testing.T
	dir      string
	folder   string
	lib      *storage.DB[storage.Library]
	keys     *secret.Keys
	free     atomic.Int64
	freeHook func()
	parallel bool
	running  sync.Mutex
	wg       sync.WaitGroup
	starts   []tasks.Start
	history  *entrytest.Recorder
	backups  *backups.Backups
}

var schemas = []storage.Schema[storage.Library]{secret.Schema(), backups.Schema()}

func newWorld(t *testing.T) *world {
	t.Helper()
	dir := t.TempDir()
	lib, err := storage.OpenLibrary(t.Context(), filepath.Join(dir, storage.LibraryFile), schemas, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	keys, err := secret.Open(t.Context(), lib, filepath.Join(dir, secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	w := &world{t: t, dir: dir, folder: filepath.Join(dir, "backups"), lib: lib, keys: keys}
	w.free.Store(1 << 40)
	t.Cleanup(w.wg.Wait)
	return w
}

func (w *world) engine(version string) *backups.Engine {
	w.t.Helper()
	index, err := backups.OpenIndex(filepath.Join(w.dir, backups.IndexFile), 0)
	if err != nil {
		w.t.Fatal(err)
	}
	timings := backups.DefaultTimings()
	timings.Scrypt = 10
	return backups.NewEngine(backups.EngineConfig{
		Dir: w.dir, Folder: w.folder, Index: index, Schemas: schemas, ControllerKey: make([]byte, 32), Version: version, Timings: timings,
		Free: func(context.Context, string) (int64, error) {
			if w.freeHook != nil {
				w.freeHook()
			}
			return w.free.Load(), nil
		},
		Clock: time.Now,
		Log:   slog.New(slog.DiscardHandler),
	})
}

// open starts a Controller of version on the data dir, as a start does.
func (w *world) open(version string) (*backups.Backups, error) {
	rec, kept := entrytest.New("backups", backups.Entries())
	w.history = kept
	b, err := backups.Open(w.t.Context(), backups.Config{
		Library: w.lib, Keys: w.keys, Engine: w.engine(version), Start: w.start,
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return false, nil },
		History:     rec,
		Log:         slog.New(slog.DiscardHandler),
	})
	w.backups = b
	return b, err
}

func (w *world) opened(version string) *backups.Backups {
	w.t.Helper()
	b, err := w.open(version)
	if err != nil {
		w.t.Fatal(err)
	}
	return b
}

func (w *world) start(ctx context.Context, st tasks.Start) (tasks.Started, error) {
	w.starts = append(w.starts, st)
	task := w.backups.Task()
	w.wg.Go(func() {
		if !w.parallel {
			w.running.Lock()
			defer w.running.Unlock()
		}
		_ = task.Run(context.WithoutCancel(ctx), tasks.Unit{Key: st.Units[0]}, &tasks.Progress{})
	})
	return tasks.Started{Added: st.Units}, nil
}

func (w *world) take(req backups.Request) (backups.Entry, error) {
	w.t.Helper()
	run, err := w.backups.Take(w.t.Context(), req)
	if err != nil {
		w.t.Fatal(err)
	}
	return run.Wait(w.t.Context())
}

func (w *world) taken(trigger archive.Trigger) backups.Entry {
	w.t.Helper()
	e, err := w.take(backups.Request{Trigger: trigger})
	if err != nil {
		w.t.Fatal(err)
	}
	return e
}

func (w *world) ledger() *backups.Ledger {
	w.t.Helper()
	index, err := backups.OpenIndex(filepath.Join(w.dir, backups.IndexFile), 0)
	if err != nil {
		w.t.Fatal(err)
	}
	l := index.Read()
	return &l
}

func (w *world) listed() []string {
	var names []string
	for _, e := range w.ledger().Backups {
		names = append(names, e.Name.String()+" "+string(e.State))
	}
	return names
}

// files is every name in the Backups folder.
func (w *world) files() []string {
	w.t.Helper()
	entries, err := os.ReadDir(w.folder)
	if err != nil {
		w.t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// rewrite changes the entry with id in backups.json as a stopped run
// left it, with no Backups service open.
func (w *world) rewrite(id archive.ID, fn func(*backups.Entry)) {
	w.t.Helper()
	index, err := backups.OpenIndex(filepath.Join(w.dir, backups.IndexFile), 0)
	if err != nil {
		w.t.Fatal(err)
	}
	if err := index.Update(func(l *backups.Ledger) error {
		for i := range l.Backups {
			if l.Backups[i].ID == id {
				fn(&l.Backups[i])
			}
		}
		return nil
	}); err != nil {
		w.t.Fatal(err)
	}
}
