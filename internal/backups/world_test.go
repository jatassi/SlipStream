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
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
	"github.com/jatassi/slipstream/internal/works"
)

// world is a Controller's data dir with a library.db, its library.key and
// a Backups folder, a Scheduler that runs each Start's units in a
// goroutine of their own, one at a time unless parallel is set, and the
// transfers of a Controller, home, with one Worker, vps, whose facts the
// test sets. Its clock runs ahead of time.Now by ahead.
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
	startMu  sync.Mutex
	starts   []tasks.Start
	history  *entrytest.Recorder
	backups  *backups.Backups
	eng      *backups.Engine
	ahead    atomic.Int64

	home, vps role.NodeID
	tr        *transfers.Transfers
	factsMu   sync.Mutex
	facts     map[role.NodeID]*tv1.Facts
	verdict   roots.Verdict
}

var schemas = []storage.Schema[storage.Library]{
	secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), health.Schema(), roots.Schema(),
	transfers.LibrarySchema(), backups.Schema(),
}

type quiet struct{}

func (quiet) Publish(context.Context, transfers.Changed) {}

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
	w := &world{
		t: t, dir: dir, folder: filepath.Join(dir, "backups"), lib: lib, keys: keys,
		home: role.NodeID{UUID: storage.NewUUID()}, vps: role.NodeID{UUID: storage.NewUUID()}, facts: map[role.NodeID]*tv1.Facts{},
	}
	w.free.Store(1 << 40)
	for i, n := range []struct {
		id   role.NodeID
		name string
	}{{w.home, "home"}, {w.vps, "vps"}} {
		key := make([]byte, 32)
		key[0] = byte(i + 1)
		if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, ?, 0)", n.id, key, n.name)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(w.wg.Wait)
	return w
}

// transfers is the Controller's transfers, as a start opens them.
func (w *world) transfers() *transfers.Transfers {
	return transfers.New(transfers.ControllerConfig{
		Library: w.lib,
		Touch:   func(...role.NodeID) {},
		Facts: func(id role.NodeID) (*tv1.Facts, bool) {
			w.factsMu.Lock()
			defer w.factsMu.Unlock()
			f, ok := w.facts[id]
			return f, ok
		},
		Board: func() transfers.Board { return nil },
		Destinations: func(context.Context, storage.Reader[storage.Library], []roots.ID) (map[roots.ID]roots.Destination, error) {
			return nil, nil
		},
		RootsMoved: func() <-chan struct{} { return nil },
		Readiness: func(context.Context, storage.Reader[storage.Library], role.NodeID, []source.Ref) (map[source.Ref]source.Readiness, error) {
			return nil, nil
		},
		Moved:        func() <-chan struct{} { return nil },
		Folders:      func(id role.NodeID) []transfers.BackupFolder { return w.backups.Folders(id) },
		FoldersNoted: func(role.NodeID) {},
		LowWait:      func(context.Context, storage.Reader[storage.Library]) (time.Duration, error) { return time.Hour, nil },
		Paused:       func(context.Context, storage.Reader[storage.Library]) (bool, error) { return false, nil },
		Sample:       func(context.Context, role.NodeID) error { return nil },
		Hub:          live.NewHub([]live.Event{transfers.ChangedEvent}),
		Changed:      quiet{},
		Timings:      transfers.DefaultTimings(),
		Log:          slog.New(slog.DiscardHandler),
	})
}

func (w *world) now() time.Time { return time.Now().Add(time.Duration(w.ahead.Load())) }

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
		Clock: w.now,
		Log:   slog.New(slog.DiscardHandler),
	})
}

// open starts a Controller of version on the data dir, as a start does.
func (w *world) open(version string) (*backups.Backups, error) {
	rec, kept := entrytest.New("backups", backups.Entries())
	w.history = kept
	w.eng = w.engine(version)
	w.tr = w.transfers()
	b, err := backups.Open(w.t.Context(), backups.Config{
		Library: w.lib, Keys: w.keys, Engine: w.eng, Start: w.start,
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return false, nil },
		History:     rec,
		Log:         slog.New(slog.DiscardHandler),
		Transfers:   w.tr,
		Self:        func() role.NodeID { return w.home },
		Check: func(_ context.Context, _ identity.Admin, _ role.NodeID, folder string) (roots.Verdict, error) {
			v := w.verdict
			v.Path = folder
			return v, nil
		},
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
	w.startMu.Lock()
	w.starts = append(w.starts, st)
	w.startMu.Unlock()
	task := w.backups.Task()
	for _, u := range st.Units {
		w.wg.Go(func() {
			if !w.parallel {
				w.running.Lock()
				defer w.running.Unlock()
			}
			_ = task.Run(context.WithoutCancel(ctx), tasks.Unit{Key: u}, &tasks.Progress{})
		})
	}
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
