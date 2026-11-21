package node

import (
	"context"
	"crypto/rand"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// marker is a test set holding one value that tells two libraries apart.
func marker() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("marker", fstest.MapFS{
		"schema/00001_marker.sql": {Data: []byte("-- +goose Up\nCREATE TABLE marker (id INTEGER PRIMARY KEY, value TEXT NOT NULL) STRICT;\n")},
	})
}

func markedEnv(t *testing.T) Env {
	t.Helper()
	env, _ := testEnv(t)
	env.Test.Library = []storage.Schema[storage.Library]{marker()}
	return env
}

// openMarked opens the library in dir as a Node does and writes value to
// its marker.
func openMarked(t *testing.T, env Env, dir DataDir, value string) *libraryDB {
	t.Helper()
	lib, err := openLibraryFiles(t.Context(), libraryFiles{
		dir:     dir,
		schemas: append(LibrarySchemas(env.Types), env.Test.Library...),
		timings: Defaults(env.Types, env.Install).Timings,
		hub:     live.NewHub(surface.Events),
		log:     slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.db.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO marker (value) VALUES (?)", value)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return lib
}

func markOf(t *testing.T, r storage.Reader[storage.Library]) []string {
	t.Helper()
	rows, err := r.QueryContext(t.Context(), "SELECT value FROM marker ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var values []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		values = append(values, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return values
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	b, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func scratchDir(t *testing.T) DataDir {
	t.Helper()
	d, err := ParseDataDir(shortTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// staged is a data dir as a restore leaves it before its restart: the
// old library live with its WAL, as a crash of the running Node leaves
// it, the restored files beside it, and a staged record.
func staged(t *testing.T, env Env) {
	t.Helper()
	dir := env.DataDir.Path()
	restored := scratchDir(t)
	if err := openMarked(t, env, restored, "restored").close(t.Context()); err != nil {
		t.Fatal(err)
	}
	copyFile(t, restored.LibraryDB(), env.DataDir.LibraryDB()+lifecycle.Restored)
	copyFile(t, restored.LibraryKey(), env.DataDir.LibraryKey()+lifecycle.Restored)
	copyFile(t, vaultIn(t, restored), filepath.Join(dir, backups.VaultFile+lifecycle.Restored))

	old := scratchDir(t)
	lib := openMarked(t, env, old, "old")
	for _, suffix := range []string{"", "-wal", "-shm"} {
		copyFile(t, old.LibraryDB()+suffix, env.DataDir.LibraryDB()+suffix)
	}
	copyFile(t, old.LibraryKey(), env.DataDir.LibraryKey())
	if err := lib.close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(env.DataDir.LibraryDB() + "-wal"); err != nil || info.Size() == 0 {
		t.Fatalf("the old library has no live WAL: %v, %v", info, err)
	}
	for name, body := range map[string]string{"backup.key": "old", "backups.json.restore": `{"format": 1, "startedVersion": "", "backups": []}`} {
		writeFile(t, filepath.Join(dir, name), body)
	}
	if err := env.DataDir.lifecycle(0).Begin(stagedRestore()); err != nil {
		t.Fatal(err)
	}
}

func stagedRestore() *lifecycle.Restore {
	return &lifecycle.Restore{ID: "r1", Stage: lifecycle.Staged, Cause: lifecycle.ByRestore, Backup: lifecycle.Backup{Name: "slipstream-20261114T040012Z-2.1.0-manual-7k3m.tar.zst.age", Version: "2.1.0"}}
}

var errCrash = errors.New("crash")

// crashAt stops a start before its stop-th restore step.
func crashAt(stop int) func(RestoreStep) error {
	var i int
	return func(RestoreStep) error {
		if i == stop {
			return errCrash
		}
		i++
		return nil
	}
}

// assertRestored checks the data dir a finished restore leaves: the
// restored library open, and nothing of the swap left beside it.
func assertRestored(t *testing.T, n *Node, env Env) {
	t.Helper()
	if got := markOf(t, n.lib.Load().db); !slices.Equal(got, []string{"restored"}) {
		t.Fatalf("the library holds %v, want the restored one", got)
	}
	entries, err := os.ReadDir(env.DataDir.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if ext := filepath.Ext(e.Name()); ext == lifecycle.Restored || ext == lifecycle.Aside || e.Name() == lifecycle.FileName || slices.Contains([]string{"library.db.pre-restore-wal", "library.db.pre-restore-shm"}, e.Name()) {
			t.Fatalf("%s is left after the restore", e.Name())
		}
	}
	keys, err := secret.ReadKeys(env.DataDir.LibraryKey())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backups.OpenVault(filepath.Join(env.DataDir.Path(), backups.VaultFile), keys, 0); err != nil {
		t.Fatalf("backup.key isn't the restored one: %v", err)
	}
}

// vaultIn writes a backup.key sealed with dir's library.key and returns
// its path.
func vaultIn(t *testing.T, dir DataDir) string {
	t.Helper()
	keys, err := secret.ReadKeys(dir.LibraryKey())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir.Path(), backups.VaultFile)
	v, err := backups.OpenVault(path, keys, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := v.Current(rand.Reader); err != nil {
		t.Fatal(err)
	}
	return path
}

// asideIsOld reports whether the swap moved the old library aside yet,
// and checks that it then opens whole, with what its WAL held.
func asideIsOld(t *testing.T, env Env) bool {
	t.Helper()
	aside := env.DataDir.LibraryDB() + lifecycle.Aside
	if !exists(aside) {
		return false
	}
	db, err := storage.OpenLibrary(t.Context(), aside, append(LibrarySchemas(env.Types), marker()), nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	got := markOf(t, db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"old"}) {
		t.Fatalf("the aside library holds %v, want the old one with its WAL", got)
	}
	return true
}

func TestACrashBeforeEachSwapStepFinishesAtTheNextStart(t *testing.T) {
	env := markedEnv(t)
	staged(t, env)
	var steps []RestoreStep
	env.Test.BeforeRestoreStage = func(s RestoreStep) error {
		steps = append(steps, s)
		return nil
	}
	n := startTest(t, env)
	assertRestored(t, n, env)
	n.Stop()
	waitDone(t, n)
	if renames := slices.IndexFunc(steps, func(s RestoreStep) bool { return s.Stage != RestoreRename }); renames != 9 {
		t.Fatalf("steps = %v, want 9 renames first", steps)
	}

	var asideOpened int
	for stop, step := range steps {
		t.Run(strconv.Itoa(stop), func(t *testing.T) {
			env := markedEnv(t)
			staged(t, env)
			env.Test.BeforeRestoreStage = crashAt(stop)
			if _, err := Start(t.Context(), env); !errors.Is(err, errCrash) {
				t.Fatalf("start that crashes before %v = %v", step, err)
			}
			if asideIsOld(t, env) {
				asideOpened++
			}
			env.Test.BeforeRestoreStage = nil
			assertRestored(t, startTest(t, env), env)
		})
	}
	if asideOpened == 0 {
		t.Fatal("no crash left an aside library to open")
	}
}

func TestASecondLifecycleOpIsRefusedAcrossARestart(t *testing.T) {
	env := markedEnv(t)
	staged(t, env)
	env.Test.BeforeRestoreStage = func(s RestoreStep) error {
		if s.Stage == RestoreAside {
			return errCrash
		}
		return nil
	}
	if _, err := Start(t.Context(), env); !errors.Is(err, errCrash) {
		t.Fatalf("start = %v, want the crash", err)
	}
	wipe := &lifecycle.Wipe{Stage: lifecycle.Deleting, JoinCode: "ABCD-EFGH-JKLM"}
	if err := env.DataDir.lifecycle(0).Begin(wipe); !errors.Is(err, lifecycle.ErrBusy) {
		t.Fatalf("a second op after the crash = %v, want ErrBusy", err)
	}
	env.Test.BeforeRestoreStage = nil
	assertRestored(t, startTest(t, env), env)
	if err := env.DataDir.lifecycle(0).Begin(wipe); err != nil {
		t.Fatalf("an op once the restore finished = %v", err)
	}
}

func TestAnUnknownRecordFormatExitsSix(t *testing.T) {
	for name, body := range map[string]string{
		"newer format": `{"format": 2, "op": "restore", "restore": {}}`,
		"unknown op":   `{"format": 1, "op": "handover", "handover": {}}`,
	} {
		t.Run(name, func(t *testing.T) {
			env, _ := testEnv(t)
			writeFile(t, filepath.Join(env.DataDir.Path(), lifecycle.FileName), body)
			_, err := Start(t.Context(), env)
			if code := ExitCode(err); code != ExitRecordRefused {
				t.Fatalf("start = %v, exit %d; want exit %d", err, code, ExitRecordRefused)
			}
			if exists(env.DataDir.LibraryDB()) {
				t.Fatal("a refused record still opened library.db")
			}
		})
	}
}

func TestLeftoverRestoreFilesGoWithoutARecord(t *testing.T) {
	env, _ := testEnv(t)
	dir := env.DataDir.Path()
	leftovers := []string{"library.db.restore", "library.db.restore-wal", "library.key.restore", "backup.key.restore", "backups.json.restore"}
	for _, name := range append(leftovers, "notes.restore.txt") {
		writeFile(t, filepath.Join(dir, name), "staged")
	}
	startTest(t, env)
	for _, name := range leftovers {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("stat %s after start = %v, want it deleted", name, err)
		}
	}
	if !exists(filepath.Join(dir, "notes.restore.txt")) {
		t.Fatal("start deleted a file that isn't a staged one")
	}
}

func TestAdminResetRefusesWhileARecordExists(t *testing.T) {
	env, _ := testEnv(t)
	if err := env.DataDir.lifecycle(0).Begin(stagedRestore()); err != nil {
		t.Fatal(err)
	}
	req := identity.ResetRequest{Email: "ana@example.com"}
	_, err := AdminReset(t.Context(), env, req)
	if !errors.Is(err, lifecycle.ErrBusy) || err.Error() != "a restore or an update is already running. Start SlipStream to finish it, then run admin reset again" {
		t.Fatalf("admin reset with a record = %v, want ErrBusy", err)
	}
	if exists(env.DataDir.LibraryDB()) {
		t.Fatal("admin reset opened library.db while a record exists")
	}
	if err := env.DataDir.lifecycle(0).Finish(); err != nil {
		t.Fatal(err)
	}
	if _, err := AdminReset(t.Context(), env, req); errors.Is(err, lifecycle.ErrBusy) {
		t.Fatalf("admin reset with no record = %v", err)
	}
}

// heldAt is a BeforeRestoreStage that holds the start at stage until
// release closes, and closes reached once it gets there.
func heldAt(stage RestoreStage) (hook func(RestoreStep) error, reached <-chan struct{}, release func()) {
	r, done := make(chan struct{}), make(chan struct{})
	var once sync.Once
	return func(s RestoreStep) error {
		if s.Stage == stage {
			once.Do(func() { close(r) })
			<-done
		}
		return nil
	}, r, sync.OnceFunc(func() { close(done) })
}

func TestStatusAnswersBeforeReady(t *testing.T) {
	env, _ := testEnv(t)
	if err := env.DataDir.lifecycle(0).Begin(stagedRestore()); err != nil {
		t.Fatal(err)
	}
	hook, reached, release := heldAt(RestoreAside)
	t.Cleanup(release)
	env.Test.BeforeRestoreStage = hook
	started := make(chan *Node, 1)
	go func() {
		n, err := Start(t.Context(), env)
		if err != nil {
			t.Error(err)
		}
		started <- n
	}()
	<-reached

	status, err := Call(t.Context(), env.Net, env.DataDir, StatusOp, StatusRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if status.Phase != PhaseRestoring || status.Claim != nil || status.URL != "" || status.PID != os.Getpid() || status.Version != "test" {
		t.Fatalf("status while restoring = %+v; want phase restoring, the process, and no claim or URL", status)
	}
	if _, err := Call(t.Context(), env.Net, env.DataDir, RestartOp, RestartRequest{}); err == nil {
		t.Fatal("restart was served before ready")
	}

	release()
	n := <-started
	if n == nil {
		t.FailNow()
	}
	t.Cleanup(func() {
		n.Stop()
		<-n.Done()
	})
	status, err = Call(t.Context(), env.Net, env.DataDir, StatusOp, StatusRequest{})
	if err != nil || status.Phase != PhaseReady || status.Claim == nil || status.URL == "" {
		t.Fatalf("status once ready = %+v, %v; want phase ready with a claim and URL", status, err)
	}
}

// dialCount is a Network that counts the dials a client makes.
type dialCount struct {
	link.Network
	dials atomic.Int32
}

func (d *dialCount) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.dials.Add(1)
	return d.Network.DialContext(ctx, network, address)
}

func TestRestartNodeWaitsForReady(t *testing.T) {
	env, _ := testEnv(t)
	hook, reached, release := heldAt(RestoreAside)
	t.Cleanup(release)
	env.Test.BeforeRestoreStage = hook
	env, _ = restartable(t, env)
	first := startTest(t, env)
	if err := env.DataDir.lifecycle(0).Begin(stagedRestore()); err != nil {
		t.Fatal(err)
	}

	client := env
	counted := &dialCount{Network: env.Net}
	client.Net = counted
	type restarted struct {
		status StatusReply
		err    error
	}
	done := make(chan restarted, 1)
	go func() {
		status, err := RestartNode(t.Context(), client)
		done <- restarted{status, err}
	}()
	select {
	case <-reached:
	case <-time.After(30 * time.Second):
		t.Fatal("the successor didn't reach the restore within 30s")
	}
	asked := counted.dials.Load()
	deadline := time.Now().Add(30 * time.Second)
	for counted.dials.Load() < asked+2 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	select {
	case r := <-done:
		t.Fatalf("RestartNode returned %+v, %v while the successor was restoring", r.status, r.err)
	default:
	}

	release()
	select {
	case r := <-done:
		if r.err != nil || r.status.Phase != PhaseReady || !r.status.StartedAt.After(first.started) {
			t.Fatalf("RestartNode = %+v, %v; want the ready successor", r.status, r.err)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("RestartNode didn't return within 60s of the successor's release")
	}
}
